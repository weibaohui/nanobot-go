package config

import (
	"encoding/json"
	"os"
)

// Config 根配置结构
type Config struct {
	Agents    AgentsConfig    `json:"agents"`
	Channels  ChannelsConfig  `json:"channels"`
	Providers ProvidersConfig `json:"providers"`
	Gateway   GatewayConfig   `json:"gateway"`
	Tools     ToolsConfig     `json:"tools"`
	Heartbeat HeartbeatConfig `json:"heartbeat"`
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
			if contains(modelLower, kw) && p.config.APIKey != "" {
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
			if contains(modelLower, prefix) {
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

// TODO 迁移到utils
// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

// TODO 迁移到utils
func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			subc := substr[j]
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
