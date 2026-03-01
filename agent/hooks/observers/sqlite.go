package observers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"go.uber.org/zap"
	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动
)

// SQLiteObserver SQLite 观察器
// 将消息事件存储到 SQLite 数据库中
type SQLiteObserver struct {
	*observer.BaseObserver
	db     *sql.DB
	dbPath string
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewSQLiteObserver 创建 SQLite 观察器
func NewSQLiteObserver(dataDir string, logger *zap.Logger, filter *observer.ObserverFilter) (*SQLiteObserver, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 数据库文件路径
	dbPath := filepath.Join(dataDir, "events.db")

	// 打开数据库连接
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite 建议单连接
	db.SetMaxIdleConns(1)

	// 初始化表结构
	if err := initDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库表失败: %w", err)
	}

	logger.Info("SQLite 观察器已初始化", zap.String("db_path", dbPath))

	return &SQLiteObserver{
		BaseObserver: observer.NewBaseObserver("sqlite", filter),
		db:           db,
		dbPath:       dbPath,
		logger:       logger,
	}, nil
}

// initDB 初始化数据库表结构
func initDB(db *sql.DB) error {
	// 创建 events 表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trace_id TEXT NOT NULL,
		span_id TEXT,
		parent_span_id TEXT,
		event_type TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		session_key TEXT,
		role TEXT,
		content TEXT,
		prompt_tokens INTEGER DEFAULT 0,
		completion_tokens INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		reasoning_tokens INTEGER DEFAULT 0,
		cached_tokens INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("创建 events 表失败: %w", err)
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);",
		"CREATE INDEX IF NOT EXISTS idx_events_session_key ON events(session_key);",
		"CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);",
		"CREATE INDEX IF NOT EXISTS idx_events_role ON events(role);",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	return nil
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

	// 插入数据库
	return o.insertEvent(&eventRecord{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   e.SessionKey,
		Role:         "user",
		Content:      e.UserInput,
	})
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
	var promptTokens, completionTokens, totalTokens, reasoningTokens, cachedTokens int
	if e.TokenUsage != nil {
		promptTokens = e.TokenUsage.PromptTokens
		completionTokens = e.TokenUsage.CompletionTokens
		totalTokens = e.TokenUsage.TotalTokens
		reasoningTokens = e.TokenUsage.CompletionTokensDetails.ReasoningTokens
		cachedTokens = e.TokenUsage.PromptTokenDetails.CachedTokens
	}

	// 插入数据库
	return o.insertEvent(&eventRecord{
		TraceID:          baseEvent.TraceID,
		SpanID:           baseEvent.SpanID,
		ParentSpanID:     baseEvent.ParentSpanID,
		EventType:        string(baseEvent.EventType),
		Timestamp:        baseEvent.Timestamp,
		SessionKey:       sessionKey,
		Role:             role,
		Content:          content,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		ReasoningTokens:  reasoningTokens,
		CachedTokens:     cachedTokens,
	})
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

	// 插入数据库
	return o.insertEvent(&eventRecord{
		TraceID:      baseEvent.TraceID,
		SpanID:       baseEvent.SpanID,
		ParentSpanID: baseEvent.ParentSpanID,
		EventType:    string(baseEvent.EventType),
		Timestamp:    baseEvent.Timestamp,
		SessionKey:   sessionKey,
		Role:         "tool_result",
		Content:      e.ToolName + ": " + content,
	})
}

// eventRecord 事件记录
type eventRecord struct {
	TraceID          string
	SpanID           string
	ParentSpanID     string
	EventType        string
	Timestamp        time.Time
	SessionKey       string
	Role             string
	Content          string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	ReasoningTokens  int
	CachedTokens     int
}

// insertEvent 插入事件到数据库
func (o *SQLiteObserver) insertEvent(record *eventRecord) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	insertSQL := `
	INSERT INTO events (trace_id, span_id, parent_span_id, event_type, timestamp, session_key, role, content, prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	result, err := o.db.Exec(insertSQL,
		record.TraceID,
		record.SpanID,
		record.ParentSpanID,
		record.EventType,
		record.Timestamp.Format(time.RFC3339),
		record.SessionKey,
		record.Role,
		record.Content,
		record.PromptTokens,
		record.CompletionTokens,
		record.TotalTokens,
		record.ReasoningTokens,
		record.CachedTokens,
	)

	if err != nil {
		o.logger.Error("插入事件失败",
			zap.Error(err),
			zap.String("event_type", record.EventType),
			zap.String("trace_id", record.TraceID),
		)
		return err
	}

	// 获取新插入记录的 ID
	newID, err := result.LastInsertId()
	if err != nil {
		return nil // 插入成功即可，去重失败不影响
	}

	// 执行去重
	o.deduplicateRecords(record.TraceID, record.Role, record.Content, newID)

	return nil
}

// deduplicateRecords 去重记录
// 对于相同 traceID、role、content 的记录：
// 1. 优先保留有 TokenUsage 信息（total_tokens > 0）的记录
// 2. 如果都没有 TokenUsage 信息，保留 ID 最小的（最早插入的）
func (o *SQLiteObserver) deduplicateRecords(traceID, role, content string, newID int64) {
	// 查找相同 traceID、role、content 的所有记录
	querySQL := `
	SELECT id, total_tokens FROM events
	WHERE trace_id = ? AND role = ? AND content = ?
	ORDER BY id ASC;`

	rows, err := o.db.Query(querySQL, traceID, role, content)
	if err != nil {
		o.logger.Error("查询重复记录失败", zap.Error(err))
		return
	}
	defer rows.Close()

	var ids []int64
	var totalTokens []int
	for rows.Next() {
		var id int64
		var tokens int
		if err := rows.Scan(&id, &tokens); err != nil {
			continue
		}
		ids = append(ids, id)
		totalTokens = append(totalTokens, tokens)
	}

	// 如果只有一条记录，无需去重
	if len(ids) <= 1 {
		return
	}

	// 找出应该保留的记录
	var keepID int64 = -1
	var hasTokenUsage bool = false

	// 优先找有 TokenUsage 的记录
	for i, id := range ids {
		if totalTokens[i] > 0 {
			if !hasTokenUsage {
				// 第一个有 TokenUsage 的记录
				keepID = id
				hasTokenUsage = true
			}
			// 如果已经有有 TokenUsage 的记录，保留 ID 较小的
		}
	}

	// 如果都没有 TokenUsage，保留 ID 最小的
	if !hasTokenUsage {
		keepID = ids[0]
	}

	// 删除其他记录
	for _, id := range ids {
		if id != keepID {
			deleteSQL := `DELETE FROM events WHERE id = ?;`
			if _, err := o.db.Exec(deleteSQL, id); err != nil {
				o.logger.Error("删除重复记录失败", zap.Error(err), zap.Int64("id", id))
			} else {
				o.logger.Debug("删除重复记录",
					zap.Int64("deleted_id", id),
					zap.Int64("kept_id", keepID),
					zap.String("trace_id", traceID),
					zap.String("role", role),
				)
			}
		}
	}
}

// Close 关闭数据库连接
func (o *SQLiteObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.db != nil {
		o.logger.Info("关闭 SQLite 数据库连接", zap.String("db_path", o.dbPath))
		return o.db.Close()
	}
	return nil
}

// GetDBPath 获取数据库文件路径
func (o *SQLiteObserver) GetDBPath() string {
	return o.dbPath
}