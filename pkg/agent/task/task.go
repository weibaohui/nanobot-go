package task

import (
	"fmt"
	"sync"
	"time"
)

// Task 后台任务
type Task struct {
	id            string
	work          string
	status        Status
	result        string
	lastLogs      []string
	logCapacity   int
	stopRequested bool
	cancel        func()
	done          chan struct{}
	mu            sync.Mutex
	channel       string
	chatID        string
	createdAt     time.Time
}

// NewTask 创建新任务
func NewTask(id, work, channel, chatID string, logCapacity int) *Task {
	return &Task{
		id:          id,
		work:        work,
		status:      StatusPending,
		logCapacity: logCapacity,
		done:        make(chan struct{}),
		channel:     channel,
		chatID:      chatID,
		createdAt:   time.Now(),
	}
}

// ID 返回任务ID
func (t *Task) ID() string {
	return t.id
}

// Work 返回任务内容
func (t *Task) Work() string {
	return t.work
}

// Status 返回任务状态
func (t *Task) Status() Status {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status
}

// Result 返回任务结果
func (t *Task) Result() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.result
}

// Channel 返回渠道
func (t *Task) Channel() string {
	return t.channel
}

// ChatID 返回聊天ID
func (t *Task) ChatID() string {
	return t.chatID
}

// CreatedAt 返回创建时间
func (t *Task) CreatedAt() time.Time {
	return t.createdAt
}

// SetStatus 设置状态
func (t *Task) SetStatus(status Status) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = status
}

// SetResult 设置结果
func (t *Task) SetResult(result string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.result = result
}

// SetCancel 设置取消函数
func (t *Task) SetCancel(cancel func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancel = cancel
}

// Cancel 取消任务
func (t *Task) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopRequested = true
	if t.cancel != nil {
		t.cancel()
	}
}

// IsStopRequested 检查是否请求停止
func (t *Task) IsStopRequested() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stopRequested
}

// Done 返回完成通道
func (t *Task) Done() chan struct{} {
	return t.done
}

// CloseDone 关闭完成通道
func (t *Task) CloseDone() {
	close(t.done)
}

// AppendLog 添加日志
func (t *Task) AppendLog(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry := fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), message)
	if len(t.lastLogs) >= t.logCapacity {
		t.lastLogs = t.lastLogs[1:]
	}
	t.lastLogs = append(t.lastLogs, entry)
}

// GetLogs 获取日志
func (t *Task) GetLogs() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]string, len(t.lastLogs))
	copy(result, t.lastLogs)
	return result
}

// ToInfo 转换为任务信息
func (t *Task) ToInfo() *Info {
	t.mu.Lock()
	defer t.mu.Unlock()

	logs := make([]string, len(t.lastLogs))
	copy(logs, t.lastLogs)

	return &Info{
		ID:            t.id,
		Status:        t.status,
		ResultSummary: t.result,
		Work:          t.work,
		Channel:       t.channel,
		ChatID:        t.chatID,
		CreatedAt:     t.createdAt,
		LastLogs:      logs,
	}
}

// ToPersistedTask 转换为持久化任务
func (t *Task) ToPersistedTask() *PersistedTask {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.toPersistedTaskInternal()
}

// toPersistedTaskInternal 内部方法，不获取锁，调用者必须持有锁
func (t *Task) toPersistedTaskInternal() *PersistedTask {
	return &PersistedTask{
		ID:          t.id,
		Work:        t.work,
		Status:      t.status,
		Result:      t.result,
		Channel:     t.channel,
		ChatID:      t.chatID,
		CreatedAt:   t.createdAt,
		CompletedAt: time.Now(),
	}
}

// appendLogInternal 内部方法，不获取锁，调用者必须持有锁
func (t *Task) appendLogInternal(message string) {
	entry := fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), message)
	if len(t.lastLogs) >= t.logCapacity {
		t.lastLogs = t.lastLogs[1:]
	}
	t.lastLogs = append(t.lastLogs, entry)
}
