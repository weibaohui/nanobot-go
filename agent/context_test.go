package agent

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// TestNewContextBuilder 测试创建上下文构建器
func TestNewContextBuilder(t *testing.T) {
	tmpDir := t.TempDir()

	builder := NewContextBuilder(tmpDir)
	if builder == nil {
		t.Fatal("NewContextBuilder 返回 nil")
	}

	if builder.workspace != tmpDir {
		t.Errorf("workspace = %q, 期望 %q", builder.workspace, tmpDir)
	}
}

// TestContextBuilder_GetSkillsLoader 测试获取技能加载器
func TestContextBuilder_GetSkillsLoader(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	loader := builder.GetSkillsLoader()
	if loader == nil {
		t.Error("GetSkillsLoader 不应该返回 nil")
	}
}

// TestContextBuilder_BuildSystemPrompt 测试构建系统提示
func TestContextBuilder_BuildSystemPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	prompt := builder.BuildSystemPrompt()
	if prompt == "" {
		t.Error("BuildSystemPrompt 不应该返回空字符串")
	}

	if !contains(prompt, "nanobot") {
		t.Error("系统提示应该包含 nanobot")
	}
}

// TestContextBuilder_loadBootstrapFiles 测试加载引导文件
func TestContextBuilder_loadBootstrapFiles(t *testing.T) {
	t.Run("无引导文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := NewContextBuilder(tmpDir)

		content := builder.loadBootstrapFiles()
		if content != "" {
			t.Errorf("无引导文件时应该返回空字符串, 得到: %q", content)
		}
	})

	t.Run("有引导文件", func(t *testing.T) {
		tmpDir := t.TempDir()

		agnetsContent := "# Agents\n这是 AGENTS.md 内容"
		os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(agnetsContent), 0644)

		builder := NewContextBuilder(tmpDir)
		content := builder.loadBootstrapFiles()

		if content == "" {
			t.Error("有引导文件时不应该返回空字符串")
		}

		if !contains(content, "AGENTS.md") {
			t.Error("内容应该包含 AGENTS.md")
		}
	})
}

// TestContextBuilder_BuildMessages 测试构建消息列表
func TestContextBuilder_BuildMessages(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	history := []map[string]any{
		{"role": "user", "content": "历史消息"},
		{"role": "assistant", "content": "历史回复"},
	}

	messages := builder.BuildMessages(history, "当前消息", nil, nil, "websocket", "chat-001")

	if len(messages) != 4 {
		t.Errorf("BuildMessages 返回 %d 条消息, 期望 4", len(messages))
	}

	if messages[0]["role"] != "system" {
		t.Error("第一条消息应该是 system 角色")
	}

	if messages[1]["role"] != "user" {
		t.Error("第二条消息应该是 user 角色")
	}

	if messages[2]["role"] != "assistant" {
		t.Error("第三条消息应该是 assistant 角色")
	}

	if messages[3]["role"] != "user" {
		t.Error("第四条消息应该是 user 角色")
	}
}

// TestContextBuilder_BuildMessages_NoHistory 测试构建消息列表（无历史）
func TestContextBuilder_BuildMessages_NoHistory(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	messages := builder.BuildMessages(nil, "当前消息", nil, nil, "", "")

	if len(messages) != 2 {
		t.Errorf("BuildMessages 返回 %d 条消息, 期望 2", len(messages))
	}
}

// TestContextBuilder_buildUserContent 测试构建用户消息内容
func TestContextBuilder_buildUserContent(t *testing.T) {
	t.Run("纯文本", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := NewContextBuilder(tmpDir)

		content := builder.buildUserContent("测试文本", nil)

		if content != "测试文本" {
			t.Errorf("buildUserContent = %v, 期望 测试文本", content)
		}
	})

	t.Run("带图片", func(t *testing.T) {
		tmpDir := t.TempDir()

		imgPath := filepath.Join(tmpDir, "test.png")
		pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		os.WriteFile(imgPath, pngHeader, 0644)

		builder := NewContextBuilder(tmpDir)
		content := builder.buildUserContent("测试文本", []string{imgPath})

		contentSlice, ok := content.([]map[string]any)
		if !ok {
			t.Error("带图片时应该返回切片")
		}

		if len(contentSlice) == 0 {
			t.Error("内容切片不应该为空")
		}
	})
}

// TestContextBuilder_AddToolResult 测试添加工具结果
func TestContextBuilder_AddToolResult(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	messages := []map[string]any{
		{"role": "user", "content": "消息"},
	}

	result := builder.AddToolResult(messages, "call-001", "test_tool", "工具结果")

	if len(result) != 2 {
		t.Errorf("AddToolResult 返回 %d 条消息, 期望 2", len(result))
	}

	if result[1]["role"] != "tool" {
		t.Error("添加的消息应该是 tool 角色")
	}

	if result[1]["tool_call_id"] != "call-001" {
		t.Errorf("tool_call_id = %q, 期望 call-001", result[1]["tool_call_id"])
	}
}

// TestContextBuilder_AddAssistantMessage 测试添加助手消息
func TestContextBuilder_AddAssistantMessage(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	messages := []map[string]any{
		{"role": "user", "content": "消息"},
	}

	result := builder.AddAssistantMessage(messages, "助手回复", nil, "")

	if len(result) != 2 {
		t.Errorf("AddAssistantMessage 返回 %d 条消息, 期望 2", len(result))
	}

	if result[1]["role"] != "assistant" {
		t.Error("添加的消息应该是 assistant 角色")
	}
}

// TestContextBuilder_AddAssistantMessage_WithToolCalls 测试添加带工具调用的助手消息
func TestContextBuilder_AddAssistantMessage_WithToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	messages := []map[string]any{
		{"role": "user", "content": "消息"},
	}

	toolCalls := []map[string]any{
		{"id": "call-001", "name": "test_tool"},
	}

	result := builder.AddAssistantMessage(messages, "助手回复", toolCalls, "推理内容")

	if result[1]["tool_calls"] == nil {
		t.Error("应该包含 tool_calls")
	}

	if result[1]["reasoning_content"] != "推理内容" {
		t.Error("应该包含 reasoning_content")
	}
}

// TestHasBinary 测试检查二进制文件是否存在
func TestHasBinary(t *testing.T) {
	t.Run("存在的命令", func(t *testing.T) {
		result := HasBinary("ls")
		if !result {
			t.Error("ls 命令应该存在")
		}
	})

	t.Run("不存在的命令", func(t *testing.T) {
		result := HasBinary("nonexistent_command_12345")
		if result {
			t.Error("不存在的命令应该返回 false")
		}
	})
}

// TestBuildMessageList 测试构建消息列表
func TestBuildMessageList(t *testing.T) {
	systemPrompt := "系统提示"
	history := []*schema.Message{
		{Role: schema.User, Content: "历史消息"},
		{Role: schema.Assistant, Content: "历史回复"},
	}

	messages := BuildMessageList(systemPrompt, history, "当前输入", "websocket", "chat-001")

	if len(messages) != 4 {
		t.Errorf("BuildMessageList 返回 %d 条消息, 期望 4", len(messages))
	}

	if messages[0].Role != schema.System {
		t.Error("第一条消息应该是 System 角色")
	}

	if messages[1].Role != schema.User {
		t.Error("第二条消息应该是 User 角色")
	}

	if messages[2].Role != schema.Assistant {
		t.Error("第三条消息应该是 Assistant 角色")
	}

	if messages[3].Role != schema.User {
		t.Error("第四条消息应该是 User 角色")
	}
}

// TestBuildMessageList_NoHistory 测试构建消息列表（无历史）
func TestBuildMessageList_NoHistory(t *testing.T) {
	systemPrompt := "系统提示"

	messages := BuildMessageList(systemPrompt, nil, "当前输入", "", "")

	if len(messages) != 2 {
		t.Errorf("BuildMessageList 返回 %d 条消息, 期望 2", len(messages))
	}
}

// TestContextBuilder_getIdentity 测试获取核心身份
func TestContextBuilder_getIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewContextBuilder(tmpDir)

	identity := builder.getIdentity()

	if identity == "" {
		t.Error("getIdentity 不应该返回空字符串")
	}

	if !contains(identity, "nanobot") {
		t.Error("身份应该包含 nanobot")
	}

	if !contains(identity, "当前时间") {
		t.Error("身份应该包含当前时间")
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBase64Encoding 测试 Base64 编码
func TestBase64Encoding(t *testing.T) {
	data := []byte("test data")
	encoded := base64.StdEncoding.EncodeToString(data)

	if encoded == "" {
		t.Error("Base64 编码不应该为空")
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Errorf("Base64 解码失败: %v", err)
	}

	if string(decoded) != "test data" {
		t.Errorf("解码结果 = %q, 期望 test data", string(decoded))
	}
}
