package observers

import (
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/conversation/database"
	"github.com/weibaohui/nanobot-go/conversation/repository"
	"github.com/weibaohui/nanobot-go/conversation/service"
	"go.uber.org/zap"
)

// NewSQLiteObserverFromConfig 从配置创建 SQLiteObserver
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

	repo := repository.NewConversationRecordRepository(dbClient.DB())
	convService := service.NewConversationService(repo)

	return NewSQLiteObserver(logger, filter,
		WithDBClient(dbClient),
		WithDedupRepository(repo),
		WithConversationCreator(convService),
	), nil
}
