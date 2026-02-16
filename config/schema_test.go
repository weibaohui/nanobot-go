package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfig 测试默认配置创建
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() 返回 nil")
	}

	// 验证默认值
	if cfg.Agents.Defaults.Workspace != "~/.nanobot/workspace" {
		t.Errorf("默认工作区 = %q, 期望 ~/.nanobot/workspace", cfg.Agents.Defaults.Workspace)
	}

	if cfg.Agents.Defaults.Model != "anthropic/claude-opus-4-5" {
		t.Errorf("默认模型 = %q, 期望 anthropic/claude-opus-4-5", cfg.Agents.Defaults.Model)
	}

	if cfg.Gateway.Port != 18790 {
		t.Errorf("默认端口 = %d, 期望 18790", cfg.Gateway.Port)
	}

	if cfg.Heartbeat.Every != "30m" {
		t.Errorf("默认心跳间隔 = %q, 期望 30m", cfg.Heartbeat.Every)
	}
}

// TestLoadConfig 测试加载配置文件
func TestLoadConfig(t *testing.T) {
	t.Run("加载不存在的文件返回默认配置", func(t *testing.T) {
		cfg, err := LoadConfig("/nonexistent/path/config.json")
		if err != nil {
			t.Errorf("加载不存在的文件应该返回默认配置，但返回错误: %v", err)
		}
		if cfg == nil {
			t.Error("加载不存在的文件应该返回默认配置，但返回 nil")
		}
	})

	t.Run("加载有效配置文件", func(t *testing.T) {
		// 创建临时配置文件
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		configContent := map[string]any{
			"agents": map[string]any{
				"defaults": map[string]any{
					"workspace": "/custom/workspace",
					"model":     "openai/gpt-4",
				},
			},
			"gateway": map[string]any{
				"port": 9000,
			},
		}

		data, _ := json.MarshalIndent(configContent, "", "  ")
		os.WriteFile(configPath, data, 0644)

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() 返回错误: %v", err)
		}

		if cfg.Agents.Defaults.Workspace != "/custom/workspace" {
			t.Errorf("工作区 = %q, 期望 /custom/workspace", cfg.Agents.Defaults.Workspace)
		}

		if cfg.Agents.Defaults.Model != "openai/gpt-4" {
			t.Errorf("模型 = %q, 期望 openai/gpt-4", cfg.Agents.Defaults.Model)
		}

		if cfg.Gateway.Port != 9000 {
			t.Errorf("端口 = %d, 期望 9000", cfg.Gateway.Port)
		}
	})

	t.Run("加载无效JSON返回错误", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.json")

		os.WriteFile(configPath, []byte("invalid json {{{"), 0644)

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("加载无效JSON应该返回错误")
		}
	})
}

// TestSaveConfig 测试保存配置文件
func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved_config.json")

	cfg := DefaultConfig()
	cfg.Agents.Defaults.Model = "test-model"
	cfg.Gateway.Port = 8888

	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() 返回错误: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("配置文件未创建")
	}

	// 重新加载验证
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("重新加载配置返回错误: %v", err)
	}

	if loaded.Agents.Defaults.Model != "test-model" {
		t.Errorf("模型 = %q, 期望 test-model", loaded.Agents.Defaults.Model)
	}

	if loaded.Gateway.Port != 8888 {
		t.Errorf("端口 = %d, 期望 8888", loaded.Gateway.Port)
	}
}

// TestGetWorkspacePath 测试配置的 GetWorkspacePath 方法
func TestConfig_GetWorkspacePath(t *testing.T) {
	cfg := DefaultConfig()
	path := cfg.GetWorkspacePath()

	if path == "" {
		t.Error("GetWorkspacePath() 返回空路径")
	}
}

// TestGetProvider 测试获取提供商配置
func TestConfig_GetProvider(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		setupConfig   func(*Config)
		expectNil     bool
		expectAPIKey  string
		expectAPIBase string
	}{
		{
			name:         "空模型使用默认模型",
			model:        "",
			setupConfig:  func(c *Config) {},
			expectNil:    true, // 默认配置没有 API key
		},
		{
			name: "匹配OpenAI模型",
			model: "openai/gpt-4",
			setupConfig: func(c *Config) {
				c.Providers.OpenAI.APIKey = "openai-key"
			},
			expectNil:    false,
			expectAPIKey: "openai-key",
		},
		{
			name: "匹配GPT关键词",
			model: "gpt-4-turbo",
			setupConfig: func(c *Config) {
				c.Providers.OpenAI.APIKey = "gpt-key"
			},
			expectNil:    false,
			expectAPIKey: "gpt-key",
		},
		{
			name: "匹配SiliconFlow模型",
			model: "siliconflow/model",
			setupConfig: func(c *Config) {
				c.Providers.SiliconFlow.APIKey = "silicon-key"
			},
			expectNil:    false,
			expectAPIKey: "silicon-key",
		},
		{
			name: "匹配SiliconFlow前缀Qwen",
			model: "Qwen/Qwen2.5-72B",
			setupConfig: func(c *Config) {
				c.Providers.SiliconFlow.APIKey = "qwen-key"
			},
			expectNil:    false,
			expectAPIKey: "qwen-key",
		},
		{
			name: "匹配SiliconFlow前缀deepseek",
			model: "deepseek-ai/deepseek-v3",
			setupConfig: func(c *Config) {
				c.Providers.SiliconFlow.APIKey = "deepseek-key"
			},
			expectNil:    false,
			expectAPIKey: "deepseek-key",
		},
		{
			name: "回退到有APIKey的提供商",
			model: "unknown-model",
			setupConfig: func(c *Config) {
				c.Providers.SiliconFlow.APIKey = "fallback-key"
			},
			expectNil:    false,
			expectAPIKey: "fallback-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setupConfig(cfg)

			provider := cfg.GetProvider(tt.model)

			if tt.expectNil {
				if provider != nil {
					t.Errorf("期望返回 nil，但返回了 %+v", provider)
				}
				return
			}

			if provider == nil {
				t.Fatal("期望返回非 nil，但返回了 nil")
			}

			if provider.APIKey != tt.expectAPIKey {
				t.Errorf("APIKey = %q, 期望 %q", provider.APIKey, tt.expectAPIKey)
			}

			if tt.expectAPIBase != "" && provider.APIBase != tt.expectAPIBase {
				t.Errorf("APIBase = %q, 期望 %q", provider.APIBase, tt.expectAPIBase)
			}
		})
	}
}

// TestGetAPIKey 测试获取 API Key
func TestConfig_GetAPIKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Providers.OpenAI.APIKey = "test-api-key"

	key := cfg.GetAPIKey("openai/gpt-4")
	if key != "test-api-key" {
		t.Errorf("GetAPIKey() = %q, 期望 test-api-key", key)
	}

	// 测试回退机制：当没有匹配的提供商时，会回退到有 API key 的提供商
	key = cfg.GetAPIKey("unknown/model")
	if key != "test-api-key" {
		t.Errorf("未知模型应该回退到有 API key 的提供商，但返回 %q", key)
	}

	// 测试没有任何提供商配置时返回空字符串
	cfg2 := DefaultConfig()
	key = cfg2.GetAPIKey("unknown/model")
	if key != "" {
		t.Errorf("没有配置 API key 时应该返回空字符串，但返回 %q", key)
	}
}

// TestGetAPIBase 测试获取 API Base URL
func TestConfig_GetAPIBase(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Providers.OpenAI.APIKey = "test-key"
	cfg.Providers.OpenAI.APIBase = "https://api.custom.com"

	base := cfg.GetAPIBase("openai/gpt-4")
	if base != "https://api.custom.com" {
		t.Errorf("GetAPIBase() = %q, 期望 https://api.custom.com", base)
	}
}

// TestProviderConfig 测试提供商配置
func TestProviderConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 设置多个提供商
	cfg.Providers.OpenAI = ProviderConfig{
		APIKey:  "openai-key",
		APIBase: "https://api.openai.com/v1",
		ExtraHeaders: map[string]string{
			"X-Custom": "value",
		},
	}

	provider := cfg.GetProvider("openai/gpt-4")
	if provider == nil {
		t.Fatal("获取 OpenAI 提供商失败")
	}

	if provider.APIKey != "openai-key" {
		t.Errorf("APIKey = %q, 期望 openai-key", provider.APIKey)
	}

	if provider.APIBase != "https://api.openai.com/v1" {
		t.Errorf("APIBase = %q, 期望 https://api.openai.com/v1", provider.APIBase)
	}

	if provider.ExtraHeaders["X-Custom"] != "value" {
		t.Errorf("ExtraHeaders[X-Custom] = %q, 期望 value", provider.ExtraHeaders["X-Custom"])
	}
}

// TestHeartbeatConfig 测试心跳配置
func TestHeartbeatConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 验证默认心跳配置
	if cfg.Heartbeat.Every != "30m" {
		t.Errorf("默认心跳间隔 = %q, 期望 30m", cfg.Heartbeat.Every)
	}

	if cfg.Heartbeat.ActiveHours.Start != "09:00" {
		t.Errorf("默认活跃开始时间 = %q, 期望 09:00", cfg.Heartbeat.ActiveHours.Start)
	}

	if cfg.Heartbeat.ActiveHours.End != "18:00" {
		t.Errorf("默认活跃结束时间 = %q, 期望 18:00", cfg.Heartbeat.ActiveHours.End)
	}
}

// TestChannelsConfig 测试渠道配置
func TestChannelsConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 测试 WebSocket 配置
	cfg.Channels.WebSocket = WebSocketConfig{
		Enabled:   true,
		Addr:      ":8088",
		Path:      "/ws",
		AllowFrom: []string{"user1", "user2"},
	}

	if !cfg.Channels.WebSocket.Enabled {
		t.Error("WebSocket 应该启用")
	}

	if len(cfg.Channels.WebSocket.AllowFrom) != 2 {
		t.Errorf("AllowFrom 长度 = %d, 期望 2", len(cfg.Channels.WebSocket.AllowFrom))
	}
}
