package task

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/weibaohui/nanobot-go/pkg/agent/task"
)

// 定义错误类型，便于调用方区分不同情况
var (
	ErrManagerNotInitialized = errors.New("task manager not initialized")
	ErrTaskNotFound          = errors.New("task not found or already completed")
)

// Service Task 服务接口
type Service interface {
	// ListTasks 获取所有任务列表
	ListTasks() ([]*TaskResponse, error)
	// ListTasksWithFilter 带筛选的任务列表
	ListTasksWithFilter(filter *TaskFilter) ([]*TaskResponse, error)
	// GetTask 获取任务详情
	GetTask(id string) (*TaskDetailResponse, error)
	// CreateTask 手动创建任务
	CreateTask(work, createdBy string) (*TaskResponse, error)
	// StopTask 停止任务
	StopTask(id string) (*TaskResponse, error)
	// RetryTask 重试任务
	RetryTask(id, createdBy string) (*TaskResponse, error)
}

// TaskFilter 任务筛选条件
type TaskFilter struct {
	Status    []string  // 状态列表
	Since     time.Time // 起始时间
	Until     time.Time // 结束时间
	Keyword   string    // 关键词
	CreatedBy string    // 创建者
	IsAdmin   bool      // 是否管理员
}

// TaskResponse 任务 API 响应结构
type TaskResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Work        string `json:"work"`
	Channel     string `json:"channel,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	Result      string `json:"result,omitempty"`
}

// TaskDetailResponse 任务详情响应结构
type TaskDetailResponse struct {
	TaskResponse
	Logs []string `json:"logs,omitempty"`
}

// service Task 服务实现
type service struct {
	manager *task.Manager
}

// NewService 创建 Task 服务
func NewService(manager *task.Manager) Service {
	return &service{manager: manager}
}

// ListTasks 获取所有任务列表
func (s *service) ListTasks() ([]*TaskResponse, error) {
	if s.manager == nil {
		return []*TaskResponse{}, nil
	}

	tasks, err := s.manager.ListTasks()
	if err != nil {
		return nil, err
	}

	results := make([]*TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		results = append(results, convertToTaskResponse(t))
	}

	return results, nil
}

// GetTask 获取任务详情
func (s *service) GetTask(id string) (*TaskDetailResponse, error) {
	if s.manager == nil {
		return nil, ErrManagerNotInitialized
	}

	info, err := s.manager.GetTask(id)
	if err != nil {
		return nil, err
	}

	if info == nil {
		return nil, nil
	}

	return &TaskDetailResponse{
		TaskResponse: *convertToTaskResponse(info),
		Logs:         info.LastLogs,
	}, nil
}

// StopTask 停止任务
func (s *service) StopTask(id string) (*TaskResponse, error) {
	if s.manager == nil {
		return nil, ErrManagerNotInitialized
	}

	stopped, status, err := s.manager.StopTask(id)
	if err != nil {
		return nil, err
	}

	if !stopped {
		return nil, ErrTaskNotFound
	}

	return &TaskResponse{
		ID:     id,
		Status: string(status),
	}, nil
}

// CreateTask 手动创建任务
func (s *service) CreateTask(work, createdBy string) (*TaskResponse, error) {
	if s.manager == nil {
		return nil, ErrManagerNotInitialized
	}

	if work == "" {
		return nil, errors.New("任务内容不能为空")
	}

	ctx := context.Background()
	taskID, status, err := s.manager.StartTask(ctx, work, "manual", "", createdBy)
	if err != nil {
		return nil, err
	}

	return &TaskResponse{
		ID:        taskID,
		Status:    string(status),
		Work:      work,
		Channel:   "manual",
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// RetryTask 重试任务
func (s *service) RetryTask(id, createdBy string) (*TaskResponse, error) {
	if s.manager == nil {
		return nil, ErrManagerNotInitialized
	}

	// 获取原任务信息
	info, err := s.manager.GetTask(id)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, ErrTaskNotFound
	}

	// 只能重试已完成、失败或已停止的任务
	if info.Status != task.StatusFinished && info.Status != task.StatusFailed && info.Status != task.StatusStopped {
		return nil, errors.New("只能重试已完成、失败或已停止的任务")
	}

	// 创建新任务，使用相同的内容
	ctx := context.Background()
	newTaskID, status, err := s.manager.StartTask(ctx, info.Work, "manual", "", createdBy)
	if err != nil {
		return nil, err
	}

	return &TaskResponse{
		ID:        newTaskID,
		Status:    string(status),
		Work:      info.Work,
		Channel:   "manual",
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// ListTasksWithFilter 带筛选的任务列表
func (s *service) ListTasksWithFilter(filter *TaskFilter) ([]*TaskResponse, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}

	// 应用筛选条件
	var results []*TaskResponse
	for _, t := range tasks {
		// 状态筛选
		if len(filter.Status) > 0 {
			matched := false
			for _, s := range filter.Status {
				if strings.ToLower(t.Status) == strings.ToLower(s) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 时间范围筛选
		if !filter.Since.IsZero() || !filter.Until.IsZero() {
			createdAt, _ := time.Parse(time.RFC3339, t.CreatedAt)
			if !filter.Since.IsZero() && createdAt.Before(filter.Since) {
				continue
			}
			if !filter.Until.IsZero() && createdAt.After(filter.Until) {
				continue
			}
		}

		// 关键词筛选（匹配任务内容）
		if filter.Keyword != "" {
			if !strings.Contains(strings.ToLower(t.Work), strings.ToLower(filter.Keyword)) {
				continue
			}
		}

		// 创建者筛选（非管理员只能看到自己的）
		if !filter.IsAdmin && filter.CreatedBy != "" {
			// TODO: 需要存储任务的创建者信息，当前暂时不处理
		}

		results = append(results, t)
	}

	return results, nil
}

// convertToTaskResponse 将 task.Info 转换为 TaskResponse
func convertToTaskResponse(info *task.Info) *TaskResponse {
	if info == nil {
		return nil
	}

	resp := &TaskResponse{
		ID:       info.ID,
		Status:   string(info.Status),
		Result:   info.ResultSummary,
		Work:     info.Work,
		Channel:  info.Channel,
		ChatID:   info.ChatID,
	}

	if !info.CreatedAt.IsZero() {
		resp.CreatedAt = info.CreatedAt.Format(time.RFC3339)
	}
	if !info.CompletedAt.IsZero() {
		resp.CompletedAt = info.CompletedAt.Format(time.RFC3339)
	}

	return resp
}
