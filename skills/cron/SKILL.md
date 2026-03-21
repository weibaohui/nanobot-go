---
name: cron
description: 安排提醒与周期性任务。
---

# Cron

使用 `cron` 工具来安排提醒或周期性任务。

## 三种模式

1. **Reminder（提醒）** - 消息会直接发送给用户
2. **Task（任务）** - 消息作为任务描述，Agent 会执行并返回结果
3. **One-time（一次性）** - 在指定时间只运行一次，随后自动删除

## 示例

固定提醒：
```
cron(action="add", message="Time to take a break!", every_seconds=1200)
```

动态任务（每次触发时由 Agent 执行）：
```
cron(action="add", message="Check HKUDS/nanobot GitHub stars and report", every_seconds=600)
```

一次性定时任务（根据当前时间计算 ISO datetime）：
```
cron(action="add", message="Remind me about the meeting", at="<ISO datetime>")
```

查看/删除：
```
cron(action="list")
cron(action="remove", job_id="abc123")
```

## 时间表达式

| 用户表达 | 参数 |
|-----------|------------|
| 每 20 分钟 | every_seconds: 1200 |
| 每小时 | every_seconds: 3600 |
| 每天早上 8 点 | cron_expr: "0 8 * * *" |
| 工作日下午 5 点 | cron_expr: "0 17 * * 1-5" |
| 在指定时间 | at: ISO datetime 字符串（根据当前时间计算） |
