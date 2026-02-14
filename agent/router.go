package agent

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/weibaohui/nanobot-go/eino_adapter"
	"go.uber.org/zap"
)

// AgentType 定义 Agent 类型
type AgentType string

const (
	AgentTypeReAct AgentType = "react"
	AgentTypePlan  AgentType = "plan"
	AgentTypeChat  AgentType = "chat"
)

// Router 路由决策器
// 根据用户输入内容决定使用哪种类型的 Agent
type Router struct {
	modeSelector *eino_adapter.ModeSelector
	logger       *zap.Logger

	// 工具调用关键词
	toolKeywords []string
}

// RouterConfig 路由配置
type RouterConfig struct {
	Logger *zap.Logger
}

// NewRouter 创建路由决策器
func NewRouter(cfg *RouterConfig) *Router {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Router{
		modeSelector: eino_adapter.NewModeSelector(),
		logger:       logger,
		toolKeywords: []string{
			// 文件操作
			"读取", "写入", "编辑", "创建", "删除", "文件", "目录", "文件夹",
			"read", "write", "edit", "create", "delete", "file", "directory", "folder",
			// 网络操作
			"搜索", "获取", "下载", "网页", "网站", "网络",
			"search", "fetch", "download", "web", "url", "http",
			// 系统操作
			"执行", "运行", "命令", "脚本", "shell", "bash",
			"execute", "run", "command", "script",
			// 消息操作
			"发送消息", "通知", "推送",
			"send message", "notify", "push",
			// 技能调用
			"使用技能", "调用技能",
			"use skill", "call skill",
		},
	}
}

// Route 决定使用哪种类型的 Agent
// 返回 AgentType 表示推荐的 Agent 类型
func (r *Router) Route(ctx context.Context, input string) AgentType {
	inputLower := strings.ToLower(input)

	// 1. 首先检查是否需要 Plan 模式（复杂任务）
	if r.modeSelector.ShouldUsePlanMode(input) {
		r.logger.Debug("路由决策: Plan Agent",
			zap.String("原因", "检测到复杂任务关键词"),
			zap.String("输入预览", truncate(input, 50)),
		)
		return AgentTypePlan
	}

	// 2. 检查是否需要工具调用
	if r.needsToolCall(inputLower) {
		r.logger.Debug("路由决策: ReAct Agent",
			zap.String("原因", "检测到工具调用需求"),
			zap.String("输入预览", truncate(input, 50)),
		)
		return AgentTypeReAct
	}

	// 3. 默认使用 Chat Agent
	r.logger.Debug("路由决策: Chat Agent",
		zap.String("原因", "简单对话或问答"),
		zap.String("输入预览", truncate(input, 50)),
	)
	return AgentTypeChat
}

// needsToolCall 检查输入是否需要工具调用
func (r *Router) needsToolCall(input string) bool {
	for _, keyword := range r.toolKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// RouteWithConfidence 带置信度的路由决策
// 返回 AgentType 和置信度 (0-1)
func (r *Router) RouteWithConfidence(ctx context.Context, input string) (AgentType, float64) {
	inputLower := strings.ToLower(input)

	// 计算各类型的得分
	planScore := r.calculatePlanScore(inputLower)
	reactScore := r.calculateReActScore(inputLower)
	chatScore := r.calculateChatScore(inputLower)

	r.logger.Debug("路由得分",
		zap.Float64("plan", planScore),
		zap.Float64("react", reactScore),
		zap.Float64("chat", chatScore),
	)

	// 选择得分最高的类型
	maxScore := planScore
	selectedType := AgentTypePlan

	if reactScore > maxScore {
		maxScore = reactScore
		selectedType = AgentTypeReAct
	}
	if chatScore > maxScore {
		maxScore = chatScore
		selectedType = AgentTypeChat
	}

	// 计算置信度（归一化）
	totalScore := planScore + reactScore + chatScore
	confidence := maxScore / totalScore

	return selectedType, confidence
}

// calculatePlanScore 计算 Plan 模式得分
func (r *Router) calculatePlanScore(input string) float64 {
	score := 0.0

	// 复杂任务关键词
	complexKeywords := []string{
		"规划", "帮我", "完成以下任务", "帮我完成", "请帮我",
		"计划", "安排", "组织", "设计", "实现",
		"一步步", "分步骤", "逐步",
		"plan", "help me", "schedule", "organize", "design", "implement",
		"step by step",
	}

	for _, keyword := range complexKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			score += 0.3
		}
	}

	// 多句子可能表示复杂任务
	sentenceCount := countSentences(input)
	if sentenceCount >= 3 {
		score += 0.2
	} else if sentenceCount >= 2 {
		score += 0.1
	}

	// 输入长度
	if utf8.RuneCountInString(input) > 100 {
		score += 0.2
	} else if utf8.RuneCountInString(input) > 50 {
		score += 0.1
	}

	// 限制最大得分
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateReActScore 计算 ReAct 模式得分
func (r *Router) calculateReActScore(input string) float64 {
	score := 0.0

	// 工具调用关键词计数
	toolKeywordCount := 0
	for _, keyword := range r.toolKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			toolKeywordCount++
		}
	}

	// 每个关键词增加得分
	score += float64(toolKeywordCount) * 0.2

	// 限制最大得分
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateChatScore 计算 Chat 模式得分
func (r *Router) calculateChatScore(input string) float64 {
	score := 0.5 // 基础得分

	// 简单问候
	greetings := []string{
		"你好", "您好", "嗨", "哈喽", "hello", "hi", "hey",
	}
	for _, greeting := range greetings {
		if strings.Contains(input, greeting) {
			score += 0.3
			break
		}
	}

	// 问句模式
	questionPatterns := []string{
		"？", "?", "吗", "呢", "what", "how", "why", "when", "where", "who",
	}
	for _, pattern := range questionPatterns {
		if strings.Contains(input, pattern) {
			score += 0.1
			break
		}
	}

	// 短输入倾向于 Chat
	if utf8.RuneCountInString(input) < 20 {
		score += 0.2
	}

	// 限制最大得分
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// countSentences 计算句子数量
func countSentences(input string) int {
	re := regexp.MustCompile(`[。！？.!?]\s*|$`)
	matches := re.FindAllStringIndex(input, -1)
	if len(matches) <= 1 {
		return 1
	}
	return len(matches) - 1
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
