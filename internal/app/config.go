package app

import (
	"github.com/weibaohui/nanobot-go/config"
)

// CreateDefaultConfig 创建默认配置
func CreateDefaultConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Model:       "",
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			MaxIterations: 15,
		},
		Database: config.DatabaseConfig{
			Enabled: true,
			DBName:  "nanobot.db",
		},
	}
}
