package config

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
// 使用系统整体配置的 API Key 和 Model
type SummarizationConfig struct {
	Model              string  `json:"model"`              // 模型名称（为空则使用系统默认模型）
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
	Compress        CompressConfig        `json:"compress"`
	ThinkingProcess ThinkingProcessConfig `json:"thinkingProcess"` // 思考过程配置
	Database        DatabaseConfig        `json:"database"`        // 数据库配置
	Memory          MemoryConfig          `json:"memory"`          // 记忆模块配置
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
	Feishu FeishuConfig `json:"feishu"`
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

		Compress: CompressConfig{
			Enabled:     false,
			MinMessages: 20,
			MinTokens:   50000,
			Model:       "",
			MaxHistory:  5,
		},
		Database: DatabaseConfig{
			Enabled:      true,
			DataDir:      "", // 空字符串表示使用固定路径 (程序目录/data)
			DBName:       "nanobot.db",
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
