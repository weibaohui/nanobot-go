package task

import (
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
)

// Status 任务状态
type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusFinished Status = "finished"
	StatusFailed   Status = "failed"
	StatusStopped  Status = "stopped"
)

// Info 任务查询结果
type Info struct {
	ID            string
	Status        Status
	ResultSummary string
	Work          string
	Channel       string
	ChatID        string
	CreatedAt     time.Time
	CompletedAt   time.Time
	LastLogs      []string
}

// ManagerConfig 任务管理器配置
type ManagerConfig struct {
	ConfigLoader          LLMConfigLoader
	Workspace             string
	Tools                 []tool.BaseTool
	Logger                *zap.Logger
	Context               ContextBuilder
	CheckpointStore       compose.CheckPointStore
	MaxIterations         int
	RegisteredTools       []string
	MaxConcurrentTasks    int
	TaskTimeoutSeconds    int
	TaskLogCapacity       int
	TaskMaxToolIterations int
	Sessions              *session.Manager
	OnTaskComplete        func(channel, chatID, taskID string, status Status, result string)
	HookManager           *hooks.HookManager
	EventBus              *bus.MessageBus // 任务事件总线，用于WebSocket推送
}

// ContextBuilder 上下文构建器接口
type ContextBuilder interface{}

// LLMConfigLoader LLM配置加载器
type LLMConfigLoader interface{}

// PersistedTask 持久化的任务结构（用于YAML存储）
type PersistedTask struct {
	ID          string    `yaml:"id"`
	Work        string    `yaml:"work,omitempty"`
	Status      Status    `yaml:"status"`
	Result      string    `yaml:"result,omitempty"`
	Channel     string    `yaml:"channel,omitempty"`
	ChatID      string    `yaml:"chat_id,omitempty"`
	CreatedAt   time.Time `yaml:"created_at"`
	CompletedAt time.Time `yaml:"completed_at,omitempty"`
}

// File YAML文件结构
type File struct {
	Date   string           `yaml:"date"`
	LastID uint32           `yaml:"last_id"`
	Tasks  []*PersistedTask `yaml:"tasks"`
}
