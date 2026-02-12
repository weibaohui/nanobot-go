package cron

import (
	"time"
)

// Schedule 调度定义
type Schedule struct {
	Kind    string `json:"kind"`    // "at", "every", "cron"
	AtMs    int    `json:"atMs"`    // 对于 "at": 时间戳（毫秒）
	EveryMs int    `json:"everyMs"` // 对于 "every": 间隔（毫秒）
	Expr    string `json:"expr"`    // 对于 "cron": cron 表达式
	Tz      string `json:"tz"`      // 时区
}

// Payload 任务负载
type Payload struct {
	Kind    string `json:"kind"`    // "system_event", "agent_turn"
	Message string `json:"message"` // 消息内容
	Deliver bool   `json:"deliver"` // 是否投递响应
	Channel string `json:"channel"` // 渠道
	To      string `json:"to"`      // 目标
}

// State 任务状态
type State struct {
	NextRunAtMs int    `json:"nextRunAtMs"` // 下次运行时间（毫秒）
	LastRunAtMs int    `json:"lastRunAtMs"` // 上次运行时间（毫秒）
	LastStatus  string `json:"lastStatus"`  // 上次状态: "ok", "error", "skipped"
	LastError   string `json:"lastError"`   // 上次错误
}

// Job 定时任务
type Job struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Enabled         bool      `json:"enabled"`
	Schedule        Schedule  `json:"schedule"`
	Payload         Payload   `json:"payload"`
	State           State     `json:"state"`
	CreatedAtMs     int       `json:"createdAtMs"`
	UpdatedAtMs     int       `json:"updatedAtMs"`
	DeleteAfterRun  bool      `json:"deleteAfterRun"`
}

// Store 任务存储
type Store struct {
	Version int    `json:"version"`
	Jobs    []*Job `json:"jobs"`
}

// nowMs 获取当前时间戳（毫秒）
func nowMs() int {
	return int(time.Now().UnixMilli())
}

// computeNextRun 计算下次运行时间
func computeNextRun(schedule *Schedule, nowMs int) int {
	switch schedule.Kind {
	case "at":
		if schedule.AtMs > nowMs {
			return schedule.AtMs
		}
		return 0

	case "every":
		if schedule.EveryMs <= 0 {
			return 0
		}
		return nowMs + schedule.EveryMs

	case "cron":
		// 简化实现：暂不支持 cron 表达式
		// 实际应使用 robfig/cron 库
		return 0
	}

	return 0
}
