package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/weibaohui/nanobot-go/internal/memory/models"
	"github.com/weibaohui/nanobot-go/internal/memory/repository"
)

var (
	// ErrMemoryDisabled 记忆模块未启用
	ErrMemoryDisabled = errors.New("memory module is disabled")
	// ErrInvalidMetadata 无效的元数据
	ErrInvalidMetadata = errors.New("invalid metadata")
	// ErrDuplicateTraceID trace_id 已存在
	ErrDuplicateTraceID = errors.New("trace_id already exists")
	// ErrUpgradeFailed 记忆升级失败
	ErrUpgradeFailed = errors.New("memory upgrade failed")
)

// MemoryService 记忆服务接口
type MemoryService interface {
	// WriteMemory 写入流水记忆
	// content: 初步总结内容
	// metadata: 包含 trace_id, session_key, channel_type, event_type 等
	WriteMemory(ctx context.Context, content string, metadata map[string]interface{}) error

	// SearchMemory 搜索记忆（分层检索）
	// 先查近期流水记忆，再查长期记忆，合并返回
	SearchMemory(ctx context.Context, query string, filters models.SearchFilters) (*models.SearchResult, error)

	// UpgradeStreamToLongTerm 将流水记忆升级为长期记忆（定时任务调用）
	UpgradeStreamToLongTerm(ctx context.Context, date string) error

	// GetUnprocessedCount 获取未处理流水记忆数量
	GetUnprocessedCount(ctx context.Context, before time.Time) (int64, error)
}

// memoryService 记忆服务实现
type memoryService struct {
	streamRepo   repository.StreamMemoryRepository
	longTermRepo repository.LongTermMemoryRepository
	summarizer   MemorySummarizer
	enabled      bool
}

// NewMemoryService 创建记忆服务实例
func NewMemoryService(
	streamRepo repository.StreamMemoryRepository,
	longTermRepo repository.LongTermMemoryRepository,
	summarizer MemorySummarizer,
	enabled bool,
) MemoryService {
	return &memoryService{
		streamRepo:   streamRepo,
		longTermRepo: longTermRepo,
		summarizer:   summarizer,
		enabled:      enabled,
	}
}

// WriteMemory 写入流水记忆（已弃用，使用 BuildFromConversations 替代）
// 暂时保留此函数以兼容现有调用，但不再实际写入数据库
func (s *memoryService) WriteMemory(ctx context.Context, content string, metadata map[string]interface{}) error {
	if !s.enabled {
		return ErrMemoryDisabled
	}

	// 新的模型结构下，短期记忆需要从对话列表手动整理
	// 此自动写入方法不再适用，直接返回 nil 避免影响业务流程
	return nil
}

// SearchMemory 搜索记忆（分层检索）
func (s *memoryService) SearchMemory(ctx context.Context, query string, filters models.SearchFilters) (*models.SearchResult, error) {
	if !s.enabled {
		return nil, ErrMemoryDisabled
	}

	startTime := time.Now()

	// 设置默认限制
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	result := &models.SearchResult{
		StreamMemories:   make([]models.MemoryDTO, 0),
		LongTermMemories: make([]models.MemoryDTO, 0),
	}

	// 默认都包含
	includeStream := filters.IncludeStream
	includeLongTerm := filters.IncludeLongTerm
	if !includeStream && !includeLongTerm {
		includeStream = true
		includeLongTerm = true
	}

	// 先查近期流水记忆
	if includeStream {
		streamMemories, err := s.searchStreamMemories(ctx, query, filters, limit/2)
		if err != nil {
			return nil, fmt.Errorf("failed to search stream memories: %w", err)
		}
		result.StreamMemories = streamMemories
	}

	// 再查长期记忆
	if includeLongTerm {
		longTermMemories, err := s.searchLongTermMemories(ctx, query, filters, limit/2)
		if err != nil {
			return nil, fmt.Errorf("failed to search long term memories: %w", err)
		}
		result.LongTermMemories = longTermMemories
	}

	result.Total = len(result.StreamMemories) + len(result.LongTermMemories)
	result.QueryTime = time.Since(startTime)

	return result, nil
}

// searchStreamMemories 搜索流水记忆
func (s *memoryService) searchStreamMemories(ctx context.Context, query string, filters models.SearchFilters, limit int) ([]models.MemoryDTO, error) {
	opts := &models.QueryOptions{
		Limit:     limit,
		Order:     "DESC",
		UserCode:  filters.UserCode,  // 用户隔离：传递 UserCode 到 Repository
		AgentCode: filters.AgentCode, // Agent隔离：传递 AgentCode 到 Repository
	}

	var memories []models.StreamMemory
	var err error

	// 根据过滤条件选择查询方式
	switch {
	case filters.StartTime != nil && filters.EndTime != nil:
		memories, err = s.streamRepo.FindByTimeRange(ctx, *filters.StartTime, *filters.EndTime, opts)
	case filters.StartTime != nil:
		// 从开始时间到现在
		memories, err = s.streamRepo.FindByTimeRange(ctx, *filters.StartTime, time.Now(), opts)
	case filters.EndTime != nil:
		// 从很久之前到结束时间
		memories, err = s.streamRepo.FindByTimeRange(ctx, time.Time{}, *filters.EndTime, opts)
	default:
		// 默认查询最近7天的流水记忆
		weekAgo := time.Now().AddDate(0, 0, -7)
		memories, err = s.streamRepo.FindByTimeRange(ctx, weekAgo, time.Now(), opts)
	}

	if err != nil {
		return nil, err
	}

	// 关键词过滤（如果指定了 query）
	if query != "" {
		memories = s.filterStreamByKeyword(memories, query)
	}

	// 转换为 DTO
	dtos := make([]models.MemoryDTO, 0, len(memories))
	for _, m := range memories {
		dtos = append(dtos, s.streamToDTO(&m))
	}

	return dtos, nil
}

// searchLongTermMemories 搜索长期记忆
func (s *memoryService) searchLongTermMemories(ctx context.Context, query string, filters models.SearchFilters, limit int) ([]models.MemoryDTO, error) {
	opts := &models.QueryOptions{
		Limit:     limit,
		Order:     "DESC",
		UserCode:  filters.UserCode,  // 用户隔离：传递 UserCode 到 Repository
		AgentCode: filters.AgentCode, // Agent隔离：传递 AgentCode 到 Repository
	}

	var memories []models.LongTermMemory
	var err error

	// 根据过滤条件选择查询方式
	switch {
	case query != "":
		// 关键词搜索
		memories, err = s.longTermRepo.SearchByKeyword(ctx, query, opts)
	case filters.StartTime != nil && filters.EndTime != nil:
		memories, err = s.longTermRepo.FindByTimeRange(ctx, *filters.StartTime, *filters.EndTime, opts)
	case filters.StartTime != nil:
		memories, err = s.longTermRepo.FindByTimeRange(ctx, *filters.StartTime, time.Now(), opts)
	case filters.EndTime != nil:
		memories, err = s.longTermRepo.FindByTimeRange(ctx, time.Time{}, *filters.EndTime, opts)
	default:
		// 默认查询最近30天的长期记忆
		monthAgo := time.Now().AddDate(0, 0, -30)
		memories, err = s.longTermRepo.FindByTimeRange(ctx, monthAgo, time.Now(), opts)
	}

	if err != nil {
		return nil, err
	}

	// 转换为 DTO
	dtos := make([]models.MemoryDTO, 0, len(memories))
	for _, m := range memories {
		dtos = append(dtos, s.longTermToDTO(&m))
	}

	return dtos, nil
}

// filterStreamByKeyword 按关键词过滤流水记忆（仅搜索摘要）
func (s *memoryService) filterStreamByKeyword(memories []models.StreamMemory, keyword string) []models.StreamMemory {
	keyword = strings.ToLower(keyword)
	filtered := make([]models.StreamMemory, 0)
	for _, m := range memories {
		if strings.Contains(strings.ToLower(m.Summary), keyword) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// streamToDTO 将 StreamMemory 转换为 MemoryDTO
func (s *memoryService) streamToDTO(m *models.StreamMemory) models.MemoryDTO {
	return models.MemoryDTO{
		ID:        m.ID,
		Type:      "stream",
		UserCode:  m.UserCode,
		AgentCode: m.AgentCode,
		Content:   m.Summary, // 短期记忆仅保留摘要，Content 字段复用 Summary
		Summary:   m.Summary,
		CreatedAt: m.CreatedAt,
	}
}

// longTermToDTO 将 LongTermMemory 转换为 MemoryDTO
func (s *memoryService) longTermToDTO(m *models.LongTermMemory) models.MemoryDTO {
	// 构建内容：总体摘要 + 发生了什么 + 结论 + 价值
	content := m.Summary
	if m.WhatHappened != "" {
		content += "\n\n发生了什么：" + m.WhatHappened
	}
	if m.Conclusion != "" {
		content += "\n结论：" + m.Conclusion
	}
	if m.Value != "" {
		content += "\n价值：" + m.Value
	}

	return models.MemoryDTO{
		ID:        m.ID,
		Type:      "longterm",
		UserCode:  m.UserCode,
		AgentCode: m.AgentCode,
		Content:   content,
		Summary:   m.Summary,
		CreatedAt: m.CreatedAt,
	}
}

// UpgradeStreamToLongTerm 将流水记忆升级为长期记忆
// 按 UserCode + AgentCode 分组处理，每个用户+Agent的记忆单独提炼为长期记忆
func (s *memoryService) UpgradeStreamToLongTerm(ctx context.Context, date string) error {
	if !s.enabled {
		return ErrMemoryDisabled
	}

	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("%w: invalid date format, expected YYYY-MM-DD: %v", ErrInvalidMetadata, err)
	}

	// 计算时间范围（当天的开始和结束）
	startOfDay := targetDate
	endOfDay := targetDate.AddDate(0, 0, 1).Add(-time.Nanosecond)

	// 查询当天的所有流水记忆（不分用户）
	opts := &models.QueryOptions{
		OrderBy: "created_at",
		Order:   "ASC",
		Limit:   1000,
	}
	streams, err := s.streamRepo.FindByTimeRange(ctx, startOfDay, endOfDay, opts)
	if err != nil {
		return fmt.Errorf("failed to find stream memories for date %s: %w", date, err)
	}

	if len(streams) == 0 {
		// 当天没有流水记忆，无需处理
		return nil
	}

	// 按 UserCode + AgentCode 分组
	userAgentStreams := make(map[string][]models.StreamMemory)
	for _, stream := range streams {
		userCode := stream.UserCode
		if userCode == "" {
			userCode = "default" // 兼容旧数据，无UserCode的归入default
		}
		agentCode := stream.AgentCode
		if agentCode == "" {
			agentCode = "default" // 兼容旧数据，无AgentCode的归入default
		}
		key := userCode + ":" + agentCode
		userAgentStreams[key] = append(userAgentStreams[key], stream)
	}

	// 对每个用户+Agent的记忆分别处理
	for key, userAgentStreamList := range userAgentStreams {
		parts := strings.SplitN(key, ":", 2)
		userCode, agentCode := parts[0], parts[1]
		if err := s.upgradeUserAgentStreamToLongTerm(ctx, date, userCode, agentCode, userAgentStreamList); err != nil {
			// 记录错误但继续处理其他用户
			// 实际生产环境可以使用日志记录
			continue
		}
	}

	return nil
}

// upgradeUserAgentStreamToLongTerm 将指定用户+Agent的流水记忆升级为长期记忆
func (s *memoryService) upgradeUserAgentStreamToLongTerm(ctx context.Context, date, userCode, agentCode string, streams []models.StreamMemory) error {
	if len(streams) == 0 {
		return nil
	}

	// 使用 summarizer 提炼长期记忆
	summary, err := s.summarizer.SummarizeToLongTerm(ctx, streams)
	if err != nil {
		return fmt.Errorf("%w: failed to summarize to long term for user %s agent %s: %v", ErrUpgradeFailed, userCode, agentCode, err)
	}

	// 构建来源ID列表
	sourceIDs := make([]string, 0, len(streams))
	for _, s := range streams {
		sourceIDs = append(sourceIDs, strconv.FormatUint(s.ID, 10))
	}
	highlightsJSON, _ := json.Marshal(summary.Highlights)

	// 创建长期记忆
	longTerm := &models.LongTermMemory{
		Date:         date,
		UserCode:     userCode,
		AgentCode:    agentCode,
		Summary:      summary.WhatHappened,
		WhatHappened: summary.WhatHappened,
		Conclusion:   summary.Conclusion,
		Value:        summary.Value,
		Highlights:   string(highlightsJSON),
		SourceIDs:    strings.Join(sourceIDs, ","),
	}

	// 检查是否已存在该用户该Agent该日期的长期记忆
	existing, err := s.longTermRepo.FindByDate(ctx, date, &models.QueryOptions{UserCode: userCode, AgentCode: agentCode})
	if err != nil {
		return fmt.Errorf("failed to check existing long term memory for user %s agent %s: %w", userCode, agentCode, err)
	}

	if existing != nil {
		// 更新现有记录
		longTerm.ID = existing.ID
		if err := s.longTermRepo.Update(ctx, longTerm); err != nil {
			return fmt.Errorf("failed to update long term memory for user %s agent %s: %w", userCode, agentCode, err)
		}
	} else {
		// 创建新记录
		if err := s.longTermRepo.Create(ctx, longTerm); err != nil {
			return fmt.Errorf("failed to create long term memory for user %s agent %s: %w", userCode, agentCode, err)
		}
	}

	// 标记该用户+Agent的流水记忆为已处理
	ids := make([]uint64, 0, len(streams))
	for _, stream := range streams {
		ids = append(ids, stream.ID)
	}
	if err := s.streamRepo.MarkAsProcessed(ctx, ids); err != nil {
		return fmt.Errorf("failed to mark stream memories as processed for user %s agent %s: %w", userCode, agentCode, err)
	}

	return nil
}

// GetUnprocessedCount 获取未处理流水记忆数量
func (s *memoryService) GetUnprocessedCount(ctx context.Context, before time.Time) (int64, error) {
	if !s.enabled {
		return 0, ErrMemoryDisabled
	}

	return s.streamRepo.CountUnprocessed(ctx, before)
}
