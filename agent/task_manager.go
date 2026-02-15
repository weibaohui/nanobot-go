package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	tasktools "github.com/weibaohui/nanobot-go/agent/tools/task"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type TaskStatus string

const (
	TaskPending  TaskStatus = "pending"
	TaskRunning  TaskStatus = "running"
	TaskFinished TaskStatus = "finished"
	TaskFailed   TaskStatus = "failed"
	TaskStopped  TaskStatus = "stopped"
)

type TaskInfo struct {
	ID            string
	Status        TaskStatus
	ResultSummary string
}

type AgentTaskManagerConfig struct {
	Cfg                   *config.Config
	Workspace             string
	Tools                 []tool.BaseTool
	Logger                *zap.Logger
	Context               *ContextBuilder
	CheckpointStore       compose.CheckPointStore
	MaxIterations         int
	RegisteredTools       []string
	MaxConcurrentTasks    int
	TaskTimeoutSeconds    int
	TaskLogCapacity       int
	TaskMaxToolIterations int
	// OnTaskComplete 任务完成回调，用于发送完成通知
	OnTaskComplete func(channel, chatID, taskID string, status TaskStatus, result string)
}

type AgentTaskManager struct {
	cfg             *config.Config
	workspace       string
	tools           []tool.BaseTool
	logger          *zap.Logger
	context         *ContextBuilder
	checkpointStore compose.CheckPointStore
	maxIterations   int
	registeredTools []string

	maxConcurrent int
	taskTimeout   time.Duration
	logCapacity   int

	// onTaskComplete 任务完成回调
	onTaskComplete func(channel, chatID, taskID string, status TaskStatus, result string)

	// taskCounter 任务ID计数器（0-999999循环）
	taskCounter uint32

	// tasksDir 任务存储目录
	tasksDir string

	// mu 保护内存中的任务
	mu sync.RWMutex
	// runningTasks 内存中的运行中/待处理任务（必须保留在内存中以便管理）
	runningTasks map[string]*AgentTask

	// persistMu 保护文件写入
	persistMu sync.Mutex
}

type AgentTask struct {
	id            string
	work          string
	status        TaskStatus
	result        string
	lastLogs      []string
	logCapacity   int
	stopRequested bool
	cancel        context.CancelFunc
	done          chan struct{}
	mu            sync.Mutex
	// 任务上下文，用于完成回调
	channel string
	chatID  string
	// 创建时间
	createdAt time.Time
}

// PersistedTask 持久化的任务结构（用于YAML存储）
type PersistedTask struct {
	ID          string     `yaml:"id"`
	Work        string     `yaml:"work,omitempty"`
	Status      TaskStatus `yaml:"status"`
	Result      string     `yaml:"result,omitempty"`
	Channel     string     `yaml:"channel,omitempty"`
	ChatID      string     `yaml:"chat_id,omitempty"`
	CreatedAt   time.Time  `yaml:"created_at"`
	CompletedAt time.Time  `yaml:"completed_at,omitempty"`
}

// TaskFile YAML文件结构
type TaskFile struct {
	Date   string           `yaml:"date"`
	LastID uint32           `yaml:"last_id"`
	Tasks  []*PersistedTask `yaml:"tasks"`
}

func NewAgentTaskManager(cfg *AgentTaskManagerConfig) (*AgentTaskManager, error) {
	if cfg == nil {
		return nil, ErrConfigNil
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

	// 任务存储目录
	tasksDir := filepath.Join(cfg.Workspace, "tasks")

	m := &AgentTaskManager{
		cfg:             cfg.Cfg,
		workspace:       cfg.Workspace,
		tools:           cfg.Tools,
		logger:          logger,
		context:         cfg.Context,
		checkpointStore: cfg.CheckpointStore,
		maxIterations:   maxIter,
		registeredTools: cfg.RegisteredTools,
		maxConcurrent:   maxConcurrent,
		taskTimeout:     timeout,
		logCapacity:     logCapacity,
		onTaskComplete:  cfg.OnTaskComplete,
		tasksDir:        tasksDir,
		runningTasks:    make(map[string]*AgentTask),
	}

	// 加载计数器状态
	m.loadCounter()

	return m, nil
}

// SetRegisteredTools 设置已注册的工具名称
func (m *AgentTaskManager) SetRegisteredTools(names []string) {
	m.registeredTools = append([]string(nil), names...)
}

func (m *AgentTaskManager) StartTask(ctx context.Context, work, channel, chatID string) (string, TaskStatus, error) {
	if work == "" {
		return "", "", fmt.Errorf("任务内容不能为空")
	}
	if m.reachedLimit() {
		return "", "", fmt.Errorf("任务并发已达上限")
	}

	// 生成6位数字任务ID（000000-999999循环）
	taskID := m.generateTaskID()
	task := &AgentTask{
		id:          taskID,
		work:        work,
		status:      TaskPending,
		logCapacity: m.logCapacity,
		done:        make(chan struct{}),
		channel:     channel,
		chatID:      chatID,
		createdAt:   time.Now(),
	}
	task.appendLog("任务已创建")

	m.mu.Lock()
	m.runningTasks[taskID] = task
	m.mu.Unlock()

	go m.runTask(ctx, task, channel, chatID)

	return taskID, TaskRunning, nil
}

// generateTaskID 生成6位数字任务ID
func (m *AgentTaskManager) generateTaskID() string {
	// 原子递增，取模1000000实现循环
	n := atomic.AddUint32(&m.taskCounter, 1) % 1000000
	return fmt.Sprintf("%06d", n)
}

// normalizeTaskID 标准化任务ID（忽略前导零）
func normalizeTaskID(taskID string) string {
	// 去除前导零
	n, err := strconv.Atoi(strings.TrimSpace(taskID))
	if err != nil {
		return taskID
	}
	return fmt.Sprintf("%06d", n)
}

func (m *AgentTaskManager) GetTask(taskID string) (*TaskInfo, error) {
	normalizedID := normalizeTaskID(taskID)

	// 先查运行中的任务
	m.mu.RLock()
	task, ok := m.runningTasks[normalizedID]
	m.mu.RUnlock()

	if ok {
		task.mu.Lock()
		defer task.mu.Unlock()
		return &TaskInfo{
			ID:            task.id,
			Status:        task.status,
			ResultSummary: task.result,
		}, nil
	}

	// 从文件加载已完成任务
	return m.loadTaskFromFile(normalizedID)
}

func (m *AgentTaskManager) StopTask(taskID string) (bool, TaskStatus, error) {
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
	case TaskFinished, TaskFailed, TaskStopped:
		return false, task.status, nil
	default:
		task.stopRequested = true
		if task.cancel != nil {
			task.cancel()
		}
		task.status = TaskStopped
		task.appendLog("任务已停止")
		return true, task.status, nil
	}
}

// ListTasks 获取所有任务列表
func (m *AgentTaskManager) ListTasks() ([]*TaskInfo, error) {
	results := make([]*TaskInfo, 0)

	// 运行中的任务
	m.mu.RLock()
	for _, task := range m.runningTasks {
		task.mu.Lock()
		info := &TaskInfo{
			ID:            task.id,
			Status:        task.status,
			ResultSummary: task.result,
		}
		task.mu.Unlock()
		results = append(results, info)
	}
	m.mu.RUnlock()

	// 从文件加载当天已完成的任务
	todayTasks, err := m.loadTodayCompletedTasks()
	if err != nil {
		m.logger.Warn("加载当天任务失败", zap.Error(err))
	} else {
		results = append(results, todayTasks...)
	}

	return results, nil
}

func (m *AgentTaskManager) runTask(ctx context.Context, task *AgentTask, channel, chatID string) {
	execCtx, cancel := m.buildTaskContext(ctx)
	task.mu.Lock()
	task.cancel = cancel
	task.status = TaskRunning
	task.appendLog("任务启动")
	task.mu.Unlock()

	result, err := m.executeTask(execCtx, task.work, channel, chatID)
	task.mu.Lock()
	defer task.mu.Unlock()
	if task.stopRequested || execCtx.Err() == context.Canceled {
		task.status = TaskStopped
		task.appendLog("任务已停止")
		close(task.done)
		m.persistTask(task)
		m.notifyComplete(task, "")
		m.removeFromRunning(task.id)
		return
	}
	if err != nil {
		task.status = TaskFailed
		task.appendLog(fmt.Sprintf("任务失败: %v", err))
	} else {
		task.status = TaskFinished
		task.result = result
		task.appendLog("任务完成")
	}
	close(task.done)
	m.persistTask(task)
	m.notifyComplete(task, result)
	m.removeFromRunning(task.id)
}

// removeFromRunning 从运行中任务列表移除
func (m *AgentTaskManager) removeFromRunning(taskID string) {
	m.mu.Lock()
	delete(m.runningTasks, taskID)
	m.mu.Unlock()
}

// persistTask 持久化任务到文件
func (m *AgentTaskManager) persistTask(task *AgentTask) {
	m.logger.Info("persistTask 被调用",
		zap.String("task_id", task.id),
		zap.String("status", string(task.status)),
		zap.Time("created_at", task.createdAt),
	)

	pt := &PersistedTask{
		ID:          task.id,
		Work:        task.work,
		Status:      task.status,
		Result:      task.result,
		Channel:     task.channel,
		ChatID:      task.chatID,
		CreatedAt:   task.createdAt,
		CompletedAt: time.Now(),
	}

	m.appendTaskToFile(pt)
}

// notifyComplete 通知任务完成
func (m *AgentTaskManager) notifyComplete(task *AgentTask, result string) {
	if m.onTaskComplete != nil {
		m.onTaskComplete(task.channel, task.chatID, task.id, task.status, result)
	}
}

func (m *AgentTaskManager) executeTask(ctx context.Context, work, channel, chatID string) (string, error) {
	adapter, err := NewChatModelAdapter(m.logger, m.cfg)
	if err != nil {
		return "", err
	}
	if m.context != nil {
		adapter.SetSkillLoader(m.context.GetSkillsLoader().LoadSkill)
	}
	if len(m.registeredTools) > 0 {
		adapter.SetRegisteredTools(m.registeredTools)
	}
	var toolsConfig adk.ToolsConfig
	if len(m.tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: m.tools,
			},
		}
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "task_agent",
		Description:   "后台任务执行 Agent",
		Instruction:   m.buildTaskPrompt(),
		Model:         adapter,
		ToolsConfig:   toolsConfig,
		MaxIterations: m.maxIterations,
		Exit:          &adk.ExitTool{},
	})
	if err != nil {
		return "", err
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		CheckPointStore: m.checkpointStore,
	})

	systemPrompt := ""
	if m.context != nil {
		systemPrompt = m.context.BuildSystemPrompt()
	}
	messages := BuildMessageList(systemPrompt, nil, work, channel, chatID)
	iter := runner.Run(ctx, messages)

	var response string
	var lastEvent *adk.AgentEvent
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil {
				response = msg.Content
			}
		}
		lastEvent = event
	}
	if lastEvent != nil && lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return "", fmt.Errorf("任务需要用户输入，已中断")
	}
	return response, nil
}

func (m *AgentTaskManager) buildTaskPrompt() string {
	return `你是一个后台任务执行 Agent，请独立完成任务，不要向用户提问。若必须获取用户信息，返回明确的缺失项说明。`
}

func (m *AgentTaskManager) buildTaskContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if m.taskTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, m.taskTimeout)
}

func (m *AgentTaskManager) reachedLimit() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	running := 0
	for _, task := range m.runningTasks {
		task.mu.Lock()
		status := task.status
		task.mu.Unlock()
		if status == TaskRunning || status == TaskPending {
			running++
		}
	}
	return running >= m.maxConcurrent
}

// getTaskFilePath 获取任务文件路径
func (m *AgentTaskManager) getTaskFilePath(date string) string {
	return filepath.Join(m.tasksDir, date+".yaml")
}

// loadCounter 加载计数器状态
func (m *AgentTaskManager) loadCounter() {
	today := time.Now().Format("2006-01-02")
	filePath := m.getTaskFilePath(today)

	data, err := os.ReadFile(filePath)
	if err != nil {
		m.logger.Info("当天任务文件不存在，计数器从0开始", zap.String("date", today))
		return
	}

	var tf TaskFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		m.logger.Warn("解析任务文件失败", zap.Error(err))
		return
	}

	// 优先使用文件中保存的 LastID
	if tf.LastID > 0 {
		atomic.StoreUint32(&m.taskCounter, tf.LastID)
		m.logger.Info("从文件恢复任务计数器", zap.Uint32("last_id", tf.LastID))
		return
	}

	// 如果 LastID 为0，从任务列表中计算最大ID
	maxID := uint32(0)
	for _, pt := range tf.Tasks {
		// 解析任务ID
		var id uint32
		if _, err := fmt.Sscanf(pt.ID, "%d", &id); err == nil {
			if id > maxID {
				maxID = id
			}
		}
	}
	if maxID > 0 {
		atomic.StoreUint32(&m.taskCounter, maxID)
		m.logger.Info("从任务列表恢复计数器", zap.Uint32("max_id", maxID))
	}
}

// loadTodayCompletedTasks 加载当天已完成的任务
func (m *AgentTaskManager) loadTodayCompletedTasks() ([]*TaskInfo, error) {
	today := time.Now().Format("2006-01-02")
	filePath := m.getTaskFilePath(today)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tf TaskFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, err
	}

	results := make([]*TaskInfo, 0, len(tf.Tasks))
	for _, pt := range tf.Tasks {
		// 只返回已完成的任务
		if pt.Status != TaskPending && pt.Status != TaskRunning {
			results = append(results, &TaskInfo{
				ID:            pt.ID,
				Status:        pt.Status,
				ResultSummary: pt.Result,
			})
		}
	}

	return results, nil
}

// loadTaskFromFile 从文件加载任务
func (m *AgentTaskManager) loadTaskFromFile(taskID string) (*TaskInfo, error) {
	// 遍历任务目录下的所有yaml文件
	files, err := filepath.Glob(filepath.Join(m.tasksDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("读取任务目录失败: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var tf TaskFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			continue
		}

		for _, pt := range tf.Tasks {
			if normalizeTaskID(pt.ID) == taskID {
				return &TaskInfo{
					ID:            pt.ID,
					Status:        pt.Status,
					ResultSummary: pt.Result,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("任务不存在")
}

// appendTaskToFile 追加任务到文件（根据创建时间决定写入哪一天的文件）
func (m *AgentTaskManager) appendTaskToFile(pt *PersistedTask) {
	m.persistMu.Lock()
	defer m.persistMu.Unlock()

	// 根据任务创建时间决定写入哪一天的文件
	date := pt.CreatedAt.Format("2006-01-02")
	filePath := m.getTaskFilePath(date)

	m.logger.Info("准备持久化任务",
		zap.String("task_id", pt.ID),
		zap.String("status", string(pt.Status)),
		zap.String("tasks_dir", m.tasksDir),
		zap.String("file", filePath),
	)

	// 确保目录存在
	if err := os.MkdirAll(m.tasksDir, 0755); err != nil {
		m.logger.Error("创建任务目录失败", zap.Error(err), zap.String("tasks_dir", m.tasksDir))
		return
	}

	// 读取现有文件
	var tf TaskFile
	data, err := os.ReadFile(filePath)
	if err == nil {
		yaml.Unmarshal(data, &tf)
	}

	// 检查是否已存在（避免重复追加）
	for _, existing := range tf.Tasks {
		if existing.ID == pt.ID {
			m.logger.Debug("任务已存在，跳过追加", zap.String("task_id", pt.ID))
			return
		}
	}

	// 更新日期、计数器和任务列表
	tf.Date = date
	tf.LastID = atomic.LoadUint32(&m.taskCounter)
	tf.Tasks = append(tf.Tasks, pt)

	// 写入文件
	out, err := yaml.Marshal(&tf)
	if err != nil {
		m.logger.Error("序列化任务失败", zap.Error(err))
		return
	}

	if err := os.WriteFile(filePath, out, 0644); err != nil {
		m.logger.Error("写入任务文件失败", zap.Error(err), zap.String("file", filePath))
		return
	}

	m.logger.Info("持久化任务成功", zap.String("task_id", pt.ID), zap.String("file", filePath))
}

// Close 关闭任务管理器
func (m *AgentTaskManager) Close() {
	// 无需额外操作，任务完成时已持久化
}

func (t *AgentTask) appendLog(message string) {
	entry := fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), message)
	if len(t.lastLogs) >= t.logCapacity {
		t.lastLogs = t.lastLogs[1:]
	}
	t.lastLogs = append(t.lastLogs, entry)
}

// TaskManagerAdapter 任务管理器工具适配器
type TaskManagerAdapter struct {
	manager *AgentTaskManager
}

// NewTaskManagerAdapter 创建任务管理器工具适配器
func NewTaskManagerAdapter(manager *AgentTaskManager) *TaskManagerAdapter {
	return &TaskManagerAdapter{manager: manager}
}

// StartTask 启动任务并返回任务ID与状态
func (a *TaskManagerAdapter) StartTask(ctx context.Context, work, channel, chatID string) (string, string, error) {
	if a.manager == nil {
		return "", "", fmt.Errorf("任务管理器未初始化")
	}
	taskID, status, err := a.manager.StartTask(ctx, work, channel, chatID)
	return taskID, string(status), err
}

// GetTask 查询任务信息
func (a *TaskManagerAdapter) GetTask(ctx context.Context, taskID string) (*tasktools.TaskInfo, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("任务管理器未初始化")
	}
	info, err := a.manager.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	return &tasktools.TaskInfo{
		ID:            info.ID,
		Status:        string(info.Status),
		ResultSummary: info.ResultSummary,
	}, nil
}

// StopTask 停止任务并返回结果
func (a *TaskManagerAdapter) StopTask(ctx context.Context, taskID string) (bool, string, error) {
	if a.manager == nil {
		return false, "", fmt.Errorf("任务管理器未初始化")
	}
	stopped, status, err := a.manager.StopTask(taskID)
	return stopped, string(status), err
}

// ListTasks 获取任务列表
func (a *TaskManagerAdapter) ListTasks() ([]*tasktools.TaskInfo, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("任务管理器未初始化")
	}
	items, err := a.manager.ListTasks()
	if err != nil {
		return nil, err
	}
	result := make([]*tasktools.TaskInfo, 0, len(items))
	for _, item := range items {
		result = append(result, &tasktools.TaskInfo{
			ID:            item.ID,
			Status:        string(item.Status),
			ResultSummary: item.ResultSummary,
		})
	}
	return result, nil
}
