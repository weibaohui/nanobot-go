package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Persistence 任务持久化
type Persistence struct {
	tasksDir string
	logger   *zap.Logger
	counter  *uint32
	mu       sync.Mutex
}

// NewPersistence 创建持久化管理器
func NewPersistence(tasksDir string, logger *zap.Logger, counter *uint32) *Persistence {
	return &Persistence{
		tasksDir: tasksDir,
		logger:   logger,
		counter:  counter,
	}
}

// GetTaskFilePath 获取任务文件路径
func (p *Persistence) GetTaskFilePath(date string) string {
	return filepath.Join(p.tasksDir, date+".yaml")
}

// LoadCounter 加载计数器状态
func (p *Persistence) LoadCounter() {
	today := time.Now().Format("2006-01-02")
	filePath := p.GetTaskFilePath(today)

	data, err := os.ReadFile(filePath)
	if err != nil {
		p.logger.Info("当天任务文件不存在，计数器从0开始", zap.String("date", today))
		return
	}

	var tf File
	if err := yaml.Unmarshal(data, &tf); err != nil {
		p.logger.Warn("解析任务文件失败", zap.Error(err))
		return
	}

	if tf.LastID > 0 {
		atomic.StoreUint32(p.counter, tf.LastID)
		p.logger.Info("从文件恢复任务计数器", zap.Uint32("last_id", tf.LastID))
		return
	}

	// 如果 LastID 为0，从任务列表中计算最大ID
	maxID := uint32(0)
	for _, pt := range tf.Tasks {
		var id uint32
		if _, err := fmt.Sscanf(pt.ID, "%d", &id); err == nil {
			if id > maxID {
				maxID = id
			}
		}
	}
	if maxID > 0 {
		atomic.StoreUint32(p.counter, maxID)
		p.logger.Info("从任务列表恢复计数器", zap.Uint32("max_id", maxID))
	}
}

// LoadTodayCompletedTasks 加载当天已完成的任务
func (p *Persistence) LoadTodayCompletedTasks() ([]*Info, error) {
	today := time.Now().Format("2006-01-02")
	filePath := p.GetTaskFilePath(today)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tf File
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, err
	}

	results := make([]*Info, 0, len(tf.Tasks))
	for _, pt := range tf.Tasks {
		if pt.Status != StatusPending && pt.Status != StatusRunning {
			results = append(results, &Info{
				ID:            pt.ID,
				Status:        pt.Status,
				ResultSummary: pt.Result,
				Work:          pt.Work,
				Channel:       pt.Channel,
				ChatID:        pt.ChatID,
				CreatedAt:     pt.CreatedAt,
				CompletedAt:   pt.CompletedAt,
			})
		}
	}

	return results, nil
}

// LoadTaskFromFile 从文件加载任务
func (p *Persistence) LoadTaskFromFile(taskID string) (*Info, error) {
	files, err := filepath.Glob(filepath.Join(p.tasksDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("读取任务目录失败: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var tf File
		if err := yaml.Unmarshal(data, &tf); err != nil {
			continue
		}

		for _, pt := range tf.Tasks {
			if normalizeTaskID(pt.ID) == taskID {
				return &Info{
					ID:            pt.ID,
					Status:        pt.Status,
					ResultSummary: pt.Result,
					Work:          pt.Work,
					Channel:       pt.Channel,
					ChatID:        pt.ChatID,
					CreatedAt:     pt.CreatedAt,
					CompletedAt:   pt.CompletedAt,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("任务不存在")
}

// AppendTaskToFile 追加任务到文件
func (p *Persistence) AppendTaskToFile(pt *PersistedTask) {
	p.mu.Lock()
	defer p.mu.Unlock()

	date := pt.CreatedAt.Format("2006-01-02")
	filePath := p.GetTaskFilePath(date)

	p.logger.Info("准备持久化任务",
		zap.String("task_id", pt.ID),
		zap.String("status", string(pt.Status)),
		zap.String("file", filePath),
	)

	if err := os.MkdirAll(p.tasksDir, 0755); err != nil {
		p.logger.Error("创建任务目录失败", zap.Error(err))
		return
	}

	var tf File
	data, err := os.ReadFile(filePath)
	if err == nil {
		yaml.Unmarshal(data, &tf)
	}

	for _, existing := range tf.Tasks {
		if existing.ID == pt.ID {
			p.logger.Debug("任务已存在，跳过追加", zap.String("task_id", pt.ID))
			return
		}
	}

	tf.Date = date
	tf.LastID = atomic.LoadUint32(p.counter)
	tf.Tasks = append(tf.Tasks, pt)

	out, err := yaml.Marshal(&tf)
	if err != nil {
		p.logger.Error("序列化任务失败", zap.Error(err))
		return
	}

	if err := os.WriteFile(filePath, out, 0644); err != nil {
		p.logger.Error("写入任务文件失败", zap.Error(err), zap.String("file", filePath))
		return
	}

	p.logger.Info("持久化任务成功", zap.String("task_id", pt.ID), zap.String("file", filePath))
}
