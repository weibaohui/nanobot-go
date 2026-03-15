package askuser

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// AskUserInfo 向用户提问的信息
type AskUserInfo struct {
	Question   string   `json:"question"`
	Options    []string `json:"options,omitempty"`
	UserAnswer string   `json:"user_answer,omitempty"`
}

// AskUserState 中断时保存的状态
type AskUserState struct {
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
}

// AskUserCallback 回调函数类型
type AskUserCallback func(channel, chatID, question string, options []string) (string, error)

func init() {
	schema.Register[*AskUserInfo]()
	schema.Register[*AskUserState]()
}

// Tool 向用户提问的工具
type Tool struct {
	callback AskUserCallback
	channel  string
	chatID   string
}

// NewTool 创建向用户提问的工具
func NewTool(callback AskUserCallback) *Tool {
	return &Tool{
		callback: callback,
	}
}

// SetContext 设置上下文
func (t *Tool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

// Name 返回工具名称
func (t *Tool) Name() string {
	return "ask_user"
}

// Info 返回工具信息
func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ask_user",
		Desc: "向用户提问以获取更多信息。当需要用户输入或选择时使用此工具。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"question": {
				Type:     schema.DataType("string"),
				Desc:     "要向用户提出的问题",
				Required: true,
			},
			"options": {
				Type:     schema.DataType("array"),
				Desc:     "可选的选项列表（如 ['yes', 'no']）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具
func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return askUser(ctx, argumentsInJSON, t.callback, t.channel, t.chatID)
}

// askUser 实际的提问逻辑
func askUser(ctx context.Context, argumentsInJSON string, callback AskUserCallback, channel, chatID string) (string, error) {
	// 解析输入
	var input struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	// 检查是否已被中断
	wasInterrupted, _, storedState := tool.GetInterruptState[*AskUserState](ctx)

	if !wasInterrupted {
		// 第一次调用，触发中断
		info := &AskUserInfo{
			Question: input.Question,
			Options:  input.Options,
		}
		state := &AskUserState{
			Question: input.Question,
			Options:  input.Options,
		}
		return "", tool.StatefulInterrupt(ctx, info, state)
	}

	// 检查是否是恢复目标
	isResumeTarget, hasData, resumeData := tool.GetResumeContext[*AskUserInfo](ctx)

	if !isResumeTarget {
		// 不是恢复目标，再次中断
		info := &AskUserInfo{
			Question: storedState.Question,
			Options:  storedState.Options,
		}
		return "", tool.StatefulInterrupt(ctx, info, storedState)
	}

	if !hasData || resumeData.UserAnswer == "" {
		return "", fmt.Errorf("工具恢复但没有用户回答")
	}

	// 返回用户的回答
	return resumeData.UserAnswer, nil
}

// GetADKTool 获取 ADK 工具实例
func (t *Tool) GetADKTool() tool.InvokableTool {
	invokable, err := utils.InferTool(
		"ask_user",
		"向用户提问以获取更多信息。当需要用户输入或选择时使用此工具。",
		func(ctx context.Context, input *struct {
			Question string   `json:"question"`
			Options  []string `json:"options"`
		}) (string, error) {
			argsJSON, _ := json.Marshal(input)
			return askUser(ctx, string(argsJSON), t.callback, t.channel, t.chatID)
		},
	)
	if err != nil {
		return nil
	}
	return invokable
}
