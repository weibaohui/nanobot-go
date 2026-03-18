package app

import (
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/database"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
	"github.com/weibaohui/nanobot-go/pkg/session"
	"go.uber.org/zap"
)

// DatabaseComponents 数据库相关组件
type DatabaseComponents struct {
	DB       *database.Client
	ConvRepo session.ConversationRecordRepository
}

// InitDatabase 初始化数据库
func InitDatabase(cfg *config.Config, logger *zap.Logger) *DatabaseComponents {
	dbConfig := database.NewConfigFromConfig(cfg)
	if dbConfig == nil {
		logger.Warn("数据库配置为空，跳过初始化")
		return nil
	}

	dbClient, err := database.NewClient(dbConfig)
	if err != nil {
		logger.Error("初始化数据库失败", zap.Error(err))
		return nil
	}

	if err := dbClient.InitSchema(); err != nil {
		logger.Error("初始化数据库 schema 失败", zap.Error(err))
		dbClient.Close()
		return nil
	}

	convRepo := conversation.NewRepository(dbClient.DB())

	logger.Info("数据库和对话记录仓库已初始化")

	return &DatabaseComponents{
		DB:       dbClient,
		ConvRepo: convRepo,
	}
}

// Close 关闭数据库连接
func (d *DatabaseComponents) Close() {
	if d.DB != nil {
		d.DB.Close()
	}
}
