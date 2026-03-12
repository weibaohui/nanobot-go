package observers

import (
	"time"

	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/database"
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
	// 使用批量写入服务提高性能
	batchService := service.NewBatchConversationService(convService, logger,
		service.WithBatchSize(50),
		service.WithFlushInterval(100*time.Millisecond),
		service.WithBufferSize(1000),
	)

	return NewSQLiteObserver(logger, filter,
		WithDBClient(dbClient),
		WithDedupRepository(repo),
		WithConversationCreator(batchService),
	), nil
}
