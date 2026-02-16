package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// TestConstants 测试常量
func TestConstants(t *testing.T) {
	if DefaultHeartbeatPrompt == "" {
		t.Error("DefaultHeartbeatPrompt 不应该为空")
	}

	if HeartbeatOKToken == "" {
		t.Error("HeartbeatOKToken 不应该为空")
	}

	if DefaultAckMaxChars <= 0 {
		t.Error("DefaultAckMaxChars 应该大于 0")
	}
}

// TestNewService 测试创建心跳服务
func TestNewService(t *testing.T) {
	cfg := &config.Config{
		Heartbeat: config.HeartbeatConfig{
			Every: "30m",
		},
	}

	service := NewService(zap.NewNop(), cfg, "/tmp", nil)

	if service == nil {
		t.Fatal("NewService 返回 nil")
	}

	if service.cfg == nil {
		t.Error("cfg 不应该为 nil")
	}
}

// TestNewService_NilLogger 测试空 logger 使用默认值
func TestNewService_NilLogger(t *testing.T) {
	cfg := &config.Config{}
	service := NewService(nil, cfg, "/tmp", nil)

	if service == nil {
		t.Fatal("NewService 返回 nil")
	}

	if service.logger == nil {
		t.Error("logger 不应该为 nil")
	}
}

// TestService_HeartbeatFile 测试获取心跳文件路径
func TestService_HeartbeatFile(t *testing.T) {
	cfg := &config.Config{}
	service := NewService(zap.NewNop(), cfg, "/workspace", nil)

	expected := "/workspace/HEARTBEAT.md"
	if service.HeartbeatFile() != expected {
		t.Errorf("HeartbeatFile() = %q, 期望 %q", service.HeartbeatFile(), expected)
	}
}

// TestService_readHeartbeatFile 测试读取心跳文件
func TestService_readHeartbeatFile(t *testing.T) {
	t.Run("文件存在", func(t *testing.T) {
		tmpDir := t.TempDir()
		heartbeatFile := filepath.Join(tmpDir, "HEARTBEAT.md")
		content := "# 任务列表\n- [ ] 任务1\n- [ ] 任务2"
		os.WriteFile(heartbeatFile, []byte(content), 0644)

		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, tmpDir, nil)

		result := service.readHeartbeatFile()
		if result != content {
			t.Errorf("readHeartbeatFile() = %q, 期望 %q", result, content)
		}
	})

	t.Run("文件不存在", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, tmpDir, nil)

		result := service.readHeartbeatFile()
		if result != "" {
			t.Errorf("readHeartbeatFile() = %q, 期望空字符串", result)
		}
	})
}

// TestIsHeartbeatEmpty 测试检查心跳文件是否为空
func TestIsHeartbeatEmpty(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"空内容", "", true},
		{"只有空行", "\n\n\n", true},
		{"只有标题", "# 标题\n## 副标题", true},
		{"只有HTML注释", "<!-- 注释 -->\n<!-- 另一个注释 -->", true},
		{"只有空复选框", "- [ ]\n* [ ]", true},
		{"只有已勾选复选框", "- [x]\n* [x]", true},
		{"有实际内容", "# 标题\n实际任务内容", false},
		{"有其他列表项", "- 普通列表项", false},
		{"混合内容", "# 标题\n- [ ]\n实际内容", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHeartbeatEmpty(tt.content)
			if result != tt.expected {
				t.Errorf("isHeartbeatEmpty() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// TestTruncateResponse 测试截断响应
func TestTruncateResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		maxChars int
		expected string
	}{
		{"短响应不截断", "短响应", 100, "短响应"},
		{"长响应截断", "abcdefghij1234567890", 10, "abcdefghij..."},
		{"刚好等于最大长度", "12345", 5, "12345"},
		{"空响应", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateResponse(tt.response, tt.maxChars)
			if result != tt.expected {
				t.Errorf("truncateResponse() = %q, 期望 %q", result, tt.expected)
			}
		})
	}
}

// TestService_getPrompt 测试获取心跳提示词
func TestService_getPrompt(t *testing.T) {
	t.Run("使用配置的提示词", func(t *testing.T) {
		cfg := &config.Config{
			Heartbeat: config.HeartbeatConfig{
				Prompt: "自定义提示词",
			},
		}
		service := NewService(zap.NewNop(), cfg, "/tmp", nil)

		if service.getPrompt() != "自定义提示词" {
			t.Errorf("getPrompt() = %q, 期望 自定义提示词", service.getPrompt())
		}
	})

	t.Run("使用默认提示词", func(t *testing.T) {
		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, "/tmp", nil)

		if service.getPrompt() != DefaultHeartbeatPrompt {
			t.Errorf("getPrompt() = %q, 期望默认提示词", service.getPrompt())
		}
	})
}

// TestService_getModel 测试获取心跳模型
func TestService_getModel(t *testing.T) {
	cfg := &config.Config{
		Heartbeat: config.HeartbeatConfig{
			Model: "gpt-4",
		},
	}
	service := NewService(zap.NewNop(), cfg, "/tmp", nil)

	if service.getModel() != "gpt-4" {
		t.Errorf("getModel() = %q, 期望 gpt-4", service.getModel())
	}
}

// TestService_getSession 测试获取心跳会话
func TestService_getSession(t *testing.T) {
	cfg := &config.Config{
		Heartbeat: config.HeartbeatConfig{
			Session: "session-001",
		},
	}
	service := NewService(zap.NewNop(), cfg, "/tmp", nil)

	if service.getSession() != "session-001" {
		t.Errorf("getSession() = %q, 期望 session-001", service.getSession())
	}
}

// TestService_getAckMaxChars 测试获取确认消息最大字符数
func TestService_getAckMaxChars(t *testing.T) {
	t.Run("使用配置值", func(t *testing.T) {
		cfg := &config.Config{
			Heartbeat: config.HeartbeatConfig{
				AckMaxChars: 1000,
			},
		}
		service := NewService(zap.NewNop(), cfg, "/tmp", nil)

		if service.getAckMaxChars() != 1000 {
			t.Errorf("getAckMaxChars() = %d, 期望 1000", service.getAckMaxChars())
		}
	})

	t.Run("使用默认值", func(t *testing.T) {
		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, "/tmp", nil)

		if service.getAckMaxChars() != DefaultAckMaxChars {
			t.Errorf("getAckMaxChars() = %d, 期望 %d", service.getAckMaxChars(), DefaultAckMaxChars)
		}
	})
}

// TestService_GetTarget 测试获取心跳目标
func TestService_GetTarget(t *testing.T) {
	cfg := &config.Config{
		Heartbeat: config.HeartbeatConfig{
			Target: "user@example.com",
		},
	}
	service := NewService(zap.NewNop(), cfg, "/tmp", nil)

	if service.GetTarget() != "user@example.com" {
		t.Errorf("GetTarget() = %q, 期望 user@example.com", service.GetTarget())
	}
}

// TestService_TriggerNow 测试手动触发心跳
func TestService_TriggerNow(t *testing.T) {
	t.Run("有回调", func(t *testing.T) {
		cfg := &config.Config{
			Heartbeat: config.HeartbeatConfig{
				Prompt: "测试提示词",
			},
		}

		called := false
		service := NewService(zap.NewNop(), cfg, "/tmp", func(ctx context.Context, cfg *config.Config, prompt string, model string, session string) (string, error) {
			called = true
			return "响应结果", nil
		})

		ctx := context.Background()
		result, err := service.TriggerNow(ctx)

		if err != nil {
			t.Errorf("TriggerNow 返回错误: %v", err)
		}

		if result != "响应结果" {
			t.Errorf("TriggerNow() = %q, 期望 响应结果", result)
		}

		if !called {
			t.Error("回调应该被调用")
		}
	})

	t.Run("无回调", func(t *testing.T) {
		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, "/tmp", nil)

		ctx := context.Background()
		result, err := service.TriggerNow(ctx)

		if err != nil {
			t.Errorf("TriggerNow 返回错误: %v", err)
		}

		if result != "" {
			t.Errorf("TriggerNow() = %q, 期望空字符串", result)
		}
	})
}

// TestService_StartStop 测试启动和停止服务
func TestService_StartStop(t *testing.T) {
	cfg := &config.Config{
		Heartbeat: config.HeartbeatConfig{
			Every: "1h",
		},
	}

	service := NewService(zap.NewNop(), cfg, "/tmp", nil)
	ctx := context.Background()

	err := service.Start(ctx)
	if err != nil {
		t.Errorf("Start 返回错误: %v", err)
	}

	if !service.IsRunning() {
		t.Error("服务应该处于运行状态")
	}

	service.Stop()

	time.Sleep(100 * time.Millisecond)
}

// TestService_IsRunning 测试检查服务运行状态
func TestService_IsRunning(t *testing.T) {
	cfg := &config.Config{
		Heartbeat: config.HeartbeatConfig{
			Every: "1h",
		},
	}

	service := NewService(zap.NewNop(), cfg, "/tmp", nil)

	if service.IsRunning() {
		t.Error("未启动的服务不应该处于运行状态")
	}

	ctx := context.Background()
	_ = service.Start(ctx)

	if !service.IsRunning() {
		t.Error("启动后服务应该处于运行状态")
	}

	service.Stop()
}

// TestService_isInActiveHours 测试检查活跃时段
func TestService_isInActiveHours(t *testing.T) {
	t.Run("无配置默认始终活跃", func(t *testing.T) {
		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, "/tmp", nil)

		if !service.isInActiveHours() {
			t.Error("无配置时应该始终活跃")
		}
	})

	t.Run("配置活跃时段", func(t *testing.T) {
		cfg := &config.Config{
			Heartbeat: config.HeartbeatConfig{
				ActiveHours: config.ActiveHours{
					Start: "00:00",
					End:   "23:59",
				},
			},
		}
		service := NewService(zap.NewNop(), cfg, "/tmp", nil)

		if !service.isInActiveHours() {
			t.Error("全天配置应该始终活跃")
		}
	})
}

// TestService_tick 测试心跳执行
func TestService_tick(t *testing.T) {
	t.Run("心跳文件为空跳过", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, tmpDir, nil)

		ctx := context.Background()
		service.tick(ctx)
	})

	t.Run("有回调执行", func(t *testing.T) {
		tmpDir := t.TempDir()
		heartbeatFile := filepath.Join(tmpDir, "HEARTBEAT.md")
		os.WriteFile(heartbeatFile, []byte("# 任务\n实际内容"), 0644)

		executed := false
		cfg := &config.Config{}
		service := NewService(zap.NewNop(), cfg, tmpDir, func(ctx context.Context, cfg *config.Config, prompt string, model string, session string) (string, error) {
			executed = true
			return "HEARTBEAT_OK", nil
		})

		ctx := context.Background()
		service.tick(ctx)

		if !executed {
			t.Error("回调应该被执行")
		}
	})
}
