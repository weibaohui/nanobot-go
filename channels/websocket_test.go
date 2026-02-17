package channels

import (
	"net/http/httptest"
	"testing"
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
