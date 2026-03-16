package conversation

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// repository 对话记录仓储实现
type repository struct {
	db *gorm.DB
}

// NewRepository 创建仓储实例
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindByID(ctx context.Context, id uint) (*models.ConversationRecord, error) {
	var record models.ConversationRecord
	if err := r.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *repository) FindByTraceID(ctx context.Context, traceID string) ([]models.ConversationRecord, error) {
	var records []models.ConversationRecord
	if err := r.db.WithContext(ctx).
		Where("trace_id = ?", traceID).
		Order("timestamp ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *repository) FindBySessionKey(ctx context.Context, sessionKey string, opts *models.QueryOptions) ([]models.ConversationRecord, error) {
	query := r.db.WithContext(ctx).Where("session_key = ?", sessionKey)

	if opts != nil {
		if len(opts.Roles) > 0 {
			query = query.Where("role IN ?", opts.Roles)
		}
		orderBy := "timestamp"
		if opts.OrderBy != "" {
			orderBy = opts.OrderBy
		}
		order := "ASC"
		if opts.Order != "" {
			order = opts.Order
		}
		query = query.Order(orderBy + " " + order)
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	} else {
		query = query.Order("timestamp ASC")
	}

	var records []models.ConversationRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *repository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, opts *models.QueryOptions) ([]models.ConversationRecord, error) {
	query := r.db.WithContext(ctx).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime)

	if opts != nil {
		if len(opts.Roles) > 0 {
			query = query.Where("role IN ?", opts.Roles)
		}
		orderBy := "timestamp"
		if opts.OrderBy != "" {
			orderBy = opts.OrderBy
		}
		order := "ASC"
		if opts.Order != "" {
			order = opts.Order
		}
		query = query.Order(orderBy + " " + order)
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	} else {
		query = query.Order("timestamp ASC")
	}

	var records []models.ConversationRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *repository) FindByUserCodeAndDate(ctx context.Context, userCode string, startTime, endTime time.Time) ([]models.ConversationRecord, error) {
	var records []models.ConversationRecord
	if err := r.db.WithContext(ctx).
		Where("user_code = ?", userCode).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime).
		Order("timestamp ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *repository) FindByTraceIDRoleAndContent(ctx context.Context, traceID, role, content string) ([]models.ConversationRecord, error) {
	var records []models.ConversationRecord
	if err := r.db.WithContext(ctx).
		Where("trace_id = ? AND role = ? AND content = ?", traceID, role, content).
		Order("id ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *repository) CountBySessionKey(ctx context.Context, sessionKey string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ConversationRecord{}).
		Where("session_key = ?", sessionKey).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) CountByTimeRange(ctx context.Context, startTime, endTime time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ConversationRecord{}).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ConversationRecord{}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) Create(ctx context.Context, record *models.ConversationRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *repository) CreateBatch(ctx context.Context, records []models.ConversationRecord) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(records, 100).Error
}

func (r *repository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ConversationRecord{}, id).Error
}

// buildStatsQuery 构建统计查询的基础条件
func (r *repository) buildStatsQuery(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&models.ConversationRecord{})

	// 使用 datetime() 函数确保时间比较正确，解决 SQLite 字符串比较问题
	// 数据库存储格式 "2026-03-16 18:00:24+08:00" 与查询格式 "2026-03-16T15:00:00Z" 不一致
	if !startTime.IsZero() {
		query = query.Where("datetime(timestamp) >= datetime(?)", startTime.UTC().Format(time.RFC3339))
	}
	if !endTime.IsZero() {
		query = query.Where("datetime(timestamp) <= datetime(?)", endTime.UTC().Format(time.RFC3339))
	}
	if len(agentCodes) > 0 {
		query = query.Where("agent_code IN ?", agentCodes)
	}
	if len(channelCodes) > 0 {
		query = query.Where("channel_code IN ?", channelCodes)
	}
	if len(roles) > 0 {
		query = query.Where("role IN ?", roles)
	}

	return query
}

func (r *repository) GetTokenStats(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) (*TokenStats, error) {
	query := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)

	var result struct {
		TotalPromptTokens     int64
		TotalCompletionTokens int64
		TotalTokens           int64
	}

	if err := query.Select(
		"COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens, " +
			"COALESCE(SUM(completion_tokens), 0) as total_completion_tokens, " +
			"COALESCE(SUM(total_tokens), 0) as total_tokens",
	).Scan(&result).Error; err != nil {
		return nil, err
	}

	stats := &TokenStats{
		TotalPromptTokens:     result.TotalPromptTokens,
		TotalCompletionTokens: result.TotalCompletionTokens,
		TotalTokens:           result.TotalTokens,
		DailyTrends:           []DailyStat{},
	}

	// 查询每日趋势（最近30天）
	type dailyResult struct {
		Date           string
		PromptTokens   int64
		CompleteTokens int64
		TotalTokens    int64
	}

	var dailyResults []dailyResult
	dailyQuery := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)
	if err := dailyQuery.Select(
		"date(timestamp) as date, " +
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
			"COALESCE(SUM(completion_tokens), 0) as complete_tokens, " +
			"COALESCE(SUM(total_tokens), 0) as total_tokens",
	).Group("date(timestamp)").Order("date(timestamp) ASC").Scan(&dailyResults).Error; err != nil {
		return nil, err
	}

	for _, d := range dailyResults {
		stats.DailyTrends = append(stats.DailyTrends, DailyStat{
			Date:           d.Date,
			PromptTokens:   d.PromptTokens,
			CompleteTokens: d.CompleteTokens,
			TotalTokens:    d.TotalTokens,
		})
	}

	return stats, nil
}

func (r *repository) GetAgentDistribution(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) ([]AgentDistribution, error) {
	query := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)

	type result struct {
		AgentCode string
		Count     int64
		Tokens    int64
	}

	var results []result
	// 使用 NULLIF 将空字符串转换为 NULL，确保空字符串和 NULL 被分到同一组
	if err := query.Select(
		"NULLIF(agent_code, '') as agent_code, " +
			"COUNT(*) as count, " +
			"COALESCE(SUM(total_tokens), 0) as tokens",
	).Group("NULLIF(agent_code, '')").Order("count DESC").Scan(&results).Error; err != nil {
		return nil, err
	}

	distributions := make([]AgentDistribution, 0, len(results))
	for _, r := range results {
		distributions = append(distributions, AgentDistribution{
			Code:   r.AgentCode,
			Name:   r.AgentCode, // 名称由上层填充
			Count:  r.Count,
			Tokens: r.Tokens,
		})
	}

	return distributions, nil
}

func (r *repository) GetChannelDistribution(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) ([]ChannelDistribution, error) {
	query := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)

	type result struct {
		ChannelType string
		Count       int64
	}

	var results []result
	if err := query.Select(
		"channel_type, " +
			"COUNT(*) as count",
	).Group("channel_type").Order("count DESC").Scan(&results).Error; err != nil {
		return nil, err
	}

	distributions := make([]ChannelDistribution, 0, len(results))
	for _, r := range results {
		distributions = append(distributions, ChannelDistribution{
			Type:  r.ChannelType,
			Count: r.Count,
		})
	}

	return distributions, nil
}

func (r *repository) GetRoleDistribution(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) ([]RoleDistribution, error) {
	query := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)

	type result struct {
		Role  string
		Count int64
	}

	var results []result
	if err := query.Select(
		"role, " +
			"COUNT(*) as count",
	).Group("role").Order("count DESC").Scan(&results).Error; err != nil {
		return nil, err
	}

	distributions := make([]RoleDistribution, 0, len(results))
	for _, r := range results {
		distributions = append(distributions, RoleDistribution{
			Role:  r.Role,
			Count: r.Count,
		})
	}

	return distributions, nil
}

func (r *repository) GetSessionStats(ctx context.Context, startTime, endTime time.Time, agentCodes, channelCodes, roles []string) (*SessionStats, error) {
	query := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)

	var totalSessions int64
	if err := query.Select("COUNT(DISTINCT session_key)").Scan(&totalSessions).Error; err != nil {
		return nil, err
	}

	// 计算平均消息数
	type sessionMsgResult struct {
		SessionKey string
		MsgCount   int64
	}

	var sessionMsgs []sessionMsgResult
	msgQuery := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)
	if err := msgQuery.Select(
		"session_key, " +
			"COUNT(*) as msg_count",
	).Group("session_key").Scan(&sessionMsgs).Error; err != nil {
		return nil, err
	}

	var totalMessages int64
	for _, s := range sessionMsgs {
		totalMessages += s.MsgCount
	}

	avgMessages := float64(0)
	if totalSessions > 0 {
		avgMessages = float64(totalMessages) / float64(totalSessions)
	}

	// 计算平均响应时间（assistant 消息的平均时间差）
	// 简化为：同一会话中，assistant 消息与前一条消息的时间差
	type timeDiffResult struct {
		SessionKey string
		Timestamp  time.Time
		Role       string
	}

	var records []timeDiffResult
	timeQuery := r.buildStatsQuery(ctx, startTime, endTime, agentCodes, channelCodes, roles)
	if err := timeQuery.Select("session_key, timestamp, role").Order("session_key, timestamp ASC").Scan(&records).Error; err != nil {
		return nil, err
	}

	var totalResponseTime int64
	var responseCount int64

	var lastTime time.Time
	var lastSession string
	for _, rec := range records {
		if rec.SessionKey != lastSession {
			lastSession = rec.SessionKey
			lastTime = rec.Timestamp
			continue
		}
		if rec.Role == "assistant" {
			diff := rec.Timestamp.Sub(lastTime).Milliseconds()
			if diff > 0 && diff < 300000 { // 排除超过5分钟的异常值
				totalResponseTime += diff
				responseCount++
			}
		}
		lastTime = rec.Timestamp
	}

	avgResponseTime := float64(0)
	if responseCount > 0 {
		avgResponseTime = float64(totalResponseTime) / float64(responseCount)
	}

	return &SessionStats{
		TotalSessions:   totalSessions,
		AvgMessages:     avgMessages,
		AvgResponseTime: avgResponseTime,
	}, nil
}
