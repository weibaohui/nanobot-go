package trace

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNewTraceID(t *testing.T) {
	traceID := NewTraceID()
	if traceID == "" {
		t.Error("TraceID 不应该为空")
	}

	// 验证是有效的 UUID
	_, err := uuid.Parse(traceID)
	if err != nil {
		t.Errorf("无效的 UUID: %v", err)
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-id"

	ctx = WithTraceID(ctx, traceID)
	retrieved := GetTraceID(ctx)

	if retrieved != traceID {
		t.Errorf("期望 %s, 得到 %s", traceID, retrieved)
	}
}

func TestGetTraceID(t *testing.T) {
	t.Run("从 context 获取", func(t *testing.T) {
		ctx := context.Background()
		traceID := "existing-trace-id"
		ctx = WithTraceID(ctx, traceID)

		retrieved := GetTraceID(ctx)
		if retrieved != traceID {
			t.Errorf("期望 %s, 得到 %s", traceID, retrieved)
		}
	})

	t.Run("不存在时生成新的", func(t *testing.T) {
		ctx := context.Background()
		retrieved := GetTraceID(ctx)

		if retrieved == "" {
			t.Error("应该生成新的 TraceID")
		}

		// 验证是有效的 UUID
		_, err := uuid.Parse(retrieved)
		if err != nil {
			t.Errorf("无效的 UUID: %v", err)
		}
	})
}

func TestMustGetTraceID(t *testing.T) {
	t.Run("存在的 TraceID", func(t *testing.T) {
		ctx := context.Background()
		traceID := "existing-trace-id"
		ctx = WithTraceID(ctx, traceID)

		retrieved := MustGetTraceID(ctx)
		if retrieved != traceID {
			t.Errorf("期望 %s, 得到 %s", traceID, retrieved)
		}
	})

	t.Run("不存在的 TraceID", func(t *testing.T) {
		ctx := context.Background()
		retrieved := MustGetTraceID(ctx)

		if retrieved != "" {
			t.Error("不存在的 TraceID 应该返回空字符串")
		}
	})
}

func TestEventTime(t *testing.T) {
	et := NewEventTime()
	if et.Timestamp.IsZero() {
		t.Error("时间戳不应该为零")
	}
}