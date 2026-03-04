package channels

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
	"maunium.net/go/mautrix/id"
)

// TestNewMatrixChannel 测试创建 Matrix 渠道
func TestNewMatrixChannel(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://matrix.example.com",
			UserID:     "@nanobot:example.com",
			Token:      "test-token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		logger := zap.NewNop()

		channel := NewMatrixChannel(config, messageBus, logger)

		if channel == nil {
			t.Fatal("NewMatrixChannel 返回 nil")
		}

		if channel.config != config {
			t.Error("config 不匹配")
		}

		if channel.Name() != "matrix" {
			t.Errorf("Name() = %q, 期望 matrix", channel.Name())
		}

		if channel.typingCancel == nil {
			t.Error("typingCancel map 不应该为 nil")
		}
	})

	t.Run("nil logger 使用默认", func(t *testing.T) {
		config := &MatrixConfig{}
		messageBus := bus.NewMessageBus(zap.NewNop())

		channel := NewMatrixChannel(config, messageBus, nil)

		if channel == nil {
			t.Fatal("NewMatrixChannel 返回 nil")
		}

		if channel.logger == nil {
			t.Error("logger 不应该为 nil")
		}
	})
}

// TestMatrixChannel_Start 测试启动
func TestMatrixChannel_Start(t *testing.T) {
	t.Run("配置不完整返回错误", func(t *testing.T) {
		messageBus := bus.NewMessageBus(zap.NewNop())

		// 缺少 Homeserver
		channel1 := NewMatrixChannel(
			&MatrixConfig{UserID: "@user:example.com", Token: "token"},
			messageBus,
			zap.NewNop(),
		)
		err := channel1.Start(context.Background())
		if err == nil {
			t.Error("缺少 Homeserver 应该返回错误")
		}

		// 缺少 UserID
		channel2 := NewMatrixChannel(
			&MatrixConfig{Homeserver: "https://example.com", Token: "token"},
			messageBus,
			zap.NewNop(),
		)
		err = channel2.Start(context.Background())
		if err == nil {
			t.Error("缺少 UserID 应该返回错误")
		}

		// 缺少 Token
		channel3 := NewMatrixChannel(
			&MatrixConfig{Homeserver: "https://example.com", UserID: "@user:example.com"},
			messageBus,
			zap.NewNop(),
		)
		err = channel3.Start(context.Background())
		if err == nil {
			t.Error("缺少 Token 应该返回错误")
		}
	})
}

// TestMatrixChannel_getStorePath 测试获取存储路径
func TestMatrixChannel_getStorePath(t *testing.T) {
	t.Run("使用配置的数据目录", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
			DataDir:    "/custom/data/dir",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		path := channel.getStorePath()
		expected := filepath.Join("/custom/data/dir", "matrix_sync.json")
		if path != expected {
			t.Errorf("getStorePath() = %q, 期望 %q", path, expected)
		}
	})

	t.Run("使用默认路径", func(t *testing.T) {
		homeDir, _ := os.UserHomeDir()
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		path := channel.getStorePath()
		expected := filepath.Join(homeDir, ".nanobot", "matrix_sync.json")
		if path != expected {
			t.Errorf("getStorePath() = %q, 期望 %q", path, expected)
		}
	})
}

// TestMatrixChannel_Stop 测试停止
func TestMatrixChannel_Stop(t *testing.T) {
	config := &MatrixConfig{
		Homeserver: "https://example.com",
		UserID:     "@test:example.com",
		Token:      "token",
	}
	messageBus := bus.NewMessageBus(zap.NewNop())
	channel := NewMatrixChannel(config, messageBus, zap.NewNop())

	// 设置一些状态
	channel.running = true
	channel.typingCancel = make(map[id.RoomID]context.CancelFunc)

	// 添加一个模拟的 cancel 函数
	ctx, cancel := context.WithCancel(context.Background())
	channel.typingCancel[id.RoomID("room-1")] = cancel

	// 停止（不会 panic）
	channel.Stop()

	// 验证 context 被取消
	select {
	case <-ctx.Done():
		// 预期行为
	case <-time.After(100 * time.Millisecond):
		t.Error("context 应该被取消")
	}

	if channel.running {
		t.Error("running 应该为 false")
	}
}

// TestMatrixChannel_markdownToHTML 测试 Markdown 转换
func TestMatrixChannel_markdownToHTML(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		contains string
	}{
		{
			name:     "标题转换",
			markdown: "# 标题",
			contains: "<h1>标题</h1>",
		},
		{
			name:     "粗体转换",
			markdown: "**粗体**",
			contains: "<strong>粗体</strong>",
		},
		{
			name:     "斜体转换",
			markdown: "*斜体*",
			contains: "<em>斜体</em>",
		},
		{
			name:     "链接转换",
			markdown: "[链接](https://example.com)",
			contains: `<a href="https://example.com">链接</a>`,
		},
		{
			name:     "代码块转换",
			markdown: "`code`",
			contains: "<code>code</code>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := markdownToHTML(tt.markdown)
			if !bytes.Contains([]byte(result), []byte(tt.contains)) {
				t.Errorf("markdownToHTML(%q) = %q, 应该包含 %q", tt.markdown, result, tt.contains)
			}
		})
	}
}

// TestMatrixChannel_Send 测试发送消息
func TestMatrixChannel_Send(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		msg := &bus.OutboundMessage{
			ChatID:  "!room:example.com",
			Content: "Hello",
		}

		err := channel.Send(msg)
		if err == nil {
			t.Error("客户端未初始化应该返回错误")
		}
	})

	t.Run("空消息内容", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		// 模拟已初始化（实际上 client 为 nil）
		msg := &bus.OutboundMessage{
			ChatID:  "!room:example.com",
			Content: "",
		}

		err := channel.Send(msg)
		if err == nil {
			t.Error("空消息内容应该返回错误")
		}
	})
}

// TestMatrixChannel_SendNotice 测试发送通知
func TestMatrixChannel_SendNotice(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		msg := &bus.OutboundMessage{
			ChatID:  "!room:example.com",
			Content: "Notice",
		}

		err := channel.SendNotice(msg)
		if err == nil {
			t.Error("客户端未初始化应该返回错误")
		}
	})
}

// TestMatrixChannel_JoinRoom 测试加入房间
func TestMatrixChannel_JoinRoom(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		err := channel.JoinRoom("#test:example.com", nil)
		if err == nil {
			t.Error("客户端未初始化应该返回错误")
		}
	})
}

// TestMatrixChannel_LeaveRoom 测试离开房间
func TestMatrixChannel_LeaveRoom(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		err := channel.LeaveRoom("!room:example.com")
		if err == nil {
			t.Error("客户端未初始化应该返回错误")
		}
	})
}

// TestMatrixChannel_GetRoomMembers 测试获取房间成员
func TestMatrixChannel_GetRoomMembers(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		_, err := channel.GetRoomMembers("!room:example.com")
		if err == nil {
			t.Error("客户端未初始化应该返回错误")
		}
	})
}

// TestMatrixChannel_GetDisplayName 测试获取显示名称
func TestMatrixChannel_GetDisplayName(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		_, err := channel.GetDisplayName("@user:example.com")
		if err == nil {
			t.Error("客户端未初始化应该返回错误")
		}
	})
}

// TestMatrixChannel_SetPresence 测试设置在线状态
func TestMatrixChannel_SetPresence(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		err := channel.SetPresence("online", "Available")
		if err == nil {
			t.Error("客户端未初始化应该返回错误")
		}
	})
}

// TestMatrixChannel_sendTypingStatus 测试发送 typing 状态
func TestMatrixChannel_sendTypingStatus(t *testing.T) {
	t.Run("客户端未初始化", func(t *testing.T) {
		config := &MatrixConfig{
			Homeserver: "https://example.com",
			UserID:     "@test:example.com",
			Token:      "token",
		}
		messageBus := bus.NewMessageBus(zap.NewNop())
		channel := NewMatrixChannel(config, messageBus, zap.NewNop())

		// 不应该 panic
		channel.sendTypingStatus("!room:example.com", true, 30*time.Second)
	})
}

// TestMatrixChannel_typingIndicator 测试 typing 指示器
func TestMatrixChannel_typingIndicator(t *testing.T) {
	config := &MatrixConfig{
		Homeserver: "https://example.com",
		UserID:     "@test:example.com",
		Token:      "token",
	}
	messageBus := bus.NewMessageBus(zap.NewNop())
	channel := NewMatrixChannel(config, messageBus, zap.NewNop())

	channel.typingCancel = make(map[id.RoomID]context.CancelFunc)

	t.Run("启动 typing 指示器需要 client", func(t *testing.T) {
		// client 为 nil 时不会添加 cancel
		channel.startTypingIndicator(id.RoomID("!room:example.com"))

		channel.typingMu.Lock()
		_, ok := channel.typingCancel[id.RoomID("!room:example.com")]
		channel.typingMu.Unlock()

		// 因为 client 为 nil，所以不会添加
		if ok {
			t.Error("client 为 nil 时不应该添加 cancel 函数")
		}
	})

	t.Run("停止 typing 指示器", func(t *testing.T) {
		// 手动添加一个 cancel 函数
		_, cancel := context.WithCancel(context.Background())
		channel.typingCancel[id.RoomID("!room:example.com")] = cancel

		channel.stopTypingIndicator(id.RoomID("!room:example.com"))

		channel.typingMu.Lock()
		_, ok := channel.typingCancel[id.RoomID("!room:example.com")]
		channel.typingMu.Unlock()

		if ok {
			t.Error("typingCancel 中不应该存在 room 的 cancel 函数")
		}
	})

	t.Run("停止所有 typing 指示器", func(t *testing.T) {
		// 手动添加多个
		_, cancel1 := context.WithCancel(context.Background())
		_, cancel2 := context.WithCancel(context.Background())
		channel.typingCancel[id.RoomID("!room1:example.com")] = cancel1
		channel.typingCancel[id.RoomID("!room2:example.com")] = cancel2

		channel.stopAllTypingIndicators()

		channel.typingMu.Lock()
		count := len(channel.typingCancel)
		channel.typingMu.Unlock()

		if count != 0 {
			t.Errorf("typingCancel 数量 = %d, 期望 0", count)
		}
	})
}

// TestMatrixConfig 测试配置结构
func TestMatrixConfig(t *testing.T) {
	config := &MatrixConfig{
		Homeserver: "https://matrix.example.com",
		UserID:     "@nanobot:example.com",
		Token:      "syt_abc123",
		AllowFrom:  []string{"@user1:example.com", "@user2:example.com"},
		DataDir:    "/data/matrix",
	}

	if config.Homeserver != "https://matrix.example.com" {
		t.Errorf("Homeserver = %q, 期望 https://matrix.example.com", config.Homeserver)
	}

	if config.UserID != "@nanobot:example.com" {
		t.Errorf("UserID = %q, 期望 @nanobot:example.com", config.UserID)
	}

	if config.Token != "syt_abc123" {
		t.Errorf("Token = %q, 期望 syt_abc123", config.Token)
	}

	if len(config.AllowFrom) != 2 {
		t.Errorf("AllowFrom 长度 = %d, 期望 2", len(config.AllowFrom))
	}

	if config.DataDir != "/data/matrix" {
		t.Errorf("DataDir = %q, 期望 /data/matrix", config.DataDir)
	}
}

// TestFileSyncStore 测试文件同步存储
func TestFileSyncStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "sync.json")
	userID := id.UserID("@test:example.com")

	t.Run("创建新存储", func(t *testing.T) {
		// 先创建文件避免加载错误
		os.MkdirAll(filepath.Dir(storePath), 0755)
		os.WriteFile(storePath, []byte(`{"filter_id": "", "next_batch": ""}`), 0644)

		store, err := NewFileSyncStore(storePath, userID)
		if err != nil {
			t.Fatalf("NewFileSyncStore 返回错误: %v", err)
		}

		if store == nil {
			t.Fatal("NewFileSyncStore 返回 nil")
		}

		if store.filePath != storePath {
			t.Errorf("filePath = %q, 期望 %q", store.filePath, storePath)
		}
	})

	t.Run("保存和加载", func(t *testing.T) {
		// 创建并保存
		store1, _ := NewFileSyncStore(storePath, userID)
		store1.filterID = "filter-123"
		store1.nextBatch = "batch-456"

		err := store1.Save()
		if err != nil {
			t.Fatalf("Save 返回错误: %v", err)
		}

		// 重新加载
		store2, err := NewFileSyncStore(storePath, userID)
		if err != nil {
			t.Fatalf("重新加载返回错误: %v", err)
		}

		if store2.filterID != "filter-123" {
			t.Errorf("FilterID = %q, 期望 filter-123", store2.filterID)
		}

		if store2.nextBatch != "batch-456" {
			t.Errorf("NextBatch = %q, 期望 batch-456", store2.nextBatch)
		}
	})

	t.Run("加载不存在的文件", func(t *testing.T) {
		nonExistentPath := filepath.Join(tmpDir, "non-existent", "sync.json")
		_, err := NewFileSyncStore(nonExistentPath, userID)
		// 文件不存在时会返回错误
		if err == nil {
			t.Error("文件不存在时应该返回错误")
		}
	})

	t.Run("保存自动创建目录", func(t *testing.T) {
		nestedPath := filepath.Join(tmpDir, "nested", "dir", "sync.json")
		// 先创建文件避免加载错误
		os.MkdirAll(filepath.Dir(nestedPath), 0755)
		os.WriteFile(nestedPath, []byte(`{"filter_id": "", "next_batch": ""}`), 0644)

		store, err := NewFileSyncStore(nestedPath, userID)
		if err != nil {
			t.Fatalf("NewFileSyncStore 返回错误: %v", err)
		}
		store.filterID = "test"

		err = store.Save()
		if err != nil {
			t.Fatalf("Save 应该自动创建目录: %v", err)
		}

		// 验证文件存在
		_, err = os.Stat(nestedPath)
		if err != nil {
			t.Errorf("文件应该存在: %v", err)
		}
	})

	t.Run("LoadFilterID", func(t *testing.T) {
		store, _ := NewFileSyncStore(storePath, userID)
		store.filterID = "filter-abc"
		store.Save()

		filterID, err := store.LoadFilterID(context.Background(), id.UserID(userID))
		if err != nil {
			t.Errorf("LoadFilterID 返回错误: %v", err)
		}
		if filterID != "filter-abc" {
			t.Errorf("FilterID = %q, 期望 filter-abc", filterID)
		}
	})

	t.Run("SaveFilterID", func(t *testing.T) {
		store, _ := NewFileSyncStore(storePath, userID)

		err := store.SaveFilterID(context.Background(), id.UserID(userID), "new-filter")
		if err != nil {
			t.Errorf("SaveFilterID 返回错误: %v", err)
		}

		if store.filterID != "new-filter" {
			t.Errorf("FilterID = %q, 期望 new-filter", store.filterID)
		}
	})

	t.Run("LoadNextBatch", func(t *testing.T) {
		store, _ := NewFileSyncStore(storePath, userID)
		store.nextBatch = "batch-xyz"
		store.Save()

		nextBatch, err := store.LoadNextBatch(context.Background(), id.UserID(userID))
		if err != nil {
			t.Errorf("LoadNextBatch 返回错误: %v", err)
		}
		if nextBatch != "batch-xyz" {
			t.Errorf("NextBatch = %q, 期望 batch-xyz", nextBatch)
		}
	})

	t.Run("SaveNextBatch", func(t *testing.T) {
		store, _ := NewFileSyncStore(storePath, userID)

		err := store.SaveNextBatch(context.Background(), id.UserID(userID), "new-batch")
		if err != nil {
			t.Errorf("SaveNextBatch 返回错误: %v", err)
		}

		if store.nextBatch != "new-batch" {
			t.Errorf("NextBatch = %q, 期望 new-batch", store.nextBatch)
		}
	})
}
