package heartbeat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

const (
	// DefaultHeartbeatPrompt 默认心跳提示
	DefaultHeartbeatPrompt = `Read HEARTBEAT.md in your workspace (if it exists).
Follow any instructions or tasks listed there.
If nothing needs attention, reply with just: HEARTBEAT_OK`

	// HeartbeatOKToken 表示"无需操作"的标记
	HeartbeatOKToken = "HEARTBEAT_OK"

	// DefaultAckMaxChars 默认确认消息最大字符数
	DefaultAckMaxChars = 500
)

// HeartbeatCallback 心跳回调函数类型
type HeartbeatCallback func(ctx context.Context, prompt string, model string, session string) (string, error)

// Service 心跳服务
type Service struct {
	cfg         *config.Config
	workspace   string
	onHeartbeat HeartbeatCallback
	cron        *cron.Cron
	jobID       cron.EntryID
	logger      *zap.Logger
	location    *time.Location
}

// NewService 创建心跳服务
func NewService(logger *zap.Logger, cfg *config.Config, workspace string, onHeartbeat HeartbeatCallback) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	// 解析时区
	loc := time.Local
	if cfg.Heartbeat.ActiveHours.Timezone != "" {
		if l, err := time.LoadLocation(cfg.Heartbeat.ActiveHours.Timezone); err == nil {
			loc = l
		} else {
			logger.Warn("解析时区失败，使用本地时区", zap.Error(err), zap.String("timezone", cfg.Heartbeat.ActiveHours.Timezone))
		}
	}

	return &Service{
		cfg:         cfg,
		workspace:   workspace,
		onHeartbeat: onHeartbeat,
		cron:        cron.New(cron.WithLocation(loc)),
		logger:      logger,
		location:    loc,
	}
}

// HeartbeatFile 返回心跳文件路径
func (s *Service) HeartbeatFile() string {
	return s.workspace + "/HEARTBEAT.md"
}

// readHeartbeatFile 读取 HEARTBEAT.md 内容
func (s *Service) readHeartbeatFile() string {
	data, err := os.ReadFile(s.HeartbeatFile())
	if err != nil {
		return ""
	}
	return string(data)
}

// isHeartbeatEmpty 检查 HEARTBEAT.md 是否没有可执行内容
func isHeartbeatEmpty(content string) bool {
	if content == "" {
		return true
	}

	// 需要跳过的模式：空行、标题、HTML注释、空复选框
	skipPatterns := []string{"- [ ]", "* [ ]", "- [x]", "* [x]"}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "<!--") {
			continue
		}
		// 检查跳过模式
		skip := false
		for _, pattern := range skipPatterns {
			if strings.TrimSpace(line) == pattern {
				skip = true
				break
			}
		}
		if !skip {
			return false // 找到可执行内容
		}
	}

	return true
}

// isInActiveHours 检查当前是否在活跃时段内
func (s *Service) isInActiveHours() bool {
	ah := s.cfg.Heartbeat.ActiveHours

	// 如果没有配置活跃时段，默认始终活跃
	if ah.Start == "" || ah.End == "" {
		return true
	}

	now := time.Now().In(s.location)

	// 解析开始和结束时间
	startTime, err := time.ParseInLocation("15:04", ah.Start, s.location)
	if err != nil {
		s.logger.Warn("解析活跃开始时间失败", zap.Error(err), zap.String("start", ah.Start))
		return true
	}

	endTime, err := time.ParseInLocation("15:04", ah.End, s.location)
	if err != nil {
		s.logger.Warn("解析活跃结束时间失败", zap.Error(err), zap.String("end", ah.End))
		return true
	}

	// 构造今天的开始和结束时间
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), 0, 0, s.location)
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), 0, 0, s.location)

	// 处理跨天情况（如 22:00 - 06:00）
	if todayEnd.Before(todayStart) {
		// 跨天：当前时间在开始之后或结束之前都算活跃
		return now.After(todayStart) || now.Before(todayEnd)
	}

	// 不跨天：当前时间在开始和结束之间
	return now.After(todayStart) && now.Before(todayEnd)
}

// getPrompt 获取心跳提示词
func (s *Service) getPrompt() string {
	if s.cfg.Heartbeat.Prompt != "" {
		return s.cfg.Heartbeat.Prompt
	}
	return DefaultHeartbeatPrompt
}

// getModel 获取心跳专用模型
func (s *Service) getModel() string {
	return s.cfg.Heartbeat.Model
}

// getSession 获取心跳会话键
func (s *Service) getSession() string {
	return s.cfg.Heartbeat.Session
}

// getAckMaxChars 获取确认消息最大字符数
func (s *Service) getAckMaxChars() int {
	if s.cfg.Heartbeat.AckMaxChars > 0 {
		return s.cfg.Heartbeat.AckMaxChars
	}
	return DefaultAckMaxChars
}

// truncateResponse 截断响应消息
func truncateResponse(response string, maxChars int) string {
	if len(response) <= maxChars {
		return response
	}
	return response[:maxChars] + "..."
}

// Start 启动心跳服务
func (s *Service) Start(ctx context.Context) error {
	// 获取心跳间隔，默认30分钟
	every := s.cfg.Heartbeat.Every
	if every == "" {
		every = "30m"
	}

	// 使用 cron 的 @every 语法创建定时任务
	everySpec := fmt.Sprintf("@every %s", every)

	jobID, err := s.cron.AddFunc(everySpec, func() {
		s.tick(ctx)
	})
	if err != nil {
		s.logger.Error("添加心跳定时任务失败", zap.Error(err))
		return err
	}
	s.jobID = jobID

	s.cron.Start()

	// 记录启动信息
	s.logger.Info("心跳服务已启动",
		zap.String("间隔", every),
		zap.Int("任务ID", int(s.jobID)),
		zap.String("活跃时段", fmt.Sprintf("%s-%s", s.cfg.Heartbeat.ActiveHours.Start, s.cfg.Heartbeat.ActiveHours.End)),
		zap.String("时区", s.cfg.Heartbeat.ActiveHours.Timezone),
		zap.String("模型", s.getModel()),
		zap.String("会话", s.getSession()),
	)
	return nil
}

// Stop 停止心跳服务
func (s *Service) Stop() {
	if s.cron != nil {
		s.cron.Stop()
		s.logger.Info("心跳服务已停止")
	}
}

// tick 执行单次心跳
func (s *Service) tick(ctx context.Context) {
	// 检查是否在活跃时段内
	if !s.isInActiveHours() {
		s.logger.Debug("心跳: 当前不在活跃时段内，跳过")
		return
	}

	content := s.readHeartbeatFile()

	// 如果 HEARTBEAT.md 为空或不存在，跳过
	if isHeartbeatEmpty(content) {
		s.logger.Debug("心跳: 无任务（HEARTBEAT.md 为空）")
		return
	}

	s.logger.Info("心跳: 检查任务...")

	if s.onHeartbeat != nil {
		response, err := s.onHeartbeat(ctx, s.getPrompt(), s.getModel(), s.getSession())
		if err != nil {
			s.logger.Error("心跳执行失败", zap.Error(err))
			return
		}

		// 检查代理是否说"无需操作"
		normalizedResponse := strings.ToUpper(strings.ReplaceAll(response, "_", ""))
		normalizedToken := strings.ToUpper(strings.ReplaceAll(HeartbeatOKToken, "_", ""))
		if strings.Contains(normalizedResponse, normalizedToken) {
			s.logger.Info("心跳: OK（无需操作）")
		} else {
			// 截断响应并记录
			truncated := truncateResponse(response, s.getAckMaxChars())
			s.logger.Info("心跳: 任务已完成", zap.String("响应", truncated))
		}
	}
}

// TriggerNow 手动触发心跳
func (s *Service) TriggerNow(ctx context.Context) (string, error) {
	if s.onHeartbeat != nil {
		return s.onHeartbeat(ctx, s.getPrompt(), s.getModel(), s.getSession())
	}
	return "", nil
}

// IsRunning 检查服务是否运行中
func (s *Service) IsRunning() bool {
	return s.cron != nil && len(s.cron.Entries()) > 0
}

// GetTarget 获取心跳目标
func (s *Service) GetTarget() string {
	return s.cfg.Heartbeat.Target
}
