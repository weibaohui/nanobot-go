package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/weibaohui/nanobot-go/utils"
)

// CompressConfig 对话压缩配置
type CompressConfig struct {
	Enabled     bool   `json:"enabled"`     // 是否启用压缩功能
	MinMessages int    `json:"minMessages"` // 最小消息数量阈值（默认20）
	MinTokens   int    `json:"minTokens"`   // 最小 Token 用量阈值（默认50000）
	Model       string `json:"model"`       // 压缩使用的模型（默认使用默认模型）
	MaxHistory  int    `json:"maxHistory"`  // 压缩后保留的最大历史消息数（默认5）
}

// ThinkingProcessConfig 思考过程配置
// 用于控制是否将 AI 的思考过程（工具调用、LLM 响应等）实时发送到 channel
type ThinkingProcessConfig struct {
	Enabled bool     `json:"enabled"` // 是否启用思考过程推送
	Events  []string `json:"events"`  // 要监听的事件类型，如 ["tool_used", "tool_completed", "llm_call_end"]
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Enabled      bool   `json:"enabled"`      // 是否启用数据库
	DataDir      string `json:"dataDir"`      // 数据目录，相对于 workspace
	DBName       string `json:"dbName"`       // 数据库文件名
	MaxOpenConns int    `json:"maxOpenConns"` // 最大打开连接数
	MaxIdleConns int    `json:"maxIdleConns"` // 最大空闲连接数
}

// MemoryConfig 记忆模块配置
type MemoryConfig struct {
	Enabled       bool                  `json:"enabled"`       // 是否启用记忆模块
	Summarization SummarizationConfig   `json:"summarization"` // 记忆归纳模型配置
	Embedding     EmbeddingConfig       `json:"embedding"`     // 向量化模型配置
	Scheduled     MemoryScheduledConfig `json:"scheduled"`     // 定时任务配置
	Storage       MemoryStorageConfig   `json:"storage"`       // 存储配置
}

// SummarizationConfig 记忆归纳模型配置
type SummarizationConfig struct {
	Model              string  `json:"model"`              // 模型名称
	APIKey             string  `json:"apiKey"`             // API Key
	BaseURL            string  `json:"baseURL"`            // API Base URL
	Temperature        float64 `json:"temperature"`        // 温度参数
	MaxTokens          int     `json:"maxTokens"`          // 最大 Token 数
	ConversationPrompt string  `json:"conversationPrompt"` // 对话总结提示词
	LongTermPrompt     string  `json:"longTermPrompt"`     // 长期记忆提炼提示词
}

// EmbeddingConfig 向量化模型配置
type EmbeddingConfig struct {
	Enabled    bool   `json:"enabled"`    // 是否启用向量化
	Model      string `json:"model"`      // 模型名称
	APIKey     string `json:"apiKey"`     // API Key
	BaseURL    string `json:"baseURL"`    // API Base URL
	Dimensions int    `json:"dimensions"` // 向量维度
}

// MemoryScheduledConfig 记忆定时任务配置
type MemoryScheduledConfig struct {
	Enabled    bool   `json:"enabled"`    // 是否启用定时任务
	TimeWindow string `json:"timeWindow"` // 时间窗口，如 "05:30-06:30"
	Timezone   string `json:"timezone"`   // 时区
	BatchSize  int    `json:"batchSize"`  // 每批处理数量
}

// MemoryStorageConfig 记忆存储配置
type MemoryStorageConfig struct {
	StreamVectorization bool `json:"streamVectorization"` // 流水记忆是否向量化
}

// Config 根配置结构
type Config struct {
	Agents          AgentsConfig          `json:"agents"`
	Channels        ChannelsConfig        `json:"channels"`
	Providers       ProvidersConfig       `json:"providers"`
	Gateway         GatewayConfig         `json:"gateway"`
	Tools           ToolsConfig           `json:"tools"`
	Heartbeat       HeartbeatConfig       `json:"heartbeat"`
	Compress        CompressConfig        `json:"compress"`
	ThinkingProcess ThinkingProcessConfig `json:"thinkingProcess"` // 思考过程配置
	Database        DatabaseConfig        `json:"database"`        // 数据库配置
	Memory          MemoryConfig          `json:"memory"`          // 记忆模块配置
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	Every       string      `json:"every,omitempty"`       // 心跳间隔，支持 "30m"/"1h" 或 cron 表达式
	ActiveHours ActiveHours `json:"activeHours,omitempty"` // 活跃时段配置
	Model       string      `json:"model,omitempty"`       // 心跳专用模型
	Session     string      `json:"session,omitempty"`     // 心跳会话键
	Target      string      `json:"target,omitempty"`      // 心跳目标: "last"/"none" 或 ChannelId
	Prompt      string      `json:"prompt,omitempty"`      // 心跳提示词
	AckMaxChars int         `json:"ackMaxChars,omitempty"` // 确认消息最大字符数
}

// ActiveHours 活跃时段配置
type ActiveHours struct {
	Start    string `json:"start,omitempty"`    // 活跃开始时间，如 "09:00"
	End      string `json:"end,omitempty"`      // 活跃结束时间，如 "18:00"
	Timezone string `json:"timezone,omitempty"` // 时区，如 "Asia/Shanghai"
}

// AgentsConfig 代理配置
type AgentsConfig struct {
	Defaults      AgentDefaults `json:"defaults"`
	MaxIterations int           `json:"maxIterations"`
}

// AgentDefaults 默认代理配置
type AgentDefaults struct {
	Workspace         string  `json:"workspace"`
	Model             string  `json:"model"`
	MaxTokens         int     `json:"maxTokens"`
	Temperature       float64 `json:"temperature"`
	MaxToolIterations int     `json:"maxToolIterations"`
}

// ChannelsConfig 渠道配置
type ChannelsConfig struct {
	WebSocket WebSocketConfig `json:"websocket"`
	Feishu    FeishuConfig    `json:"feishu"`
	DingTalk  DingTalkConfig  `json:"dingtalk"`
	Matrix    MatrixConfig    `json:"matrix"`
}

// WebSocketConfig WebSocket 渠道配置
type WebSocketConfig struct {
	Enabled   bool     `json:"enabled"`
	Addr      string   `json:"addr"`      // 监听地址，如 ":8088"
	Path      string   `json:"path"`      // WebSocket 路径，如 "/ws"
	AllowFrom []string `json:"allowFrom"` // 允许的用户 ID 列表
}

// FeishuConfig 飞书渠道配置
type FeishuConfig struct {
	Enabled           bool     `json:"enabled"`
	AppID             string   `json:"appId"`
	AppSecret         string   `json:"appSecret"`
	EncryptKey        string   `json:"encryptKey"`
	VerificationToken string   `json:"verificationToken"`
	AllowFrom         []string `json:"allowFrom"`
}

// DingTalkConfig 钉钉渠道配置
type DingTalkConfig struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	AllowFrom    []string `json:"allowFrom"`
}

// MatrixConfig Matrix 渠道配置
type MatrixConfig struct {
	Enabled    bool     `json:"enabled"`
	Homeserver string   `json:"homeserver"` // Matrix 服务器地址，如 https://matrix.example.com
	UserID     string   `json:"userId"`     // 用户 ID，如 @nanobot:example.com
	Token      string   `json:"token"`      // 访问令牌
	AllowFrom  []string `json:"allowFrom"`  // 允许的用户白名单
	DataDir    string   `json:"dataDir"`    // 数据存储目录，用于持久化同步状态
}

// ProvidersConfig LLM 提供商配置
type ProvidersConfig struct {
	Anthropic   ProviderConfig `json:"anthropic"`
	OpenAI      ProviderConfig `json:"openai"`
	OpenRouter  ProviderConfig `json:"openrouter"`
	DeepSeek    ProviderConfig `json:"deepseek"`
	Groq        ProviderConfig `json:"groq"`
	Zhipu       ProviderConfig `json:"zhipu"`
	DashScope   ProviderConfig `json:"dashscope"`
	VLLM        ProviderConfig `json:"vllm"`
	Gemini      ProviderConfig `json:"gemini"`
	Moonshot    ProviderConfig `json:"moonshot"`
	MiniMax     ProviderConfig `json:"minimax"`
	AiHubMix    ProviderConfig `json:"aihubmix"`
	SiliconFlow ProviderConfig `json:"siliconflow"`
}

// ProviderConfig LLM 提供商配置
type ProviderConfig struct {
	APIKey       string            `json:"apiKey"`
	APIBase      string            `json:"apiBase"`
	ExtraHeaders map[string]string `json:"extraHeaders"`
}

// GatewayConfig 网关配置
type GatewayConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// WebSearchConfig 网络搜索工具配置
type WebSearchConfig struct {
	MaxResults int `json:"maxResults"`
}

// WebToolsConfig Web 工具配置
type WebToolsConfig struct {
	Search WebSearchConfig `json:"search"`
}

// ExecToolConfig Shell 执行工具配置
type ExecToolConfig struct {
	Timeout int `json:"timeout"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	Web                 WebToolsConfig `json:"web"`
	Exec                ExecToolConfig `json:"exec"`
	RestrictToWorkspace bool           `json:"restrictToWorkspace"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:         "~/.nanobot/workspace",
				Model:             "anthropic/claude-opus-4-5",
				MaxTokens:         8192,
				Temperature:       0.7,
				MaxToolIterations: 20,
			},
		},
		ThinkingProcess: ThinkingProcessConfig{
			Enabled: true,
			Events:  []string{"tool_used", "tool_completed"},
		},
		Channels: ChannelsConfig{
			Matrix: MatrixConfig{
				Homeserver: "https://matrix.example.com",
				UserID:     "@nanobot:example.com",
				Token:      "your-access-token",
			},
			DingTalk: DingTalkConfig{
				Enabled:      true,
				ClientID:     "your-client-id",
				ClientSecret: "your-client-secret",
			},
		},
		Gateway: GatewayConfig{
			Host: "0.0.0.0",
			Port: 18790,
		},
		Tools: ToolsConfig{
			Web: WebToolsConfig{
				Search: WebSearchConfig{
					MaxResults: 5,
				},
			},
			Exec: ExecToolConfig{
				Timeout: 60,
			},
		},
		Heartbeat: HeartbeatConfig{
			Every:       "30m",
			ActiveHours: ActiveHours{Start: "09:00", End: "18:00"},
		},
		Compress: CompressConfig{
			Enabled:     false,
			MinMessages: 20,
			MinTokens:   50000,
			Model:       "",
			MaxHistory:  5,
		},
		Database: DatabaseConfig{
			Enabled:      true,
			DataDir:      ".data",
			DBName:       "events.db",
			MaxOpenConns: 1, // SQLite 建议单连接
			MaxIdleConns: 1,
		},
		Memory: MemoryConfig{
			Enabled: false, // 默认关闭，需要手动启用
			Summarization: SummarizationConfig{
				Model:              "",
				Temperature:        0.3,
				MaxTokens:          2048,
				ConversationPrompt: "",
				LongTermPrompt:     "",
			},
			Embedding: EmbeddingConfig{
				Enabled:    false,
				Dimensions: 1024,
			},
			Scheduled: MemoryScheduledConfig{
				Enabled:    true,
				TimeWindow: "05:30-06:30",
				Timezone:   "Asia/Shanghai",
				BatchSize:  100,
			},
			Storage: MemoryStorageConfig{
				StreamVectorization: false,
			},
		},
	}
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = GetConfigPath()
	}

	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig 保存配置文件
func SaveConfig(config *Config, path string) error {
	if path == "" {
		path = GetConfigPath()
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetWorkspacePath 获取工作区路径
func (c *Config) GetWorkspacePath() string {
	return GetWorkspacePath(c.Agents.Defaults.Workspace)
}

// GetProvider 获取匹配的提供商配置
func (c *Config) GetProvider(model string) *ProviderConfig {
	modelLower := model
	if modelLower == "" {
		modelLower = c.Agents.Defaults.Model
	}

	// 按关键词匹配（优先级从高到低）
	providers := []struct {
		name     string
		keywords []string
		config   ProviderConfig
	}{

		{"siliconflow", []string{"siliconflow"}, c.Providers.SiliconFlow},
		{"openai", []string{"openai", "gpt"}, c.Providers.OpenAI},
	}

	for _, p := range providers {
		for _, kw := range p.keywords {
			if utils.ContainsInsensitive(modelLower, kw) && p.config.APIKey != "" {
				return &p.config
			}
		}
	}

	// TODO 配置中改为 可配置多个模型，存在配置的进行匹配
	// 改成 多个列表配置，意思为可以从这些模型中任选一个
	// 现在这个模式配置，要从模型名称进行识别，不友好
	// 特殊处理：模型格式为 "Org/Model" 时，检查常见云平台
	// SiliconFlow 支持多种开源模型如 Qwen/, deepseek-ai/, meta-llama/ 等
	if c.Providers.SiliconFlow.APIKey != "" {
		siliconflowPrefixes := []string{"Qwen/", "deepseek-ai/", "meta-llama/", "THUDM/", "mistralai/", "google/"}
		for _, prefix := range siliconflowPrefixes {
			if utils.ContainsInsensitive(modelLower, prefix) {
				return &c.Providers.SiliconFlow
			}
		}
	}

	// 回退：按优先级检查有 API key 的提供商
	fallbackOrder := []ProviderConfig{
		c.Providers.SiliconFlow,
		c.Providers.OpenAI,
	}
	for _, cfg := range fallbackOrder {
		if cfg.APIKey != "" {
			return &cfg
		}
	}

	return nil
}

// GetAPIKey 获取指定模型的 API key
func (c *Config) GetAPIKey(model string) string {
	p := c.GetProvider(model)
	if p != nil {
		return p.APIKey
	}
	return ""
}

// GetAPIBase 获取指定模型的 API base URL
func (c *Config) GetAPIBase(model string) string {
	p := c.GetProvider(model)
	if p != nil {
		return p.APIBase
	}
	return ""
}

// GetDatabaseDataDir 获取数据库数据目录的完整路径
// 数据目录位于 workspace 下的 Database.DataDir 子目录
func (c *Config) GetDatabaseDataDir() string {
	workspacePath := c.GetWorkspacePath()
	return filepath.Join(workspacePath, c.Database.DataDir)
}
