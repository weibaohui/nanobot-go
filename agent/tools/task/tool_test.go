package task

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
)

// mockManager 模拟任务管理器
type mockManager struct {
	startTaskFunc func(ctx context.Context, work, channel, chatID string) (string, string, error)
	getTaskFunc   func(ctx context.Context, taskID string) (*TaskInfo, error)
	stopTaskFunc  func(ctx context.Context, taskID string) (bool, string, error)
	listTasksFunc func() ([]*TaskInfo, error)
}

func (m *mockManager) StartTask(ctx context.Context, work, channel, chatID string) (string, string, error) {
	if m.startTaskFunc != nil {
		return m.startTaskFunc(ctx, work, channel, chatID)
	}
	return "task-001", "running", nil
}

func (m *mockManager) GetTask(ctx context.Context, taskID string) (*TaskInfo, error) {
	if m.getTaskFunc != nil {
		return m.getTaskFunc(ctx, taskID)
	}
	return &TaskInfo{ID: taskID, Status: "running", ResultSummary: "测试摘要"}, nil
}

func (m *mockManager) StopTask(ctx context.Context, taskID string) (bool, string, error) {
	if m.stopTaskFunc != nil {
		return m.stopTaskFunc(ctx, taskID)
	}
	return true, "stopped", nil
}

func (m *mockManager) ListTasks() ([]*TaskInfo, error) {
	if m.listTasksFunc != nil {
		return m.listTasksFunc()
	}
	return []*TaskInfo{
		{ID: "task-001", Status: "running", ResultSummary: "任务1"},
		{ID: "task-002", Status: "done", ResultSummary: "任务2"},
	}, nil
}

// TestStartTool_Name 测试工具名称
func TestStartTool_Name(t *testing.T) {
	tool := &StartTool{}
	if tool.Name() != "start_task" {
		t.Errorf("Name() = %q, 期望 start_task", tool.Name())
	}
}

// TestStartTool_Info 测试工具信息
func TestStartTool_Info(t *testing.T) {
	tool := &StartTool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "start_task" {
		t.Errorf("Info.Name = %q, 期望 start_task", info.Name)
	}
}

// TestStartTool_Run 测试执行工具
func TestStartTool_Run(t *testing.T) {
	t.Run("正常执行", func(t *testing.T) {
		tool := &StartTool{
			Manager: &mockManager{},
			Logger:  zap.NewNop(),
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"work": "测试任务"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("空任务内容", func(t *testing.T) {
		tool := &StartTool{
			Manager: &mockManager{},
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"work": ""}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 任务内容不能为空" {
			t.Errorf("Run() = %q, 期望错误提示", result)
		}
	})

	t.Run("无管理器", func(t *testing.T) {
		tool := &StartTool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"work": "测试任务"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 任务管理器未配置" {
			t.Errorf("Run() = %q, 期望错误提示", result)
		}
	})

	t.Run("管理器返回错误", func(t *testing.T) {
		tool := &StartTool{
			Manager: &mockManager{
				startTaskFunc: func(ctx context.Context, work, channel, chatID string) (string, string, error) {
					return "", "", errors.New("启动失败")
				},
			},
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"work": "测试任务"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 创建任务失败: 启动失败" {
			t.Errorf("Run() = %q, 期望错误提示", result)
		}
	})
}

// TestStartTool_SetContext 测试设置上下文
func TestStartTool_SetContext(t *testing.T) {
	tool := &StartTool{}
	tool.SetContext("websocket", "chat-001")

	if tool.Channel != "websocket" {
		t.Errorf("Channel = %q, 期望 websocket", tool.Channel)
	}

	if tool.ChatID != "chat-001" {
		t.Errorf("ChatID = %q, 期望 chat-001", tool.ChatID)
	}
}

// TestGetTool_Name 测试工具名称
func TestGetTool_Name(t *testing.T) {
	tool := &GetTool{}
	if tool.Name() != "get_task" {
		t.Errorf("Name() = %q, 期望 get_task", tool.Name())
	}
}

// TestGetTool_Info 测试工具信息
func TestGetTool_Info(t *testing.T) {
	tool := &GetTool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "get_task" {
		t.Errorf("Info.Name = %q, 期望 get_task", info.Name)
	}
}

// TestGetTool_Run 测试执行工具
func TestGetTool_Run(t *testing.T) {
	t.Run("正常执行", func(t *testing.T) {
		tool := &GetTool{
			Manager: &mockManager{},
			Logger:  zap.NewNop(),
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"task_id": "task-001"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("空任务ID", func(t *testing.T) {
		tool := &GetTool{
			Manager: &mockManager{},
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"task_id": ""}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 任务ID不能为空" {
			t.Errorf("Run() = %q, 期望错误提示", result)
		}
	})

	t.Run("无管理器", func(t *testing.T) {
		tool := &GetTool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"task_id": "task-001"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 任务管理器未配置" {
			t.Errorf("Run() = %q, 期望错误提示", result)
		}
	})
}

// TestStopTool_Name 测试工具名称
func TestStopTool_Name(t *testing.T) {
	tool := &StopTool{}
	if tool.Name() != "stop_task" {
		t.Errorf("Name() = %q, 期望 stop_task", tool.Name())
	}
}

// TestStopTool_Info 测试工具信息
func TestStopTool_Info(t *testing.T) {
	tool := &StopTool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "stop_task" {
		t.Errorf("Info.Name = %q, 期望 stop_task", info.Name)
	}
}

// TestStopTool_Run 测试执行工具
func TestStopTool_Run(t *testing.T) {
	t.Run("正常执行", func(t *testing.T) {
		tool := &StopTool{
			Manager: &mockManager{},
			Logger:  zap.NewNop(),
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"task_id": "task-001"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("空任务ID", func(t *testing.T) {
		tool := &StopTool{
			Manager: &mockManager{},
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"task_id": ""}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 任务ID不能为空" {
			t.Errorf("Run() = %q, 期望错误提示", result)
		}
	})
}

// TestListTool_Name 测试工具名称
func TestListTool_Name(t *testing.T) {
	tool := &ListTool{}
	if tool.Name() != "list_task" {
		t.Errorf("Name() = %q, 期望 list_task", tool.Name())
	}
}

// TestListTool_Info 测试工具信息
func TestListTool_Info(t *testing.T) {
	tool := &ListTool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "list_task" {
		t.Errorf("Info.Name = %q, 期望 list_task", info.Name)
	}
}

// TestListTool_Run 测试执行工具
func TestListTool_Run(t *testing.T) {
	t.Run("正常执行", func(t *testing.T) {
		tool := &ListTool{
			Manager: &mockManager{},
			Logger:  zap.NewNop(),
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("空列表", func(t *testing.T) {
		tool := &ListTool{
			Manager: &mockManager{
				listTasksFunc: func() ([]*TaskInfo, error) {
					return []*TaskInfo{}, nil
				},
			},
		}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "任务列表为空" {
			t.Errorf("Run() = %q, 期望 任务列表为空", result)
		}
	})

	t.Run("无管理器", func(t *testing.T) {
		tool := &ListTool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "错误: 任务管理器未配置" {
			t.Errorf("Run() = %q, 期望错误提示", result)
		}
	})
}

// TestTaskInfo 测试任务信息结构
func TestTaskInfo(t *testing.T) {
	info := &TaskInfo{
		ID:            "task-001",
		Status:        "running",
		ResultSummary: "测试摘要",
	}

	if info.ID != "task-001" {
		t.Errorf("ID = %q, 期望 task-001", info.ID)
	}

	if info.Status != "running" {
		t.Errorf("Status = %q, 期望 running", info.Status)
	}

	if info.ResultSummary != "测试摘要" {
		t.Errorf("ResultSummary = %q, 期望 测试摘要", info.ResultSummary)
	}
}

// TestManager_Interface 测试 Manager 接口实现
func TestManager_Interface(t *testing.T) {
	var _ Manager = &mockManager{}
}

// TestInvokableRun 测试 InvokableRun 方法
func TestInvokableRun(t *testing.T) {
	t.Run("StartTool", func(t *testing.T) {
		tool := &StartTool{
			Manager: &mockManager{},
		}
		ctx := context.Background()

		result, err := tool.InvokableRun(ctx, `{"work": "测试任务"}`)
		if err != nil {
			t.Errorf("InvokableRun() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("InvokableRun() 不应该返回空结果")
		}
	})

	t.Run("GetTool", func(t *testing.T) {
		tool := &GetTool{
			Manager: &mockManager{},
		}
		ctx := context.Background()

		result, err := tool.InvokableRun(ctx, `{"task_id": "task-001"}`)
		if err != nil {
			t.Errorf("InvokableRun() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("InvokableRun() 不应该返回空结果")
		}
	})

	t.Run("StopTool", func(t *testing.T) {
		tool := &StopTool{
			Manager: &mockManager{},
		}
		ctx := context.Background()

		result, err := tool.InvokableRun(ctx, `{"task_id": "task-001"}`)
		if err != nil {
			t.Errorf("InvokableRun() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("InvokableRun() 不应该返回空结果")
		}
	})

	t.Run("ListTool", func(t *testing.T) {
		tool := &ListTool{
			Manager: &mockManager{},
		}
		ctx := context.Background()

		result, err := tool.InvokableRun(ctx, `{}`)
		if err != nil {
			t.Errorf("InvokableRun() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("InvokableRun() 不应该返回空结果")
		}
	})
}
