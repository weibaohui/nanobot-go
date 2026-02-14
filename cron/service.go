package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OnJobCallback 任务执行回调
type OnJobCallback func(job *Job) (string, error)

// Service 定时任务服务
type Service struct {
	storePath string
	onJob     OnJobCallback
	store     *Store
	mu        sync.RWMutex
	running   bool
	timer     *time.Timer
	logger    *zap.Logger
}

// NewService 创建定时任务服务
func NewService(storePath string, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		storePath: storePath,
		logger:    logger,
	}
}

// Start 启动服务
func (s *Service) Start(ctx context.Context) error {
	s.loadStore()
	s.recomputeNextRuns()
	s.saveStore()
	s.armTimer(ctx)

	s.logger.Info("定时任务服务已启动",
		zap.Int("任务数量", len(s.store.Jobs)),
	)

	return nil
}

// Stop 停止服务
func (s *Service) Stop() {
	s.running = false
	if s.timer != nil {
		s.timer.Stop()
	}
}

// loadStore 加载任务存储
func (s *Service) loadStore() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store = &Store{Version: 1}

	data, err := os.ReadFile(s.storePath)
	if err != nil {
		return
	}

	if err := json.Unmarshal(data, s.store); err != nil {
		s.logger.Warn("加载任务存储失败", zap.Error(err))
	}
}

// saveStore 保存任务存储
func (s *Service) saveStore() {
	s.mu.RLock()
	store := s.store
	s.mu.RUnlock()

	if store == nil {
		return
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		s.logger.Error("序列化任务存储失败", zap.Error(err))
		return
	}

	os.MkdirAll(filepath.Dir(s.storePath), 0755)
	if err := os.WriteFile(s.storePath, data, 0644); err != nil {
		s.logger.Error("保存任务存储失败", zap.Error(err))
	}
}

// recomputeNextRuns 重新计算所有任务的下次运行时间
func (s *Service) recomputeNextRuns() {
	now := nowMs()
	for _, job := range s.store.Jobs {
		if job.Enabled {
			job.State.NextRunAtMs = computeNextRun(&job.Schedule, now)
		}
	}
}

// getNextWakeMs 获取最早的下次运行时间
func (s *Service) getNextWakeMs() int {
	var minMs int
	for _, job := range s.store.Jobs {
		if job.Enabled && job.State.NextRunAtMs > 0 {
			if minMs == 0 || job.State.NextRunAtMs < minMs {
				minMs = job.State.NextRunAtMs
			}
		}
	}
	return minMs
}

// armTimer 设置定时器
func (s *Service) armTimer(ctx context.Context) {
	if s.timer != nil {
		s.timer.Stop()
	}

	nextWake := s.getNextWakeMs()
	if nextWake == 0 {
		return
	}

	delayMs := nextWake - nowMs()
	if delayMs < 0 {
		delayMs = 0
	}

	s.timer = time.AfterFunc(time.Duration(delayMs)*time.Millisecond, func() {
		if s.running {
			s.onTimer(ctx)
		}
	})
}

// onTimer 定时器触发
func (s *Service) onTimer(ctx context.Context) {
	now := nowMs()

	var dueJobs []*Job
	for _, job := range s.store.Jobs {
		if job.Enabled && job.State.NextRunAtMs > 0 && now >= job.State.NextRunAtMs {
			dueJobs = append(dueJobs, job)
		}
	}

	for _, job := range dueJobs {
		s.executeJob(ctx, job)
	}

	s.saveStore()
	s.armTimer(ctx)
}

// executeJob 执行任务
func (s *Service) executeJob(ctx context.Context, job *Job) {
	startMs := nowMs()
	s.logger.Info("执行定时任务",
		zap.String("名称", job.Name),
		zap.String("ID", job.ID),
	)

	var status string
	var errMsg string

	if s.onJob != nil {
		if _, err := s.onJob(job); err != nil {
			status = "error"
			errMsg = err.Error()
			s.logger.Error("任务执行失败",
				zap.String("名称", job.Name),
				zap.Error(err),
			)
		} else {
			status = "ok"
			s.logger.Info("任务执行完成", zap.String("名称", job.Name))
		}
	}

	job.State.LastRunAtMs = startMs
	job.State.LastStatus = status
	job.State.LastError = errMsg
	job.UpdatedAtMs = nowMs()

	// 处理一次性任务
	if job.Schedule.Kind == "at" {
		if job.DeleteAfterRun {
			s.removeJobByID(job.ID)
		} else {
			job.Enabled = false
			job.State.NextRunAtMs = 0
		}
	} else {
		job.State.NextRunAtMs = computeNextRun(&job.Schedule, nowMs())
	}
}

// ListJobs 列出所有任务
func (s *Service) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.store == nil {
		return nil
	}

	var jobs []*Job
	for _, job := range s.store.Jobs {
		if job.Enabled {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// AddJob 添加任务
func (s *Service) AddJob(name string, schedule *Schedule, message string, deliver bool, channel, to string, deleteAfterRun bool) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.store == nil {
		s.store = &Store{Version: 1}
	}

	now := nowMs()
	job := &Job{
		ID:             generateID(),
		Name:           name,
		Enabled:        true,
		Schedule:       *schedule,
		Payload:        Payload{Kind: "agent_turn", Message: message, Deliver: deliver, Channel: channel, To: to},
		State:          State{NextRunAtMs: computeNextRun(schedule, now)},
		CreatedAtMs:    now,
		UpdatedAtMs:    now,
		DeleteAfterRun: deleteAfterRun,
	}

	s.store.Jobs = append(s.store.Jobs, job)
	s.saveStore()

	s.logger.Info("添加定时任务",
		zap.String("名称", name),
		zap.String("ID", job.ID),
	)

	return job
}

// RemoveJob 删除任务
func (s *Service) RemoveJob(jobID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := s.removeJobByID(jobID)
	if removed {
		s.saveStore()
		s.logger.Info("删除定时任务", zap.String("ID", jobID))
	}
	return removed
}

// removeJobByID 内部删除任务
func (s *Service) removeJobByID(jobID string) bool {
	for i, job := range s.store.Jobs {
		if job.ID == jobID {
			s.store.Jobs = append(s.store.Jobs[:i], s.store.Jobs[i+1:]...)
			return true
		}
	}
	return false
}

// SetOnJobCallback 设置任务执行回调
func (s *Service) SetOnJobCallback(callback OnJobCallback) {
	s.onJob = callback
}

// Status 获取服务状态
func (s *Service) Status() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobCount := 0
	if s.store != nil {
		jobCount = len(s.store.Jobs)
	}

	return map[string]any{
		"enabled":      s.running,
		"jobs":         jobCount,
		"nextWakeAtMs": s.getNextWakeMs(),
	}
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
}
