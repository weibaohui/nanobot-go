package task

import (
	"time"

	"github.com/weibaohui/nanobot-go/pkg/agent/task"
)

// Service Task 服务接口
type Service interface {
	// ListTasks 获取所有任务列表
	ListTasks() ([]*TaskResponse, error)
	// GetTask 获取任务详情
	GetTask(id string) (*TaskDetailResponse, error)
	// StopTask 停止任务
	StopTask(id string) (*TaskResponse, error)
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
		return nil, nil
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
		return nil, nil
	}

	stopped, status, err := s.manager.StopTask(id)
	if err != nil {
		return nil, err
	}

	if !stopped {
		return nil, nil
	}

	return &TaskResponse{
		ID:     id,
		Status: string(status),
	}, nil
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
