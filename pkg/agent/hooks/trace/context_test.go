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

func TestNewSpanID(t *testing.T) {
	spanID := NewSpanID()
	if spanID == "" {
		t.Error("SpanID 不应该为空")
	}

	// 验证是有效的 UUID
	_, err := uuid.Parse(spanID)
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

func TestWithSpanID(t *testing.T) {
	ctx := context.Background()
	spanID := "test-span-id"

	ctx = WithSpanID(ctx, spanID)
	retrieved := GetSpanID(ctx)

	if retrieved != spanID {
		t.Errorf("期望 %s, 得到 %s", spanID, retrieved)
	}
}

func TestWithParentSpanID(t *testing.T) {
	ctx := context.Background()
	parentSpanID := "test-parent-span-id"

	ctx = WithParentSpanID(ctx, parentSpanID)
	retrieved := GetParentSpanID(ctx)

	if retrieved != parentSpanID {
		t.Errorf("期望 %s, 得到 %s", parentSpanID, retrieved)
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

func TestGetSpanID(t *testing.T) {
	t.Run("从 context 获取", func(t *testing.T) {
		ctx := context.Background()
		spanID := "existing-span-id"
		ctx = WithSpanID(ctx, spanID)

		retrieved := GetSpanID(ctx)
		if retrieved != spanID {
			t.Errorf("期望 %s, 得到 %s", spanID, retrieved)
		}
	})

	t.Run("不存在时生成新的", func(t *testing.T) {
		ctx := context.Background()
		retrieved := GetSpanID(ctx)

		if retrieved == "" {
			t.Error("应该生成新的 SpanID")
		}

		// 验证是有效的 UUID
		_, err := uuid.Parse(retrieved)
		if err != nil {
			t.Errorf("无效的 UUID: %v", err)
		}
	})
}

func TestGetParentSpanID(t *testing.T) {
	t.Run("从 context 获取", func(t *testing.T) {
		ctx := context.Background()
		parentSpanID := "existing-parent-span-id"
		ctx = WithParentSpanID(ctx, parentSpanID)

		retrieved := GetParentSpanID(ctx)
		if retrieved != parentSpanID {
			t.Errorf("期望 %s, 得到 %s", parentSpanID, retrieved)
		}
	})

	t.Run("不存在时返回空字符串", func(t *testing.T) {
		ctx := context.Background()
		retrieved := GetParentSpanID(ctx)

		if retrieved != "" {
			t.Error("不存在的 ParentSpanID 应该返回空字符串")
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

func TestMustGetSpanID(t *testing.T) {
	t.Run("存在的 SpanID", func(t *testing.T) {
		ctx := context.Background()
		spanID := "existing-span-id"
		ctx = WithSpanID(ctx, spanID)

		retrieved := MustGetSpanID(ctx)
		if retrieved != spanID {
			t.Errorf("期望 %s, 得到 %s", spanID, retrieved)
		}
	})

	t.Run("不存在的 SpanID", func(t *testing.T) {
		ctx := context.Background()
		retrieved := MustGetSpanID(ctx)

		if retrieved != "" {
			t.Error("不存在的 SpanID 应该返回空字符串")
		}
	})
}

func TestStartSpan(t *testing.T) {
	t.Run("根 Span (没有父)", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithTraceID(ctx, "root-trace-id")

		newCtx, spanID := StartSpan(ctx)

		// SpanID 应该是新的 UUID
		if spanID == "" {
			t.Error("SpanID 不应该为空")
		}

		// ParentSpanID 应该为空（根 Span）
		parentSpanID := GetParentSpanID(newCtx)
		if parentSpanID != "" {
			t.Errorf("根 Span 的 ParentSpanID 应该为空，得到 %s", parentSpanID)
		}

		// TraceID 应该保持不变
		traceID := GetTraceID(newCtx)
		if traceID != "root-trace-id" {
			t.Errorf("TraceID 应该保持不变，得到 %s", traceID)
		}
	})

	t.Run("子 Span (有父)", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithTraceID(ctx, "root-trace-id")
		ctx = WithSpanID(ctx, "parent-span-id")

		newCtx, spanID := StartSpan(ctx)

		// SpanID 应该是新的 UUID
		if spanID == "" {
			t.Error("SpanID 不应该为空")
		}

		// 新的 SpanID 应该与父 SpanID 不同
		if spanID == "parent-span-id" {
			t.Error("子 SpanID 应该与父 SpanID 不同")
		}

		// ParentSpanID 应该指向父 Span
		parentSpanID := GetParentSpanID(newCtx)
		if parentSpanID != "parent-span-id" {
			t.Errorf("ParentSpanID 应该为 parent-span-id，得到 %s", parentSpanID)
		}

		// TraceID 应该保持不变
		traceID := GetTraceID(newCtx)
		if traceID != "root-trace-id" {
			t.Errorf("TraceID 应该保持不变，得到 %s", traceID)
		}
	})

	t.Run("多级调用链", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithTraceID(ctx, "root-trace-id")

		// 第一级
		ctx1, span1 := StartSpan(ctx)
		if GetParentSpanID(ctx1) != "" {
			t.Error("第一级 Span 不应该有 ParentSpanID")
		}

		// 第二级
		ctx2, span2 := StartSpan(ctx1)
		if GetParentSpanID(ctx2) != span1 {
			t.Error("第二级 Span 的 ParentSpanID 应该指向第一级")
		}

		// 第三级
		ctx3, span3 := StartSpan(ctx2)
		_ = span3 // 使用 span3 避免未使用变量错误
		if GetParentSpanID(ctx3) != span2 {
			t.Error("第三级 Span 的 ParentSpanID 应该指向第二级")
		}

		// 所有 Span 应该有相同的 TraceID
		if GetTraceID(ctx1) != "root-trace-id" ||
			GetTraceID(ctx2) != "root-trace-id" ||
			GetTraceID(ctx3) != "root-trace-id" {
			t.Error("所有 Span 应该有相同的 TraceID")
		}
	})
}

func TestEventTime(t *testing.T) {
	et := NewEventTime()
	if et.Timestamp.IsZero() {
		t.Error("时间戳不应该为零")
	}
}