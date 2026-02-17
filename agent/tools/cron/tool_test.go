package cron

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/cron"
	"go.uber.org/zap"
)

// TestTool_Name 测试工具名称
func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	if tool.Name() != "cron" {
		t.Errorf("Name() = %q, 期望 cron", tool.Name())
	}
}

// TestTool_Info 测试工具信息
func TestTool_Info(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "cron" {
		t.Errorf("Info.Name = %q, 期望 cron", info.Name)
	}
}

// TestTool_SetContext 测试设置上下文
func TestTool_SetContext(t *testing.T) {
	tool := &Tool{}
	tool.SetContext("websocket", "chat-001")

	if tool.Channel != "websocket" {
		t.Errorf("Channel = %q, 期望 websocket", tool.Channel)
	}

	if tool.ChatID != "chat-001" {
		t.Errorf("ChatID = %q, 期望 chat-001", tool.ChatID)
	}
}

// TestTool_Run 测试执行工具
func TestTool_Run(t *testing.T) {
	t.Run("未知操作", func(t *testing.T) {
		tool := &Tool{}
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"action": "unknown"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result != "未知操作: unknown" {
			t.Errorf("Run() = %q, 期望 未知操作: unknown", result)
		}
	})
}

// TestTool_listJobs 测试列出任务
func TestTool_listJobs(t *testing.T) {
	t.Run("无任务", func(t *testing.T) {
		tmpFile := "/tmp/test_cron_list.json"
		service := cron.NewService(tmpFile, zap.NewNop())
		tool := &Tool{CronService: service}

		result, err := tool.listJobs()
		if err != nil {
			t.Errorf("listJobs() 返回错误: %v", err)
		}

		if result != "没有计划任务" {
			t.Errorf("listJobs() = %q, 期望 没有计划任务", result)
		}
	})
}

// TestTool_addJob 测试添加任务
func TestTool_addJob(t *testing.T) {
	t.Run("无消息参数", func(t *testing.T) {
		tool := &Tool{}

		result, err := tool.addJob(Args{})
		if err != nil {
			t.Errorf("addJob() 返回错误: %v", err)
		}

		if result != "错误: 需要消息参数" {
			t.Errorf("addJob() = %q, 期望 错误: 需要消息参数", result)
		}
	})

	t.Run("无会话上下文", func(t *testing.T) {
		tool := &Tool{}

		result, err := tool.addJob(Args{Message: "测试消息"})
		if err != nil {
			t.Errorf("addJob() 返回错误: %v", err)
		}

		if result != "错误: 没有会话上下文" {
			t.Errorf("addJob() = %q, 期望 错误: 没有会话上下文", result)
		}
	})

	t.Run("无调度参数", func(t *testing.T) {
		tool := &Tool{Channel: "websocket", ChatID: "chat-001"}

		result, err := tool.addJob(Args{Message: "测试消息"})
		if err != nil {
			t.Errorf("addJob() 返回错误: %v", err)
		}

		if result != "错误: 需要 every_seconds 或 cron_expr 参数" {
			t.Errorf("addJob() = %q, 期望 错误: 需要 every_seconds 或 cron_expr 参数", result)
		}
	})
}

// TestTool_removeJob 测试删除任务
func TestTool_removeJob(t *testing.T) {
	t.Run("无 job_id 参数", func(t *testing.T) {
		tool := &Tool{}

		result, err := tool.removeJob(Args{})
		if err != nil {
			t.Errorf("removeJob() 返回错误: %v", err)
		}

		if result != "错误: 需要 job_id 参数" {
			t.Errorf("removeJob() = %q, 期望 错误: 需要 job_id 参数", result)
		}
	})
}

// TestTool_InvokableRun 测试 InvokableRun 方法
func TestTool_InvokableRun(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	result, err := tool.InvokableRun(ctx, `{"action": "unknown"}`)
	if err != nil {
		t.Errorf("InvokableRun() 返回错误: %v", err)
	}

	if result == "" {
		t.Error("InvokableRun() 不应该返回空结果")
	}
}

// TestTool_InfoParams 测试工具参数信息
func TestTool_InfoParams(t *testing.T) {
	tool := &Tool{}
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.ParamsOneOf == nil {
		t.Error("ParamsOneOf 不应该为 nil")
	}
}

// TestTool_Interface 测试工具接口实现
func TestTool_Interface(t *testing.T) {
	tool := &Tool{}

	var _ interface {
		Name() string
		Info(ctx context.Context) (*schema.ToolInfo, error)
	} = tool
}

// TestArgs 测试参数结构
func TestArgs(t *testing.T) {
	args := Args{
		Action:       "add",
		Message:      "测试消息",
		EverySeconds: 60,
		CronExpr:     "0 * * * *",
		JobID:        "job-001",
	}

	if args.Action != "add" {
		t.Errorf("Action = %q, 期望 add", args.Action)
	}

	if args.Message != "测试消息" {
		t.Errorf("Message = %q, 期望 测试消息", args.Message)
	}

	if args.EverySeconds != 60 {
		t.Errorf("EverySeconds = %f, 期望 60", args.EverySeconds)
	}

	if args.CronExpr != "0 * * * *" {
		t.Errorf("CronExpr = %q, 期望 0 * * * *", args.CronExpr)
	}

	if args.JobID != "job-001" {
		t.Errorf("JobID = %q, 期望 job-001", args.JobID)
	}
}

// TestTool_CompleteWorkflow 测试完整工作流
func TestTool_CompleteWorkflow(t *testing.T) {
	tmpFile := "/tmp/test_cron_workflow.json"
	service := cron.NewService(tmpFile, zap.NewNop())
	tool := &Tool{
		CronService: service,
		Channel:     "websocket",
		ChatID:      "chat-001",
	}
	ctx := context.Background()

	listResult, err := tool.Run(ctx, `{"action": "list"}`)
	if err != nil {
		t.Errorf("Run(list) 返回错误: %v", err)
	}
	if listResult != "没有计划任务" {
		t.Errorf("listResult = %q, 期望 没有计划任务", listResult)
	}
}
