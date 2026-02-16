package agent

import (
	"context"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// TestInterruptType 测试中断类型常量
func TestInterruptType(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  InterruptType
		expected string
	}{
		{"AskUser", InterruptTypeAskUser, "ask_user"},
		{"PlanApproval", InterruptTypePlanApproval, "plan_approval"},
		{"ToolConfirm", InterruptTypeToolConfirm, "tool_confirm"},
		{"FileOperation", InterruptTypeFileOperation, "file_operation"},
		{"Custom", InterruptTypeCustom, "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.typeVal) != tt.expected {
				t.Errorf("InterruptType = %q, 期望 %q", tt.typeVal, tt.expected)
			}
		})
	}
}

// TestInterruptStatus 测试中断状态常量
func TestInterruptStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   InterruptStatus
		expected string
	}{
		{"Pending", InterruptStatusPending, "pending"},
		{"Resolved", InterruptStatusResolved, "resolved"},
		{"Cancelled", InterruptStatusCancelled, "cancelled"},
		{"Expired", InterruptStatusExpired, "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("InterruptStatus = %q, 期望 %q", tt.status, tt.expected)
			}
		})
	}
}

// TestInterruptInfo 测试中断信息结构体
func TestInterruptInfo(t *testing.T) {
	now := time.Now()
	info := InterruptInfo{
		CheckpointID: "checkpoint-001",
		InterruptID:  "interrupt-001",
		Channel:      "websocket",
		ChatID:       "chat-001",
		SessionKey:   "session-001",
		Question:     "测试问题",
		Options:      []string{"选项A", "选项B"},
		Type:         InterruptTypeAskUser,
		Status:       InterruptStatusPending,
		CreatedAt:    now,
		Priority:     10,
	}

	if info.CheckpointID != "checkpoint-001" {
		t.Errorf("CheckpointID = %q, 期望 checkpoint-001", info.CheckpointID)
	}

	if len(info.Options) != 2 {
		t.Errorf("Options 长度 = %d, 期望 2", len(info.Options))
	}
}

// TestNewInterruptManager 测试创建中断管理器
func TestNewInterruptManager(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	if mgr == nil {
		t.Fatal("NewInterruptManager 返回 nil")
	}

	if mgr.pending == nil {
		t.Error("pending 不应该为 nil")
	}

	if mgr.handlers == nil {
		t.Error("handlers 不应该为 nil")
	}
}

// TestNewInterruptManagerWithConfig 测试使用配置创建中断管理器
func TestNewInterruptManagerWithConfig(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())

	cfg := &InterruptManagerConfig{
		Bus:            messageBus,
		Logger:         zap.NewNop(),
		DefaultTimeout: 10 * time.Minute,
		MaxPending:     50,
		MaxHistory:     500,
	}

	mgr := NewInterruptManagerWithConfig(cfg)
	if mgr == nil {
		t.Fatal("NewInterruptManagerWithConfig 返回 nil")
	}

	if mgr.defaultTimeout != 10*time.Minute {
		t.Errorf("defaultTimeout = %v, 期望 10m", mgr.defaultTimeout)
	}

	if mgr.maxPending != 50 {
		t.Errorf("maxPending = %d, 期望 50", mgr.maxPending)
	}
}

// TestNewInterruptManagerWithConfig_Defaults 测试配置默认值
func TestNewInterruptManagerWithConfig_Defaults(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())

	cfg := &InterruptManagerConfig{
		Bus: messageBus,
	}

	mgr := NewInterruptManagerWithConfig(cfg)
	if mgr == nil {
		t.Fatal("NewInterruptManagerWithConfig 返回 nil")
	}

	if mgr.maxPending != 100 {
		t.Errorf("默认 maxPending = %d, 期望 100", mgr.maxPending)
	}

	if mgr.maxHistory != 1000 {
		t.Errorf("默认 maxHistory = %d, 期望 1000", mgr.maxHistory)
	}
}

// TestInterruptManager_HandleInterrupt 测试处理中断
func TestInterruptManager_HandleInterrupt(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	info := &InterruptInfo{
		CheckpointID: "checkpoint-001",
		InterruptID:  "interrupt-001",
		Channel:      "test",
		ChatID:       "chat-001",
		SessionKey:   "session-001",
		Question:     "测试问题",
		Type:         InterruptTypeAskUser,
	}

	mgr.HandleInterrupt(info)

	if len(mgr.pending) != 1 {
		t.Errorf("pending 长度 = %d, 期望 1", len(mgr.pending))
	}

	if mgr.pending["checkpoint-001"] == nil {
		t.Error("中断应该被添加到 pending")
	}
}

// TestInterruptManager_GetPendingInterrupt 测试获取待处理中断
func TestInterruptManager_GetPendingInterrupt(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	info := &InterruptInfo{
		CheckpointID: "checkpoint-001",
		SessionKey:   "session-001",
		Question:     "测试问题",
		Type:         InterruptTypeAskUser,
	}

	mgr.HandleInterrupt(info)

	pending := mgr.GetPendingInterrupt("session-001")
	if pending == nil {
		t.Fatal("GetPendingInterrupt 返回 nil")
	}

	if pending.CheckpointID != "checkpoint-001" {
		t.Errorf("CheckpointID = %q, 期望 checkpoint-001", pending.CheckpointID)
	}
}

// TestInterruptManager_ClearInterrupt 测试清除中断
func TestInterruptManager_ClearInterrupt(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	info := &InterruptInfo{
		CheckpointID: "checkpoint-001",
		SessionKey:   "session-001",
		Question:     "测试问题",
		Type:         InterruptTypeAskUser,
	}

	mgr.HandleInterrupt(info)
	mgr.ClearInterrupt("checkpoint-001")

	if len(mgr.pending) != 0 {
		t.Errorf("pending 长度 = %d, 期望 0", len(mgr.pending))
	}

	if mgr.GetPendingInterrupt("session-001") != nil {
		t.Error("session 中断应该被清除")
	}
}

// TestInterruptManager_CancelInterrupt 测试取消中断
func TestInterruptManager_CancelInterrupt(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	info := &InterruptInfo{
		CheckpointID: "checkpoint-001",
		SessionKey:   "session-001",
		Question:     "测试问题",
		Type:         InterruptTypeAskUser,
	}

	mgr.HandleInterrupt(info)
	mgr.CancelInterrupt("checkpoint-001")

	if mgr.pending["checkpoint-001"] != nil {
		t.Error("中断应该被移除")
	}
}

// TestInterruptManager_SubmitUserResponse 测试提交用户响应
func TestInterruptManager_SubmitUserResponse(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	info := &InterruptInfo{
		CheckpointID: "checkpoint-001",
		SessionKey:   "session-001",
		Question:     "测试问题",
		Type:         InterruptTypeAskUser,
	}

	mgr.HandleInterrupt(info)

	response := &UserResponse{
		CheckpointID: "checkpoint-001",
		Answer:       "用户回答",
	}

	err := mgr.SubmitUserResponse(response)
	if err != nil {
		t.Errorf("SubmitUserResponse 返回错误: %v", err)
	}
}

// TestInterruptManager_SubmitUserResponse_NotFound 测试提交响应到不存在的中断
func TestInterruptManager_SubmitUserResponse_NotFound(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	response := &UserResponse{
		CheckpointID: "nonexistent",
		Answer:       "用户回答",
	}

	err := mgr.SubmitUserResponse(response)
	if err == nil {
		t.Error("提交到不存在的中断应该返回错误")
	}
}

// TestInterruptManager_GetInterruptHistory 测试获取中断历史
func TestInterruptManager_GetInterruptHistory(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	for i := 0; i < 5; i++ {
		info := &InterruptInfo{
			CheckpointID: "checkpoint-" + string(rune('0'+i)),
			Question:     "问题",
			Type:         InterruptTypeAskUser,
		}
		mgr.HandleInterrupt(info)
		mgr.ClearInterrupt(info.CheckpointID)
	}

	history := mgr.GetInterruptHistory(3)
	if len(history) != 3 {
		t.Errorf("GetInterruptHistory(3) 返回 %d 条记录, 期望 3", len(history))
	}

	allHistory := mgr.GetInterruptHistory(0)
	if len(allHistory) != 5 {
		t.Errorf("GetInterruptHistory(0) 返回 %d 条记录, 期望 5", len(allHistory))
	}
}

// TestInterruptManager_GetInterruptStats 测试获取中断统计
func TestInterruptManager_GetInterruptStats(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	info := &InterruptInfo{
		CheckpointID: "checkpoint-001",
		Question:     "问题",
		Type:         InterruptTypeAskUser,
	}
	mgr.HandleInterrupt(info)

	stats := mgr.GetInterruptStats()
	if stats == nil {
		t.Fatal("GetInterruptStats 返回 nil")
	}

	if stats["pending_count"].(int) != 1 {
		t.Errorf("pending_count = %v, 期望 1", stats["pending_count"])
	}
}

// TestCreateAskUserInterrupt 测试创建用户提问中断
func TestCreateAskUserInterrupt(t *testing.T) {
	info := CreateAskUserInterrupt(
		"checkpoint-001",
		"interrupt-001",
		"websocket",
		"chat-001",
		"session-001",
		"请选择一个选项",
		[]string{"选项A", "选项B"},
	)

	if info.CheckpointID != "checkpoint-001" {
		t.Errorf("CheckpointID = %q, 期望 checkpoint-001", info.CheckpointID)
	}

	if info.Type != InterruptTypeAskUser {
		t.Errorf("Type = %q, 期望 ask_user", info.Type)
	}

	if len(info.Options) != 2 {
		t.Errorf("Options 长度 = %d, 期望 2", len(info.Options))
	}
}

// TestCreatePlanApprovalInterrupt 测试创建计划审批中断
func TestCreatePlanApprovalInterrupt(t *testing.T) {
	info := CreatePlanApprovalInterrupt(
		"checkpoint-001",
		"interrupt-001",
		"websocket",
		"chat-001",
		"session-001",
		"plan-001",
		"计划内容",
		[]string{"步骤1", "步骤2"},
	)

	if info.Type != InterruptTypePlanApproval {
		t.Errorf("Type = %q, 期望 plan_approval", info.Type)
	}

	if info.Metadata["plan_id"] != "plan-001" {
		t.Errorf("plan_id = %v, 期望 plan-001", info.Metadata["plan_id"])
	}
}

// TestCreateToolConfirmInterrupt 测试创建工具确认中断
func TestCreateToolConfirmInterrupt(t *testing.T) {
	info := CreateToolConfirmInterrupt(
		"checkpoint-001",
		"interrupt-001",
		"websocket",
		"chat-001",
		"session-001",
		"execute_command",
		map[string]any{"command": "ls"},
		"high",
	)

	if info.Type != InterruptTypeToolConfirm {
		t.Errorf("Type = %q, 期望 tool_confirm", info.Type)
	}

	if info.Metadata["tool_name"] != "execute_command" {
		t.Errorf("tool_name = %v, 期望 execute_command", info.Metadata["tool_name"])
	}
}

// TestAskUserHandler 测试用户提问处理器
func TestAskUserHandler(t *testing.T) {
	handler := &AskUserHandler{}

	t.Run("FormatQuestion", func(t *testing.T) {
		info := &InterruptInfo{
			Question: "请选择",
			Options:  []string{"A", "B", "C"},
		}

		question := handler.FormatQuestion(info)
		if question == "" {
			t.Error("FormatQuestion 返回空字符串")
		}
	})

	t.Run("Validate", func(t *testing.T) {
		err := handler.Validate(&UserResponse{Answer: "回答"})
		if err != nil {
			t.Errorf("Validate 返回错误: %v", err)
		}

		err = handler.Validate(&UserResponse{Answer: ""})
		if err == nil {
			t.Error("空回答应该返回错误")
		}
	})
}

// TestPlanApprovalHandler 测试计划审批处理器
func TestPlanApprovalHandler(t *testing.T) {
	handler := &PlanApprovalHandler{}

	t.Run("FormatQuestion", func(t *testing.T) {
		info := &InterruptInfo{
			Question: "请审批计划",
			Metadata: map[string]any{
				"steps": []string{"步骤1", "步骤2"},
			},
		}

		question := handler.FormatQuestion(info)
		if question == "" {
			t.Error("FormatQuestion 返回空字符串")
		}
	})

	t.Run("Validate", func(t *testing.T) {
		err := handler.Validate(&UserResponse{Answer: "确认"})
		if err != nil {
			t.Errorf("Validate 返回错误: %v", err)
		}
	})
}

// TestToolConfirmHandler 测试工具确认处理器
func TestToolConfirmHandler(t *testing.T) {
	handler := &ToolConfirmHandler{}

	t.Run("FormatQuestion", func(t *testing.T) {
		info := &InterruptInfo{
			Metadata: map[string]any{
				"tool_name":  "test_tool",
				"risk_level": "high",
				"tool_args":  map[string]any{"arg1": "value1"},
			},
		}

		question := handler.FormatQuestion(info)
		if question == "" {
			t.Error("FormatQuestion 返回空字符串")
		}
	})
}

// TestFileOperationHandler 测试文件操作处理器
func TestFileOperationHandler(t *testing.T) {
	handler := &FileOperationHandler{}

	t.Run("FormatQuestion", func(t *testing.T) {
		info := &InterruptInfo{
			Metadata: map[string]any{
				"operation":  "write",
				"file_path":  "/tmp/test.txt",
			},
		}

		question := handler.FormatQuestion(info)
		if question == "" {
			t.Error("FormatQuestion 返回空字符串")
		}
	})
}

// TestInMemoryCheckpointStore 测试内存 Checkpoint 存储
func TestInMemoryCheckpointStore(t *testing.T) {
	store := NewInMemoryCheckpointStore().(*InMemoryCheckpointStore)
	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		err := store.Set(ctx, "key1", []byte("value1"))
		if err != nil {
			t.Errorf("Set 返回错误: %v", err)
		}

		val, ok, err := store.Get(ctx, "key1")
		if err != nil {
			t.Errorf("Get 返回错误: %v", err)
		}

		if !ok {
			t.Error("应该找到 key1")
		}

		if string(val) != "value1" {
			t.Errorf("值 = %q, 期望 value1", string(val))
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, ok, err := store.Get(ctx, "nonexistent")
		if err != nil {
			t.Errorf("Get 返回错误: %v", err)
		}

		if ok {
			t.Error("不应该找到 nonexistent")
		}
	})
}

// TestInterruptManager_RegisterHandler 测试注册处理器
func TestInterruptManager_RegisterHandler(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	customHandler := &AskUserHandler{}
	mgr.RegisterHandler(InterruptTypeCustom, customHandler)

	if mgr.handlers[InterruptTypeCustom] == nil {
		t.Error("处理器应该被注册")
	}
}

// TestInterruptManager_GetCheckpointStore 测试获取 CheckpointStore
func TestInterruptManager_GetCheckpointStore(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	mgr := NewInterruptManager(messageBus, zap.NewNop())

	store := mgr.GetCheckpointStore()
	if store == nil {
		t.Error("GetCheckpointStore 不应该返回 nil")
	}
}
