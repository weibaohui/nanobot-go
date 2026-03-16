package cron

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// TestNewService 测试创建定时任务服务
func TestNewService(t *testing.T) {
	logger := zap.NewNop()
	service := NewService("/tmp/test_jobs.json", logger)

	if service == nil {
		t.Fatal("NewService 返回 nil")
	}

	if service.storePath != "/tmp/test_jobs.json" {
		t.Errorf("storePath = %q, 期望 /tmp/test_jobs.json", service.storePath)
	}
}

// TestNewService_NilLogger 测试空 logger 使用默认值
func TestNewService_NilLogger(t *testing.T) {
	service := NewService("/tmp/test.json", nil)
	if service == nil {
		t.Fatal("NewService 返回 nil")
	}

	if service.logger == nil {
		t.Error("logger 不应该为 nil")
	}
}

// TestService_loadStore 测试加载任务存储
func TestService_loadStore(t *testing.T) {
	t.Run("文件不存在", func(t *testing.T) {
		tmpDir := t.TempDir()
		storePath := filepath.Join(tmpDir, "nonexistent.json")

		service := NewService(storePath, zap.NewNop())
		service.loadStore()

		if service.store == nil {
			t.Error("store 不应该为 nil")
		}
	})

	t.Run("文件存在但无效", func(t *testing.T) {
		tmpDir := t.TempDir()
		storePath := filepath.Join(tmpDir, "invalid.json")

		os.WriteFile(storePath, []byte("invalid json"), 0644)

		service := NewService(storePath, zap.NewNop())
		service.loadStore()

		if service.store == nil {
			t.Error("store 不应该为 nil")
		}
	})
}

// TestService_saveStore 测试保存任务存储
func TestService_saveStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")

	service := NewService(storePath, zap.NewNop())
	service.store = &Store{Version: 1}

	service.saveStore()

	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Error("存储文件应该被创建")
	}
}

// TestService_saveStore_NilStore 测试保存空存储
func TestService_saveStore_NilStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")

	service := NewService(storePath, zap.NewNop())
	service.store = nil

	service.saveStore()

	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Error("空存储不应该创建文件")
	}
}

// TestService_recomputeNextRuns 测试重新计算下次运行时间
func TestService_recomputeNextRuns(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")

	service := NewService(storePath, zap.NewNop())
	service.store = &Store{
		Version: 1,
		Jobs: []*Job{
			{
				Enabled:  true,
				Schedule: Schedule{Kind: "every", EveryMs: 5 * 60 * 1000},
			},
		},
	}

	service.recomputeNextRuns()

	if service.store.Jobs[0].State.NextRunAtMs == 0 {
		t.Error("NextRunAtMs 应该被计算")
	}
}

// TestService_getNextWakeMs 测试获取最早唤醒时间
func TestService_getNextWakeMs(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	now := nowMs()

	service.store = &Store{
		Version: 1,
		Jobs: []*Job{
			{
				Enabled: true,
				State:   State{NextRunAtMs: now + 1000},
			},
			{
				Enabled: true,
				State:   State{NextRunAtMs: now + 500},
			},
		},
	}

	nextWake := service.getNextWakeMs()
	expected := now + 500

	if nextWake != expected {
		t.Errorf("getNextWakeMs = %d, 期望 %d", nextWake, expected)
	}
}

// TestService_getNextWakeMs_NoJobs 测试无任务时的唤醒时间
func TestService_getNextWakeMs_NoJobs(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	service.store = &Store{Version: 1}

	nextWake := service.getNextWakeMs()
	if nextWake != 0 {
		t.Errorf("无任务时 getNextWakeMs = %d, 期望 0", nextWake)
	}
}

// TestGenerateID 测试生成 ID
func TestGenerateID(t *testing.T) {
	id := generateID()

	if id == "" {
		t.Error("ID 不应该为空")
	}

	if len(id) != 8 {
		t.Errorf("ID 长度 = %d, 期望 8", len(id))
	}
}

// TestService_removeJobByID 测试内部删除任务
func TestService_removeJobByID(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	service.store = &Store{
		Version: 1,
		Jobs: []*Job{
			{ID: "job-001"},
			{ID: "job-002"},
		},
	}

	removed := service.removeJobByID("job-001")
	if !removed {
		t.Error("removeJobByID 应该返回 true")
	}

	if len(service.store.Jobs) != 1 {
		t.Errorf("Jobs 长度 = %d, 期望 1", len(service.store.Jobs))
	}

	removed = service.removeJobByID("job-001")
	if removed {
		t.Error("再次删除应该返回 false")
	}
}

// TestService_Status 测试获取服务状态
func TestService_Status(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	service.store = &Store{
		Version: 1,
		Jobs: []*Job{
			{Enabled: true},
		},
	}

	status := service.Status()
	if status == nil {
		t.Fatal("Status 返回 nil")
	}

	if status["jobs"].(int) != 1 {
		t.Errorf("jobs = %v, 期望 1", status["jobs"])
	}
}

// TestService_SetOnJobCallback 测试设置任务执行回调
func TestService_SetOnJobCallback(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())

	callback := func(job *Job) (string, error) {
		return "执行成功", nil
	}

	service.SetOnJobCallback(callback)

	if service.onJob == nil {
		t.Error("onJob 应该被设置")
	}
}

// TestService_ListJobs 测试列出任务
func TestService_ListJobs(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	service.store = &Store{
		Version: 1,
		Jobs: []*Job{
			{ID: "job-001", Enabled: true},
			{ID: "job-002", Enabled: true},
			{ID: "job-003", Enabled: false},
		},
	}

	jobs := service.ListJobs()
	if len(jobs) != 2 {
		t.Errorf("ListJobs 返回 %d 个任务, 期望 2", len(jobs))
	}
}

// TestService_ExecuteJob 测试执行任务
func TestService_ExecuteJob(t *testing.T) {
	executed := false
	service := NewService("/tmp/test.json", zap.NewNop())
	service.SetOnJobCallback(func(job *Job) (string, error) {
		executed = true
		return "成功", nil
	})

	ctx := context.Background()

	job := &Job{
		ID:       "test-job-001",
		Name:     "测试任务",
		Enabled:  true,
		Schedule: Schedule{Kind: "every", EveryMs: 5 * 60 * 1000},
	}
	service.store = &Store{Version: 1, Jobs: []*Job{job}}

	service.executeJob(ctx, job)

	if !executed {
		t.Error("任务应该被执行")
	}

	if job.State.LastStatus != "ok" {
		t.Errorf("LastStatus = %q, 期望 ok", job.State.LastStatus)
	}
}

// TestService_ExecuteJob_Error 测试执行任务出错
func TestService_ExecuteJob_Error(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	service.SetOnJobCallback(func(job *Job) (string, error) {
		return "", os.ErrNotExist
	})

	ctx := context.Background()

	job := &Job{
		ID:       "test-job-002",
		Name:     "测试任务",
		Enabled:  true,
		Schedule: Schedule{Kind: "every", EveryMs: 5 * 60 * 1000},
	}
	service.store = &Store{Version: 1, Jobs: []*Job{job}}

	service.executeJob(ctx, job)

	if job.State.LastStatus != "error" {
		t.Errorf("LastStatus = %q, 期望 error", job.State.LastStatus)
	}

	if job.State.LastError == "" {
		t.Error("LastError 不应该为空")
	}
}

// TestService_ExecuteJob_OneTime 测试执行一次性任务
func TestService_ExecuteJob_OneTime(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	service.SetOnJobCallback(func(job *Job) (string, error) {
		return "成功", nil
	})

	ctx := context.Background()

	job := &Job{
		ID:             "test-job-003",
		Name:           "一次性任务",
		Enabled:        true,
		Schedule:       Schedule{Kind: "at"},
		DeleteAfterRun: true,
	}
	service.store = &Store{Version: 1, Jobs: []*Job{job}}

	service.executeJob(ctx, job)

	if len(service.store.Jobs) != 0 {
		t.Error("一次性任务执行后应该被删除")
	}
}

// TestService_ExecuteJob_OneTime_NoDelete 测试执行一次性任务不删除
func TestService_ExecuteJob_OneTime_NoDelete(t *testing.T) {
	service := NewService("/tmp/test.json", zap.NewNop())
	service.SetOnJobCallback(func(job *Job) (string, error) {
		return "成功", nil
	})

	ctx := context.Background()

	job := &Job{
		ID:             "test-job-004",
		Name:           "一次性任务",
		Enabled:        true,
		Schedule:       Schedule{Kind: "at"},
		DeleteAfterRun: false,
	}
	service.store = &Store{Version: 1, Jobs: []*Job{job}}

	service.executeJob(ctx, job)

	if len(service.store.Jobs) != 1 {
		t.Error("任务应该保留")
	}

	if job.Enabled {
		t.Error("任务应该被禁用")
	}
}
