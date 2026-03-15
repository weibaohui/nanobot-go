package observers

import (
	"context"
	"fmt"

	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/trace"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/service/conversation"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DedupRepository 去重仓储接口
type DedupRepository interface {
	FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.ConversationRecord, error)
	DeleteByID(ctx context.Context, id uint) error
}

// ConversationCreator 对话创建接口
type ConversationCreator interface {
	Create(ctx context.Context, dto *conversation.ConversationDTO) error
}

// DBClient 数据库客户端接口
type DBClient interface {
	DBPath() string
	Close() error
	DB() *gorm.DB
}

// SQLiteObserver SQLite 观察器
type SQLiteObserver struct {
	*observer.BaseObserver
	dbClient DBClient
	logger   *zap.Logger
	repo     DedupRepository
	creator  ConversationCreator
}

// SQLiteObserverOption 构造选项
type SQLiteObserverOption func(*SQLiteObserver)

// WithDedupRepository 设置去重仓储
func WithDedupRepository(repo DedupRepository) SQLiteObserverOption {
	return func(o *SQLiteObserver) { o.repo = repo }
}

// WithConversationCreator 设置对话创建服务
func WithConversationCreator(creator ConversationCreator) SQLiteObserverOption {
	return func(o *SQLiteObserver) { o.creator = creator }
}

// WithDBClient 设置数据库客户端
func WithDBClient(dbClient DBClient) SQLiteObserverOption {
	return func(o *SQLiteObserver) { o.dbClient = dbClient }
}

// NewSQLiteObserver 创建 SQLite 观察器
func NewSQLiteObserver(logger *zap.Logger, filter *observer.ObserverFilter, opts ...SQLiteObserverOption) *SQLiteObserver {
	if logger == nil {
		logger = zap.NewNop()
	}

	obs := &SQLiteObserver{
		BaseObserver: observer.NewBaseObserver("sqlite", filter),
		logger:       logger,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(obs)
		}
	}

	return obs
}

// OnEvent 处理事件
func (o *SQLiteObserver) OnEvent(ctx context.Context, event events.Event) error {
	switch event.GetEventType() {
	case events.EventPromptSubmitted:
		return o.handlePromptSubmitted(ctx, event)
	case events.EventLLMCallEnd:
		return o.handleLLMCallEnd(ctx, event)
	case events.EventToolCompleted:
		return o.handleToolCompleted(ctx, event)
	default:
		o.logger.Debug("未处理事件类型", zap.String("event_type", string(event.GetEventType())))
		return nil
	}
}

func (o *SQLiteObserver) handlePromptSubmitted(ctx context.Context, event events.Event) error {
	e, ok := event.(*events.PromptSubmittedEvent)
	if !ok || e.UserInput == "" || e.SessionKey == "" {
		return nil
	}

	if o.creator == nil {
		return fmt.Errorf("ConversationCreator 未注入")
	}

	baseEvent := event.ToBaseEvent()
	dto := &conversation.ConversationDTO{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   e.SessionKey,
		Role:         "user",
		Content:      e.UserInput,
		ChannelCode:  trace.GetChannelCode(ctx),
		AgentCode:    trace.GetAgentCode(ctx),
		UserCode:     trace.GetUserCode(ctx),
	}

	if err := o.creator.Create(ctx, dto); err != nil {
		o.logger.Error("插入对话记录失败", zap.Error(err), zap.String("trace_id", dto.TraceID))
		return err
	}

	return nil
}

func (o *SQLiteObserver) handleLLMCallEnd(ctx context.Context, event events.Event) error {
	e, ok := event.(*events.LLMCallEndEvent)
	if !ok {
		return nil
	}

	sessionKey := getCtxSessionKey(ctx)
	if sessionKey == "" {
		return nil
	}

	if o.creator == nil {
		return fmt.Errorf("ConversationCreator 未注入")
	}

	baseEvent := event.ToBaseEvent()

	var role, content string
	if len(e.ToolCalls) > 0 {
		role = "tool"
		for _, tc := range e.ToolCalls {
			content += tc.Function.Name + "(" + tc.Function.Arguments + ") "
		}
	} else {
		role = "assistant"
		content = e.ResponseContent
	}

	if content == "" {
		return nil
	}

	dto := &conversation.ConversationDTO{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   sessionKey,
		Role:         role,
		Content:      content,
		ChannelCode:  trace.GetChannelCode(ctx),
		AgentCode:    trace.GetAgentCode(ctx),
		UserCode:     trace.GetUserCode(ctx),
	}

	if e.TokenUsage != nil {
		dto.TokenUsage = &conversation.TokenUsageDTO{
			PromptTokens:     e.TokenUsage.PromptTokens,
			CompletionTokens: e.TokenUsage.CompletionTokens,
			TotalTokens:      e.TokenUsage.TotalTokens,
			ReasoningTokens:  e.TokenUsage.CompletionTokensDetails.ReasoningTokens,
			CachedTokens:     e.TokenUsage.PromptTokenDetails.CachedTokens,
		}
	}

	if err := o.creator.Create(ctx, dto); err != nil {
		o.logger.Error("插入对话记录失败", zap.Error(err), zap.String("trace_id", dto.TraceID))
		return err
	}

	return nil
}

func (o *SQLiteObserver) handleToolCompleted(ctx context.Context, event events.Event) error {
	e, ok := event.(*events.ToolCompletedEvent)
	if !ok {
		return nil
	}

	sessionKey := getCtxSessionKey(ctx)
	if sessionKey == "" {
		return nil
	}

	if o.creator == nil {
		return fmt.Errorf("ConversationCreator 未注入")
	}

	baseEvent := event.ToBaseEvent()
	content := e.Response
	if content == "" {
		content = "(无输出)"
	}

	dto := &conversation.ConversationDTO{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   sessionKey,
		Role:         "tool_result",
		Content:      e.ToolName + ": " + content,
		ChannelCode:  trace.GetChannelCode(ctx),
		AgentCode:    trace.GetAgentCode(ctx),
		UserCode:     trace.GetUserCode(ctx),
	}

	if err := o.creator.Create(ctx, dto); err != nil {
		o.logger.Error("插入对话记录失败", zap.Error(err), zap.String("trace_id", dto.TraceID))
		return err
	}

	o.deduplicate(ctx, baseEvent.TraceID, "tool_result", e.ToolName+": "+content)
	return nil
}

func (o *SQLiteObserver) deduplicate(ctx context.Context, traceID, role, content string) {
	if o.repo == nil {
		return
	}

	records, err := o.repo.FindByTraceIDRoleAndContent(ctx, traceID, role, content)
	if err != nil {
		o.logger.Error("查询重复记录失败", zap.Error(err))
		return
	}

	if len(records) <= 1 {
		return
	}

	var keepID uint
	var hasTokenUsage bool

	for _, r := range records {
		if r.TotalTokens > 0 {
			if !hasTokenUsage {
				keepID = r.ID
				hasTokenUsage = true
			}
			if r.ID < keepID {
				keepID = r.ID
			}
		}
	}

	if !hasTokenUsage {
		keepID = records[0].ID
	}

	for _, r := range records {
		if r.ID != keepID {
			if err := o.repo.DeleteByID(ctx, r.ID); err != nil {
				o.logger.Error("删除重复记录失败", zap.Error(err), zap.Uint("id", r.ID))
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

// GetDBClient 获取数据库客户端
func (o *SQLiteObserver) GetDBClient() DBClient {
	return o.dbClient
}

// getCtxSessionKey 从上下文获取 sessionKey
// 使用 trace.GetSessionKey 获取通过 trace.WithSessionKey 注入的 session key
func getCtxSessionKey(ctx context.Context) string {
	return trace.GetSessionKey(ctx)
}
