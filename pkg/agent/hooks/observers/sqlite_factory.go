package observers

import (
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
	"github.com/weibaohui/nanobot-go/internal/database"
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

	repo := conversation.NewRepository(dbClient.DB())
	convService := conversation.NewService(repo)

	return NewSQLiteObserver(logger, filter,
		WithDBClient(dbClient),
		WithDedupRepository(repo),
		WithConversationCreator(convService),
	), nil
}
