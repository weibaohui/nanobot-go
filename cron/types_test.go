package cron

import (
	"testing"
	"time"
)

// TestNowMs 测试获取当前时间戳（毫秒）
func TestNowMs(t *testing.T) {
	before := int(time.Now().UnixMilli())
	result := nowMs()
	after := int(time.Now().UnixMilli())

	if result < before || result > after {
		t.Errorf("nowMs() = %d, 应该在 %d 和 %d 之间", result, before, after)
	}
}

// TestComputeNextRun 测试计算下次运行时间
func TestComputeNextRun(t *testing.T) {
	now := int(time.Now().UnixMilli())

	t.Run("at 类型 - 未来时间", func(t *testing.T) {
		futureTime := now + 3600000 // 1小时后
		schedule := &Schedule{
			Kind: "at",
			AtMs: futureTime,
		}

		result := computeNextRun(schedule, now)
		if result != futureTime {
			t.Errorf("computeNextRun() = %d, 期望 %d", result, futureTime)
		}
	})

	t.Run("at 类型 - 过去时间", func(t *testing.T) {
		pastTime := now - 3600000 // 1小时前
		schedule := &Schedule{
			Kind: "at",
			AtMs: pastTime,
		}

		result := computeNextRun(schedule, now)
		if result != 0 {
			t.Errorf("computeNextRun() = %d, 期望 0 (过去时间)", result)
		}
	})

	t.Run("every 类型 - 正常间隔", func(t *testing.T) {
		interval := 60000 // 1分钟
		schedule := &Schedule{
			Kind:    "every",
			EveryMs: interval,
		}

		result := computeNextRun(schedule, now)
		expected := now + interval

		if result != expected {
			t.Errorf("computeNextRun() = %d, 期望 %d", result, expected)
		}
	})

	t.Run("every 类型 - 零间隔", func(t *testing.T) {
		schedule := &Schedule{
			Kind:    "every",
			EveryMs: 0,
		}

		result := computeNextRun(schedule, now)
		if result != 0 {
			t.Errorf("computeNextRun() = %d, 期望 0 (零间隔)", result)
		}
	})

	t.Run("every 类型 - 负间隔", func(t *testing.T) {
		schedule := &Schedule{
			Kind:    "every",
			EveryMs: -1000,
		}

		result := computeNextRun(schedule, now)
		if result != 0 {
			t.Errorf("computeNextRun() = %d, 期望 0 (负间隔)", result)
		}
	})

	t.Run("cron 类型 - 暂不支持", func(t *testing.T) {
		schedule := &Schedule{
			Kind: "cron",
			Expr: "0 * * * *",
		}

		result := computeNextRun(schedule, now)
		if result != 0 {
			t.Errorf("computeNextRun() = %d, 期望 0 (cron 暂不支持)", result)
		}
	})

	t.Run("未知类型", func(t *testing.T) {
		schedule := &Schedule{
			Kind: "unknown",
		}

		result := computeNextRun(schedule, now)
		if result != 0 {
			t.Errorf("computeNextRun() = %d, 期望 0 (未知类型)", result)
		}
	})
}

// TestScheduleStruct 测试 Schedule 结构体
func TestScheduleStruct(t *testing.T) {
	schedule := Schedule{
		Kind:    "every",
		EveryMs: 30000,
		Tz:      "Asia/Shanghai",
	}

	if schedule.Kind != "every" {
		t.Errorf("Kind = %q, 期望 every", schedule.Kind)
	}

	if schedule.EveryMs != 30000 {
		t.Errorf("EveryMs = %d, 期望 30000", schedule.EveryMs)
	}

	if schedule.Tz != "Asia/Shanghai" {
		t.Errorf("Tz = %q, 期望 Asia/Shanghai", schedule.Tz)
	}
}

// TestPayloadStruct 测试 Payload 结构体
func TestPayloadStruct(t *testing.T) {
	payload := Payload{
		Kind:    "agent_turn",
		Message: "执行任务",
		Deliver: true,
		Channel: "websocket",
		To:      "user123",
	}

	if payload.Kind != "agent_turn" {
		t.Errorf("Kind = %q, 期望 agent_turn", payload.Kind)
	}

	if !payload.Deliver {
		t.Error("Deliver 应该为 true")
	}
}

// TestStateStruct 测试 State 结构体
func TestStateStruct(t *testing.T) {
	state := State{
		NextRunAtMs: 1700000000000,
		LastRunAtMs: 1699999999000,
		LastStatus:  "ok",
		LastError:   "",
	}

	if state.LastStatus != "ok" {
		t.Errorf("LastStatus = %q, 期望 ok", state.LastStatus)
	}
}

// TestJobStruct 测试 Job 结构体
func TestJobStruct(t *testing.T) {
	job := Job{
		ID:      "job-001",
		Name:    "测试任务",
		Enabled: true,
		Schedule: Schedule{
			Kind:    "every",
			EveryMs: 60000,
		},
		Payload: Payload{
			Kind:    "agent_turn",
			Message: "定时执行",
		},
		State: State{
			LastStatus: "ok",
		},
		CreatedAtMs:    1700000000000,
		UpdatedAtMs:    1700000001000,
		DeleteAfterRun: false,
	}

	if job.ID != "job-001" {
		t.Errorf("ID = %q, 期望 job-001", job.ID)
	}

	if !job.Enabled {
		t.Error("Enabled 应该为 true")
	}

	if job.Schedule.Kind != "every" {
		t.Errorf("Schedule.Kind = %q, 期望 every", job.Schedule.Kind)
	}
}

// TestStoreStruct 测试 Store 结构体
func TestStoreStruct(t *testing.T) {
	store := Store{
		Version: 1,
		Jobs: []*Job{
			{ID: "job-1", Name: "任务1"},
			{ID: "job-2", Name: "任务2"},
		},
	}

	if store.Version != 1 {
		t.Errorf("Version = %d, 期望 1", store.Version)
	}

	if len(store.Jobs) != 2 {
		t.Errorf("Jobs 长度 = %d, 期望 2", len(store.Jobs))
	}
}

// TestComputeNextRun_EdgeCases 测试边界情况
func TestComputeNextRun_EdgeCases(t *testing.T) {
	t.Run("at 类型 - 当前时间", func(t *testing.T) {
		now := int(time.Now().UnixMilli())
		schedule := &Schedule{
			Kind: "at",
			AtMs: now,
		}

		result := computeNextRun(schedule, now)
		// 当前时间等于 AtMs 时，应该返回 0（已经过期）
		if result != 0 {
			t.Errorf("computeNextRun() = %d, 期望 0 (当前时间)", result)
		}
	})

	t.Run("at 类型 - 刚好过未来1毫秒", func(t *testing.T) {
		now := int(time.Now().UnixMilli())
		futureTime := now + 1
		schedule := &Schedule{
			Kind: "at",
			AtMs: futureTime,
		}

		result := computeNextRun(schedule, now)
		if result != futureTime {
			t.Errorf("computeNextRun() = %d, 期望 %d", result, futureTime)
		}
	})

	t.Run("every 类型 - 大间隔", func(t *testing.T) {
		now := int(time.Now().UnixMilli())
		interval := 86400000 // 1天
		schedule := &Schedule{
			Kind:    "every",
			EveryMs: interval,
		}

		result := computeNextRun(schedule, now)
		expected := now + interval

		if result != expected {
			t.Errorf("computeNextRun() = %d, 期望 %d", result, expected)
		}
	})
}
