package provider

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ContextKey session key 的 context key
type ContextKey string

const SessionKeyContextKey ContextKey = "session_key"

// SkillLoader 技能加载函数类型
type SkillLoader func(name string) string

// HookCallback Hook 回调函数类型
type HookCallback func(eventType events.EventType, data map[string]interface{})

// LLMConfig LLM 配置信息，用于创建 LLM 客户端
type LLMConfig struct {
	APIKey       string            `json:"api_key"`
	APIBase      string            `json:"api_base"`
	DefaultModel string            `json:"default_model"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}

// LLMConfigLoader LLM 配置加载器函数类型
type LLMConfigLoader func(ctx context.Context) (*LLMConfig, error)

// ChatModelAdapter 包装 eino 的 ChatModel，添加工具调用拦截功能
type ChatModelAdapter struct {
	logger        *zap.Logger
	chatModel     model.ToolCallingChatModel
	registeredMap map[string]bool
	skillLoader   SkillLoader
	sessions      *session.Manager
	hookCallback  HookCallback
}

// Sentinel errors
var (
	ErrNilConfig       = fmt.Errorf("配置不能为空")
	ErrCreateChatModel = fmt.Errorf("创建 ChatModel 失败")
	ErrNilAPIKey       = fmt.Errorf("API Key 不能为空")
)

// adapterConfig 用于选项模式的内部配置
type adapterConfig struct {
	db            *gorm.DB
	configLoader  LLMConfigLoader
	logger        *zap.Logger
	sessions      *session.Manager
}

// AdapterOption 创建 ChatModelAdapter 的选项函数类型
type AdapterOption func(*adapterConfig)

// WithDB 使用数据库作为配置来源
func WithDB(db *gorm.DB) AdapterOption {
	return func(c *adapterConfig) {
		c.db = db
	}
}

// WithConfigLoader 使用自定义 ConfigLoader
func WithConfigLoader(loader LLMConfigLoader) AdapterOption {
	return func(c *adapterConfig) {
		c.configLoader = loader
	}
}

// WithLogger 设置 Logger
func WithLogger(logger *zap.Logger) AdapterOption {
	return func(c *adapterConfig) {
		c.logger = logger
	}
}

// WithSessions 设置 Session Manager
func WithSessions(sessions *session.Manager) AdapterOption {
	return func(c *adapterConfig) {
		c.sessions = sessions
	}
}
