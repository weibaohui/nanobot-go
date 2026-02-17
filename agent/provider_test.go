package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// TestCreateChatModelConfig 测试创建聊天模型配置
func TestCreateChatModelConfig(t *testing.T) {
	t.Run("配置为空", func(t *testing.T) {
		logger := zap.NewNop()
		apiKey, apiBase, modelName, err := createChatModelConfig(logger, nil)

		if err != ErrNilConfig {
			t.Errorf("createChatModelConfig() error = %v, 期望 %v", err, ErrNilConfig)
		}
		if apiKey != "" {
			t.Errorf("createChatModelConfig() apiKey = %q, 期望空", apiKey)
		}
		if apiBase != "" {
			t.Errorf("createChatModelConfig() apiBase = %q, 期望空", apiBase)
		}
		if modelName != "" {
			t.Errorf("createChatModelConfig() modelName = %q, 期望空", modelName)
		}
	})

	t.Run("配置无APIKey", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &config.Config{}

		apiKey, _, modelName, err := createChatModelConfig(logger, cfg)

		if err != ErrNilAPIKey {
			t.Errorf("createChatModelConfig() error = %v, 期望 %v", err, ErrNilAPIKey)
		}
		if apiKey != "" {
			t.Errorf("createChatModelConfig() apiKey = %q, 期望空", apiKey)
		}
		if modelName != "gpt-4o-mini" {
			t.Errorf("createChatModelConfig() modelName = %q, 期望 gpt-4o-mini", modelName)
		}
	})
}

// TestChatModelAdapter_SetSkillLoader 测试设置技能加载器
func TestChatModelAdapter_SetSkillLoader(t *testing.T) {
	adapter := &ChatModelAdapter{
		logger:        zap.NewNop(),
		registeredMap: make(map[string]bool),
	}

	loader := func(name string) string {
		return "skill content for " + name
	}

	adapter.SetSkillLoader(loader)

	if adapter.skillLoader == nil {
		t.Error("SetSkillLoader() 未设置 skillLoader")
	}
}

// TestChatModelAdapter_SetRegisteredTools 测试设置已注册工具
func TestChatModelAdapter_SetRegisteredTools(t *testing.T) {
	adapter := &ChatModelAdapter{
		logger:        zap.NewNop(),
		registeredMap: make(map[string]bool),
	}

	tools := []string{"read_file", "write_file", "exec"}
	adapter.SetRegisteredTools(tools)

	if len(adapter.registeredMap) != 3 {
		t.Errorf("SetRegisteredTools() registeredMap 长度 = %d, 期望 3", len(adapter.registeredMap))
	}

	for _, tool := range tools {
		if !adapter.registeredMap[tool] {
			t.Errorf("SetRegisteredTools() 工具 %s 未注册", tool)
		}
	}
}

// TestChatModelAdapter_IsRegisteredTool 测试检查工具是否已注册
func TestChatModelAdapter_IsRegisteredTool(t *testing.T) {
	adapter := &ChatModelAdapter{
		logger:        zap.NewNop(),
		registeredMap: map[string]bool{"read_file": true, "write_file": true},
	}

	tests := []struct {
		name     string
		toolName string
		expected bool
	}{
		{"已注册工具", "read_file", true},
		{"已注册工具", "write_file", true},
		{"未注册工具", "exec", false},
		{"空名称", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.isRegisteredTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("isRegisteredTool(%q) = %v, 期望 %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

// TestChatModelAdapter_IsKnownSkill 测试检查是否是已知技能
func TestChatModelAdapter_IsKnownSkill(t *testing.T) {
	t.Run("无技能加载器", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger: zap.NewNop(),
		}

		result := adapter.isKnownSkill("test_skill")
		if result {
			t.Error("isKnownSkill() 无加载器时应返回 false")
		}
	})

	t.Run("有技能加载器", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger: zap.NewNop(),
			skillLoader: func(name string) string {
				if name == "existing_skill" {
					return "skill content"
				}
				return ""
			},
		}

		if !adapter.isKnownSkill("existing_skill") {
			t.Error("isKnownSkill(existing_skill) 应返回 true")
		}

		if adapter.isKnownSkill("nonexistent_skill") {
			t.Error("isKnownSkill(nonexistent_skill) 应返回 false")
		}
	})
}

// TestChatModelAdapter_InterceptToolCall 测试工具调用拦截
func TestChatModelAdapter_InterceptToolCall(t *testing.T) {
	t.Run("已注册工具不拦截", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger:        zap.NewNop(),
			registeredMap: map[string]bool{"read_file": true},
		}

		name, args, err := adapter.interceptToolCall("read_file", `{"path": "/test"}`)
		if err != nil {
			t.Errorf("interceptToolCall() 返回错误: %v", err)
		}
		if name != "read_file" {
			t.Errorf("interceptToolCall() name = %q, 期望 read_file", name)
		}
		if args != `{"path": "/test"}` {
			t.Errorf("interceptToolCall() args = %q, 期望原始值", args)
		}
	})

	t.Run("技能转换为use_skill", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger: zap.NewNop(),
			skillLoader: func(name string) string {
				if name == "my_skill" {
					return "skill content"
				}
				return ""
			},
		}

		name, args, err := adapter.interceptToolCall("my_skill", `{"action": "run", "param": "value"}`)
		if err != nil {
			t.Errorf("interceptToolCall() 返回错误: %v", err)
		}
		if name != "use_skill" {
			t.Errorf("interceptToolCall() name = %q, 期望 use_skill", name)
		}

		var parsedArgs map[string]any
		if err := json.Unmarshal([]byte(args), &parsedArgs); err != nil {
			t.Errorf("interceptToolCall() 返回的参数不是有效 JSON: %v", err)
		}

		if parsedArgs["skill_name"] != "my_skill" {
			t.Errorf("interceptToolCall() skill_name = %v, 期望 my_skill", parsedArgs["skill_name"])
		}
		if parsedArgs["action"] != "run" {
			t.Errorf("interceptToolCall() action = %v, 期望 run", parsedArgs["action"])
		}
	})

	t.Run("既不是工具也不是技能", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger:        zap.NewNop(),
			registeredMap: make(map[string]bool),
		}

		name, args, err := adapter.interceptToolCall("unknown_tool", `{"test": "value"}`)
		if err != nil {
			t.Errorf("interceptToolCall() 返回错误: %v", err)
		}
		if name != "unknown_tool" {
			t.Errorf("interceptToolCall() name = %q, 期望 unknown_tool", name)
		}
		if args != `{"test": "value"}` {
			t.Errorf("interceptToolCall() args = %q, 期望原始值", args)
		}
	})

	t.Run("无效JSON参数", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger: zap.NewNop(),
			skillLoader: func(name string) string {
				return "skill content"
			},
		}

		name, _, err := adapter.interceptToolCall("my_skill", `invalid json`)
		if err != nil {
			t.Errorf("interceptToolCall() 返回错误: %v", err)
		}
		if name != "use_skill" {
			t.Errorf("interceptToolCall() name = %q, 期望 use_skill", name)
		}
	})
}

// TestChatModelAdapter_InterceptToolCalls 测试批量工具调用拦截
func TestChatModelAdapter_InterceptToolCalls(t *testing.T) {
	t.Run("无工具调用", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger: zap.NewNop(),
		}

		msg := &schema.Message{
			Role:    schema.Assistant,
			Content: "Hello",
		}

		adapter.interceptToolCalls(msg)

		if len(msg.ToolCalls) != 0 {
			t.Errorf("interceptToolCalls() 不应该修改无工具调用的消息")
		}
	})

	t.Run("有工具调用", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger:        zap.NewNop(),
			registeredMap: map[string]bool{"read_file": true},
			skillLoader: func(name string) string {
				if name == "my_skill" {
					return "skill content"
				}
				return ""
			},
		}

		msg := &schema.Message{
			Role:    schema.Assistant,
			Content: "",
			ToolCalls: []schema.ToolCall{
				{
					Function: schema.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path": "/test"}`,
					},
				},
				{
					Function: schema.FunctionCall{
						Name:      "my_skill",
						Arguments: `{"action": "run"}`,
					},
				},
			},
		}

		adapter.interceptToolCalls(msg)

		if msg.ToolCalls[0].Function.Name != "read_file" {
			t.Errorf("已注册工具不应被修改")
		}
		if msg.ToolCalls[1].Function.Name != "use_skill" {
			t.Errorf("技能应被转换为 use_skill")
		}
	})
}

// TestChatModelAdapter_RecordTokenUsage 测试记录 Token 用量
func TestChatModelAdapter_RecordTokenUsage(t *testing.T) {
	t.Run("无 session manager", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger: zap.NewNop(),
		}

		msg := &schema.Message{
			Role:    schema.Assistant,
			Content: "Hello",
		}

		adapter.recordTokenUsage(context.Background(), msg)
	})

	t.Run("无 session key", func(t *testing.T) {
		adapter := &ChatModelAdapter{
			logger:   zap.NewNop(),
			sessions: nil,
		}

		ctx := context.Background()
		msg := &schema.Message{
			Role:    schema.Assistant,
			Content: "Hello",
		}

		adapter.recordTokenUsage(ctx, msg)
	})
}

// TestContextKey 测试 ContextKey 类型
func TestContextKey(t *testing.T) {
	key := SessionKeyContextKey
	if string(key) != "session_key" {
		t.Errorf("SessionKeyContextKey = %q, 期望 session_key", key)
	}
}

// containsString 检查字符串是否包含子串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
