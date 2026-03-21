package task

import (
	"time"

	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// TaskEventType 任务事件类型
type TaskEventType string

const (
	// TaskEventCreated 任务创建事件
	TaskEventCreated TaskEventType = "task_created"
	// TaskEventUpdated 任务更新事件
	TaskEventUpdated TaskEventType = "task_updated"
	// TaskEventCompleted 任务完成事件
	TaskEventCompleted TaskEventType = "task_completed"
	// TaskEventLog 任务日志事件
	TaskEventLog TaskEventType = "task_log"
)

// TaskEvent 任务事件
type TaskEvent struct {
	Type      TaskEventType `json:"type"`
	TaskID    string        `json:"task_id"`
	Payload   any           `json:"payload"`
	Timestamp time.Time     `json:"timestamp"`
}

// TaskCreatedPayload 任务创建事件载荷
type TaskCreatedPayload struct {
	ID        string    `json:"id"`
	Status    Status    `json:"status"`
	Work      string    `json:"work"`
	Channel   string    `json:"channel,omitempty"`
	ChatID    string    `json:"chat_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by,omitempty"`
}

// TaskUpdatedPayload 任务更新事件载荷
type TaskUpdatedPayload struct {
	ID        string    `json:"id"`
	Status    Status    `json:"status"`
	Result    string    `json:"result,omitempty"`
	Logs      []string  `json:"logs,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskCompletedPayload 任务完成事件载荷
type TaskCompletedPayload struct {
	ID              string    `json:"id"`
	Status          Status    `json:"status"`
	Result          string    `json:"result,omitempty"`
	Logs            []string  `json:"logs,omitempty"`
	CompletedAt     time.Time `json:"completed_at"`
	DurationSeconds int       `json:"duration_seconds"`
}

// TaskLogPayload 任务日志事件载荷
type TaskLogPayload struct {
	ID        string    `json:"id"`
	Log       string    `json:"log"`
	Timestamp time.Time `json:"timestamp"`
}

// EventPublisher 任务事件发布器
type EventPublisher struct {
	bus    *bus.MessageBus
	logger *zap.Logger
}

// NewEventPublisher 创建任务事件发布器
func NewEventPublisher(b *bus.MessageBus, logger *zap.Logger) *EventPublisher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EventPublisher{
		bus:    b,
		logger: logger,
	}
}

// PublishTaskCreated 发布任务创建事件
func (p *EventPublisher) PublishTaskCreated(task *Task, createdBy string) {
	event := &TaskEvent{
		Type:   TaskEventCreated,
		TaskID: task.ID(),
		Payload: TaskCreatedPayload{
			ID:        task.ID(),
			Status:    task.Status(),
			Work:      task.Work(),
			Channel:   task.Channel(),
			ChatID:    task.ChatID(),
			CreatedAt: task.CreatedAt(),
			CreatedBy: createdBy,
		},
		Timestamp: time.Now(),
	}
	p.publish(event)
}

// PublishTaskUpdated 发布任务更新事件
func (p *EventPublisher) PublishTaskUpdated(task *Task) {
	event := &TaskEvent{
		Type:   TaskEventUpdated,
		TaskID: task.ID(),
		Payload: TaskUpdatedPayload{
			ID:        task.ID(),
			Status:    task.Status(),
			Result:    task.Result(),
			Logs:      task.GetLogs(),
			UpdatedAt: time.Now(),
		},
		Timestamp: time.Now(),
	}
	p.publish(event)
}

// PublishTaskCompleted 发布任务完成事件
func (p *EventPublisher) PublishTaskCompleted(task *Task, duration time.Duration) {
	event := &TaskEvent{
		Type:   TaskEventCompleted,
		TaskID: task.ID(),
		Payload: TaskCompletedPayload{
			ID:              task.ID(),
			Status:          task.Status(),
			Result:          task.Result(),
			Logs:            task.GetLogs(),
			CompletedAt:     time.Now(),
			DurationSeconds: int(duration.Seconds()),
		},
		Timestamp: time.Now(),
	}
	p.publish(event)
}

// PublishTaskLog 发布任务日志事件
func (p *EventPublisher) PublishTaskLog(taskID string, log string) {
	event := &TaskEvent{
		Type:   TaskEventLog,
		TaskID: taskID,
		Payload: TaskLogPayload{
			ID:        taskID,
			Log:       log,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}
	p.publish(event)
}

// publish 发布事件到MessageBus
func (p *EventPublisher) publish(event *TaskEvent) {
	if p.bus == nil {
		return
	}

	// 使用 MessageBus 的 PublishTaskEvent 方法分发任务事件
	payload := make(map[string]any)
	payload["event_type"] = event.Type
	payload["task_event"] = event
	p.bus.PublishTaskEvent(string(event.Type), payload)
}
