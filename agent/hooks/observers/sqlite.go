package observers

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/agent/database"
	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/agent/models"
	"github.com/weibaohui/nanobot-go/agent/repository"
	"github.com/weibaohui/nanobot-go/agent/service"
	"go.uber.org/zap"
)

// SQLiteObserver SQLite 观察器
// 将消息事件存储到 SQLite 数据库中
type SQLiteObserver struct {
	*observer.BaseObserver
	dbClient     *database.Client
	logger       *zap.Logger
	conversation service.ConversationService
}

// NewSQLiteObserver 创建 SQLite 观察器
func NewSQLiteObserver(dataDir string, logger *zap.Logger, filter *observer.ObserverFilter) (*SQLiteObserver, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// 创建数据库客户端
	dbClient, err := database.NewClient(&database.Config{
		DataDir: dataDir,
		DBName:  "events.db",
	})
	if err != nil {
		return nil, fmt.Errorf("创建数据库客户端失败: %w", err)
	}

	// 初始化表结构
	if err := dbClient.InitSchema(); err != nil {
		dbClient.Close()
		return nil, fmt.Errorf("初始化数据库表失败: %w", err)
	}

	logger.Info("SQLite 观察器已初始化", zap.String("db_path", dbClient.DBPath()))

	// 创建 Repository 和 Service
	repo := repository.NewEventRepository(dbClient.DB())
	convService := service.NewConversationService(repo)

	return &SQLiteObserver{
		BaseObserver: observer.NewBaseObserver("sqlite", filter),
		dbClient:     dbClient,
		logger:       logger,
		conversation: convService,
	}, nil
}

// OnEvent 处理事件（实现 Observer 接口）
// 处理 PromptSubmitted、LLMCallEnd 和 ToolCompleted 事件
func (o *SQLiteObserver) OnEvent(ctx context.Context, event events.Event) error {
	switch event.GetEventType() {
	case events.EventPromptSubmitted:
		return o.handlePromptSubmitted(ctx, event)
	case events.EventLLMCallEnd:
		return o.handleLLMCallEnd(ctx, event)
	case events.EventToolCompleted:
		return o.handleToolCompleted(ctx, event)
	}
	return nil
}

// handlePromptSubmitted 处理用户提交 Prompt 事件
func (o *SQLiteObserver) handlePromptSubmitted(ctx context.Context, event events.Event) error {
	e, ok := event.(*events.PromptSubmittedEvent)
	if !ok {
		return nil
	}

	if e.UserInput == "" || e.SessionKey == "" {
		return nil
	}

	// 提取通用字段
	baseEvent := event.ToBaseEvent()

	// 创建 DTO
	dto := &service.ConversationDTO{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   e.SessionKey,
		Role:         "user",
		Content:      e.UserInput,
	}

	// 插入数据库
	if err := o.conversation.Create(ctx, dto); err != nil {
		o.logger.Error("插入事件失败",
			zap.Error(err),
			zap.String("event_type", dto.EventType),
			zap.String("trace_id", dto.TraceID),
		)
		return err
	}

	// 执行去重（去重逻辑需要保留，但需要调整实现）
	o.deduplicateRecords(ctx, dto.TraceID, dto.Role, dto.Content)

	return nil
}

// handleLLMCallEnd 处理 LLM 调用结束事件
func (o *SQLiteObserver) handleLLMCallEnd(ctx context.Context, event events.Event) error {
	e, ok := event.(*events.LLMCallEndEvent)
	if !ok {
		return nil
	}

	// 从 context 获取 sessionKey
	sessionKey := getCtxSessionKey(ctx)
	if sessionKey == "" {
		return nil
	}

	// 提取通用字段
	baseEvent := event.ToBaseEvent()

	// 确定 role 和 content
	var role, content string
	if len(e.ToolCalls) > 0 {
		role = "tool"
		// 拼接所有工具调用信息
		for _, tc := range e.ToolCalls {
			content += tc.Function.Name + "(" + tc.Function.Arguments + ") "
		}
	} else {
		role = "assistant"
		content = e.ResponseContent
	}

	// 空内容不保存
	if content == "" {
		return nil
	}

	// 提取 Token Usage 信息
	var tokenUsage *service.TokenUsageDTO
	if e.TokenUsage != nil {
		tokenUsage = &service.TokenUsageDTO{
			PromptTokens:     e.TokenUsage.PromptTokens,
			CompletionTokens: e.TokenUsage.CompletionTokens,
			TotalTokens:      e.TokenUsage.TotalTokens,
			ReasoningTokens:  e.TokenUsage.CompletionTokensDetails.ReasoningTokens,
			CachedTokens:     e.TokenUsage.PromptTokenDetails.CachedTokens,
		}
	}

	// 创建 DTO
	dto := &service.ConversationDTO{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   sessionKey,
		Role:         role,
		Content:      content,
		TokenUsage:   tokenUsage,
	}

	// 插入数据库
	if err := o.conversation.Create(ctx, dto); err != nil {
		o.logger.Error("插入事件失败",
			zap.Error(err),
			zap.String("event_type", dto.EventType),
			zap.String("trace_id", dto.TraceID),
		)
		return err
	}

	// 执行去重
	o.deduplicateRecords(ctx, dto.TraceID, dto.Role, dto.Content)

	return nil
}

// handleToolCompleted 处理工具执行完成事件
func (o *SQLiteObserver) handleToolCompleted(ctx context.Context, event events.Event) error {
	e, ok := event.(*events.ToolCompletedEvent)
	if !ok {
		return nil
	}

	// 从 context 获取 sessionKey
	sessionKey := getCtxSessionKey(ctx)
	if sessionKey == "" {
		return nil
	}

	// 提取通用字段
	baseEvent := event.ToBaseEvent()

	// 工具执行结果
	content := e.Response
	if content == "" {
		content = "(无输出)"
	}

	// 创建 DTO
	dto := &service.ConversationDTO{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   sessionKey,
		Role:         "tool_result",
		Content:      e.ToolName + ": " + content,
	}

	// 插入数据库
	if err := o.conversation.Create(ctx, dto); err != nil {
		o.logger.Error("插入事件失败",
			zap.Error(err),
			zap.String("event_type", dto.EventType),
			zap.String("trace_id", dto.TraceID),
		)
		return err
	}

	// 执行去重
	o.deduplicateRecords(ctx, dto.TraceID, dto.Role, dto.Content)

	return nil
}

// deduplicateRecords 去重记录
// 对于相同 traceID、role、content 的记录：
// 1. 优先保留有 TokenUsage 信息（total_tokens > 0）的记录
// 2. 如果都没有 TokenUsage 信息，保留 ID 最小的（最早插入的）
func (o *SQLiteObserver) deduplicateRecords(ctx context.Context, traceID, role, content string) {
	// 查找相同 traceID、role、content 的所有记录
	var records []models.Event
	if err := o.dbClient.DB().WithContext(ctx).
		Where("trace_id = ? AND role = ? AND content = ?", traceID, role, content).
		Order("id ASC").
		Find(&records).Error; err != nil {
		o.logger.Error("查询重复记录失败", zap.Error(err))
		return
	}

	// 如果只有一条记录，无需去重
	if len(records) <= 1 {
		return
	}

	// 找出应该保留的记录
	var keepID uint
	var hasTokenUsage bool

	// 优先找有 TokenUsage 的记录
	for _, r := range records {
		if r.TotalTokens > 0 {
			if !hasTokenUsage {
				keepID = r.ID
				hasTokenUsage = true
			}
			// 如果已经有有 TokenUsage 的记录，保留 ID 较小的
			if r.ID < keepID {
				keepID = r.ID
			}
		}
	}

	// 如果都没有 TokenUsage，保留 ID 最小的
	if !hasTokenUsage {
		keepID = records[0].ID
	}

	// 删除其他记录
	for _, r := range records {
		if r.ID != keepID {
			if err := o.dbClient.DB().WithContext(ctx).Delete(&r).Error; err != nil {
				o.logger.Error("删除重复记录失败", zap.Error(err), zap.Uint("id", r.ID))
			} else {
				o.logger.Debug("删除重复记录",
					zap.Uint("deleted_id", r.ID),
					zap.Uint("kept_id", keepID),
					zap.String("trace_id", traceID),
					zap.String("role", role),
				)
			}
		}
	}
}

// Close 关闭数据库连接
func (o *SQLiteObserver) Close() error {
	o.logger.Info("关闭 SQLite 数据库连接", zap.String("db_path", o.dbClient.DBPath()))
	return o.dbClient.Close()
}

// GetDBPath 获取数据库文件路径
func (o *SQLiteObserver) GetDBPath() string {
	return o.dbClient.DBPath()
}

// GetConversationService 获取对话服务（供外部查询使用）
func (o *SQLiteObserver) GetConversationService() service.ConversationService {
	return o.conversation
}

// GetDBClient 获取数据库客户端（供其他模块使用）
func (o *SQLiteObserver) GetDBClient() *database.Client {
	return o.dbClient
}