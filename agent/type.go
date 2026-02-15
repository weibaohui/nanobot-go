package agent

import (
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
)

// Sentinel errors 定义包级别的错误常量
var (
	ErrConfigNil        = errors.New("配置不能为空")
	ErrSubAgentCreate   = errors.New("创建子 Agent 失败")
	ErrSupervisorInit   = errors.New("Supervisor 初始化失败")
	ErrMasterInit       = errors.New("Master 初始化失败")
	ErrChatModelAdapter = errors.New("创建 ChatModel 适配器失败")
	ErrPlannerCreate    = errors.New("创建 Planner 失败")
	ErrExecutorCreate   = errors.New("创建 Executor 失败")
	ErrReplannerCreate  = errors.New("创建 Replanner 失败")
	ErrAgentCreate      = errors.New("创建 Agent 失败")
	ErrSupervisorCreate = errors.New("创建 Supervisor 编排失败")
	ErrMasterCreate     = errors.New("创建 Master 编排失败")
	ErrADKRunnerNil     = errors.New("ADK Runner 未初始化")
	ErrResumeFailed     = errors.New("恢复执行失败")
)

// AgentType 定义 Agent 类型
type AgentType string

const (
	AgentTypeReAct AgentType = "react_agent"
	AgentTypePlan  AgentType = "plan_agent"
	AgentTypeChat  AgentType = "chat_agent"
)

// SubAgent 子 Agent 接口
type SubAgent interface {
	Name() string
	Description() string
	Type() AgentType
	GetADKAgent() adk.Agent
}

// WrapError 包装错误，添加上下文信息
func WrapError(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
