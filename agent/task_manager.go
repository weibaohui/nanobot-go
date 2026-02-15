package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/google/uuid"
	tasktools "github.com/weibaohui/nanobot-go/agent/tools/task"
	"github.com/weibaohui/nanobot-go/config"
	"go.uber.org/zap"
)

type TaskStatus string

const (
	TaskPending  TaskStatus = "pending"
	TaskRunning  TaskStatus = "running"
	TaskFinished TaskStatus = "finished"
	TaskFailed   TaskStatus = "failed"
	TaskStopped  TaskStatus = "stopped"
)

type TaskContext struct {
	SessionKey string
	Channel    string
	ChatID     string
}

type TaskInfo struct {
	ID            string
	Status        TaskStatus
	LastLogs      []string
	ResultSummary string
}

// AgentTaskManagerInterface 任务管理器接口
type AgentTaskManagerInterface interface {
	// StartTask 创建后台任务并返回任务 ID
	StartTask(ctx context.Context, work string, taskCtx TaskContext) (string, TaskStatus, error)
	// GetTask 查询任务状态与最近日志
	GetTask(taskID string, requesterKey string) (*TaskInfo, error)
	// StopTask 停止后台任务并返回是否成功
	StopTask(taskID string, requesterKey string) (bool, TaskStatus, error)
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

	mu    sync.RWMutex
	tasks map[string]*AgentTask
}

type AgentTask struct {
	id            string
	ownerKey      string
	work          string
	status        TaskStatus
	result        string
	lastLogs      []string
	logCapacity   int
	stopRequested bool
	cancel        context.CancelFunc
	done          chan struct{}
	mu            sync.Mutex
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

	return &AgentTaskManager{
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
		tasks:           make(map[string]*AgentTask),
	}, nil
}

// SetRegisteredTools 设置已注册的工具名称
func (m *AgentTaskManager) SetRegisteredTools(names []string) {
	m.registeredTools = append([]string(nil), names...)
}

func (m *AgentTaskManager) StartTask(ctx context.Context, work string, taskCtx TaskContext) (string, TaskStatus, error) {
	if work == "" {
		return "", "", fmt.Errorf("任务内容不能为空")
	}
	if m.reachedLimit() {
		return "", "", fmt.Errorf("任务并发已达上限")
	}

	taskID := uuid.NewString()
	task := &AgentTask{
		id:          taskID,
		ownerKey:    taskCtx.SessionKey,
		work:        work,
		status:      TaskPending,
		logCapacity: m.logCapacity,
		done:        make(chan struct{}),
	}
	task.appendLog("任务已创建")

	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	go m.runTask(ctx, task, taskCtx)

	return taskID, TaskRunning, nil
}

func (m *AgentTaskManager) GetTask(taskID string, requesterKey string) (*TaskInfo, error) {
	task := m.getTask(taskID)
	if task == nil {
		return nil, fmt.Errorf("任务不存在")
	}
	if !task.isOwner(requesterKey) {
		return nil, fmt.Errorf("无权限访问任务")
	}
	task.mu.Lock()
	defer task.mu.Unlock()
	return &TaskInfo{
		ID:            task.id,
		Status:        task.status,
		LastLogs:      append([]string(nil), task.lastLogs...),
		ResultSummary: task.result,
	}, nil
}

func (m *AgentTaskManager) StopTask(taskID string, requesterKey string) (bool, TaskStatus, error) {
	task := m.getTask(taskID)
	if task == nil {
		return false, "", fmt.Errorf("任务不存在")
	}
	if !task.isOwner(requesterKey) {
		return false, "", fmt.Errorf("无权限访问任务")
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

func (m *AgentTaskManager) runTask(ctx context.Context, task *AgentTask, taskCtx TaskContext) {
	taskCtxLocal := taskCtx
	execCtx, cancel := m.buildTaskContext(ctx)
	task.mu.Lock()
	task.cancel = cancel
	task.status = TaskRunning
	task.appendLog("任务启动")
	task.mu.Unlock()

	result, err := m.executeTask(execCtx, task.work, taskCtxLocal)
	task.mu.Lock()
	defer task.mu.Unlock()
	if task.stopRequested || execCtx.Err() == context.Canceled {
		task.status = TaskStopped
		task.appendLog("任务已停止")
		close(task.done)
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
}

func (m *AgentTaskManager) executeTask(ctx context.Context, work string, taskCtx TaskContext) (string, error) {
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
	messages := BuildMessageList(systemPrompt, nil, work, taskCtx.Channel, taskCtx.ChatID)
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
	for _, task := range m.tasks {
		task.mu.Lock()
		status := task.status
		task.mu.Unlock()
		if status == TaskRunning || status == TaskPending {
			running++
		}
	}
	return running >= m.maxConcurrent
}

func (m *AgentTaskManager) getTask(taskID string) *AgentTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[taskID]
}

func (t *AgentTask) appendLog(message string) {
	entry := fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), message)
	if len(t.lastLogs) >= t.logCapacity {
		t.lastLogs = t.lastLogs[1:]
	}
	t.lastLogs = append(t.lastLogs, entry)
}

func (t *AgentTask) isOwner(requesterKey string) bool {
	if t.ownerKey == "" {
		return requesterKey == ""
	}
	return requesterKey != "" && requesterKey == t.ownerKey
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
func (a *TaskManagerAdapter) StartTask(ctx context.Context, work, sessionKey, channel, chatID string) (string, string, error) {
	if a.manager == nil {
		return "", "", fmt.Errorf("任务管理器未初始化")
	}
	taskID, status, err := a.manager.StartTask(ctx, work, TaskContext{
		SessionKey: sessionKey,
		Channel:    channel,
		ChatID:     chatID,
	})
	return taskID, string(status), err
}

// GetTask 查询任务信息
func (a *TaskManagerAdapter) GetTask(ctx context.Context, taskID, requesterKey string) (*tasktools.TaskInfo, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("任务管理器未初始化")
	}
	info, err := a.manager.GetTask(taskID, requesterKey)
	if err != nil {
		return nil, err
	}
	return &tasktools.TaskInfo{
		ID:            info.ID,
		Status:        string(info.Status),
		LastLogs:      append([]string(nil), info.LastLogs...),
		ResultSummary: info.ResultSummary,
	}, nil
}

// StopTask 停止任务并返回结果
func (a *TaskManagerAdapter) StopTask(ctx context.Context, taskID, requesterKey string) (bool, string, error) {
	if a.manager == nil {
		return false, "", fmt.Errorf("任务管理器未初始化")
	}
	stopped, status, err := a.manager.StopTask(taskID, requesterKey)
	return stopped, string(status), err
}
