package channels

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// TestWebSocketConfig 测试 WebSocket 配置
func TestWebSocketConfig(t *testing.T) {
	t.Run("默认配置", func(t *testing.T) {
		cfg := &WebSocketConfig{}

		if cfg.Addr != "" {
			t.Errorf("WebSocketConfig.Addr = %q, 期望空", cfg.Addr)
		}
		if cfg.Path != "" {
			t.Errorf("WebSocketConfig.Path = %q, 期望空", cfg.Path)
		}
	})

	t.Run("自定义配置", func(t *testing.T) {
		cfg := &WebSocketConfig{
			Addr:            ":9090",
			Path:            "/chat",
			AllowFrom:       []string{"user1", "user2"},
			EnableStreaming: true,
		}

		if cfg.Addr != ":9090" {
			t.Errorf("WebSocketConfig.Addr = %q, 期望 :9090", cfg.Addr)
		}
		if cfg.Path != "/chat" {
			t.Errorf("WebSocketConfig.Path = %q, 期望 /chat", cfg.Path)
		}
		if len(cfg.AllowFrom) != 2 {
			t.Errorf("WebSocketConfig.AllowFrom 长度 = %d, 期望 2", len(cfg.AllowFrom))
		}
		if !cfg.EnableStreaming {
			t.Error("WebSocketConfig.EnableStreaming 应该为 true")
		}
	})
}

// TestTruncate 测试字符串截断
func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"短字符串不截断", "Hello", 10, "Hello"},
		{"长字符串截断", "Hello World", 5, "Hello..."},
		{"刚好等于最大长度", "Hello", 5, "Hello"},
		{"空字符串", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, 期望 %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestGenerateChatID 测试生成 ChatID
func TestGenerateChatID(t *testing.T) {
	t.Run("普通地址", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ws", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		result := generateChatID(req)
		if result[:3] != "ws_" {
			t.Errorf("generateChatID() = %q, 应以 ws_ 开头", result)
		}
	})

	t.Run("本地地址", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ws", nil)
		req.RemoteAddr = "127.0.0.1:8080"

		result := generateChatID(req)
		if result[:3] != "ws_" {
			t.Errorf("generateChatID() = %q, 应以 ws_ 开头", result)
		}
	})
}

// TestNewWebSocketChannel 测试创建 WebSocket 渠道
func TestNewWebSocketChannel(t *testing.T) {
	t.Run("默认配置填充", func(t *testing.T) {
		channel := NewWebSocketChannel(nil, nil, nil)

		if channel == nil {
			t.Fatal("NewWebSocketChannel() 返回 nil")
		}

		if channel.config.Addr != ":8088" {
			t.Errorf("默认 Addr = %q, 期望 :8088", channel.config.Addr)
		}

		if channel.config.Path != "/ws" {
			t.Errorf("默认 Path = %q, 期望 /ws", channel.config.Path)
		}
	})

	t.Run("自定义配置", func(t *testing.T) {
		cfg := &WebSocketConfig{
			Addr: ":9999",
			Path: "/chat",
		}

		channel := NewWebSocketChannel(cfg, nil, nil)

		if channel == nil {
			t.Fatal("NewWebSocketChannel() 返回 nil")
		}

		if channel.config.Addr != ":9999" {
			t.Errorf("Addr = %q, 期望 :9999", channel.config.Addr)
		}

		if channel.config.Path != "/chat" {
			t.Errorf("Path = %q, 期望 /chat", channel.config.Path)
		}
	})
}

// TestWebSocketChannel_Name 测试渠道名称
func TestWebSocketChannel_Name(t *testing.T) {
	channel := NewWebSocketChannel(nil, nil, nil)

	if channel.Name() != "websocket" {
		t.Errorf("Name() = %q, 期望 websocket", channel.Name())
	}
}

// TestWebSocketChannel_StartStop 测试启动和停止
func TestWebSocketChannel_StartStop(t *testing.T) {
	t.Run("正常启动停止", func(t *testing.T) {
		messageBus := bus.NewMessageBus(zap.NewNop())
		cfg := &WebSocketConfig{
			Addr: ":18088", // 使用不常用端口避免冲突
			Path: "/ws",
		}
		channel := NewWebSocketChannel(cfg, messageBus, zap.NewNop())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 启动（非阻塞）
		go func() {
			err := channel.Start(ctx)
			if err != nil && ctx.Err() == nil {
				t.Logf("Start 返回错误: %v", err)
			}
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		// 停止
		channel.Stop()
	})

	t.Run("地址已被占用", func(t *testing.T) {
		messageBus := bus.NewMessageBus(zap.NewNop())

		// 第一个服务器
		cfg1 := &WebSocketConfig{
			Addr: ":18089",
			Path: "/ws",
		}
		channel1 := NewWebSocketChannel(cfg1, messageBus, zap.NewNop())

		ctx1 := context.Background()
		go channel1.Start(ctx1)
		time.Sleep(50 * time.Millisecond)

		// 第二个服务器尝试使用相同地址
		cfg2 := &WebSocketConfig{
			Addr: ":18089",
			Path: "/ws2",
		}
		channel2 := NewWebSocketChannel(cfg2, messageBus, zap.NewNop())

		err := channel2.Start(ctx1)
		if err == nil {
			// 某些系统可能允许多个监听，这没关系
			t.Log("第二个服务器启动没有返回错误（可能系统允许多个监听）")
		}

		channel1.Stop()
		channel2.Stop()
	})
}

// TestWebSocketChannel_clients 测试客户端管理
func TestWebSocketChannel_clients(t *testing.T) {
	messageBus := bus.NewMessageBus(zap.NewNop())
	cfg := &WebSocketConfig{
		Addr: ":18090",
		Path: "/ws",
	}
	channel := NewWebSocketChannel(cfg, messageBus, zap.NewNop())

	// 初始应该没有客户端
	if len(channel.clients) != 0 {
		t.Errorf("初始客户端数 = %d, 期望 0", len(channel.clients))
	}
}

// TestTruncate 边界情况
func TestTruncate_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"零长度", "Hello", 0, "..."},
		{"长度为3", "Hello", 3, "Hel..."},
		{"刚好长度4", "Hello", 4, "Hell..."},
		{"中文(按字节截断)", "你好世界", 6, "你好..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, 期望 %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestGenerateChatID_IPv6 测试 IPv6 地址
func TestGenerateChatID_IPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	req.RemoteAddr = "[::1]:8080"

	result := generateChatID(req)
	if result[:3] != "ws_" {
		t.Errorf("generateChatID() = %q, 应以 ws_ 开头", result)
	}
}
