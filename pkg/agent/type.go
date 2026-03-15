package agent

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors 定义包级别的错误常量
var (
	ErrConfigNil        = errors.New("配置不能为空")
	ErrMasterInit       = errors.New("Master 初始化失败")
	ErrChatModelAdapter = errors.New("创建 ChatModel 适配器失败")
	ErrAgentCreate      = errors.New("创建 Agent 失败")
	ErrMasterCreate     = errors.New("创建 Master 编排失败")
	ErrADKRunnerNil     = errors.New("ADK Runner 未初始化")
	ErrResumeFailed     = errors.New("恢复执行失败")
)

// WrapError 包装错误，添加上下文信息
func WrapError(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// IsInterruptError 检查是否是中断错误
func IsInterruptError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "INTERRUPT:") {
		return true
	}
	if strings.Contains(msg, "interrupt signal:") {
		return true
	}
	if strings.Contains(msg, "interrupt happened") {
		return true
	}
	return false
}
