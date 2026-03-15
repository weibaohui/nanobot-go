package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/utils/pagination"
)

// service 对话服务实现
type service struct {
	repo Repository
}

// NewService 创建服务实例
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetByTraceID(ctx context.Context, traceID string) ([]ConversationDTO, error) {
	if traceID == "" {
		return nil, fmt.Errorf("%w: traceID cannot be empty", ErrInvalidParameter)
	}

	records, err := s.repo.FindByTraceID(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return s.recordsToDTOs(records), nil
}

func (s *service) ListBySessionKey(ctx context.Context, sessionKey string, page, pageSize int) (*ConversationListResult, error) {
	if sessionKey == "" {
		return nil, fmt.Errorf("%w: sessionKey cannot be empty", ErrInvalidParameter)
	}

	page, pageSize = pagination.NormalizeDefault(page, pageSize)

	total, err := s.repo.CountBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	offset := pagination.CalculateOffset(page, pageSize)
	records, err := s.repo.FindBySessionKey(ctx, sessionKey, &models.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return &ConversationListResult{
		Conversations: s.recordsToDTOs(records),
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func (s *service) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) (*ConversationListResult, error) {
	page, pageSize = pagination.NormalizeDefault(page, pageSize)

	total, err := s.repo.CountByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	offset := pagination.CalculateOffset(page, pageSize)
	records, err := s.repo.FindByTimeRange(ctx, startTime, endTime, &models.QueryOptions{
		OrderBy: "timestamp",
		Order:   "ASC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return &ConversationListResult{
		Conversations: s.recordsToDTOs(records),
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func (s *service) ListByUserAndDate(ctx context.Context, userCode string, date string) ([]ConversationDTO, error) {
	if userCode == "" {
		return nil, fmt.Errorf("%w: userCode cannot be empty", ErrInvalidParameter)
	}
	if date == "" {
		return nil, fmt.Errorf("%w: date cannot be empty", ErrInvalidParameter)
	}

	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid date format, expected YYYY-MM-DD: %v", ErrInvalidParameter, err)
	}

	// 计算时间范围（当天的开始和结束）
	startOfDay := targetDate
	endOfDay := targetDate.AddDate(0, 0, 1).Add(-time.Nanosecond)

	// 查询该用户当天的所有对话
	records, err := s.repo.FindByUserCodeAndDate(ctx, userCode, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return s.recordsToDTOs(records), nil
}

func (s *service) ListRecent(ctx context.Context, page, pageSize int) (*ConversationListResult, error) {
	page, pageSize = pagination.NormalizeDefault(page, pageSize)

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	offset := pagination.CalculateOffset(page, pageSize)
	records, err := s.repo.FindByTimeRange(ctx, time.Time{}, time.Now(), &models.QueryOptions{
		OrderBy: "timestamp",
		Order:   "DESC",
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	result := s.recordsToDTOs(records)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return &ConversationListResult{
		Conversations: result,
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func (s *service) Create(ctx context.Context, dto *ConversationDTO) error {
	if dto == nil {
		return fmt.Errorf("%w: dto cannot be nil", ErrInvalidParameter)
	}

	record := s.dtoToRecord(dto)
	if err := s.repo.Create(ctx, &record); err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	dto.ID = record.ID
	return nil
}

func (s *service) CreateBatch(ctx context.Context, dtos []ConversationDTO) error {
	if len(dtos) == 0 {
		return nil
	}

	records := make([]models.ConversationRecord, 0, len(dtos))
	for _, dto := range dtos {
		records = append(records, s.dtoToRecord(&dto))
	}

	if err := s.repo.CreateBatch(ctx, records); err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseOperation, err)
	}

	return nil
}

func (s *service) GetStats(ctx context.Context, req *StatsRequest) (*StatsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request cannot be nil", ErrInvalidParameter)
	}

	tokenStats, err := s.repo.GetTokenStats(ctx, req.StartTime, req.EndTime, req.AgentCodes, req.ChannelCodes, req.Roles)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get token stats: %w", ErrDatabaseOperation, err)
	}

	agentDist, err := s.repo.GetAgentDistribution(ctx, req.StartTime, req.EndTime, req.AgentCodes, req.ChannelCodes, req.Roles)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get agent distribution: %w", ErrDatabaseOperation, err)
	}

	channelDist, err := s.repo.GetChannelDistribution(ctx, req.StartTime, req.EndTime, req.AgentCodes, req.ChannelCodes, req.Roles)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get channel distribution: %w", ErrDatabaseOperation, err)
	}

	roleDist, err := s.repo.GetRoleDistribution(ctx, req.StartTime, req.EndTime, req.AgentCodes, req.ChannelCodes, req.Roles)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get role distribution: %w", ErrDatabaseOperation, err)
	}

	sessionStats, err := s.repo.GetSessionStats(ctx, req.StartTime, req.EndTime, req.AgentCodes, req.ChannelCodes, req.Roles)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get session stats: %w", ErrDatabaseOperation, err)
	}

	return &StatsResponse{
		TokenStats:          *tokenStats,
		AgentDistribution:   agentDist,
		ChannelDistribution: channelDist,
		RoleDistribution:    roleDist,
		SessionStats:        *sessionStats,
	}, nil
}

func (s *service) recordsToDTOs(records []models.ConversationRecord) []ConversationDTO {
	dtos := make([]ConversationDTO, 0, len(records))
	for _, record := range records {
		dtos = append(dtos, s.recordToDTO(&record))
	}
	return dtos
}

func (s *service) recordToDTO(record *models.ConversationRecord) ConversationDTO {
	dto := ConversationDTO{
		ID:           record.ID,
		TraceID:      record.TraceID,
		SpanID:       record.SpanID,
		ParentSpanID: record.ParentSpanID,
		EventType:    record.EventType,
		Timestamp:    record.Timestamp,
		SessionKey:   record.SessionKey,
		Role:         record.Role,
		Content:      record.Content,
		CreatedAt:    record.CreatedAt,
		UserCode:     record.UserCode,
		AgentCode:    record.AgentCode,
		ChannelCode:  record.ChannelCode,
		ChannelType:  record.ChannelType,
	}

	if record.TotalTokens > 0 || record.PromptTokens > 0 || record.CompletionTokens > 0 {
		dto.TokenUsage = &TokenUsageDTO{
			PromptTokens:     record.PromptTokens,
			CompletionTokens: record.CompletionTokens,
			TotalTokens:      record.TotalTokens,
			ReasoningTokens:  record.ReasoningTokens,
			CachedTokens:     record.CachedTokens,
		}
	}

	return dto
}

func (s *service) dtoToRecord(dto *ConversationDTO) models.ConversationRecord {
	record := models.ConversationRecord{
		ID:           dto.ID,
		TraceID:      dto.TraceID,
		SpanID:       dto.SpanID,
		ParentSpanID: dto.ParentSpanID,
		EventType:    dto.EventType,
		Timestamp:    dto.Timestamp,
		SessionKey:   dto.SessionKey,
		Role:         dto.Role,
		Content:      dto.Content,
		CreatedAt:    dto.CreatedAt,
		UserCode:     dto.UserCode,
		AgentCode:    dto.AgentCode,
		ChannelCode:  dto.ChannelCode,
		ChannelType:  dto.ChannelType,
	}

	if dto.TokenUsage != nil {
		record.PromptTokens = dto.TokenUsage.PromptTokens
		record.CompletionTokens = dto.TokenUsage.CompletionTokens
		record.TotalTokens = dto.TokenUsage.TotalTokens
		record.ReasoningTokens = dto.TokenUsage.ReasoningTokens
		record.CachedTokens = dto.TokenUsage.CachedTokens
	}

	return record
}
