package agent

import (
	"errors"
	"testing"

	"github.com/cloudwego/eino/adk"
)

// TestSentinelErrors 测试哨兵错误定义
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrConfigNil", ErrConfigNil, "配置不能为空"},
		{"ErrSubAgentCreate", ErrSubAgentCreate, "创建子 Agent 失败"},
		{"ErrSupervisorInit", ErrSupervisorInit, "Supervisor 初始化失败"},
		{"ErrMasterInit", ErrMasterInit, "Master 初始化失败"},
		{"ErrChatModelAdapter", ErrChatModelAdapter, "创建 ChatModel 适配器失败"},
		{"ErrPlannerCreate", ErrPlannerCreate, "创建 Planner 失败"},
		{"ErrExecutorCreate", ErrExecutorCreate, "创建 Executor 失败"},
		{"ErrReplannerCreate", ErrReplannerCreate, "创建 Replanner 失败"},
		{"ErrAgentCreate", ErrAgentCreate, "创建 Agent 失败"},
		{"ErrSupervisorCreate", ErrSupervisorCreate, "创建 Supervisor 编排失败"},
		{"ErrMasterCreate", ErrMasterCreate, "创建 Master 编排失败"},
		{"ErrADKRunnerNil", ErrADKRunnerNil, "ADK Runner 未初始化"},
		{"ErrResumeFailed", ErrResumeFailed, "恢复执行失败"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("错误消息 = %q, 期望 %q", tt.err.Error(), tt.expected)
			}
		})
	}
}

// TestAgentType 测试 Agent 类型常量
func TestAgentType(t *testing.T) {
	tests := []struct {
		name     string
		agentType AgentType
		expected string
	}{
		{"ReAct Agent", AgentTypeReAct, "react_agent"},
		{"Plan Agent", AgentTypePlan, "plan_agent"},
		{"Chat Agent", AgentTypeChat, "chat_agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.agentType) != tt.expected {
				t.Errorf("AgentType = %q, 期望 %q", string(tt.agentType), tt.expected)
			}
		})
	}
}

// TestWrapError 测试错误包装
func TestWrapError(t *testing.T) {
	t.Run("包装非空错误", func(t *testing.T) {
		originalErr := errors.New("原始错误")
		wrappedErr := WrapError(originalErr, "操作失败: %s", "测试操作")

		if wrappedErr == nil {
			t.Fatal("WrapError 返回 nil")
		}

		expectedMsg := "操作失败: 测试操作: 原始错误"
		if wrappedErr.Error() != expectedMsg {
			t.Errorf("错误消息 = %q, 期望 %q", wrappedErr.Error(), expectedMsg)
		}

		if !errors.Is(wrappedErr, originalErr) {
			t.Error("应该能通过 errors.Is 找到原始错误")
		}
	})

	t.Run("包装空错误", func(t *testing.T) {
		wrappedErr := WrapError(nil, "操作失败: %s", "测试操作")

		if wrappedErr != nil {
			t.Errorf("包装 nil 错误应该返回 nil, 但返回 %v", wrappedErr)
		}
	})

	t.Run("包装哨兵错误", func(t *testing.T) {
		wrappedErr := WrapError(ErrConfigNil, "初始化失败")

		if wrappedErr == nil {
			t.Fatal("WrapError 返回 nil")
		}

		if !errors.Is(wrappedErr, ErrConfigNil) {
			t.Error("应该能通过 errors.Is 找到 ErrConfigNil")
		}
	})

	t.Run("无格式参数", func(t *testing.T) {
		originalErr := errors.New("原始错误")
		wrappedErr := WrapError(originalErr, "简单包装")

		expectedMsg := "简单包装: 原始错误"
		if wrappedErr.Error() != expectedMsg {
			t.Errorf("错误消息 = %q, 期望 %q", wrappedErr.Error(), expectedMsg)
		}
	})
}

// TestWrapError_Chained 测试链式错误包装
func TestWrapError_Chained(t *testing.T) {
	originalErr := errors.New("底层错误")
	firstWrap := WrapError(originalErr, "第一层包装")
	secondWrap := WrapError(firstWrap, "第二层包装")

	if !errors.Is(secondWrap, originalErr) {
		t.Error("应该能通过 errors.Is 找到最底层的错误")
	}

	if !errors.Is(secondWrap, firstWrap) {
		t.Error("应该能通过 errors.Is 找到中间层的错误")
	}
}

// TestAgentType_Equality 测试 AgentType 相等比较
func TestAgentType_Equality(t *testing.T) {
	tests := []struct {
		name     string
		type1    AgentType
		type2    AgentType
		expected bool
	}{
		{"相同类型", AgentTypeReAct, AgentTypeReAct, true},
		{"不同类型", AgentTypeReAct, AgentTypePlan, false},
		{"空类型比较", AgentType(""), AgentType(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.type1 == tt.type2
			if result != tt.expected {
				t.Errorf("比较结果 = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// TestSubAgent_Interface 测试 SubAgent 接口方法签名
func TestSubAgent_Interface(t *testing.T) {
	var _ SubAgent = (*mockSubAgent)(nil)
}

type mockSubAgent struct {
	name        string
	description string
	agentType   AgentType
}

func (m *mockSubAgent) Name() string            { return m.name }
func (m *mockSubAgent) Description() string     { return m.description }
func (m *mockSubAgent) Type() AgentType         { return m.agentType }
func (m *mockSubAgent) GetADKAgent() adk.Agent  { return nil }

// TestMockSubAgent 测试模拟 SubAgent 实现
func TestMockSubAgent(t *testing.T) {
	agent := &mockSubAgent{
		name:        "test-agent",
		description: "测试代理",
		agentType:   AgentTypeReAct,
	}

	if agent.Name() != "test-agent" {
		t.Errorf("Name() = %q, 期望 test-agent", agent.Name())
	}

	if agent.Description() != "测试代理" {
		t.Errorf("Description() = %q, 期望 测试代理", agent.Description())
	}

	if agent.Type() != AgentTypeReAct {
		t.Errorf("Type() = %q, 期望 %q", agent.Type(), AgentTypeReAct)
	}
}

// TestErrorUnwrap 测试错误解包
func TestErrorUnwrap(t *testing.T) {
	originalErr := errors.New("原始错误")
	wrappedErr := WrapError(originalErr, "包装")

	unwrapped := errors.Unwrap(wrappedErr)
	if unwrapped == nil {
		t.Fatal("Unwrap 返回 nil")
	}

	if unwrapped != originalErr {
		t.Error("Unwrap 应该返回原始错误")
	}
}

// TestMultipleWrapError 测试多次包装同一错误
func TestMultipleWrapError(t *testing.T) {
	originalErr := errors.New("原始错误")

	wrapped1 := WrapError(originalErr, "包装1")
	wrapped2 := WrapError(originalErr, "包装2")

	if wrapped1.Error() == wrapped2.Error() {
		t.Error("不同包装应该产生不同的错误消息")
	}

	if !errors.Is(wrapped1, originalErr) || !errors.Is(wrapped2, originalErr) {
		t.Error("两个包装都应该能找到原始错误")
	}
}
