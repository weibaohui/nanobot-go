package channels

import (
	"fmt"
	"testing"

	"github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/stretchr/testify/assert"
	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

func TestNewFeishuChannel(t *testing.T) {
	tests := []struct {
		name   string
		config *FeishuConfig
		logger *zap.Logger
	}{
		{
			name: "正常创建",
			config: &FeishuConfig{
				AppID:     "test-app-id",
				AppSecret: "test-app-secret",
			},
			logger: zap.NewNop(),
		},
		{
			name: "nil logger 使用默认",
			config: &FeishuConfig{
				AppID:     "test-app-id",
				AppSecret: "test-app-secret",
			},
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageBus := bus.NewMessageBus(nil)
			channel := NewFeishuChannel(tt.config, messageBus, tt.logger)
			assert.NotNil(t, channel)
			assert.Equal(t, "feishu", channel.Name())
			assert.Equal(t, tt.config, channel.config)
			assert.NotNil(t, channel.processedMsgIDs)
		})
	}
}

func TestFeishuChannel_Start(t *testing.T) {
	tests := []struct {
		name    string
		config  *FeishuConfig
		wantErr bool
	}{
		{
			name:    "配置不完整返回错误",
			config:  &FeishuConfig{},
			wantErr: true,
		},
		{
			name: "缺少 AppID",
			config: &FeishuConfig{
				AppSecret: "test-secret",
			},
			wantErr: true,
		},
		{
			name: "缺少 AppSecret",
			config: &FeishuConfig{
				AppID: "test-app-id",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageBus := bus.NewMessageBus(nil)
			channel := NewFeishuChannel(tt.config, messageBus, zap.NewNop())
			err := channel.Start(nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// 正常启动会阻塞，这里只测试配置检查
				// 实际测试需要模拟 WebSocket 连接
			}
		})
	}
}

func TestFeishuChannel_Stop(t *testing.T) {
	messageBus := bus.NewMessageBus(nil)
	config := &FeishuConfig{
		AppID:     "test-app-id",
		AppSecret: "test-app-secret",
	}
	channel := NewFeishuChannel(config, messageBus, zap.NewNop())

	// 测试停止（即使未启动也不应 panic）
	channel.Stop()
	assert.False(t, channel.running)
}

func TestSyncMap(t *testing.T) {
	t.Run("添加和检查", func(t *testing.T) {
		m := newSyncMap(100)
		
		// 第一次添加应成功
		assert.True(t, m.add("key1"))
		
		// 重复添加应失败
		assert.False(t, m.add("key1"))
		
		// 新 key 添加应成功
		assert.True(t, m.add("key2"))
	})

	t.Run("自动清理", func(t *testing.T) {
		m := newSyncMap(10)
		
		// 添加超过限制的元素
		for i := 0; i < 15; i++ {
			m.add(fmt.Sprintf("key%d", i))
		}
		
		// 清理后应还能添加新元素
		assert.True(t, m.add("new_key"))
	})
}

func TestFeishuChannel_parseMessageContent(t *testing.T) {
	messageBus := bus.NewMessageBus(nil)
	channel := NewFeishuChannel(&FeishuConfig{}, messageBus, zap.NewNop())

	tests := []struct {
		name     string
		message  *larkim.EventMessage
		expected string
	}{
		{
			name:     "nil message",
			message:  nil,
			expected: "",
		},
		{
			name: "text message",
			message: &larkim.EventMessage{
				MessageType: strPtr("text"),
				Content:     strPtr(`{"text": "Hello World"}`),
			},
			expected: "Hello World",
		},
		{
			name: "image message",
			message: &larkim.EventMessage{
				MessageType: strPtr("image"),
			},
			expected: "[图片]",
		},
		{
			name: "audio message",
			message: &larkim.EventMessage{
				MessageType: strPtr("audio"),
			},
			expected: "[语音]",
		},
		{
			name: "file message",
			message: &larkim.EventMessage{
				MessageType: strPtr("file"),
			},
			expected: "[文件]",
		},
		{
			name: "sticker message",
			message: &larkim.EventMessage{
				MessageType: strPtr("sticker"),
			},
			expected: "[表情]",
		},
		{
			name: "unknown message type",
			message: &larkim.EventMessage{
				MessageType: strPtr("unknown"),
			},
			expected: "[unknown]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := channel.parseMessageContent(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFeishuChannel_buildCard(t *testing.T) {
	messageBus := bus.NewMessageBus(nil)
	channel := NewFeishuChannel(&FeishuConfig{}, messageBus, zap.NewNop())

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "纯文本",
			content: "Hello World",
		},
		{
			name: "带表格的文本",
			content: `查询结果：
| 名称 | 值 |
|------|------|
| CPU | 80% |
| 内存 | 60% |`,
		},
		{
			name:    "空内容",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := channel.buildCard(tt.content)
			assert.NotNil(t, card)
			assert.Contains(t, card, "config")
			assert.Contains(t, card, "elements")
		})
	}
}

func TestFeishuChannel_parseMarkdownTable(t *testing.T) {
	messageBus := bus.NewMessageBus(nil)
	channel := NewFeishuChannel(&FeishuConfig{}, messageBus, zap.NewNop())

	tests := []struct {
		name     string
		table    string
		hasTable bool
	}{
		{
			name: "有效表格",
			table: `| 名称 | 值 |
|------|------|
| CPU | 80% |
| 内存 | 60% |`,
			hasTable: true,
		},
		{
			name:     "无效表格（行数不足）",
			table:    "| 名称 | 值 |",
			hasTable: false,
		},
		{
			name:     "空表格",
			table:    "",
			hasTable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := channel.parseMarkdownTable(tt.table)
			if tt.hasTable {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestFeishuChannel_splitTableRow(t *testing.T) {
	messageBus := bus.NewMessageBus(nil)
	channel := NewFeishuChannel(&FeishuConfig{}, messageBus, zap.NewNop())

	tests := []struct {
		name     string
		row      string
		expected []string
	}{
		{
			name:     "正常行",
			row:      "| 单元格1 | 单元格2 |",
			expected: []string{"单元格1", "单元格2"},
		},
		{
			name:     "带空格的行",
			row:      "|  单元格1  |  单元格2  |",
			expected: []string{"单元格1", "单元格2"},
		},
		{
			name:     "分隔符行",
			row:      "|------|------|",
			expected: []string{"------", "------"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := channel.splitTableRow(tt.row)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFeishuConfig(t *testing.T) {
	config := FeishuConfig{
		AppID:             "test-app-id",
		AppSecret:         "test-app-secret",
		EncryptKey:        "test-encrypt-key",
		VerificationToken: "test-verification-token",
		AllowFrom:         []string{"user1", "user2"},
	}

	assert.Equal(t, "test-app-id", config.AppID)
	assert.Equal(t, "test-app-secret", config.AppSecret)
	assert.Equal(t, "test-encrypt-key", config.EncryptKey)
	assert.Equal(t, "test-verification-token", config.VerificationToken)
	assert.Equal(t, []string{"user1", "user2"}, config.AllowFrom)
}

// Helper function
func strPtr(s string) *string {
	return &s
}
