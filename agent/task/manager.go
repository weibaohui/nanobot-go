package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/weibaohui/nanobot-go/agent/hooks"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// Manager 任务管理器
type Manager struct {
	configLoader      LLMConfigLoader
	workspace         string
	tools             []tool.BaseTool
	logger            *zap.Logger
	context           ContextBuilder
	checkpointStore   compose.CheckPointStore
	maxIterations     int
	registeredTools   []string
	sessions          *session.Manager
	maxConcurrent     int
	taskTimeout       time.Duration
	logCapacity       int
	onTaskComplete    func(channel, chatID, taskID string, status Status, result string)
	hookManager       *hooks.HookManager
	taskCounter       uint32
	tasksDir          string
	mu                sync.RWMutex
	runningTasks      map[string]*Task
	persistence       *Persistence
}

// NewManager 创建后台任务管理器
func NewManager(cfg *ManagerConfig) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 10
	}

	maxConcurrent := cfg.MaxConcurrentTasks
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}

	logCapacity := cfg.TaskLogCapacity
	if logCapacity <= 0 {
		logCapacity = 10
	}

	timeout := time.Duration(cfg.TaskTimeoutSeconds) * time.Second
	if cfg.TaskTimeoutSeconds <= 0 {
		timeout = 0
	}

	tasksDir := filepath.Join(cfg.Workspace, "tasks")

	m := &Manager{
		configLoader:    cfg.ConfigLoader,
		workspace:       cfg.Workspace,
		tools:           cfg.Tools,
		logger:          logger,
		context:         cfg.Context,
		checkpointStore: cfg.CheckpointStore,
		maxIterations:   maxIter,
		registeredTools: cfg.RegisteredTools,
		sessions:        cfg.Sessions,
		maxConcurrent:   maxConcurrent,
		taskTimeout:     timeout,
		logCapacity:     logCapacity,
		onTaskComplete:  cfg.OnTaskComplete,
		tasksDir:        tasksDir,
		runningTasks:    make(map[string]*Task),
		hookManager:     cfg.HookManager,
	}

	m.persistence = NewPersistence(tasksDir, logger, &m.taskCounter)
	m.persistence.LoadCounter()

	return m, nil
}

// SetRegisteredTools 设置已注册的工具名称
func (m *Manager) SetRegisteredTools(names []string) {
	m.registeredTools = append([]string(nil), names...)
}

// StartTask 启动任务
func (m *Manager) StartTask(ctx context.Context, work, channel, chatID string) (string, Status, error) {
	if work == "" {
		return "", "", fmt.Errorf("任务内容不能为空")
	}
	if m.reachedLimit() {
		return "", "", fmt.Errorf("任务并发已达上限")
	}

	taskID := m.generateTaskID()
	task := NewTask(taskID, work, channel, chatID, m.logCapacity)
	task.AppendLog("任务已创建")

	m.mu.Lock()
	m.runningTasks[taskID] = task
	m.mu.Unlock()

	go m.runTask(ctx, task, channel, chatID)

	return taskID, StatusRunning, nil
}

// generateTaskID 生成6位数字任务ID
func (m *Manager) generateTaskID() string {
	n := atomic.AddUint32(&m.taskCounter, 1) % 1000000
	return fmt.Sprintf("%06d", n)
}

// normalizeTaskID 标准化任务ID
func normalizeTaskID(taskID string) string {
	n, err := strconv.Atoi(strings.TrimSpace(taskID))
	if err != nil {
		return taskID
	}
	return fmt.Sprintf("%06d", n)
}

// GetTask 获取任务
func (m *Manager) GetTask(taskID string) (*Info, error) {
	normalizedID := normalizeTaskID(taskID)

	m.mu.RLock()
	task, ok := m.runningTasks[normalizedID]
	m.mu.RUnlock()

	if ok {
		return task.ToInfo(), nil
	}

	return m.persistence.LoadTaskFromFile(normalizedID)
}

// StopTask 停止任务
func (m *Manager) StopTask(taskID string) (bool, Status, error) {
	normalizedID := normalizeTaskID(taskID)

	m.mu.RLock()
	task, ok := m.runningTasks[normalizedID]
	m.mu.RUnlock()

	if !ok {
		return false, "", fmt.Errorf("任务不存在或已完成")
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	switch task.status {
	case StatusFinished, StatusFailed, StatusStopped:
		return false, task.status, nil
	default:
		task.stopRequested = true
		if task.cancel != nil {
			task.cancel()
		}
		task.status = StatusStopped
		task.AppendLog("任务已停止")
		return true, task.status, nil
	}
}

// ListTasks 获取所有任务列表
func (m *Manager) ListTasks() ([]*Info, error) {
	results := make([]*Info, 0)

	m.mu.RLock()
	for _, task := range m.runningTasks {
		results = append(results, task.ToInfo())
	}
	m.mu.RUnlock()

	todayTasks, err := m.persistence.LoadTodayCompletedTasks()
	if err != nil {
		m.logger.Warn("加载当天任务失败", zap.Error(err))
	} else {
		results = append(results, todayTasks...)
	}

	return results, nil
}

func (m *Manager) runTask(ctx context.Context, task *Task, channel, chatID string) {
	execCtx, cancel := m.buildTaskContext(ctx)
	task.SetCancel(cancel)
	task.SetStatus(StatusRunning)
	task.AppendLog("任务启动")

	result, err := m.executeTask(execCtx, task.Work(), channel, chatID)

	task.mu.Lock()
	defer task.mu.Unlock()

	if task.IsStopRequested() || execCtx.Err() == context.Canceled {
		task.status = StatusStopped
		task.AppendLog("任务已停止")
		task.CloseDone()
		m.persistTask(task)
		m.notifyComplete(task, "")
		m.removeFromRunning(task.ID())
		return
	}

	if err != nil {
		task.status = StatusFailed
		task.AppendLog(fmt.Sprintf("任务失败: %v", err))
	} else {
		task.status = StatusFinished
		task.result = result
		task.AppendLog("任务完成")
	}

	task.CloseDone()
	m.persistTask(task)
	m.notifyComplete(task, result)
	m.removeFromRunning(task.ID())
}

func (m *Manager) removeFromRunning(taskID string) {
	m.mu.Lock()
	delete(m.runningTasks, taskID)
	m.mu.Unlock()
}

func (m *Manager) persistTask(task *Task) {
	m.persistence.AppendTaskToFile(task.ToPersistedTask())
}

func (m *Manager) notifyComplete(task *Task, result string) {
	if m.onTaskComplete != nil {
		m.onTaskComplete(task.Channel(), task.ChatID(), task.ID(), task.Status(), result)
	}
}

func (m *Manager) executeTask(ctx context.Context, work, channel, chatID string) (string, error) {
	// 执行任务的具体逻辑在 executor 中实现
	return "", fmt.Errorf("executeTask not implemented")
}

func (m *Manager) buildTaskContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if m.taskTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, m.taskTimeout)
}

func (m *Manager) reachedLimit() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	running := 0
	for _, task := range m.runningTasks {
		task.mu.Lock()
		status := task.status
		task.mu.Unlock()
		if status == StatusRunning || status == StatusPending {
			running++
		}
	}
	return running >= m.maxConcurrent
}

// Close 关闭任务管理器
func (m *Manager) Close() {}
