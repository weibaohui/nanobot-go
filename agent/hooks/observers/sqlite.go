package observers

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/agent/models"
	"github.com/weibaohui/nanobot-go/agent/repository"
	"github.com/weibaohui/nanobot-go/agent/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EventRepository 事件仓储接口（用于去重操作）
type EventRepository interface {
	FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.Event, error)
	DeleteByID(ctx context.Context, id uint) error
}

// ConversationService 对话服务接口
type ConversationService interface {
	Create(ctx context.Context, dto *service.ConversationDTO) error
}

// DBClient 数据库客户端接口（用于生命周期管理）
type DBClient interface {
	DBPath() string
	Close() error
	DB() *gorm.DB // 返回底层 DB 连接（供测试等特殊场景使用）
}

// SQLiteObserver SQLite 观察器
// 将消息事件存储到 SQLite 数据库中
type SQLiteObserver struct {
	*observer.BaseObserver
	dbClient     DBClient
	logger       *zap.Logger
	repository   EventRepository
	conversation ConversationService
}

// SQLiteObserverOption SQLiteObserver 构造选项
type SQLiteObserverOption func(*SQLiteObserver) error

// WithRepository 设置自定义仓储实现
func WithRepository(repo EventRepository) SQLiteObserverOption {
	return func(o *SQLiteObserver) error {
		o.repository = repo
		return nil
	}
}

// WithConversationService 设置自定义对话服务实现
func WithConversationService(convService ConversationService) SQLiteObserverOption {
	return func(o *SQLiteObserver) error {
		o.conversation = convService
		return nil
	}
}

// WithDBClient 设置自定义数据库客户端
func WithDBClient(dbClient DBClient) SQLiteObserverOption {
	return func(o *SQLiteObserver) error {
		o.dbClient = dbClient
		return nil
	}
}

// NewSQLiteObserver 创建 SQLite 观察器
// 所有依赖必须通过选项注入，不自动创建任何依赖
func NewSQLiteObserver(logger *zap.Logger, filter *observer.ObserverFilter, opts ...SQLiteObserverOption) (*SQLiteObserver, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	obs := &SQLiteObserver{
		BaseObserver: observer.NewBaseObserver("sqlite", filter),
		logger:       logger,
	}

	// 应用选项
	for _, opt := range opts {
		if opt != nil {
			if err := opt(obs); err != nil {
				return nil, err
			}
		}
	}

	return obs, nil
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

	if o.conversation == nil {
		return fmt.Errorf("ConversationService 未设置，请使用 WithConversationService 选项注入")
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

	// 执行去重
	o.deduplicateRecords(ctx, baseEvent.TraceID, "user", e.UserInput)

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

	if o.conversation == nil {
		return fmt.Errorf("ConversationService 未设置，请使用 WithConversationService 选项注入")
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
	}

	// 提取 Token Usage 信息
	if e.TokenUsage != nil {
		dto.TokenUsage = &service.TokenUsageDTO{
			PromptTokens:     e.TokenUsage.PromptTokens,
			CompletionTokens: e.TokenUsage.CompletionTokens,
			TotalTokens:      e.TokenUsage.TotalTokens,
			ReasoningTokens:  e.TokenUsage.CompletionTokensDetails.ReasoningTokens,
			CachedTokens:     e.TokenUsage.PromptTokenDetails.CachedTokens,
		}
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
	o.deduplicateRecords(ctx, baseEvent.TraceID, role, content)

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

	if o.conversation == nil {
		return fmt.Errorf("ConversationService 未设置，请使用 WithConversationService 选项注入")
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
	o.deduplicateRecords(ctx, baseEvent.TraceID, "tool_result", e.ToolName+": "+content)

	return nil
}

// deduplicateRecords 去重记录
// 对于相同 traceID、role、content 的记录：
// 1. 优先保留有 TokenUsage 信息（total_tokens > 0）的记录
// 2. 如果都没有 TokenUsage 信息，保留 ID 最小的（最早插入的）
func (o *SQLiteObserver) deduplicateRecords(ctx context.Context, traceID, role, content string) {
	if o.repository == nil {
		return
	}

	// 查找相同 traceID、role、content 的所有记录
	records, err := o.repository.FindByTraceIDRoleAndContent(ctx, traceID, role, content)
	if err != nil {
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
			if err := o.repository.DeleteByID(ctx, r.ID); err != nil {
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
	if o.dbClient != nil {
		o.logger.Info("关闭 SQLite 数据库连接", zap.String("db_path", o.dbClient.DBPath()))
		return o.dbClient.Close()
	}
	return nil
}

// GetDBPath 获取数据库文件路径
func (o *SQLiteObserver) GetDBPath() string {
	if o.dbClient != nil {
		return o.dbClient.DBPath()
	}
	return ""
}

// GetDBClient 获取数据库客户端（供其他模块使用）
func (o *SQLiteObserver) GetDBClient() DBClient {
	return o.dbClient
}

// 适配器类型，用于类型转换

// EventRepositoryAdapter 将 repository.EventRepository 适配为 observers.EventRepository
type EventRepositoryAdapter struct {
	Repo repository.EventRepository
}

// FindByTraceIDRoleAndContent 实现 EventRepository 接口
func (a *EventRepositoryAdapter) FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.Event, error) {
	return a.Repo.FindByTraceIDRoleAndContent(ctx, traceID, role, content)
}

// DeleteByID 实现 EventRepository 接口
func (a *EventRepositoryAdapter) DeleteByID(ctx context.Context, id uint) error {
	return a.Repo.DeleteByID(ctx, id)
}

// ConversationServiceAdapter 将 service.ConversationService 适配为 observers.ConversationService
type ConversationServiceAdapter struct {
	Svc service.ConversationService
}

// Create 实现 ConversationService 接口
func (a *ConversationServiceAdapter) Create(ctx context.Context, dto *service.ConversationDTO) error {
	return a.Svc.Create(ctx, dto)
}
