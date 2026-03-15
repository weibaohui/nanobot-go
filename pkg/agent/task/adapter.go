package task

import (
	"context"
	"fmt"

	tasktools "github.com/weibaohui/nanobot-go/pkg/agent/tools/task"
)

// Adapter 任务管理器工具适配器
type Adapter struct {
	manager *Manager
}

// NewAdapter 创建任务管理器工具适配器
func NewAdapter(manager *Manager) *Adapter {
	return &Adapter{manager: manager}
}

// StartTask 启动任务并返回任务ID与状态
func (a *Adapter) StartTask(ctx context.Context, work, channel, chatID string) (string, string, error) {
	if a.manager == nil {
		return "", "", fmt.Errorf("任务管理器未初始化")
	}
	taskID, status, err := a.manager.StartTask(ctx, work, channel, chatID)
	return taskID, string(status), err
}

// GetTask 查询任务信息
func (a *Adapter) GetTask(ctx context.Context, taskID string) (*tasktools.TaskInfo, error) {
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
func (a *Adapter) StopTask(ctx context.Context, taskID string) (bool, string, error) {
	if a.manager == nil {
		return false, "", fmt.Errorf("任务管理器未初始化")
	}
	stopped, status, err := a.manager.StopTask(taskID)
	return stopped, string(status), err
}

// ListTasks 获取任务列表
func (a *Adapter) ListTasks() ([]*tasktools.TaskInfo, error) {
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
