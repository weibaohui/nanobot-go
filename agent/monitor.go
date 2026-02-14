package agent

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Monitor 监控器
type Monitor struct {
	logger *zap.Logger

	// 指标收集
	metrics     *MonitorMetrics
	metricsMu   sync.RWMutex

	// 事件日志
	eventLog     []*MonitorEvent
	eventLogMu   sync.RWMutex
	maxEventLog  int

	// 告警规则
	alertRules   []*AlertRule
	alertChan    chan *Alert

	// 健康检查
	healthChecks map[string]HealthCheckFunc
	healthMu     sync.RWMutex
}

// MonitorMetrics 监控指标
type MonitorMetrics struct {
	// 请求统计
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	RequestRate        float64

	// 响应时间
	AvgResponseTime    time.Duration
	MinResponseTime    time.Duration
	MaxResponseTime    time.Duration
	P95ResponseTime    time.Duration
	P99ResponseTime    time.Duration

	// Agent 统计
	AgentInvocations   map[string]int64
	AgentSuccessRate   map[string]float64
	AgentAvgLatency    map[string]time.Duration

	// 错误统计
	ErrorsByType       map[string]int64
	LastErrorTime      time.Time

	// 资源使用
	ActiveSessions     int64
	PendingTasks       int64
	QueueLength        int64

	// 时间戳
	StartTime          time.Time
	LastUpdateTime     time.Time
}

// MonitorEvent 监控事件
type MonitorEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	AgentType   string                 `json:"agent_type,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// AlertRule 告警规则
type AlertRule struct {
	Name        string
	Condition   func(*MonitorMetrics) bool
	Severity    string
	Message     string
	Cooldown    time.Duration
	LastTrigger time.Time
}

// Alert 告警
type Alert struct {
	RuleName  string
	Severity  string
	Message   string
	Timestamp time.Time
	Metrics   *MonitorMetrics
}

// HealthCheckFunc 健康检查函数
type HealthCheckFunc func(ctx context.Context) error

// MonitorConfig 监控配置
type MonitorConfig struct {
	Logger       *zap.Logger
	MaxEventLog  int
	AlertChan    chan *Alert
}

// NewMonitor 创建监控器
func NewMonitor(config *MonitorConfig) *Monitor {
	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxEventLog := config.MaxEventLog
	if maxEventLog <= 0 {
		maxEventLog = 1000
	}

	return &Monitor{
		logger:       logger,
		metrics:      &MonitorMetrics{
			AgentInvocations: make(map[string]int64),
			AgentSuccessRate: make(map[string]float64),
			AgentAvgLatency:  make(map[string]time.Duration),
			ErrorsByType:     make(map[string]int64),
			StartTime:        time.Now(),
		},
		eventLog:      make([]*MonitorEvent, 0, maxEventLog),
		maxEventLog:   maxEventLog,
		alertRules:    make([]*AlertRule, 0),
		alertChan:     config.AlertChan,
		healthChecks:  make(map[string]HealthCheckFunc),
	}
}

// 事件记录方法

// RecordEvent 记录事件
func (m *Monitor) RecordEvent(eventType, level, message string, details map[string]interface{}) {
	event := &MonitorEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		Level:     level,
		Message:   message,
		Details:   details,
	}

	m.addEvent(event)

	// 根据级别记录日志
	switch level {
	case "error":
		m.logger.Error(message,
			zap.String("event_type", eventType),
			zap.Any("details", details),
		)
	case "warn":
		m.logger.Warn(message,
			zap.String("event_type", eventType),
			zap.Any("details", details),
		)
	case "info":
		m.logger.Info(message,
			zap.String("event_type", eventType),
			zap.Any("details", details),
		)
	default:
		m.logger.Debug(message,
			zap.String("event_type", eventType),
			zap.Any("details", details),
		)
	}
}

// RecordAgentInvocation 记录 Agent 调用
func (m *Monitor) RecordAgentInvocation(agentType string, duration time.Duration, err error) {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	m.metrics.AgentInvocations[agentType]++
	m.metrics.TotalRequests++

	if err != nil {
		m.metrics.FailedRequests++
		m.metrics.ErrorsByType[err.Error()]++
		m.metrics.LastErrorTime = time.Now()

		m.RecordEvent("agent_error", "error", "Agent 执行失败",
			map[string]interface{}{
				"agent_type": agentType,
				"duration":   duration.String(),
				"error":      err.Error(),
			})
	} else {
		m.metrics.SuccessfulRequests++

		m.RecordEvent("agent_success", "info", "Agent 执行成功",
			map[string]interface{}{
				"agent_type": agentType,
				"duration":   duration.String(),
			})
	}

	// 更新平均延迟
	total := m.metrics.AgentInvocations[agentType]
	oldAvg := m.metrics.AgentAvgLatency[agentType]
	m.metrics.AgentAvgLatency[agentType] = time.Duration(
		(int64(oldAvg)*(total-1) + int64(duration)) / total,
	)

	// 更新成功率
	successCount := m.metrics.AgentInvocations[agentType] - m.metrics.ErrorsByType[agentType]
	m.metrics.AgentSuccessRate[agentType] = float64(successCount) / float64(m.metrics.AgentInvocations[agentType])

	// 更新响应时间统计
	m.updateResponseTimeStats(duration)

	m.metrics.LastUpdateTime = time.Now()

	// 检查告警规则
	m.checkAlertRules()
}

// RecordRoutingDecision 记录路由决策
func (m *Monitor) RecordRoutingDecision(fromAgent, toAgent string, confidence float64) {
	m.RecordEvent("routing", "info", "路由决策",
		map[string]interface{}{
			"from_agent": fromAgent,
			"to_agent":   toAgent,
			"confidence": confidence,
		})
}

// RecordInterrupt 记录中断
func (m *Monitor) RecordInterrupt(interruptType string, resolved bool, waitTime time.Duration) {
	level := "info"
	message := "中断已解决"
	if !resolved {
		level = "warn"
		message = "中断等待中"
	}

	m.RecordEvent("interrupt", level, message,
		map[string]interface{}{
			"interrupt_type": interruptType,
			"resolved":       resolved,
			"wait_time":      waitTime.String(),
		})
}

// RecordCacheOperation 记录缓存操作
func (m *Monitor) RecordCacheOperation(operation string, hit bool) {
	m.RecordEvent("cache", "debug", "缓存操作",
		map[string]interface{}{
			"operation": operation,
			"hit":       hit,
		})
}

// addEvent 添加事件到日志
func (m *Monitor) addEvent(event *MonitorEvent) {
	m.eventLogMu.Lock()
	defer m.eventLogMu.Unlock()

	if len(m.eventLog) >= m.maxEventLog {
		m.eventLog = m.eventLog[1:]
	}
	m.eventLog = append(m.eventLog, event)
}

// updateResponseTimeStats 更新响应时间统计
func (m *Monitor) updateResponseTimeStats(duration time.Duration) {
	// 更新最小/最大值
	if m.metrics.MinResponseTime == 0 || duration < m.metrics.MinResponseTime {
		m.metrics.MinResponseTime = duration
	}
	if duration > m.metrics.MaxResponseTime {
		m.metrics.MaxResponseTime = duration
	}

	// 更新平均值
	total := m.metrics.TotalRequests
	oldAvg := m.metrics.AvgResponseTime
	m.metrics.AvgResponseTime = time.Duration(
		(int64(oldAvg)*(total-1) + int64(duration)) / total,
	)
}

// 指标获取方法

// GetMetrics 获取当前指标
func (m *Monitor) GetMetrics() *MonitorMetrics {
	m.metricsMu.RLock()
	defer m.metricsMu.RUnlock()

	// 复制指标
	metrics := &MonitorMetrics{
		TotalRequests:      m.metrics.TotalRequests,
		SuccessfulRequests: m.metrics.SuccessfulRequests,
		FailedRequests:     m.metrics.FailedRequests,
		RequestRate:        m.metrics.RequestRate,
		AvgResponseTime:    m.metrics.AvgResponseTime,
		MinResponseTime:    m.metrics.MinResponseTime,
		MaxResponseTime:    m.metrics.MaxResponseTime,
		P95ResponseTime:    m.metrics.P95ResponseTime,
		P99ResponseTime:    m.metrics.P99ResponseTime,
		ActiveSessions:     m.metrics.ActiveSessions,
		PendingTasks:       m.metrics.PendingTasks,
		QueueLength:        m.metrics.QueueLength,
		StartTime:          m.metrics.StartTime,
		LastUpdateTime:     m.metrics.LastUpdateTime,
		LastErrorTime:      m.metrics.LastErrorTime,
		AgentInvocations:   make(map[string]int64),
		AgentSuccessRate:   make(map[string]float64),
		AgentAvgLatency:    make(map[string]time.Duration),
		ErrorsByType:       make(map[string]int64),
	}

	for k, v := range m.metrics.AgentInvocations {
		metrics.AgentInvocations[k] = v
	}
	for k, v := range m.metrics.AgentSuccessRate {
		metrics.AgentSuccessRate[k] = v
	}
	for k, v := range m.metrics.AgentAvgLatency {
		metrics.AgentAvgLatency[k] = v
	}
	for k, v := range m.metrics.ErrorsByType {
		metrics.ErrorsByType[k] = v
	}

	return metrics
}

// GetEventLog 获取事件日志
func (m *Monitor) GetEventLog(limit int) []*MonitorEvent {
	m.eventLogMu.RLock()
	defer m.eventLogMu.RUnlock()

	if limit <= 0 || limit > len(m.eventLog) {
		limit = len(m.eventLog)
	}

	start := len(m.eventLog) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*MonitorEvent, limit)
	copy(result, m.eventLog[start:])
	return result
}

// GetEventLogJSON 获取 JSON 格式的事件日志
func (m *Monitor) GetEventLogJSON(limit int) string {
	events := m.GetEventLog(limit)
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// 告警相关方法

// AddAlertRule 添加告警规则
func (m *Monitor) AddAlertRule(rule *AlertRule) {
	m.alertRules = append(m.alertRules, rule)
	m.logger.Info("添加告警规则",
		zap.String("name", rule.Name),
		zap.String("severity", rule.Severity),
	)
}

// checkAlertRules 检查告警规则
func (m *Monitor) checkAlertRules() {
	for _, rule := range m.alertRules {
		// 检查冷却时间
		if time.Since(rule.LastTrigger) < rule.Cooldown {
			continue
		}

		// 检查条件
		if rule.Condition(m.metrics) {
			rule.LastTrigger = time.Now()

			alert := &Alert{
				RuleName:  rule.Name,
				Severity:  rule.Severity,
				Message:   rule.Message,
				Timestamp: time.Now(),
				Metrics:   m.GetMetrics(),
			}

			// 发送告警
			if m.alertChan != nil {
				select {
				case m.alertChan <- alert:
				default:
					m.logger.Warn("告警通道已满，丢弃告警",
						zap.String("rule", rule.Name),
					)
				}
			}

			// 记录告警事件
			m.RecordEvent("alert", rule.Severity, rule.Message,
				map[string]interface{}{
					"rule_name": rule.Name,
				})
		}
	}
}

// 健康检查方法

// RegisterHealthCheck 注册健康检查
func (m *Monitor) RegisterHealthCheck(name string, check HealthCheckFunc) {
	m.healthMu.Lock()
	defer m.healthMu.Unlock()
	m.healthChecks[name] = check
}

// RunHealthChecks 运行所有健康检查
func (m *Monitor) RunHealthChecks(ctx context.Context) map[string]error {
	m.healthMu.RLock()
	defer m.healthMu.RUnlock()

	results := make(map[string]error)
	for name, check := range m.healthChecks {
		err := check(ctx)
		results[name] = err

		if err != nil {
			m.RecordEvent("health_check", "error", "健康检查失败",
				map[string]interface{}{
					"check_name": name,
					"error":      err.Error(),
				})
		}
	}

	return results
}

// IsHealthy 检查是否健康
func (m *Monitor) IsHealthy(ctx context.Context) bool {
	results := m.RunHealthChecks(ctx)
	for _, err := range results {
		if err != nil {
			return false
		}
	}
	return true
}

// 统计报告方法

// GetSummary 获取监控摘要
func (m *Monitor) GetSummary() map[string]interface{} {
	metrics := m.GetMetrics()

	uptime := time.Since(metrics.StartTime)
	successRate := float64(0)
	if metrics.TotalRequests > 0 {
		successRate = float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests) * 100
	}

	return map[string]interface{}{
		"uptime":            uptime.String(),
		"total_requests":    metrics.TotalRequests,
		"success_rate":      successRate,
		"avg_response_time": metrics.AvgResponseTime.String(),
		"active_sessions":   metrics.ActiveSessions,
		"pending_tasks":     metrics.PendingTasks,
		"agent_stats": map[string]interface{}{
			"invocations": metrics.AgentInvocations,
			"success_rate": metrics.AgentSuccessRate,
			"avg_latency":  metrics.AgentAvgLatency,
		},
		"errors": map[string]interface{}{
			"total":      metrics.FailedRequests,
			"by_type":    metrics.ErrorsByType,
			"last_error": metrics.LastErrorTime,
		},
	}
}

// GetDetailedReport 获取详细报告
func (m *Monitor) GetDetailedReport() map[string]interface{} {
	summary := m.GetSummary()
	metrics := m.GetMetrics()

	summary["detailed_metrics"] = map[string]interface{}{
		"min_response_time": metrics.MinResponseTime.String(),
		"max_response_time": metrics.MaxResponseTime.String(),
		"p95_response_time": metrics.P95ResponseTime.String(),
		"p99_response_time": metrics.P99ResponseTime.String(),
		"start_time":        metrics.StartTime,
		"last_update":       metrics.LastUpdateTime,
	}

	summary["recent_events"] = m.GetEventLog(10)

	return summary
}

// Reset 重置监控数据
func (m *Monitor) Reset() {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	m.metrics = &MonitorMetrics{
		AgentInvocations: make(map[string]int64),
		AgentSuccessRate: make(map[string]float64),
		AgentAvgLatency:  make(map[string]time.Duration),
		ErrorsByType:     make(map[string]int64),
		StartTime:        time.Now(),
	}

	m.eventLogMu.Lock()
	m.eventLog = make([]*MonitorEvent, 0, m.maxEventLog)
	m.eventLogMu.Unlock()

	m.logger.Info("监控数据已重置")
}

// 预定义告警规则

// NewHighErrorRateAlertRule 创建高错误率告警规则
func NewHighErrorRateAlertRule(threshold float64, cooldown time.Duration) *AlertRule {
	return &AlertRule{
		Name: "high_error_rate",
		Condition: func(m *MonitorMetrics) bool {
			if m.TotalRequests == 0 {
				return false
			}
			errorRate := float64(m.FailedRequests) / float64(m.TotalRequests)
			return errorRate > threshold
		},
		Severity: "error",
		Message:  "错误率超过阈值",
		Cooldown: cooldown,
	}
}

// NewSlowResponseAlertRule 创建慢响应告警规则
func NewSlowResponseAlertRule(threshold time.Duration, cooldown time.Duration) *AlertRule {
	return &AlertRule{
		Name: "slow_response",
		Condition: func(m *MonitorMetrics) bool {
			return m.AvgResponseTime > threshold
		},
		Severity: "warn",
		Message:  "平均响应时间超过阈值",
		Cooldown: cooldown,
	}
}

// NewHighQueueLengthAlertRule 创建高队列长度告警规则
func NewHighQueueLengthAlertRule(threshold int64, cooldown time.Duration) *AlertRule {
	return &AlertRule{
		Name: "high_queue_length",
		Condition: func(m *MonitorMetrics) bool {
			return m.QueueLength > threshold
		},
		Severity: "warn",
		Message:  "队列长度超过阈值",
		Cooldown: cooldown,
	}
}
