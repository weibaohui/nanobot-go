package observers

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/agent/database"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/agent/repository"
	"github.com/weibaohui/nanobot-go/agent/service"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

// SQLiteObserverBuilder SQLiteObserver 构建器
// 封装所有依赖的创建逻辑，简化 main 函数
type SQLiteObserverBuilder struct {
	cfg    *config.Config
	logger *zap.Logger
	filter *observer.ObserverFilter
	err    error

	dbClient     *database.Client
	repo         repository.EventRepository
	convService  service.ConversationService
}

// NewSQLiteObserverBuilder 创建构建器
func NewSQLiteObserverBuilder(cfg *config.Config, logger *zap.Logger) *SQLiteObserverBuilder {
	return &SQLiteObserverBuilder{
		cfg:    cfg,
		logger: logger,
	}
}

// WithFilter 设置过滤器
func (b *SQLiteObserverBuilder) WithFilter(filter *observer.ObserverFilter) *SQLiteObserverBuilder {
	if b.err != nil {
		return b
	}
	b.filter = filter
	return b
}

// initDatabase 初始化数据库相关依赖
func (b *SQLiteObserverBuilder) initDatabase() *SQLiteObserverBuilder {
	if b.err != nil {
		return b
	}

	// 从配置创建数据库配置
	dbConfig := database.NewConfigFromConfig(b.cfg)
	if dbConfig == nil {
		// 数据库未启用，不是错误，只是跳过创建
		return b
	}

	// 创建数据库客户端
	dbClient, err := database.NewClient(dbConfig)
	if err != nil {
		b.err = fmt.Errorf("创建数据库客户端失败: %w", err)
		return b
	}

	// 初始化表结构
	if err := dbClient.InitSchema(); err != nil {
		dbClient.Close()
		b.err = fmt.Errorf("初始化数据库表失败: %w", err)
		return b
	}

	b.dbClient = dbClient
	b.repo = repository.NewEventRepository(dbClient.DB())
	b.convService = service.NewConversationService(b.repo)

	return b
}

// Build 构建 SQLiteObserver
func (b *SQLiteObserverBuilder) Build() (*SQLiteObserver, error) {
	// 初始化数据库依赖
	b.initDatabase()
	if b.err != nil {
		return nil, b.err
	}

	// 如果数据库未启用，返回 nil（不是错误）
	if b.dbClient == nil {
		return nil, nil
	}

	// 创建观察器
	obs, err := NewSQLiteObserver(b.logger, b.filter,
		WithDBClient(b.dbClient),
		WithRepository(&EventRepositoryAdapter{Repo: b.repo}),
		WithConversationService(&ConversationServiceAdapter{Svc: b.convService}),
	)
	if err != nil {
		b.dbClient.Close()
		return nil, err
	}

	return obs, nil
}

// NewSQLiteObserverFromConfig 从配置创建 SQLiteObserver（便捷函数）
// 返回 nil 表示数据库未启用，不是错误
func NewSQLiteObserverFromConfig(cfg *config.Config, logger *zap.Logger, filter *observer.ObserverFilter) (*SQLiteObserver, error) {
	return NewSQLiteObserverBuilder(cfg, logger).
		WithFilter(filter).
		Build()
}
