package observers

import (
	"github.com/weibaohui/nanobot-go/agent/database"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/agent/repository"
	"github.com/weibaohui/nanobot-go/agent/service"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// NewSQLiteObserverFromConfig 从配置创建 SQLiteObserver
// 返回 nil 表示数据库未启用
func NewSQLiteObserverFromConfig(cfg *config.Config, logger *zap.Logger, filter *observer.ObserverFilter) (*SQLiteObserver, error) {
	dbConfig := database.NewConfigFromConfig(cfg)
	if dbConfig == nil {
		return nil, nil
	}

	dbClient, err := database.NewClient(dbConfig)
	if err != nil {
		return nil, err
	}

	if err := dbClient.InitSchema(); err != nil {
		dbClient.Close()
		return nil, err
	}

	repo := repository.NewEventRepository(dbClient.DB())
	convService := service.NewConversationService(repo)

	return NewSQLiteObserver(logger, filter,
		WithDBClient(dbClient),
		WithDedupRepository(repo),
		WithConversationCreator(convService),
	), nil
}
