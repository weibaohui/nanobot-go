package job

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"github.com/weibaohui/nanobot-go/memory/service"
)

// MemoryUpgradeJob 记忆升级定时任务
type MemoryUpgradeJob struct {
	memoryService service.MemoryService
	logger        *zap.Logger
	enabled       bool
	timeWindow    string // 时间窗口，如 "05:30-06:30"
	timezone      string
}

// NewMemoryUpgradeJob 创建记忆升级定时任务
func NewMemoryUpgradeJob(
	memoryService service.MemoryService,
	logger *zap.Logger,
	enabled bool,
	timeWindow string,
	timezone string,
) *MemoryUpgradeJob {
	return &MemoryUpgradeJob{
		memoryService: memoryService,
		logger:        logger,
		enabled:       enabled,
		timeWindow:    timeWindow,
		timezone:      timezone,
	}
}

// Run 执行记忆升级任务
// 这是定时任务的入口方法
func (j *MemoryUpgradeJob) Run() {
	if !j.enabled {
		j.logger.Debug("记忆升级任务未启用，跳过执行")
		return
	}

	// 检查是否在时间窗口内
	if !j.isInTimeWindow() {
		j.logger.Debug("不在时间窗口内，跳过执行",
			zap.String("time_window", j.timeWindow),
		)
		return
	}

	ctx := context.Background()

	// 计算昨日的日期
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	j.logger.Info("开始执行记忆升级任务",
		zap.String("target_date", yesterday),
	)

	// 执行升级
	if err := j.memoryService.UpgradeStreamToLongTerm(ctx, yesterday); err != nil {
		j.logger.Error("记忆升级任务失败",
			zap.String("date", yesterday),
			zap.Error(err),
		)
		return
	}

	j.logger.Info("记忆升级任务完成",
		zap.String("date", yesterday),
	)
}

// RunForDate 为指定日期执行记忆升级
// 用于手动触发或测试
func (j *MemoryUpgradeJob) RunForDate(date string) error {
	if !j.enabled {
		return fmt.Errorf("记忆升级任务未启用")
	}

	ctx := context.Background()

	j.logger.Info("手动执行记忆升级任务",
		zap.String("target_date", date),
	)

	if err := j.memoryService.UpgradeStreamToLongTerm(ctx, date); err != nil {
		return fmt.Errorf("记忆升级失败: %w", err)
	}

	j.logger.Info("记忆升级任务完成",
		zap.String("date", date),
	)
	return nil
}

// isInTimeWindow 检查当前时间是否在时间窗口内
func (j *MemoryUpgradeJob) isInTimeWindow() bool {
	if j.timeWindow == "" {
		// 没有时间窗口限制，随时可执行
		return true
	}

	// 解析时间窗口
	parts := splitTimeWindow(j.timeWindow)
	if len(parts) != 2 {
		j.logger.Warn("时间窗口格式无效，使用默认值",
			zap.String("time_window", j.timeWindow),
		)
		return true
	}

	startTime, err1 := parseTime(parts[0])
	endTime, err2 := parseTime(parts[1])
	if err1 != nil || err2 != nil {
		j.logger.Warn("解析时间窗口失败",
			zap.String("time_window", j.timeWindow),
			zap.Error(err1),
			zap.Error(err2),
		)
		return true
	}

	// 获取当前时间（只比较时分）
	now := time.Now()
	currentTime := now.Hour()*60 + now.Minute()

	return currentTime >= startTime && currentTime <= endTime
}

// splitTimeWindow 分割时间窗口字符串
func splitTimeWindow(timeWindow string) []string {
	// 支持 "-" 或 "~" 作为分隔符
	for _, sep := range []string{"-", "~"} {
		if parts := splitBySep(timeWindow, sep); len(parts) == 2 {
			return parts
		}
	}
	return nil
}

// splitBySep 按分隔符分割字符串
func splitBySep(s, sep string) []string {
	var result []string
	idx := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[idx:i])
			idx = i + len(sep)
		}
	}
	result = append(result, s[idx:])
	return result
}

// parseTime 解析时间字符串（如 "05:30"）为分钟数
func parseTime(timeStr string) (int, error) {
	timeStr = trimSpace(timeStr)

	var hour, minute int
	_, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("invalid time value: %s", timeStr)
	}

	return hour*60 + minute, nil
}

// trimSpace 去除字符串首尾空白
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// GetUnprocessedCount 获取未处理的流水记忆数量
func (j *MemoryUpgradeJob) GetUnprocessedCount() (int64, error) {
	if !j.enabled {
		return 0, nil
	}

	ctx := context.Background()
	before := time.Now()
	return j.memoryService.GetUnprocessedCount(ctx, before)
}

// MemoryUpgradeJobFunc 返回一个适配现有 cron 系统的任务函数
func MemoryUpgradeJobFunc(job *MemoryUpgradeJob) func() {
	return func() {
		job.Run()
	}
}
