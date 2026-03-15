package interrupt

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
)

// InMemoryCheckpointStore 内存 Checkpoint 存储
type InMemoryCheckpointStore struct {
	mem      map[string]*checkpointEntry
	mu       sync.RWMutex
	maxSize  int
	stopChan chan struct{}
}

// checkpointEntry 带时间戳的 checkpoint 条目
type checkpointEntry struct {
	value     []byte
	createdAt time.Time
}

// NewInMemoryCheckpointStore 创建内存 Checkpoint 存储
func NewInMemoryCheckpointStore() compose.CheckPointStore {
	store := &InMemoryCheckpointStore{
		mem:      make(map[string]*checkpointEntry),
		maxSize:  1000,
		stopChan: make(chan struct{}),
	}
	go store.cleanupLoop()
	return store
}

// cleanupLoop 定期清理过期的 checkpoint
func (s *InMemoryCheckpointStore) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanExpired()
		case <-s.stopChan:
			return
		}
	}
}

// cleanExpired 清理过期的 checkpoint（超过 1 小时）
func (s *InMemoryCheckpointStore) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	expireThreshold := time.Now().Add(-1 * time.Hour)
	for key, entry := range s.mem {
		if entry.createdAt.Before(expireThreshold) {
			delete(s.mem, key)
		}
	}
}

// Close 停止清理 goroutine
func (s *InMemoryCheckpointStore) Close() {
	close(s.stopChan)
}

// Delete 删除 checkpoint
func (s *InMemoryCheckpointStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.mem, key)
	return nil
}

// Set 保存 checkpoint
func (s *InMemoryCheckpointStore) Set(ctx context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.mem) >= s.maxSize {
		s.evictOldestLocked()
	}

	s.mem[key] = &checkpointEntry{
		value:     value,
		createdAt: time.Now(),
	}
	return nil
}

// Get 获取 checkpoint
func (s *InMemoryCheckpointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.mem[key]
	if !ok {
		return nil, false, nil
	}
	return entry.value, true, nil
}

// evictOldestLocked 清理最旧的条目（调用时已持有锁）
func (s *InMemoryCheckpointStore) evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range s.mem {
		if first || entry.createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.createdAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(s.mem, oldestKey)
	}
}
