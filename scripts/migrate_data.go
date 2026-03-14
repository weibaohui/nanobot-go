package main

import (
	"fmt"
	"os"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("用法: go run migrate_data.go <旧数据库路径> <新数据库路径>")
		fmt.Println("示例: go run migrate_data.go data/backup/nanobot_xxx.db data/nanobot.db")
		os.Exit(1)
	}

	oldDBPath := os.Args[1]
	newDBPath := os.Args[2]

	// 打开旧数据库
	oldDB, err := gorm.Open(sqlite.Open(oldDBPath), &gorm.Config{})
	if err != nil {
		fmt.Printf("打开旧数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 打开新数据库
	newDB, err := gorm.Open(sqlite.Open(newDBPath), &gorm.Config{})
	if err != nil {
		fmt.Printf("打开新数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 Code 生成服务
	codeService := service.NewCodeService()

	fmt.Println("开始数据迁移...")

	// 迁移用户
	if err := migrateUsers(oldDB, newDB, codeService); err != nil {
		fmt.Printf("迁移用户失败: %v\n", err)
	}

	// 迁移 Agents
	if err := migrateAgents(oldDB, newDB, codeService); err != nil {
		fmt.Printf("迁移 Agents 失败: %v\n", err)
	}

	// 迁移 Channels
	if err := migrateChannels(oldDB, newDB, codeService); err != nil {
		fmt.Printf("迁移 Channels 失败: %v\n", err)
	}

	// 迁移 Providers
	if err := migrateProviders(oldDB, newDB); err != nil {
		fmt.Printf("迁移 Providers 失败: %v\n", err)
	}

	// 迁移 Sessions
	if err := migrateSessions(oldDB, newDB); err != nil {
		fmt.Printf("迁移 Sessions 失败: %v\n", err)
	}

	// 迁移对话记录
	if err := migrateConversations(oldDB, newDB); err != nil {
		fmt.Printf("迁移对话记录失败: %v\n", err)
	}

	fmt.Println("数据迁移完成！")
}

// 旧模型定义（用于读取旧数据）
type OldUser struct {
	ID           uint
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
	IsActive     bool
}

func migrateUsers(oldDB, newDB *gorm.DB, codeService service.CodeService) error {
	var oldUsers []OldUser
	if err := oldDB.Table("users").Find(&oldUsers).Error; err != nil {
		return err
	}

	fmt.Printf("迁移 %d 个用户...\n", len(oldUsers))

	for _, old := range oldUsers {
		// 生成 UserCode
		userCode, err := codeService.GenerateUserCode()
		if err != nil {
			return err
		}

		newUser := models.User{
			ID:           old.ID,
			UserCode:     userCode,
			Username:     old.Username,
			Email:        old.Email,
			PasswordHash: old.PasswordHash,
			DisplayName:  old.DisplayName,
			IsActive:     old.IsActive,
		}

		if err := newDB.Create(&newUser).Error; err != nil {
			return err
		}
		fmt.Printf("  用户 %s -> Code: %s\n", old.Username, userCode)
	}

	return nil
}

// 旧 Agent 模型
type OldAgent struct {
	ID          uint
	UserID      uint
	Name        string
	Description string
	IsActive    bool
}

func migrateAgents(oldDB, newDB *gorm.DB, codeService service.CodeService) error {
	var oldAgents []OldAgent
	if err := oldDB.Table("agents").Find(&oldAgents).Error; err != nil {
		return err
	}

	fmt.Printf("迁移 %d 个 Agents...\n", len(oldAgents))

	// 获取用户ID到Code的映射
	var users []models.User
	if err := newDB.Find(&users).Error; err != nil {
		return err
	}
	userCodeMap := make(map[uint]string)
	for _, u := range users {
		userCodeMap[u.ID] = u.UserCode
	}

	for _, old := range oldAgents {
		// 生成 AgentCode
		agentCode, err := codeService.GenerateAgentCode()
		if err != nil {
			return err
		}

		// 从旧数据库获取完整数据
		var fullAgent models.Agent
		if err := oldDB.Table("agents").First(&fullAgent, old.ID).Error; err != nil {
			return err
		}

		fullAgent.AgentCode = agentCode
		fullAgent.UserCode = userCodeMap[old.UserID]

		if err := newDB.Create(&fullAgent).Error; err != nil {
			return err
		}
		fmt.Printf("  Agent %s -> Code: %s\n", old.Name, agentCode)
	}

	return nil
}

// 旧 Channel 模型
type OldChannel struct {
	ID      uint
	UserID  uint
	AgentID *uint
	Name    string
	Type    string
}

func migrateChannels(oldDB, newDB *gorm.DB, codeService service.CodeService) error {
	var oldChannels []OldChannel
	if err := oldDB.Table("channels").Find(&oldChannels).Error; err != nil {
		return err
	}

	fmt.Printf("迁移 %d 个 Channels...\n", len(oldChannels))

	// 获取映射
	var users []models.User
	newDB.Find(&users)
	userCodeMap := make(map[uint]string)
	for _, u := range users {
		userCodeMap[u.ID] = u.UserCode
	}

	var agents []models.Agent
	newDB.Find(&agents)
	agentCodeMap := make(map[uint]string)
	for _, a := range agents {
		agentCodeMap[a.ID] = a.AgentCode
	}

	for _, old := range oldChannels {
		// 生成 ChannelCode
		channelCode, err := codeService.GenerateChannelCode()
		if err != nil {
			return err
		}

		// 从旧数据库获取完整数据
		var fullChannel models.Channel
		if err := oldDB.Table("channels").First(&fullChannel, old.ID).Error; err != nil {
			return err
		}

		fullChannel.ChannelCode = channelCode
		fullChannel.UserCode = userCodeMap[old.UserID]
		if old.AgentID != nil {
			fullChannel.AgentCode = agentCodeMap[*old.AgentID]
		}

		if err := newDB.Create(&fullChannel).Error; err != nil {
			return err
		}
		fmt.Printf("  Channel %s -> Code: %s\n", old.Name, channelCode)
	}

	return nil
}

// 旧的 Provider 结构（用于读取旧数据）
type OldProvider struct {
	ID     uint
	UserID uint // 关联的用户ID
}

func migrateProviders(oldDB, newDB *gorm.DB) error {
	var providers []models.LLMProvider
	if err := oldDB.Table("llm_providers").Find(&providers).Error; err != nil {
		return err
	}

	fmt.Printf("迁移 %d 个 Providers...\n", len(providers))

	// 获取用户Code映射
	var users []models.User
	newDB.Find(&users)
	userCodeMap := make(map[uint]string)
	for _, u := range users {
		userCodeMap[u.ID] = u.UserCode
	}

	// 获取旧Provider的UserID映射
	var oldProviders []OldProvider
	if err := oldDB.Table("llm_providers").Select("id, user_id").Find(&oldProviders).Error; err != nil {
		return fmt.Errorf("读取旧Provider数据失败: %w", err)
	}
	providerUserMap := make(map[uint]uint)
	for _, op := range oldProviders {
		providerUserMap[op.ID] = op.UserID
	}

	for _, p := range providers {
		// 使用Provider关联的UserID查找UserCode
		if userID, ok := providerUserMap[p.ID]; ok {
			p.UserCode = userCodeMap[userID]
		}
		if err := newDB.Create(&p).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateSessions(oldDB, newDB *gorm.DB) error {
	// 获取映射
	var users []models.User
	newDB.Find(&users)
	userCodeMap := make(map[uint]string)
	for _, u := range users {
		userCodeMap[u.ID] = u.UserCode
	}

	var channels []models.Channel
	newDB.Find(&channels)
	channelCodeMap := make(map[uint]string)
	for _, c := range channels {
		channelCodeMap[c.ID] = c.ChannelCode
	}

	var agents []models.Agent
	newDB.Find(&agents)
	agentCodeMap := make(map[uint]string)
	for _, a := range agents {
		agentCodeMap[a.ID] = a.AgentCode
	}

	// 旧 Session 模型
	type OldSession struct {
		ID        uint
		UserID    uint
		ChannelID uint
		AgentID   *uint
	}

	var oldSessions []OldSession
	if err := oldDB.Table("sessions").Find(&oldSessions).Error; err != nil {
		return err
	}

	fmt.Printf("迁移 %d 个 Sessions...\n", len(oldSessions))

	for _, old := range oldSessions {
		var fullSession models.Session
		if err := oldDB.Table("sessions").First(&fullSession, old.ID).Error; err != nil {
			continue
		}

		fullSession.UserCode = userCodeMap[old.UserID]
		fullSession.ChannelCode = channelCodeMap[old.ChannelID]
		if old.AgentID != nil {
			fullSession.AgentCode = agentCodeMap[*old.AgentID]
		}

		if err := newDB.Create(&fullSession).Error; err != nil {
			continue
		}
	}

	return nil
}

func migrateConversations(oldDB, newDB *gorm.DB) error {
	// 获取映射
	var users []models.User
	newDB.Find(&users)
	userCodeMap := make(map[uint]string)
	for _, u := range users {
		userCodeMap[u.ID] = u.UserCode
	}

	var channels []models.Channel
	newDB.Find(&channels)
	channelCodeMap := make(map[uint]string)
	for _, c := range channels {
		channelCodeMap[c.ID] = c.ChannelCode
	}

	var agents []models.Agent
	newDB.Find(&agents)
	agentCodeMap := make(map[uint]string)
	for _, a := range agents {
		agentCodeMap[a.ID] = a.AgentCode
	}

	// 旧对话记录模型
	type OldConv struct {
		ID        uint
		UserID    *uint
		ChannelID *uint
		AgentID   *uint
	}

	var oldConvs []OldConv
	if err := oldDB.Table("conversation_records").Find(&oldConvs).Error; err != nil {
		return err
	}

	fmt.Printf("迁移 %d 条对话记录...\n", len(oldConvs))

	for _, old := range oldConvs {
		var fullConv models.ConversationRecord
		if err := oldDB.Table("conversation_records").First(&fullConv, old.ID).Error; err != nil {
			continue
		}

		if old.UserID != nil {
			fullConv.UserCode = userCodeMap[*old.UserID]
		}
		if old.ChannelID != nil {
			fullConv.ChannelCode = channelCodeMap[*old.ChannelID]
		}
		if old.AgentID != nil {
			fullConv.AgentCode = agentCodeMap[*old.AgentID]
		}

		if err := newDB.Create(&fullConv).Error; err != nil {
			continue
		}
	}

	return nil
}
