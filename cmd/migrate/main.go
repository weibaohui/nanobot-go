package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/weibaohui/nanobot-go/internal/database"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
	"github.com/weibaohui/nanobot-go/internal/service"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// OldConfig 旧版配置结构
type OldConfig struct {
	Agents struct {
		Defaults struct {
			Workspace         string  `json:"workspace"`
			Model             string  `json:"model"`
			MaxTokens         int     `json:"maxTokens"`
			Temperature       float64 `json:"temperature"`
			MaxToolIterations int     `json:"maxToolIterations"`
		} `json:"defaults"`
	} `json:"agents"`
	Channels struct {
		Feishu struct {
			Enabled           bool     `json:"enabled"`
			AppID             string   `json:"appId"`
			AppSecret         string   `json:"appSecret"`
			EncryptKey        string   `json:"encryptKey"`
			VerificationToken string   `json:"verificationToken"`
			AllowFrom         []string `json:"allowFrom"`
		} `json:"feishu"`
	} `json:"channels"`
	Providers struct {
		SiliconFlow struct {
			APIKey       string            `json:"apiKey"`
			APIBase      string            `json:"apiBase"`
			ExtraHeaders map[string]string `json:"extraHeaders"`
		} `json:"siliconflow"`
	} `json:"providers"`
	Database struct {
		Enabled bool   `json:"enabled"`
		DataDir string `json:"dataDir"`
		DBName  string `json:"dbName"`
	} `json:"database"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: migrate <config.json路径>")
		fmt.Println("示例: migrate ~/.nanobot/config.json")
		os.Exit(1)
	}

	configPath := os.Args[1]

	// 1. 读取旧配置
	oldConfig, err := loadOldConfig(configPath)
	if err != nil {
		fmt.Printf("读取配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化数据库（使用配置文件中的数据库设置）
	db, err := initDatabase(oldConfig)
	if err != nil {
		fmt.Printf("初始化数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 3. 执行迁移
	if err := migrate(db, oldConfig); err != nil {
		fmt.Printf("迁移失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ 配置迁移完成!")
}

// loadOldConfig 读取旧版配置文件
func loadOldConfig(path string) (*OldConfig, error) {
	// 展开 ~ 为家目录
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, path[1:])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var cfg OldConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return &cfg, nil
}

// initDatabase 初始化数据库连接
// 优先使用配置文件中的数据库设置，如果没有则使用默认值
func initDatabase(cfg *OldConfig) (*gorm.DB, error) {
	// 获取 workspace 路径
	workspace := cfg.Agents.Defaults.Workspace
	if workspace == "" {
		workspace = "~/.nanobot/workspace"
	}
	// 展开 ~ 为家目录
	if workspace[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		workspace = filepath.Join(home, workspace[1:])
	}

	// 数据库配置（优先使用配置文件中的设置）
	dataDir := cfg.Database.DataDir
	if dataDir == "" {
		dataDir = ".nanobot"
	}
	dbName := cfg.Database.DBName
	if dbName == "" {
		dbName = "nanobot.db"
	}

	// 构建完整的数据目录路径（workspace + dataDir）
	fullDataDir := filepath.Join(workspace, dataDir)

	fmt.Printf("数据库路径: %s/%s\n", fullDataDir, dbName)

	dbCfg := &database.Config{
		DataDir:      fullDataDir,
		DBName:       dbName,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	client, err := database.NewClient(dbCfg)
	if err != nil {
		return nil, err
	}

	// 初始化数据库表结构
	if err := client.InitSchema(); err != nil {
		return nil, err
	}

	return client.DB(), nil
}

// migrate 执行配置迁移
func migrate(db *gorm.DB, oldCfg *OldConfig) error {
	// 初始化 repository 和 service
	userRepo := repository.NewUserRepository(db)
	agentRepo := repository.NewAgentRepository(db)
	channelRepo := repository.NewChannelRepository(db)
	providerService := service.NewProviderService(db)

	// 1. 创建或获取默认用户
	user, err := createOrGetDefaultUser(userRepo)
	if err != nil {
		return fmt.Errorf("创建默认用户失败: %w", err)
	}
	fmt.Printf("✓ 用户已创建/获取: ID=%d, Username=%s\n", user.ID, user.Username)

	// 2. 迁移 LLM Provider 配置
	if err := migrateProviders(providerService, user.ID, oldCfg); err != nil {
		return fmt.Errorf("迁移LLM Provider失败: %w", err)
	}

	// 3. 创建默认 Agent
	agent, err := createDefaultAgent(agentRepo, user.ID, oldCfg)
	if err != nil {
		return fmt.Errorf("创建默认Agent失败: %w", err)
	}
	fmt.Printf("✓ 默认Agent已创建: ID=%d, Name=%s\n", agent.ID, agent.Name)

	// 4. 创建飞书 Channel（如果启用）
	if oldCfg.Channels.Feishu.Enabled {
		channel, err := createFeishuChannel(channelRepo, user.ID, agent.ID, oldCfg)
		if err != nil {
			return fmt.Errorf("创建飞书Channel失败: %w", err)
		}
		fmt.Printf("✓ 飞书Channel已创建: ID=%d, Name=%s\n", channel.ID, channel.Name)
	}

	return nil
}

// createOrGetDefaultUser 创建或获取默认用户
func createOrGetDefaultUser(repo repository.UserRepository) (*models.User, error) {
	// 先尝试查找已有用户
	users, _, err := repo.List(0, 1)
	if err != nil {
		return nil, err
	}
	if len(users) > 0 {
		return &users[0], nil
	}

	// 创建默认用户
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     "admin",
		Email:        "admin@nanobot.local",
		PasswordHash: string(passwordHash),
		DisplayName:  "Administrator",
		IsActive:     true,
	}

	if err := repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// createDefaultAgent 创建默认 Agent
func createDefaultAgent(repo repository.AgentRepository, userID uint, oldCfg *OldConfig) (*models.Agent, error) {
	// 检查是否已有默认 Agent
	existingAgents, err := repo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	for _, agent := range existingAgents {
		if agent.IsDefault {
			fmt.Printf("  ℹ️ 默认Agent已存在，跳过创建\n")
			return &agent, nil
		}
	}

	// 使用旧配置的模型，如果没有则使用默认
	model := oldCfg.Agents.Defaults.Model
	if model == "" {
		model = "anthropic/claude-opus-4-5"
	}

	// 转换 maxIterations -> maxToolIterations (旧配置有该字段)
	maxIterations := oldCfg.Agents.Defaults.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 20
	}

	maxTokens := oldCfg.Agents.Defaults.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	temperature := oldCfg.Agents.Defaults.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// 构建 Agent 描述（包含提供商信息）
	description := "从旧配置迁移的默认Agent"
	if oldCfg.Providers.SiliconFlow.APIKey != "" {
		description = fmt.Sprintf("%s\n提供商: SiliconFlow\nAPI Base: %s",
			description,
			oldCfg.Providers.SiliconFlow.APIBase)
	}

	// 序列化空列表
	skillsJSON, _ := json.Marshal([]string{})
	toolsJSON, _ := json.Marshal([]string{})

	agent := &models.Agent{
		UserID:        userID,
		Name:          "默认Agent",
		Description:   description,
		Model:         model,
		MaxTokens:     maxTokens,
		Temperature:   temperature,
		MaxIterations: maxIterations,
		SkillsList:    string(skillsJSON),
		ToolsList:     string(toolsJSON),
		IsActive:      true,
		IsDefault:     true,
	}

	if err := repo.Create(agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// createFeishuChannel 创建飞书 Channel
func createFeishuChannel(repo repository.ChannelRepository, userID uint, agentID uint, oldCfg *OldConfig) (*models.Channel, error) {
	// 检查是否已有同名 Channel
	existingChannels, err := repo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	for _, ch := range existingChannels {
		if ch.Type == models.ChannelTypeFeishu {
			fmt.Printf("  ℹ️ 飞书Channel已存在，跳过创建\n")
			return &ch, nil
		}
	}

	feishuCfg := oldCfg.Channels.Feishu

	// 构建配置 JSON
	configMap := map[string]interface{}{
		"app_id":             feishuCfg.AppID,
		"app_secret":         feishuCfg.AppSecret,
		"encrypt_key":        feishuCfg.EncryptKey,
		"verification_token": feishuCfg.VerificationToken,
	}

	configJSON, err := json.Marshal(configMap)
	if err != nil {
		return nil, err
	}

	allowFromJSON, err := json.Marshal(feishuCfg.AllowFrom)
	if err != nil {
		return nil, err
	}

	agentIDPtr := &agentID
	channel := &models.Channel{
		UserID:    userID,
		AgentID:   agentIDPtr,
		Name:      "飞书",
		Type:      models.ChannelTypeFeishu,
		IsActive:  true,
		Config:    string(configJSON),
		AllowFrom: string(allowFromJSON),
	}

	if err := repo.Create(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// migrateProviders 迁移 LLM Provider 配置
func migrateProviders(providerService service.ProviderService, userID uint, oldCfg *OldConfig) error {
	ctx := context.Background()

	// 检查是否已有 Provider
	existingProviders, _, err := providerService.List(ctx, userID, 0, 10)
	if err != nil {
		return fmt.Errorf("查询现有Provider失败: %w", err)
	}
	if len(existingProviders) > 0 {
		fmt.Printf("  ℹ️ 已有 %d 个Provider存在，跳过迁移\n", len(existingProviders))
		return nil
	}

	// 迁移 SiliconFlow Provider
	if oldCfg.Providers.SiliconFlow.APIKey != "" {
		// 序列化 extra headers
		var extraHeaders string
		if oldCfg.Providers.SiliconFlow.ExtraHeaders != nil {
			headersJSON, err := json.Marshal(oldCfg.Providers.SiliconFlow.ExtraHeaders)
			if err != nil {
				return fmt.Errorf("序列化extra headers失败: %w", err)
			}
			extraHeaders = string(headersJSON)
		}

		// 使用默认模型或配置中的模型
		defaultModel := oldCfg.Agents.Defaults.Model
		if defaultModel == "" {
			defaultModel = "MiniMax-M2.5"
		}

		req := service.CreateProviderRequest{
			ProviderKey:  "siliconflow",
			ProviderName: "SiliconFlow",
			APIKey:       oldCfg.Providers.SiliconFlow.APIKey,
			APIBase:      oldCfg.Providers.SiliconFlow.APIBase,
			ExtraHeaders: extraHeaders,
			DefaultModel: defaultModel,
			IsDefault:    true,
			Priority:     10,
		}

		provider, err := providerService.Create(ctx, userID, req)
		if err != nil {
			return fmt.Errorf("创建SiliconFlow Provider失败: %w", err)
		}
		fmt.Printf("✓ LLM Provider已创建: ID=%d, Name=%s, Model=%s\n",
			provider.ID, provider.ProviderName, defaultModel)
	} else {
		fmt.Printf("  ℹ️ 配置文件中没有找到SiliconFlow API Key，跳过Provider迁移\n")
	}

	return nil
}
