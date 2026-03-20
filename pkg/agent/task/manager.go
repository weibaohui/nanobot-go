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
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks"
	"github.com/weibaohui/nanobot-go/pkg/session"
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
	eventPublisher    *EventPublisher // 任务事件发布器
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

	// 初始化事件发布器
	m.eventPublisher = NewEventPublisher(cfg.EventBus, logger)

	return m, nil
}

// SetRegisteredTools 设置已注册的工具名称
func (m *Manager) SetRegisteredTools(names []string) {
	m.registeredTools = append([]string(nil), names...)
}

// StartTask 启动任务
func (m *Manager) StartTask(ctx context.Context, work, channel, chatID string, createdBy ...string) (string, Status, error) {
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

	creator := ""
	if len(createdBy) > 0 {
		creator = createdBy[0]
	}
	go m.runTask(ctx, task, channel, chatID, creator)

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
	// 先复制任务指针列表，避免在持有锁时调用 task.ToInfo() 导致死锁
	m.mu.RLock()
	tasks := make([]*Task, 0, len(m.runningTasks))
	for _, task := range m.runningTasks {
		tasks = append(tasks, task)
	}
	m.mu.RUnlock()

	// 释放锁后再调用 ToInfo()，避免与 runTask() 中的锁顺序冲突
	results := make([]*Info, 0, len(tasks))
	for _, task := range tasks {
		results = append(results, task.ToInfo())
	}

	todayTasks, err := m.persistence.LoadTodayCompletedTasks()
	if err != nil {
		m.logger.Warn("加载当天任务失败", zap.Error(err))
	} else {
		results = append(results, todayTasks...)
	}

	return results, nil
}

func (m *Manager) runTask(ctx context.Context, task *Task, channel, chatID string, createdBy string) {
	startTime := time.Now()

	execCtx, cancel := m.buildTaskContext(ctx)
	task.SetCancel(cancel)
	task.SetStatus(StatusRunning)
	task.AppendLog("任务启动")

	// 发布任务创建事件
	if m.eventPublisher != nil {
		m.eventPublisher.PublishTaskCreated(task, createdBy)
	}

	result, err := m.executeTask(execCtx, task.Work(), channel, chatID)

	// 在锁内准备需要的数据，然后释放锁再调用可能获取 m.mu 的函数
	// 注意：这里直接访问字段而不是调用方法，因为方法内部也会加锁，会导致死锁
	task.mu.Lock()
	var (
		status        Status
		taskID        string
		shouldPersist bool
		persistedTask *PersistedTask
		notifyResult  string
	)
	if task.stopRequested || execCtx.Err() == context.Canceled {
		task.status = StatusStopped
		task.appendLogInternal("任务已停止")
		close(task.done)
		status = task.status
		taskID = task.id
		shouldPersist = true
		persistedTask = task.toPersistedTaskInternal()
		notifyResult = ""
	} else if err != nil {
		task.status = StatusFailed
		task.appendLogInternal(fmt.Sprintf("任务失败: %v", err))
		close(task.done)
		status = task.status
		taskID = task.id
		shouldPersist = true
		persistedTask = task.toPersistedTaskInternal()
		notifyResult = result
	} else {
		task.status = StatusFinished
		task.result = result
		task.appendLogInternal("任务完成")
		close(task.done)
		status = task.status
		taskID = task.id
		shouldPersist = true
		persistedTask = task.toPersistedTaskInternal()
		notifyResult = result
	}
	task.mu.Unlock()

	// 在释放 task.mu 后再调用可能获取 m.mu 的函数，避免死锁
	if shouldPersist {
		m.persistence.AppendTaskToFile(persistedTask)
	}

	// 发布任务完成事件
	if m.eventPublisher != nil {
		duration := time.Since(startTime)
		m.eventPublisher.PublishTaskCompleted(task, duration)
	}

	if m.onTaskComplete != nil {
		m.onTaskComplete(channel, chatID, taskID, status, notifyResult)
	}
	m.removeFromRunning(taskID)
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
	// 先复制任务指针列表，避免在持有锁时调用 task.mu.Lock() 导致死锁
	m.mu.RLock()
	tasks := make([]*Task, 0, len(m.runningTasks))
	for _, task := range m.runningTasks {
		tasks = append(tasks, task)
	}
	m.mu.RUnlock()

	running := 0
	for _, task := range tasks {
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
