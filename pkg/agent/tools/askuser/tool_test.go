package askuser

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// mockAskUserCallback 模拟向用户提问回调
func mockAskUserCallback(channel, chatID, question string, options []string) (string, error) {
	if question == "error" {
		return "", errors.New("模拟错误")
	}
	return "用户回答", nil
}

// TestTool_Name 测试工具名称
func TestTool_Name(t *testing.T) {
	tool := NewTool(mockAskUserCallback)
	if tool.Name() != "ask_user" {
		t.Errorf("Name() = %q, 期望 ask_user", tool.Name())
	}
}

// TestTool_Info 测试工具信息
func TestTool_Info(t *testing.T) {
	tool := NewTool(mockAskUserCallback)
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "ask_user" {
		t.Errorf("Info.Name = %q, 期望 ask_user", info.Name)
	}
}

// TestTool_SetContext 测试设置上下文
func TestTool_SetContext(t *testing.T) {
	tool := NewTool(mockAskUserCallback)
	tool.SetContext("websocket", "chat-001")

	if tool.channel != "websocket" {
		t.Errorf("channel = %q, 期望 websocket", tool.channel)
	}

	if tool.chatID != "chat-001" {
		t.Errorf("chatID = %q, 期望 chat-001", tool.chatID)
	}
}

// TestNewTool 测试创建工具
func TestNewTool(t *testing.T) {
	tool := NewTool(mockAskUserCallback)

	if tool == nil {
		t.Fatal("NewTool 返回 nil")
	}

	if tool.callback == nil {
		t.Error("callback 不应该为 nil")
	}
}

// TestAskUserInfo 测试提问信息结构
func TestAskUserInfo(t *testing.T) {
	info := AskUserInfo{
		Question:   "测试问题",
		Options:    []string{"选项1", "选项2"},
		UserAnswer: "用户回答",
	}

	if info.Question != "测试问题" {
		t.Errorf("Question = %q, 期望 测试问题", info.Question)
	}

	if len(info.Options) != 2 {
		t.Errorf("Options 长度 = %d, 期望 2", len(info.Options))
	}

	if info.UserAnswer != "用户回答" {
		t.Errorf("UserAnswer = %q, 期望 用户回答", info.UserAnswer)
	}
}

// TestAskUserState 测试中断状态结构
func TestAskUserState(t *testing.T) {
	state := AskUserState{
		Question: "测试问题",
		Options:  []string{"选项1", "选项2"},
	}

	if state.Question != "测试问题" {
		t.Errorf("Question = %q, 期望 测试问题", state.Question)
	}

	if len(state.Options) != 2 {
		t.Errorf("Options 长度 = %d, 期望 2", len(state.Options))
	}
}

// TestAskUserCallback 测试回调函数类型
func TestAskUserCallback(t *testing.T) {
	var callback AskUserCallback = mockAskUserCallback

	answer, err := callback("websocket", "chat-001", "测试问题", []string{"是", "否"})
	if err != nil {
		t.Errorf("callback 返回错误: %v", err)
	}

	if answer != "用户回答" {
		t.Errorf("answer = %q, 期望 用户回答", answer)
	}
}

// TestGetADKTool 测试获取 ADK 工具
func TestGetADKTool(t *testing.T) {
	tool := NewTool(mockAskUserCallback)

	adkTool := tool.GetADKTool()
	if adkTool == nil {
		t.Error("GetADKTool 不应该返回 nil")
	}
}

// TestSchemaRegistration 测试 schema 注册
func TestSchemaRegistration(t *testing.T) {
	info := &AskUserInfo{
		Question:   "测试问题",
		Options:    []string{"是", "否"},
		UserAnswer: "是",
	}

	if info.Question == "" {
		t.Error("Question 不应该为空")
	}

	state := &AskUserState{
		Question: "测试问题",
		Options:  []string{"是", "否"},
	}

	if state.Question == "" {
		t.Error("Question 不应该为空")
	}
}

// TestTool_NilCallback 测试空回调
func TestTool_NilCallback(t *testing.T) {
	tool := NewTool(nil)

	if tool.callback != nil {
		t.Error("callback 应该为 nil")
	}
}

// TestTool_InfoParams 测试工具参数信息
func TestTool_InfoParams(t *testing.T) {
	tool := NewTool(mockAskUserCallback)
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.ParamsOneOf == nil {
		t.Error("ParamsOneOf 不应该为 nil")
	}
}

// TestAskUserInfo_WithEmptyOptions 测试空选项
func TestAskUserInfo_WithEmptyOptions(t *testing.T) {
	info := AskUserInfo{
		Question:   "测试问题",
		Options:    nil,
		UserAnswer: "用户回答",
	}

	if info.Options != nil {
		t.Errorf("Options 应该为 nil, 得到 %v", info.Options)
	}
}

// TestAskUserState_WithEmptyOptions 测试空选项状态
func TestAskUserState_WithEmptyOptions(t *testing.T) {
	state := AskUserState{
		Question: "测试问题",
		Options:  nil,
	}

	if state.Options != nil {
		t.Errorf("Options 应该为 nil, 得到 %v", state.Options)
	}
}

// TestTool_InvokableRun 测试 InvokableRun 方法
func TestTool_InvokableRun(t *testing.T) {
	tool := NewTool(mockAskUserCallback)
	tool.SetContext("websocket", "chat-001")
	ctx := context.Background()

	_, err := tool.InvokableRun(ctx, `{"question": "测试问题", "options": ["是", "否"]}`)
	if err == nil {
		t.Error("InvokableRun 应该返回中断错误")
	}
}

// TestTool_Interface 测试工具接口实现
func TestTool_Interface(t *testing.T) {
	tool := NewTool(mockAskUserCallback)

	var _ interface {
		Name() string
		Info(ctx context.Context) (*schema.ToolInfo, error)
	} = tool
}
