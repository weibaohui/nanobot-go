package interrupt

import (
	"context"
	"encoding/json"
	"fmt"
)

// Handler 中断处理器接口
type Handler interface {
	Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error)
	Validate(response *UserResponse) error
	FormatQuestion(info *InterruptInfo) string
}

// AskUserHandler 用户提问处理器
type AskUserHandler struct{}

func (h *AskUserHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *AskUserHandler) Validate(response *UserResponse) error {
	if response.Answer == "" {
		return fmt.Errorf("回答不能为空")
	}
	return nil
}

func (h *AskUserHandler) FormatQuestion(info *InterruptInfo) string {
	question := info.Question
	if len(info.Options) > 0 {
		question += "\n\n可选答案:"
		for i, opt := range info.Options {
			question += fmt.Sprintf("\n%d. %s", i+1, opt)
		}
	}
	return question
}

// PlanApprovalHandler 计划审批处理器
type PlanApprovalHandler struct{}

func (h *PlanApprovalHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *PlanApprovalHandler) Validate(response *UserResponse) error {
	return nil
}

func (h *PlanApprovalHandler) FormatQuestion(info *InterruptInfo) string {
	question := info.Question
	if steps, ok := info.Metadata["steps"].([]string); ok {
		question += "\n\n执行步骤:"
		for i, step := range steps {
			question += fmt.Sprintf("\n%d. %s", i+1, step)
		}
	}
	question += "\n\n请回复 '确认' 或 '批准' 继续，或提出修改意见。"
	return question
}

// ToolConfirmHandler 工具确认处理器
type ToolConfirmHandler struct{}

func (h *ToolConfirmHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *ToolConfirmHandler) Validate(response *UserResponse) error {
	return nil
}

func (h *ToolConfirmHandler) FormatQuestion(info *InterruptInfo) string {
	toolName, _ := info.Metadata["tool_name"].(string)
	riskLevel, _ := info.Metadata["risk_level"].(string)
	toolArgs, _ := info.Metadata["tool_args"].(map[string]any)

	argsJSON, _ := json.MarshalIndent(toolArgs, "", "  ")

	return fmt.Sprintf(`⚠️ 需要确认执行工具

工具名称: %s
风险等级: %s
参数:
%s

请回复 '确认' 或 '批准' 继续，或 '取消' 拒绝执行。`, toolName, riskLevel, string(argsJSON))
}

// FileOperationHandler 文件操作处理器
type FileOperationHandler struct{}

func (h *FileOperationHandler) Handle(ctx context.Context, info *InterruptInfo) (*UserResponse, error) {
	return &UserResponse{
		CheckpointID: info.CheckpointID,
	}, nil
}

func (h *FileOperationHandler) Validate(response *UserResponse) error {
	return nil
}

func (h *FileOperationHandler) FormatQuestion(info *InterruptInfo) string {
	operation, _ := info.Metadata["operation"].(string)
	filePath, _ := info.Metadata["file_path"].(string)

	return fmt.Sprintf(`📁 文件操作确认

操作类型: %s
文件路径: %s

请回复 '确认' 继续，或 '取消' 拒绝操作。`, operation, filePath)
}
