package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	memorymodels "github.com/weibaohui/nanobot-go/internal/memory/models"
	memservice "github.com/weibaohui/nanobot-go/internal/memory/service"
	"gorm.io/gorm"
)

// StreamMemoryService 短期记忆服务接口
type StreamMemoryService interface {
	// 查询方法
	List(ctx context.Context, userCode string, offset int, limit int) ([]memorymodels.StreamMemory, int64, error)
	Get(ctx context.Context, id uint64) (*memorymodels.StreamMemory, error)
	GetByUserAndDate(ctx context.Context, userCode string, date string) (*memorymodels.StreamMemory, error)
	GetUnprocessed(ctx context.Context) ([]memorymodels.StreamMemory, error)

	// 管理方法
	Create(ctx context.Context, memory *memorymodels.StreamMemory) error
	Update(ctx context.Context, id uint64, memory *memorymodels.StreamMemory) error
	Delete(ctx context.Context, id uint64) error
	MarkProcessed(ctx context.Context, id uint64) error

	// 核心方法：追加对话内容到指定日期的短期记忆（自动聚合）
	AppendConversation(ctx context.Context, userCode string, date string, conversationContent string, conversationID string) error

	// 从对话记录创建/更新短期记忆
	BuildFromConversations(ctx context.Context, userCode string, date string, conversationIDs []string, contents []string) error
}

// streamMemoryService 短期记忆服务实现
type streamMemoryService struct {
	db         *gorm.DB
	summarizer memservice.MemorySummarizer
}

// NewStreamMemoryService 创建短期记忆服务
func NewStreamMemoryService(db *gorm.DB, summarizer memservice.MemorySummarizer) StreamMemoryService {
	return &streamMemoryService{db: db, summarizer: summarizer}
}

// List 获取短期记忆列表（按用户+日期聚合后的记录）
func (s *streamMemoryService) List(ctx context.Context, userCode string, offset int, limit int) ([]memorymodels.StreamMemory, int64, error) {
	var memories []memorymodels.StreamMemory
	var total int64

	query := s.db.WithContext(ctx).Model(&memorymodels.StreamMemory{})

	if userCode != "" {
		query = query.Where("user_code = ?", userCode)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("date DESC, created_at DESC").Offset(offset).Limit(limit).Find(&memories).Error; err != nil {
		return nil, 0, err
	}

	return memories, total, nil
}

// Get 获取单条短期记忆
func (s *streamMemoryService) Get(ctx context.Context, id uint64) (*memorymodels.StreamMemory, error) {
	var memory memorymodels.StreamMemory
	if err := s.db.WithContext(ctx).First(&memory, id).Error; err != nil {
		return nil, err
	}
	return &memory, nil
}

// GetByUserAndDate 根据用户和日期获取短期记忆
func (s *streamMemoryService) GetByUserAndDate(ctx context.Context, userCode string, date string) (*memorymodels.StreamMemory, error) {
	var memory memorymodels.StreamMemory
	if err := s.db.WithContext(ctx).
		Where("user_code = ? AND date = ?", userCode, date).
		First(&memory).Error; err != nil {
		return nil, err
	}
	return &memory, nil
}

// Create 创建短期记忆
func (s *streamMemoryService) Create(ctx context.Context, memory *memorymodels.StreamMemory) error {
	return s.db.WithContext(ctx).Create(memory).Error
}

// Update 更新短期记忆
func (s *streamMemoryService) Update(ctx context.Context, id uint64, memory *memorymodels.StreamMemory) error {
	return s.db.WithContext(ctx).Model(&memorymodels.StreamMemory{}).Where("id = ?", id).Updates(memory).Error
}

// Delete 删除短期记忆
func (s *streamMemoryService) Delete(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&memorymodels.StreamMemory{}, id).Error
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// MarkProcessed 标记为已处理
func (s *streamMemoryService) MarkProcessed(ctx context.Context, id uint64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&memorymodels.StreamMemory{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"processed":    true,
			"processed_at": now,
		}).Error
}

// GetUnprocessed 获取未处理的短期记忆
func (s *streamMemoryService) GetUnprocessed(ctx context.Context) ([]memorymodels.StreamMemory, error) {
	var memories []memorymodels.StreamMemory
	if err := s.db.WithContext(ctx).
		Where("processed = ?", false).
		Order("date ASC, created_at ASC").
		Find(&memories).Error; err != nil {
		return nil, err
	}
	return memories, nil
}

// AppendConversation 追加对话内容到指定日期的短期记忆
// 如果不存在则创建，存在则追加内容
func (s *streamMemoryService) AppendConversation(ctx context.Context, userCode string, date string, conversationContent string, conversationID string) error {
	if userCode == "" || date == "" {
		return fmt.Errorf("user_code and date are required")
	}

	// 查找是否已存在该用户+日期的记录
	var memory memorymodels.StreamMemory
	err := s.db.WithContext(ctx).
		Where("user_code = ? AND date = ?", userCode, date).
		First(&memory).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// 不存在，创建新记录
		memory = memorymodels.StreamMemory{
			UserCode:  userCode,
			Date:      date,
			Content:   conversationContent,
			SourceIDs: conversationID,
			CreatedAt: now,
			UpdatedAt: now,
			Processed: false,
		}
		return s.db.WithContext(ctx).Create(&memory).Error
	} else if err != nil {
		return err
	}

	// 已存在，追加内容
	var newContent string
	if memory.Content != "" {
		newContent = memory.Content + "\n\n---\n\n" + conversationContent
	} else {
		newContent = conversationContent
	}

	// 追加 source_ids
	var newSourceIDs string
	if memory.SourceIDs != "" {
		newSourceIDs = memory.SourceIDs + "," + conversationID
	} else {
		newSourceIDs = conversationID
	}

	return s.db.WithContext(ctx).Model(&memory).
		Updates(map[string]interface{}{
			"content":    newContent,
			"source_ids": newSourceIDs,
			"updated_at": now,
		}).Error
}

// BuildFromConversations 从对话记录构建短期记忆
// 使用 LLM 对对话内容进行总结，生成摘要
func (s *streamMemoryService) BuildFromConversations(ctx context.Context, userCode string, date string, conversationIDs []string, contents []string) error {
	if len(conversationIDs) == 0 || len(contents) == 0 {
		return fmt.Errorf("conversationIDs and contents are required")
	}
	if len(conversationIDs) != len(contents) {
		return fmt.Errorf("conversationIDs and contents must have the same length")
	}


	// 构建消息列表用于 LLM 总结
	messages := make([]memorymodels.Message, 0, len(contents))
	var allContent strings.Builder
	for i, content := range contents {
		// 从内容中提取角色信息（如果有 [user] 或 [assistant] 标记）
		role := "user"
		cleanContent := content
		if strings.HasPrefix(content, "[user]") {
			role = "user"
			cleanContent = strings.TrimPrefix(content, "[user] ")
		} else if strings.HasPrefix(content, "[assistant]") {
			role = "assistant"
			cleanContent = strings.TrimPrefix(content, "[assistant] ")
		}

		messages = append(messages, memorymodels.Message{
			Role:      role,
			Content:   cleanContent,
			Timestamp: time.Now(),
		})

		// 同时聚合原始内容用于存储
		if i > 0 {
			allContent.WriteString("\n\n---\n\n")
		}
		allContent.WriteString(fmt.Sprintf("【对话 %d】\n%s", i+1, content))
	}

	// 使用 LLM 生成摘要
	var summaryStr string
	if s.summarizer != nil {
		summary, err := s.summarizer.SummarizeConversation(ctx, messages)
		if err != nil {
			// LLM 失败时使用简单的内容截断作为摘要
			summaryStr = truncateString(allContent.String(), 200)
		} else {
			summaryStr = summary.Summary
			if summary.KeyPoints != "" {
				summaryStr += "\n\n关键要点：\n" + summary.KeyPoints
			}
		}
	} else {
		// 没有 summarizer 时使用简单的内容截断
		summaryStr = truncateString(allContent.String(), 200)
	}

	sourceIDs := strings.Join(conversationIDs, ",")

	// 保存到数据库
	var memory memorymodels.StreamMemory
	err := s.db.WithContext(ctx).
		Where("user_code = ? AND date = ?", userCode, date).
		First(&memory).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		memory = memorymodels.StreamMemory{
			UserCode:  userCode,
			Date:      date,
			Content:   allContent.String(),
			Summary:   summaryStr,
			SourceIDs: sourceIDs,
			CreatedAt: now,
			UpdatedAt: now,
			Processed: false,
		}
		return s.db.WithContext(ctx).Create(&memory).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录（覆盖内容）
	return s.db.WithContext(ctx).Model(&memory).
		Updates(map[string]interface{}{
			"content":    allContent.String(),
			"summary":    summaryStr,
			"source_ids": sourceIDs,
			"updated_at": now,
			"processed":  false, // 重置处理状态
		}).Error
}
