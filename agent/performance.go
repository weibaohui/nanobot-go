package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	Logger *zap.Logger

	// 并发控制
	MaxConcurrentTasks int
	MaxConcurrentAgent int

	// 缓存配置
	CacheEnabled bool
	CacheTTL     time.Duration
	MaxCacheSize int

	// 超时配置
	DefaultTimeout time.Duration
	AgentTimeout   time.Duration
	ToolTimeout    time.Duration
}

// DefaultPerformanceConfig 默认性能配置
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		MaxConcurrentTasks: 10,
		MaxConcurrentAgent: 5,
		CacheEnabled:       true,
		CacheTTL:           5 * time.Minute,
		MaxCacheSize:       1000,
		DefaultTimeout:     30 * time.Second,
		AgentTimeout:       5 * time.Minute,
		ToolTimeout:        1 * time.Minute,
	}
}

// PerformanceManager 性能管理器
type PerformanceManager struct {
	config *PerformanceConfig
	logger *zap.Logger

	// 并发控制
	taskSemaphore chan struct{}
	agentSemaphore chan struct{}

	// 缓存
	cache     map[string]*CacheEntry
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration

	// 指标收集
	metrics     *PerformanceMetrics
	metricsMu   sync.RWMutex
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
	HitCount   int
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// 任务统计
	TotalTasks       int64
	SuccessfulTasks  int64
	FailedTasks      int64
	AverageTaskTime  time.Duration

	// Agent 统计
	AgentCalls       map[string]int64
	AgentAvgTime     map[string]time.Duration
	AgentErrors      map[string]int64

	// 路由统计
	RouteDecisions   int64
	RouteAvgTime     time.Duration
	RouteByAgent     map[string]int64

	// 缓存统计
	CacheHits        int64
	CacheMisses      int64
	CacheSize        int

	// 中断统计
	TotalInterrupts  int64
	ResolvedInterrupts int64
	AverageWaitTime  time.Duration

	// 时间窗口统计
	LastHourTasks    int64
	LastHourErrors   int64
}

// NewPerformanceManager 创建性能管理器
func NewPerformanceManager(config *PerformanceConfig) *PerformanceManager {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	pm := &PerformanceManager{
		config:        config,
		logger:        logger,
		taskSemaphore: make(chan struct{}, config.MaxConcurrentTasks),
		agentSemaphore: make(chan struct{}, config.MaxConcurrentAgent),
		cache:         make(map[string]*CacheEntry),
		cacheTTL:      config.CacheTTL,
		metrics: &PerformanceMetrics{
			AgentCalls:   make(map[string]int64),
			AgentAvgTime: make(map[string]time.Duration),
			AgentErrors:  make(map[string]int64),
			RouteByAgent: make(map[string]int64),
		},
	}

	// 初始化信号量
	for i := 0; i < config.MaxConcurrentTasks; i++ {
		pm.taskSemaphore <- struct{}{}
	}
	for i := 0; i < config.MaxConcurrentAgent; i++ {
		pm.agentSemaphore <- struct{}{}
	}

	return pm
}

// AcquireTaskSlot 获取任务槽位
func (pm *PerformanceManager) AcquireTaskSlot(ctx context.Context) error {
	select {
	case <-pm.taskSemaphore:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReleaseTaskSlot 释放任务槽位
func (pm *PerformanceManager) ReleaseTaskSlot() {
	pm.taskSemaphore <- struct{}{}
}

// AcquireAgentSlot 获取 Agent 槽位
func (pm *PerformanceManager) AcquireAgentSlot(ctx context.Context) error {
	select {
	case <-pm.agentSemaphore:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReleaseAgentSlot 释放 Agent 槽位
func (pm *PerformanceManager) ReleaseAgentSlot() {
	pm.agentSemaphore <- struct{}{}
}

// WithTaskConcurrency 使用并发控制执行任务
func (pm *PerformanceManager) WithTaskConcurrency(ctx context.Context, fn func() error) error {
	if err := pm.AcquireTaskSlot(ctx); err != nil {
		return err
	}
	defer pm.ReleaseTaskSlot()

	return fn()
}

// WithAgentConcurrency 使用并发控制执行 Agent
func (pm *PerformanceManager) WithAgentConcurrency(ctx context.Context, agentName string, fn func() error) error {
	if err := pm.AcquireAgentSlot(ctx); err != nil {
		return err
	}
	defer pm.ReleaseAgentSlot()

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// 记录指标
	pm.recordAgentCall(agentName, duration, err)

	return err
}

// 缓存相关方法

// GetFromCache 从缓存获取
func (pm *PerformanceManager) GetFromCache(key string) (interface{}, bool) {
	if !pm.config.CacheEnabled {
		return nil, false
	}

	pm.cacheMu.RLock()
	defer pm.cacheMu.RUnlock()

	entry, ok := pm.cache[key]
	if !ok {
		pm.recordCacheMiss()
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(entry.Expiration) {
		pm.recordCacheMiss()
		return nil, false
	}

	// 更新命中计数
	entry.HitCount++
	pm.recordCacheHit()

	return entry.Value, true
}

// SetToCache 设置缓存
func (pm *PerformanceManager) SetToCache(key string, value interface{}) {
	if !pm.config.CacheEnabled {
		return
	}

	pm.cacheMu.Lock()
	defer pm.cacheMu.Unlock()

	// 检查缓存大小
	if len(pm.cache) >= pm.config.MaxCacheSize {
		pm.cleanExpiredCacheLocked()
		if len(pm.cache) >= pm.config.MaxCacheSize {
			// 删除最旧的条目
			pm.evictOldestLocked()
		}
	}

	pm.cache[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(pm.cacheTTL),
		HitCount:   0,
	}
}

// cleanExpiredCacheLocked 清理过期缓存（调用时已持有锁）
func (pm *PerformanceManager) cleanExpiredCacheLocked() {
	now := time.Now()
	for key, entry := range pm.cache {
		if now.After(entry.Expiration) {
			delete(pm.cache, key)
		}
	}
}

// evictOldestLocked 驱逐最旧的条目（调用时已持有锁）
func (pm *PerformanceManager) evictOldestLocked() {
	// 简单实现：删除一个条目
	for key := range pm.cache {
		delete(pm.cache, key)
		break
	}
}

// ClearCache 清空缓存
func (pm *PerformanceManager) ClearCache() {
	pm.cacheMu.Lock()
	defer pm.cacheMu.Unlock()
	pm.cache = make(map[string]*CacheEntry)
}

// GetCacheStats 获取缓存统计
func (pm *PerformanceManager) GetCacheStats() map[string]interface{} {
	pm.cacheMu.RLock()
	defer pm.cacheMu.RUnlock()

	var totalHits int
	for _, entry := range pm.cache {
		totalHits += entry.HitCount
	}

	return map[string]interface{}{
		"size":      len(pm.cache),
		"max_size":  pm.config.MaxCacheSize,
		"total_hits": totalHits,
		"enabled":    pm.config.CacheEnabled,
	}
}

// 指标记录方法

// recordAgentCall 记录 Agent 调用
func (pm *PerformanceManager) recordAgentCall(agentName string, duration time.Duration, err error) {
	pm.metricsMu.Lock()
	defer pm.metricsMu.Unlock()

	pm.metrics.AgentCalls[agentName]++

	// 计算平均时间
	totalCalls := pm.metrics.AgentCalls[agentName]
	oldAvg := pm.metrics.AgentAvgTime[agentName]
	pm.metrics.AgentAvgTime[agentName] = time.Duration(
		(int64(oldAvg)*(totalCalls-1) + int64(duration)) / totalCalls,
	)

	if err != nil {
		pm.metrics.AgentErrors[agentName]++
	}
}

// recordRouteDecision 记录路由决策
func (pm *PerformanceManager) recordRouteDecision(agentType string, duration time.Duration) {
	pm.metricsMu.Lock()
	defer pm.metricsMu.Unlock()

	pm.metrics.RouteDecisions++
	pm.metrics.RouteByAgent[agentType]++

	// 计算平均时间
	total := pm.metrics.RouteDecisions
	oldAvg := pm.metrics.RouteAvgTime
	pm.metrics.RouteAvgTime = time.Duration(
		(int64(oldAvg)*(total-1) + int64(duration)) / total,
	)
}

// recordTaskComplete 记录任务完成
func (pm *PerformanceManager) recordTaskComplete(success bool, duration time.Duration) {
	pm.metricsMu.Lock()
	defer pm.metricsMu.Unlock()

	pm.metrics.TotalTasks++
	if success {
		pm.metrics.SuccessfulTasks++
	} else {
		pm.metrics.FailedTasks++
	}

	// 计算平均时间
	total := pm.metrics.TotalTasks
	oldAvg := pm.metrics.AverageTaskTime
	pm.metrics.AverageTaskTime = time.Duration(
		(int64(oldAvg)*(total-1) + int64(duration)) / total,
	)
}

// recordCacheHit 记录缓存命中
func (pm *PerformanceManager) recordCacheHit() {
	pm.metricsMu.Lock()
	defer pm.metricsMu.Unlock()
	pm.metrics.CacheHits++
}

// recordCacheMiss 记录缓存未命中
func (pm *PerformanceManager) recordCacheMiss() {
	pm.metricsMu.Lock()
	defer pm.metricsMu.Unlock()
	pm.metrics.CacheMisses++
}

// recordInterrupt 记录中断
func (pm *PerformanceManager) recordInterrupt(resolved bool, waitTime time.Duration) {
	pm.metricsMu.Lock()
	defer pm.metricsMu.Unlock()

	pm.metrics.TotalInterrupts++
	if resolved {
		pm.metrics.ResolvedInterrupts++
	}

	// 计算平均等待时间
	total := pm.metrics.TotalInterrupts
	oldAvg := pm.metrics.AverageWaitTime
	pm.metrics.AverageWaitTime = time.Duration(
		(int64(oldAvg)*(total-1) + int64(waitTime)) / total,
	)
}

// GetMetrics 获取性能指标
func (pm *PerformanceManager) GetMetrics() *PerformanceMetrics {
	pm.metricsMu.RLock()
	defer pm.metricsMu.RUnlock()

	// 复制指标
	metrics := &PerformanceMetrics{
		TotalTasks:        pm.metrics.TotalTasks,
		SuccessfulTasks:   pm.metrics.SuccessfulTasks,
		FailedTasks:       pm.metrics.FailedTasks,
		AverageTaskTime:   pm.metrics.AverageTaskTime,
		RouteDecisions:    pm.metrics.RouteDecisions,
		RouteAvgTime:      pm.metrics.RouteAvgTime,
		CacheHits:         pm.metrics.CacheHits,
		CacheMisses:       pm.metrics.CacheMisses,
		CacheSize:         len(pm.cache),
		TotalInterrupts:   pm.metrics.TotalInterrupts,
		ResolvedInterrupts: pm.metrics.ResolvedInterrupts,
		AverageWaitTime:   pm.metrics.AverageWaitTime,
		AgentCalls:        make(map[string]int64),
		AgentAvgTime:      make(map[string]time.Duration),
		AgentErrors:       make(map[string]int64),
		RouteByAgent:      make(map[string]int64),
	}

	// 复制 map
	for k, v := range pm.metrics.AgentCalls {
		metrics.AgentCalls[k] = v
	}
	for k, v := range pm.metrics.AgentAvgTime {
		metrics.AgentAvgTime[k] = v
	}
	for k, v := range pm.metrics.AgentErrors {
		metrics.AgentErrors[k] = v
	}
	for k, v := range pm.metrics.RouteByAgent {
		metrics.RouteByAgent[k] = v
	}

	return metrics
}

// GetSummary 获取性能摘要
func (pm *PerformanceManager) GetSummary() map[string]interface{} {
	metrics := pm.GetMetrics()

	successRate := float64(0)
	if metrics.TotalTasks > 0 {
		successRate = float64(metrics.SuccessfulTasks) / float64(metrics.TotalTasks) * 100
	}

	cacheHitRate := float64(0)
	totalCacheOps := metrics.CacheHits + metrics.CacheMisses
	if totalCacheOps > 0 {
		cacheHitRate = float64(metrics.CacheHits) / float64(totalCacheOps) * 100
	}

	interruptResolveRate := float64(0)
	if metrics.TotalInterrupts > 0 {
		interruptResolveRate = float64(metrics.ResolvedInterrupts) / float64(metrics.TotalInterrupts) * 100
	}

	return map[string]interface{}{
		"tasks": map[string]interface{}{
			"total":      metrics.TotalTasks,
			"successful": metrics.SuccessfulTasks,
			"failed":     metrics.FailedTasks,
			"success_rate": fmt.Sprintf("%.2f%%", successRate),
			"avg_time":   metrics.AverageTaskTime.String(),
		},
		"routing": map[string]interface{}{
			"decisions":    metrics.RouteDecisions,
			"avg_time":     metrics.RouteAvgTime.String(),
			"by_agent":     metrics.RouteByAgent,
		},
		"cache": map[string]interface{}{
			"hits":        metrics.CacheHits,
			"misses":      metrics.CacheMisses,
			"hit_rate":    fmt.Sprintf("%.2f%%", cacheHitRate),
			"size":        metrics.CacheSize,
		},
		"interrupts": map[string]interface{}{
			"total":         metrics.TotalInterrupts,
			"resolved":      metrics.ResolvedInterrupts,
			"resolve_rate":  fmt.Sprintf("%.2f%%", interruptResolveRate),
			"avg_wait_time": metrics.AverageWaitTime.String(),
		},
		"agents": map[string]interface{}{
			"calls":    metrics.AgentCalls,
			"errors":   metrics.AgentErrors,
			"avg_time": metrics.AgentAvgTime,
		},
		"concurrency": map[string]interface{}{
			"available_task_slots":  len(pm.taskSemaphore),
			"available_agent_slots": len(pm.agentSemaphore),
		},
	}
}

// ResetMetrics 重置指标
func (pm *PerformanceManager) ResetMetrics() {
	pm.metricsMu.Lock()
	defer pm.metricsMu.Unlock()

	pm.metrics = &PerformanceMetrics{
		AgentCalls:   make(map[string]int64),
		AgentAvgTime: make(map[string]time.Duration),
		AgentErrors:  make(map[string]int64),
		RouteByAgent: make(map[string]int64),
	}
}
