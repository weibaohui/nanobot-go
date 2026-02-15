package heartbeat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

const (
	// HeartbeatPrompt 心跳时发送给代理的提示
	HeartbeatPrompt = `Read HEARTBEAT.md in your workspace (if it exists).
Follow any instructions or tasks listed there.
If nothing needs attention, reply with just: HEARTBEAT_OK`

	// HeartbeatOKToken 表示"无需操作"的标记
	HeartbeatOKToken = "HEARTBEAT_OK"
)

// HeartbeatCallback 心跳回调函数类型
type HeartbeatCallback func(ctx context.Context, prompt string) (string, error)

// Service 心跳服务
type Service struct {
	cfg         *config.Config
	workspace   string
	onHeartbeat HeartbeatCallback
	cron        *cron.Cron
	jobID       cron.EntryID
	logger      *zap.Logger
}

// NewService 创建心跳服务
func NewService(logger *zap.Logger, cfg *config.Config, workspace string, onHeartbeat HeartbeatCallback) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		workspace:   workspace,
		onHeartbeat: onHeartbeat,
		cron:        cron.New(),
		logger:      logger,
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
	s.logger.Info("心跳服务已启动", zap.String("间隔", every), zap.Int("任务ID", int(s.jobID)))
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
	content := s.readHeartbeatFile()

	// 如果 HEARTBEAT.md 为空或不存在，跳过
	if isHeartbeatEmpty(content) {
		s.logger.Debug("心跳: 无任务（HEARTBEAT.md 为空）")
		return
	}

	s.logger.Info("心跳: 检查任务...")

	if s.onHeartbeat != nil {
		response, err := s.onHeartbeat(ctx, HeartbeatPrompt)
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
			s.logger.Info("心跳: 任务已完成")
		}
	}
}

// TriggerNow 手动触发心跳
func (s *Service) TriggerNow(ctx context.Context) (string, error) {
	if s.onHeartbeat != nil {
		return s.onHeartbeat(ctx, HeartbeatPrompt)
	}
	return "", nil
}

// IsRunning 检查服务是否运行中
func (s *Service) IsRunning() bool {
	return s.cron != nil && len(s.cron.Entries()) > 0
}
