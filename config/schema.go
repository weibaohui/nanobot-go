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
	WhatsApp  WhatsAppConfig  `json:"whatsapp"`
	Telegram  TelegramConfig  `json:"telegram"`
	Discord   DiscordConfig   `json:"discord"`
	Feishu    FeishuConfig    `json:"feishu"`
	Mochat    MochatConfig    `json:"mochat"`
	DingTalk  DingTalkConfig  `json:"dingtalk"`
	Email     EmailConfig     `json:"email"`
	Slack     SlackConfig     `json:"slack"`
	QQ        QQConfig        `json:"qq"`
	Matrix    MatrixConfig    `json:"matrix"`
}

// WebSocketConfig WebSocket 渠道配置
type WebSocketConfig struct {
	Enabled   bool     `json:"enabled"`
	Addr      string   `json:"addr"`      // 监听地址，如 ":8088"
	Path      string   `json:"path"`      // WebSocket 路径，如 "/ws"
	AllowFrom []string `json:"allowFrom"` // 允许的用户 ID 列表
}

// WhatsAppConfig WhatsApp 渠道配置
type WhatsAppConfig struct {
	Enabled   bool     `json:"enabled"`
	BridgeURL string   `json:"bridgeUrl"`
	AllowFrom []string `json:"allowFrom"`
}

// TelegramConfig Telegram 渠道配置
type TelegramConfig struct {
	Enabled   bool     `json:"enabled"`
	Token     string   `json:"token"`
	AllowFrom []string `json:"allowFrom"`
	Proxy     string   `json:"proxy"`
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

// DiscordConfig Discord 渠道配置
type DiscordConfig struct {
	Enabled    bool     `json:"enabled"`
	Token      string   `json:"token"`
	AllowFrom  []string `json:"allowFrom"`
	GatewayURL string   `json:"gatewayUrl"`
	Intents    int      `json:"intents"`
}

// EmailConfig 邮件渠道配置
type EmailConfig struct {
	Enabled             bool     `json:"enabled"`
	ConsentGranted      bool     `json:"consentGranted"`
	IMAPHost            string   `json:"imapHost"`
	IMAPPort            int      `json:"imapPort"`
	IMAPUsername        string   `json:"imapUsername"`
	IMAPPassword        string   `json:"imapPassword"`
	IMAPMailbox         string   `json:"imapMailbox"`
	IMAPUseSSL          bool     `json:"imapUseSSL"`
	SMTPHost            string   `json:"smtpHost"`
	SMTPPort            int      `json:"smtpPort"`
	SMTPUsername        string   `json:"smtpUsername"`
	SMTPPassword        string   `json:"smtpPassword"`
	SMTPUseTLS          bool     `json:"smtpUseTls"`
	SMTPUseSSL          bool     `json:"smtpUseSsl"`
	FromAddress         string   `json:"fromAddress"`
	AutoReplyEnabled    bool     `json:"autoReplyEnabled"`
	PollIntervalSeconds int      `json:"pollIntervalSeconds"`
	MarkSeen            bool     `json:"markSeen"`
	MaxBodyChars        int      `json:"maxBodyChars"`
	SubjectPrefix       string   `json:"subjectPrefix"`
	AllowFrom           []string `json:"allowFrom"`
}

// MochatMentionConfig Mochat 提及行为配置
type MochatMentionConfig struct {
	RequireInGroups bool `json:"requireInGroups"`
}

// MochatGroupRule Mochat 群组提及规则
type MochatGroupRule struct {
	RequireMention bool `json:"requireMention"`
}

// MochatConfig Mochat 渠道配置
type MochatConfig struct {
	Enabled                   bool                       `json:"enabled"`
	BaseURL                   string                     `json:"baseUrl"`
	SocketURL                 string                     `json:"socketUrl"`
	SocketPath                string                     `json:"socketPath"`
	SocketDisableMsgpack      bool                       `json:"socketDisableMsgpack"`
	SocketReconnectDelayMs    int                        `json:"socketReconnectDelayMs"`
	SocketMaxReconnectDelayMs int                        `json:"socketMaxReconnectDelayMs"`
	SocketConnectTimeoutMs    int                        `json:"socketConnectTimeoutMs"`
	RefreshIntervalMs         int                        `json:"refreshIntervalMs"`
	WatchTimeoutMs            int                        `json:"watchTimeoutMs"`
	WatchLimit                int                        `json:"watchLimit"`
	RetryDelayMs              int                        `json:"retryDelayMs"`
	MaxRetryAttempts          int                        `json:"maxRetryAttempts"`
	ClawToken                 string                     `json:"clawToken"`
	AgentUserID               string                     `json:"agentUserId"`
	Sessions                  []string                   `json:"sessions"`
	Panels                    []string                   `json:"panels"`
	AllowFrom                 []string                   `json:"allowFrom"`
	Mention                   MochatMentionConfig        `json:"mention"`
	Groups                    map[string]MochatGroupRule `json:"groups"`
	ReplyDelayMode            string                     `json:"replyDelayMode"`
	ReplyDelayMs              int                        `json:"replyDelayMs"`
}

// SlackDMConfig Slack DM 策略配置
type SlackDMConfig struct {
	Enabled   bool     `json:"enabled"`
	Policy    string   `json:"policy"`
	AllowFrom []string `json:"allowFrom"`
}

// SlackConfig Slack 渠道配置
type SlackConfig struct {
	Enabled           bool          `json:"enabled"`
	Mode              string        `json:"mode"`
	WebhookPath       string        `json:"webhookPath"`
	BotToken          string        `json:"botToken"`
	AppToken          string        `json:"appToken"`
	UserTokenReadOnly bool          `json:"userTokenReadOnly"`
	GroupPolicy       string        `json:"groupPolicy"`
	GroupAllowFrom    []string      `json:"groupAllowFrom"`
	DM                SlackDMConfig `json:"dm"`
}

// QQConfig QQ 渠道配置
type QQConfig struct {
	Enabled   bool     `json:"enabled"`
	AppID     string   `json:"appId"`
	Secret    string   `json:"secret"`
	AllowFrom []string `json:"allowFrom"`
}

// MatrixConfig Matrix 渠道配置
type MatrixConfig struct {
	Enabled    bool     `json:"enabled"`
	Homeserver string   `json:"homeserver"` // Matrix 服务器地址，如 https://matrix.example.com
	UserID     string   `json:"userId"`     // 用户 ID，如 @nanobot:example.com
	Token      string   `json:"token"`      // 访问令牌
	AllowFrom  []string `json:"allowFrom"`  // 允许的用户白名单
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
	MaxResults int    `json:"maxResults"`
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
			WhatsApp: WhatsAppConfig{
				BridgeURL: "ws://localhost:3001",
			},
			Discord: DiscordConfig{
				GatewayURL: "wss://gateway.discord.gg/?v=10&encoding=json",
				Intents:    37377,
			},
			Slack: SlackConfig{
				Mode:        "socket",
				WebhookPath: "/slack/events",
				GroupPolicy: "mention",
				DM: SlackDMConfig{
					Enabled: true,
					Policy:  "open",
				},
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
		// 网关优先（支持多模型）
		{"openrouter", []string{"openrouter"}, c.Providers.OpenRouter},
		{"aihubmix", []string{"aihubmix"}, c.Providers.AiHubMix},
		{"siliconflow", []string{"siliconflow"}, c.Providers.SiliconFlow},
		// 专属模型提供商
		{"anthropic", []string{"anthropic", "claude"}, c.Providers.Anthropic},
		{"openai", []string{"openai", "gpt"}, c.Providers.OpenAI},
		{"deepseek", []string{"deepseek"}, c.Providers.DeepSeek},
		{"gemini", []string{"gemini"}, c.Providers.Gemini},
		{"zhipu", []string{"zhipu", "glm", "zai"}, c.Providers.Zhipu},
		{"dashscope", []string{"dashscope"}, c.Providers.DashScope},
		{"moonshot", []string{"moonshot", "kimi"}, c.Providers.Moonshot},
		{"minimax", []string{"minimax"}, c.Providers.MiniMax},
		{"vllm", []string{"vllm"}, c.Providers.VLLM},
		{"groq", []string{"groq"}, c.Providers.Groq},
	}

	for _, p := range providers {
		for _, kw := range p.keywords {
			if contains(modelLower, kw) && p.config.APIKey != "" {
				return &p.config
			}
		}
	}

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
		c.Providers.OpenRouter,
		c.Providers.DeepSeek,
		c.Providers.Anthropic,
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

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

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
