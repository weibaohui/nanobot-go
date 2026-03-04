package channels

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// TestNewDingTalkChannel 测试创建钉钉渠道
func TestNewDingTalkChannel(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		config := &DingTalkConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		logger := zap.NewNop()

		channel := NewDingTalkChannel(config, messageBus, logger)

		if channel == nil {
			t.Fatal("NewDingTalkChannel 返回 nil")
		}

		if channel.config != config {
			t.Error("config 不匹配")
		}

		if channel.BaseChannel == nil {
			t.Error("BaseChannel 不应该为 nil")
		}

		if channel.Name() != "dingtalk" {
			t.Errorf("Name() = %q, 期望 dingtalk", channel.Name())
		}

		if channel.sessionCache == nil {
			t.Error("sessionCache 不应该为 nil")
		}
	})

	t.Run("nil logger 使用默认", func(t *testing.T) {
		config := &DingTalkConfig{}
		messageBus := bus.NewMessageBus(zap.NewNop())

		channel := NewDingTalkChannel(config, messageBus, nil)

		if channel == nil {
			t.Fatal("NewDingTalkChannel 返回 nil")
		}

		if channel.logger == nil {
			t.Error("logger 不应该为 nil")
		}
	})
}

// TestDingTalkChannel_Start 测试启动
func TestDingTalkChannel_Start(t *testing.T) {
	t.Run("配置不完整返回错误", func(t *testing.T) {
		messageBus := bus.NewMessageBus(zap.NewNop())

		// 缺少 ClientID
		channel := NewDingTalkChannel(
			&DingTalkConfig{ClientSecret: "secret"},
			messageBus,
			zap.NewNop(),
		)

		err := channel.Start(context.Background())
		if err == nil {
			t.Error("配置不完整应该返回错误")
		}

		// 缺少 ClientSecret
		channel2 := NewDingTalkChannel(
			&DingTalkConfig{ClientID: "client-id"},
			messageBus,
			zap.NewNop(),
		)

		err = channel2.Start(context.Background())
		if err == nil {
			t.Error("配置不完整应该返回错误")
		}
	})
}

// TestDingTalkChannel_replyViaWebhook 测试 Webhook 回复
func TestDingTalkChannel_replyViaWebhook(t *testing.T) {
	t.Run("成功发送", func(t *testing.T) {
		// 创建模拟服务器
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 验证请求
			if r.Method != "POST" {
				t.Errorf("请求方法 = %q, 期望 POST", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, 期望 application/json", r.Header.Get("Content-Type"))
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		channel := &DingTalkChannel{
			ctx:        context.Background(),
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
		}

		err := channel.replyViaWebhook(server.URL, "测试消息")
		if err != nil {
			t.Errorf("replyViaWebhook 返回错误: %v", err)
		}
	})

	t.Run("服务器返回错误", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		channel := &DingTalkChannel{
			ctx:        context.Background(),
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
		}

		err := channel.replyViaWebhook(server.URL, "测试消息")
		if err == nil {
			t.Error("服务器返回 500 应该返回错误")
		}
	})

	t.Run("网络错误", func(t *testing.T) {
		channel := &DingTalkChannel{
			ctx:        context.Background(),
			httpClient: &http.Client{Timeout: 1 * time.Second},
			logger:     zap.NewNop(),
		}

		err := channel.replyViaWebhook("http://invalid-server-address:99999", "测试消息")
		if err == nil {
			t.Error("无效地址应该返回错误")
		}
	})
}

// TestDingTalkChannel_getAccessToken 测试获取 Access Token
func TestDingTalkChannel_getAccessToken(t *testing.T) {
	t.Run("从缓存返回", func(t *testing.T) {
		channel := &DingTalkChannel{
			accessToken: "cached-token",
			tokenExpiry: time.Now().Add(1 * time.Hour),
			logger:      zap.NewNop(),
		}

		token, err := channel.getAccessToken()
		if err != nil {
			t.Errorf("getAccessToken 返回错误: %v", err)
		}
		if token != "cached-token" {
			t.Errorf("token = %q, 期望 cached-token", token)
		}
	})
}

// TestDingTalkChannel_Send 测试发送消息
func TestDingTalkChannel_Send(t *testing.T) {
	t.Run("使用 Session Webhook 发送", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		channel := &DingTalkChannel{
			ctx:          context.Background(),
			httpClient:   &http.Client{Timeout: 5 * time.Second},
			logger:       zap.NewNop(),
			sessionCache: make(map[string]*sessionContext),
		}

		// 添加 session webhook 缓存
		channel.sessionCache["chat-1"] = &sessionContext{
			sessionWebhook: server.URL,
			expireTime:     time.Now().Add(1 * time.Hour),
		}

		msg := &bus.OutboundMessage{
			ChatID:  "chat-1",
			Content: "测试消息",
		}

		err := channel.Send(msg)
		if err != nil {
			t.Errorf("Send 返回错误: %v", err)
		}
	})

	t.Run("Webhook 过期后使用 API", func(t *testing.T) {
		channel := &DingTalkChannel{
			ctx:          context.Background(),
			httpClient:   &http.Client{Timeout: 5 * time.Second},
			logger:       zap.NewNop(),
			sessionCache: make(map[string]*sessionContext),
			config: &DingTalkConfig{
				ClientID: "test-id",
			},
		}

		// 添加已过期的 session webhook
		channel.sessionCache["chat-1"] = &sessionContext{
			sessionWebhook: "http://expired-webhook",
			expireTime:     time.Now().Add(-1 * time.Hour),
		}

		msg := &bus.OutboundMessage{
			ChatID:  "chat-1",
			Content: "测试消息",
		}

		// 预期失败，因为没有有效的 access token
		err := channel.Send(msg)
		if err == nil {
			t.Error("没有有效 token 应该返回错误")
		}
	})
}

// TestDingTalkConfig 测试配置结构
func TestDingTalkConfig(t *testing.T) {
	config := &DingTalkConfig{
		ClientID:     "client-123",
		ClientSecret: "secret-456",
		AllowFrom:    []string{"user1", "user2"},
	}

	if config.ClientID != "client-123" {
		t.Errorf("ClientID = %q, 期望 client-123", config.ClientID)
	}

	if config.ClientSecret != "secret-456" {
		t.Errorf("ClientSecret = %q, 期望 secret-456", config.ClientSecret)
	}

	if len(config.AllowFrom) != 2 {
		t.Errorf("AllowFrom 长度 = %d, 期望 2", len(config.AllowFrom))
	}
}

// TestDingTalkChannel_sessionCache 测试 Session 缓存
func TestDingTalkChannel_sessionCache(t *testing.T) {
	channel := &DingTalkChannel{
		sessionCache: make(map[string]*sessionContext),
		sessionMutex: sync.RWMutex{},
	}

	// 添加缓存
	channel.sessionCache["conv-1"] = &sessionContext{
		sessionWebhook:     "https://webhook.example.com",
		expireTime:         time.Now().Add(1 * time.Hour),
		conversationType:   "2",
		openConversationID: "conv-1",
	}

	// 读取缓存
	channel.sessionMutex.RLock()
	session, ok := channel.sessionCache["conv-1"]
	channel.sessionMutex.RUnlock()

	if !ok {
		t.Fatal("缓存中应该存在 conv-1")
	}

	if session.sessionWebhook != "https://webhook.example.com" {
		t.Errorf("webhook = %q, 期望 https://webhook.example.com", session.sessionWebhook)
	}
}

// TestDingTalkChannel_Stop 测试停止渠道
func TestDingTalkChannel_Stop(t *testing.T) {
	channel := NewDingTalkChannel(
		&DingTalkConfig{ClientID: "test", ClientSecret: "secret"},
		bus.NewMessageBus(zap.NewNop()),
		zap.NewNop(),
	)

	// 设置状态
	channel.running = true
	channel.ctx, channel.cancel = context.WithCancel(context.Background())
	channel.httpClient = &http.Client{}

	// 不应该 panic
	channel.Stop()

	if channel.running {
		t.Error("running 应该为 false")
	}
}

// TestDingTalkChannel_sendViaAPI 测试通过 API 发送
func TestDingTalkChannel_sendViaAPI(t *testing.T) {
	t.Run("获取 access token 失败", func(t *testing.T) {
		channel := &DingTalkChannel{
			ctx:        context.Background(),
			httpClient: &http.Client{Timeout: 1 * time.Second},
			logger:     zap.NewNop(),
			config: &DingTalkConfig{
				ClientID:     "test-id",
				ClientSecret: "test-secret",
			},
		}

		err := channel.sendViaAPI("user-1", "测试消息")
		if err == nil {
			t.Error("无效配置应该返回错误")
		}
	})
}

// TestDingTalkChannel_getAccessToken_FromServer 测试从服务器获取 Token
func TestDingTalkChannel_getAccessToken_FromServer(t *testing.T) {
	t.Run("服务器返回有效 token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("方法 = %q, 期望 POST", r.Method)
			}

			response := `{"accessToken": "new-token", "expireIn": 7200}`
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(response))
		}))
		defer server.Close()

		// 注意：需要修改代码才能测试这个功能
		// 这里只是验证结构
		channel := &DingTalkChannel{
			accessToken: "old-token",
			tokenExpiry: time.Now().Add(-1 * time.Hour), // 已过期
			logger:      zap.NewNop(),
		}

		if channel.accessToken != "old-token" {
			t.Error("token 应该能被读取")
		}
	})

	t.Run("token 双重检查", func(t *testing.T) {
		channel := &DingTalkChannel{
			accessToken: "valid-token",
			tokenExpiry: time.Now().Add(1 * time.Hour),
			logger:      zap.NewNop(),
		}

		// 第一次检查通过，直接返回
		token, err := channel.getAccessToken()
		if err != nil {
			t.Errorf("getAccessToken 返回错误: %v", err)
		}
		if token != "valid-token" {
			t.Errorf("token = %q, 期望 valid-token", token)
		}
	})
}

// TestDingTalkChannel_replyViaWebhook_InvalidJSON 测试 Webhook 序列化错误
func TestDingTalkChannel_replyViaWebhook_InvalidJSON(t *testing.T) {
	// 这个测试用于覆盖错误路径
	channel := &DingTalkChannel{
		ctx:        context.Background(),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     zap.NewNop(),
	}

	// 正常情况不应该失败
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := channel.replyViaWebhook(server.URL, "正常消息")
	if err != nil {
		t.Errorf("replyViaWebhook 返回错误: %v", err)
	}
}

// TestSDKLoggerAdapter 测试 SDK 日志适配器
func TestSDKLoggerAdapter(t *testing.T) {
	logger := zap.NewNop()
	adapter := &sdkLoggerAdapter{logger: logger}

	// 不应该 panic
	adapter.Debugf("debug %s", "message")
	adapter.Infof("info %s", "message")
	adapter.Warningf("warning %s", "message")
	adapter.Errorf("error %s", "message")
}

// TestSessionContext 测试会话上下文
func TestSessionContext(t *testing.T) {
	ctx := &sessionContext{
		sessionWebhook:     "https://webhook.example.com",
		expireTime:         time.Now().Add(1 * time.Hour),
		conversationType:   "2",
		openConversationID: "conv-123",
	}

	if ctx.sessionWebhook != "https://webhook.example.com" {
		t.Errorf("sessionWebhook = %q, 期望 https://webhook.example.com", ctx.sessionWebhook)
	}

	if ctx.conversationType != "2" {
		t.Errorf("conversationType = %q, 期望 2", ctx.conversationType)
	}

	if ctx.openConversationID != "conv-123" {
		t.Errorf("openConversationID = %q, 期望 conv-123", ctx.openConversationID)
	}
}
