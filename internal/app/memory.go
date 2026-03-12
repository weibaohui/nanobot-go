package app

import (
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/memory/job"
	"github.com/weibaohui/nanobot-go/memory/repository"
	"github.com/weibaohui/nanobot-go/memory/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MemoryComponents 记忆模块组件
type MemoryComponents struct {
	Service      service.MemoryService
	UpgradeJob   *job.MemoryUpgradeJob
}

// InitMemory 初始化记忆模块
func InitMemory(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *MemoryComponents {
	if !cfg.Memory.Enabled || db == nil {
		return nil
	}

	// 创建记忆仓库
	streamRepo := repository.NewStreamMemoryRepository(db)
	longTermRepo := repository.NewLongTermMemoryRepository(db)

	// 创建 LLM 客户端
	llmClient := service.NewSystemLLMClient(cfg, logger)

	// 创建总结器
	summarizer := service.NewMemorySummarizer(
		llmClient,
		cfg.Memory.Summarization.ConversationPrompt,
		cfg.Memory.Summarization.LongTermPrompt,
	)

	// 创建记忆服务
	memoryService := service.NewMemoryService(
		streamRepo,
		longTermRepo,
		summarizer,
		cfg.Memory.Enabled,
	)

	// 创建定时任务
	upgradeJob := job.NewMemoryUpgradeJob(
		memoryService,
		logger,
		cfg.Memory.Enabled,
		cfg.Memory.Scheduled.TimeWindow,
		cfg.Memory.Scheduled.Timezone,
	)

	logger.Info("记忆模块已初始化",
		zap.Bool("enabled", cfg.Memory.Enabled),
		zap.String("summarization_model", cfg.Memory.Summarization.Model),
	)

	return &MemoryComponents{
		Service:    memoryService,
		UpgradeJob: upgradeJob,
	}
}

// RunMemoryUpgrade 执行记忆升级
func RunMemoryUpgrade(cfg *config.Config, db *gorm.DB, logger *zap.Logger, targetDate string) error {
	if !cfg.Memory.Enabled {
		return nil
	}

	// 创建记忆模块组件
	streamRepo := repository.NewStreamMemoryRepository(db)
	longTermRepo := repository.NewLongTermMemoryRepository(db)

	// 创建 LLM 客户端
	llmClient := service.NewSystemLLMClient(cfg, logger)

	// 创建总结器
	summarizer := service.NewMemorySummarizer(
		llmClient,
		cfg.Memory.Summarization.ConversationPrompt,
		cfg.Memory.Summarization.LongTermPrompt,
	)

	// 创建记忆服务
	memoryService := service.NewMemoryService(
		streamRepo,
		longTermRepo,
		summarizer,
		cfg.Memory.Enabled,
	)

	// 创建升级任务
	upgradeJob := job.NewMemoryUpgradeJob(
		memoryService,
		logger,
		cfg.Memory.Enabled,
		cfg.Memory.Scheduled.TimeWindow,
		cfg.Memory.Scheduled.Timezone,
	)

	return upgradeJob.RunForDate(targetDate)
}
