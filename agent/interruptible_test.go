package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// TestNewInterruptible 测试创建中断处理能力
func TestNewInterruptible(t *testing.T) {
	t.Run("配置为空返回错误", func(t *testing.T) {
		ctx := context.Background()
		i, err := newInterruptible(ctx, nil)

		if err != ErrConfigNil {
			t.Errorf("newInterruptible() error = %v, 期望 %v", err, ErrConfigNil)
		}
		if i != nil {
			t.Error("newInterruptible() 应该返回 nil")
		}
	})

	t.Run("基本配置", func(t *testing.T) {
		ctx := context.Background()
		logger := zap.NewNop()
		cfg := &config.Config{}
		sessionMgr := session.NewManager(cfg, "/tmp")
		messageBus := bus.NewMessageBus(logger)

		i, err := newInterruptible(ctx, &interruptibleConfig{
			Cfg:           cfg,
			Workspace:     "/tmp/workspace",
			Logger:        logger,
			Sessions:      sessionMgr,
			Bus:           messageBus,
			MaxIterations: 10,
			AgentType:     "test",
		})

		if err != nil {
			t.Errorf("newInterruptible() 返回错误: %v", err)
		}
		if i == nil {
			t.Fatal("newInterruptible() 返回 nil")
		}

		if i.workspace != "/tmp/workspace" {
			t.Errorf("interruptible.workspace = %q, 期望 /tmp/workspace", i.workspace)
		}

		if i.maxIterations != 10 {
			t.Errorf("interruptible.maxIterations = %d, 期望 10", i.maxIterations)
		}

		if i.agentType != "test" {
			t.Errorf("interruptible.agentType = %q, 期望 test", i.agentType)
		}
	})

	t.Run("无logger使用默认", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{}

		i, err := newInterruptible(ctx, &interruptibleConfig{
			Cfg:       cfg,
			Workspace: "/tmp/workspace",
		})

		if err != nil {
			t.Errorf("newInterruptible() 返回错误: %v", err)
		}
		if i == nil {
			t.Fatal("newInterruptible() 返回 nil")
		}

		if i.logger == nil {
			t.Error("interruptible.logger 应该使用默认的 nop logger")
		}
	})

	t.Run("默认MaxIterations", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{}

		i, err := newInterruptible(ctx, &interruptibleConfig{
			Cfg:       cfg,
			Workspace: "/tmp/workspace",
		})

		if err != nil {
			t.Errorf("newInterruptible() 返回错误: %v", err)
		}
		if i == nil {
			t.Fatal("newInterruptible() 返回 nil")
		}

		if i.maxIterations != 10 {
			t.Errorf("interruptible.maxIterations = %d, 期望默认值 10", i.maxIterations)
		}
	})
}

// TestInterruptible_BuildChatModelAdapter 测试构建 ChatModelAdapter
func TestInterruptible_BuildChatModelAdapter(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	sessionMgr := session.NewManager(cfg, "/tmp")

	i := &interruptible{
		cfg:      cfg,
		logger:   logger,
		sessions: sessionMgr,
		registeredTools: []string{"read_file", "write_file"},
	}

	adapter, err := i.BuildChatModelAdapter()
	if err == nil {
		if adapter == nil {
			t.Error("BuildChatModelAdapter() 不应该返回 nil adapter")
		}
	}
}

// TestInterruptible_ConvertHistory 测试转换会话历史
func TestInterruptible_ConvertHistory(t *testing.T) {
	i := &interruptible{
		logger: zap.NewNop(),
	}

	history := []map[string]any{
		{"role": "user", "content": "Hello"},
		{"role": "assistant", "content": "Hi there!"},
		{"role": "user", "content": "How are you?"},
	}

	messages := i.convertHistory(history)

	if len(messages) != 3 {
		t.Errorf("convertHistory() 返回 %d 条消息, 期望 3", len(messages))
	}

	if messages[0].Role != schema.User {
		t.Errorf("messages[0].Role = %v, 期望 User", messages[0].Role)
	}

	if messages[1].Role != schema.Assistant {
		t.Errorf("messages[1].Role = %v, 期望 Assistant", messages[1].Role)
	}

	if messages[0].Content != "Hello" {
		t.Errorf("messages[0].Content = %q, 期望 Hello", messages[0].Content)
	}
}

// TestInterruptible_ConvertHistory_MultiPartContent 测试多模态内容转换
func TestInterruptible_ConvertHistory_MultiPartContent(t *testing.T) {
	i := &interruptible{
		logger: zap.NewNop(),
	}

	// 模拟包含图片的历史消息
	imageURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	history := []map[string]any{
		{
			"role":    "user",
			"content": []map[string]any{
				{"type": "image_url", "image_url": map[string]any{"url": imageURL}},
				{"type": "text", "text": "这是什么图片？"},
			},
		},
		{"role": "assistant", "content": "这是一张简单的图片。"},
		{"role": "user", "content": "接下来请分析一下"},
	}

	messages := i.convertHistory(history)

	if len(messages) != 3 {
		t.Fatalf("convertHistory() 返回 %d 条消息, 期望 3", len(messages))
	}

	// 验证第一条带图片的消息
	if messages[0].Role != schema.User {
		t.Errorf("messages[0].Role = %v, 期望 User", messages[0].Role)
	}

	// 多模态内容应该在 UserInputMultiContent 中
	if len(messages[0].UserInputMultiContent) != 2 {
		t.Errorf("messages[0].UserInputMultiContent 长度 = %d, 期望 2", len(messages[0].UserInputMultiContent))
	}

	if messages[0].UserInputMultiContent[0].Type != schema.ChatMessagePartTypeImageURL {
		t.Errorf("第一部分类型 = %v, 期望 ImageURL", messages[0].UserInputMultiContent[0].Type)
	}

	if messages[0].UserInputMultiContent[1].Type != schema.ChatMessagePartTypeText {
		t.Errorf("第二部分类型 = %v, 期望 Text", messages[0].UserInputMultiContent[1].Type)
	}

	if messages[0].UserInputMultiContent[1].Text != "这是什么图片？" {
		t.Errorf("第二部分文本 = %q, 期望 这是什么图片？", messages[0].UserInputMultiContent[1].Text)
	}

	// 验证第二条纯文本消息
	if messages[1].Role != schema.Assistant {
		t.Errorf("messages[1].Role = %v, 期望 Assistant", messages[1].Role)
	}

	if messages[1].Content != "这是一张简单的图片。" {
		t.Errorf("messages[1].Content = %q, 期望 这是一张简单的图片。", messages[1].Content)
	}

	// 验证第三条纯文本消息
	if messages[2].Role != schema.User {
		t.Errorf("messages[2].Role = %v, 期望 User", messages[2].Role)
	}

	if messages[2].Content != "接下来请分析一下" {
		t.Errorf("messages[2].Content = %q, 期望 接下来请分析一下", messages[2].Content)
	}
}

// TestInterruptible_BuildResumePayload 测试构建恢复参数
func TestInterruptible_BuildResumePayload(t *testing.T) {
	i := &interruptible{
		logger: zap.NewNop(),
	}

	t.Run("AskUser 类型", func(t *testing.T) {
		payload := i.buildResumePayload(true, "user answer")
		if payload == nil {
			t.Error("buildResumePayload(true, ...) 不应该返回 nil")
		}
	})

	t.Run("普通类型", func(t *testing.T) {
		payload := i.buildResumePayload(false, "user answer")
		if payload == nil {
			t.Error("buildResumePayload(false, ...) 不应该返回 nil")
		}

		if m, ok := payload.(map[string]any); ok {
			if m["user_answer"] != "user answer" {
				t.Errorf("payload[user_answer] = %v, 期望 user answer", m["user_answer"])
			}
		} else {
			t.Error("buildResumePayload(false, ...) 应该返回 map[string]any")
		}
	})
}

// TestInterruptibleConfig 测试 interruptibleConfig 结构
func TestInterruptibleConfig(t *testing.T) {
	logger := zap.NewNop()
	cfg := &interruptibleConfig{
		Workspace:     "/test/workspace",
		MaxIterations: 15,
		AgentType:     "master",
		Logger:        logger,
	}

	if cfg.Workspace != "/test/workspace" {
		t.Errorf("interruptibleConfig.Workspace = %q, 期望 /test/workspace", cfg.Workspace)
	}

	if cfg.MaxIterations != 15 {
		t.Errorf("interruptibleConfig.MaxIterations = %d, 期望 15", cfg.MaxIterations)
	}

	if cfg.AgentType != "master" {
		t.Errorf("interruptibleConfig.AgentType = %q, 期望 master", cfg.AgentType)
	}
}

// TestBuildChatModelAdapter 测试包级别的构建函数
func TestBuildChatModelAdapter(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	sessionMgr := session.NewManager(cfg, "/tmp")

	t.Run("无技能加载器", func(t *testing.T) {
		adapter, err := buildChatModelAdapter(logger, cfg, sessionMgr, nil, []string{"read_file"})
		if err == nil {
			if adapter == nil {
				t.Error("buildChatModelAdapter() 不应该返回 nil adapter")
			}
		}
	})

	t.Run("有技能加载器", func(t *testing.T) {
		skillLoader := func(name string) string {
			return "skill: " + name
		}

		adapter, err := buildChatModelAdapter(logger, cfg, sessionMgr, skillLoader, []string{"read_file"})
		if err == nil {
			if adapter == nil {
				t.Error("buildChatModelAdapter() 不应该返回 nil adapter")
			}
		}
	})
}
