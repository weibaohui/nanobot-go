package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
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

// IntentType 意图类型
type IntentType string

const (
	IntentFileOperation    IntentType = "file_operation"
	IntentWebSearch        IntentType = "web_search"
	IntentCodeExecution    IntentType = "code_execution"
	IntentProjectPlanning  IntentType = "project_planning"
	IntentTaskDelegation   IntentType = "task_delegation"
	IntentSimpleQuestion   IntentType = "simple_question"
	IntentCasualChat       IntentType = "casual_chat"
	IntentInformationQuery IntentType = "information_query"
	IntentUnknown          IntentType = "unknown"
)

// RouteDecision 路由决策结果
type RouteDecision struct {
	AgentType    AgentType
	IntentType   IntentType
	Confidence   float64
	Reason       string
	MatchedRules []string
	Timestamp    time.Time
}

// Router 路由决策器
// 根据用户输入内容决定使用哪种类型的 Agent
type Router struct {
	modeSelector *eino_adapter.ModeSelector
	logger       *zap.Logger

	// 关键词配置
	toolKeywords    []string
	planKeywords    []string
	chatKeywords    []string
	intentKeywords  map[IntentType][]string

	// 规则引擎
	rules []RoutingRule

	// 决策历史（用于学习和分析）
	decisionHistory []*RouteDecision
	historyMutex    sync.RWMutex
	maxHistorySize  int

	// 缓存
	decisionCache map[string]*RouteDecision
	cacheMutex    sync.RWMutex
	cacheTTL      time.Duration
}

// RoutingRule 路由规则
type RoutingRule struct {
	Name        string
	Pattern     string
	regex       *regexp.Regexp
	AgentType   AgentType
	IntentType  IntentType
	Priority    int
	Description string
}

// RouterConfig 路由配置
type RouterConfig struct {
	Logger         *zap.Logger
	MaxHistorySize int
	CacheTTL       time.Duration
}

// NewRouter 创建路由决策器
func NewRouter(cfg *RouterConfig) *Router {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	maxHistory := cfg.MaxHistorySize
	if maxHistory <= 0 {
		maxHistory = 1000
	}

	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}

	router := &Router{
		modeSelector:   eino_adapter.NewModeSelector(),
		logger:         logger,
		maxHistorySize: maxHistory,
		decisionCache:  make(map[string]*RouteDecision),
		cacheTTL:       cacheTTL,
	}

	// 初始化关键词
	router.initKeywords()

	// 初始化规则
	router.initRules()

	return router
}

// initKeywords 初始化关键词配置
func (r *Router) initKeywords() {
	r.toolKeywords = []string{
		// 文件操作
		"读取", "写入", "编辑", "创建", "删除", "文件", "目录", "文件夹", "保存", "修改",
		"read", "write", "edit", "create", "delete", "file", "directory", "folder", "save", "modify",
		// 网络操作
		"搜索", "获取", "下载", "网页", "网站", "网络", "抓取", "爬取",
		"search", "fetch", "download", "web", "url", "http", "crawl", "scrape",
		// 系统操作
		"执行", "运行", "命令", "脚本", "shell", "bash", "终端",
		"execute", "run", "command", "script", "terminal",
		// 消息操作
		"发送消息", "通知", "推送", "广播",
		"send message", "notify", "push", "broadcast",
		// 技能调用
		"使用技能", "调用技能", "技能",
		"use skill", "call skill", "skill",
	}

	r.planKeywords = []string{
		"规划", "帮我", "完成以下任务", "帮我完成", "请帮我",
		"计划", "安排", "组织", "设计", "实现", "部署",
		"一步步", "分步骤", "逐步", "按步骤",
		"项目", "流程", "方案",
		"plan", "help me", "schedule", "organize", "design", "implement", "deploy",
		"step by step", "project", "workflow", "solution",
	}

	r.chatKeywords = []string{
		"你好", "您好", "嗨", "哈喽", "早上好", "下午好", "晚上好",
		"hello", "hi", "hey", "good morning", "good afternoon", "good evening",
		"怎么样", "如何", "什么是", "为什么", "谁", "哪里",
		"how", "what", "why", "who", "where",
	}

	// 意图关键词映射
	r.intentKeywords = map[IntentType][]string{
		IntentFileOperation: {
			"读取", "写入", "编辑", "创建", "删除", "文件", "目录",
			"read", "write", "edit", "create", "delete", "file", "directory",
		},
		IntentWebSearch: {
			"搜索", "查找", "查询", "网页", "网站",
			"search", "find", "query", "web", "website",
		},
		IntentCodeExecution: {
			"执行", "运行", "命令", "脚本", "shell", "bash",
			"execute", "run", "command", "script",
		},
		IntentProjectPlanning: {
			"规划", "计划", "项目", "流程", "方案", "设计",
			"plan", "project", "workflow", "solution", "design",
		},
		IntentTaskDelegation: {
			"帮我", "请帮我", "帮我完成", "完成以下任务",
			"help me", "do this", "complete this task",
		},
		IntentSimpleQuestion: {
			"什么是", "如何", "为什么", "怎么样", "吗", "呢",
			"what is", "how to", "why", "how about",
		},
		IntentCasualChat: {
			"你好", "嗨", "哈喽", "早上好", "下午好", "晚上好",
			"hello", "hi", "hey", "good morning", "good afternoon",
		},
		IntentInformationQuery: {
			"查询", "查找", "获取", "检索",
			"query", "find", "get", "retrieve",
		},
	}
}

// initRules 初始化路由规则
func (r *Router) initRules() {
	r.rules = []RoutingRule{
		// 高优先级规则：明确的文件操作
		{
			Name:        "file_operation",
			Pattern:     `(读取|写入|编辑|创建|删除|保存).{0,10}(文件|目录|文件夹)`,
			AgentType:   AgentTypeReAct,
			IntentType:  IntentFileOperation,
			Priority:    100,
			Description: "文件操作任务",
		},
		// 高优先级规则：明确的代码执行
		{
			Name:        "code_execution",
			Pattern:     `(执行|运行).{0,10}(命令|脚本|shell|bash)`,
			AgentType:   AgentTypeReAct,
			IntentType:  IntentCodeExecution,
			Priority:    100,
			Description: "代码执行任务",
		},
		// 高优先级规则：项目规划
		{
			Name:        "project_planning",
			Pattern:     `(帮我|请帮我).{0,20}(规划|计划|设计|实现).{0,20}(项目|方案|流程)`,
			AgentType:   AgentTypePlan,
			IntentType:  IntentProjectPlanning,
			Priority:    100,
			Description: "项目规划任务",
		},
		// 中优先级规则：网络搜索
		{
			Name:        "web_search",
			Pattern:     `(搜索|查找|查询).{0,10}(网络|网页|网站)`,
			AgentType:   AgentTypeReAct,
			IntentType:  IntentWebSearch,
			Priority:    80,
			Description: "网络搜索任务",
		},
		// 中优先级规则：简单问候
		{
			Name:        "casual_chat",
			Pattern:     `^(你好|您好|嗨|哈喽|hello|hi|hey)[\s!！。.]*$`,
			AgentType:   AgentTypeChat,
			IntentType:  IntentCasualChat,
			Priority:    90,
			Description: "简单问候",
		},
		// 中优先级规则：任务委托
		{
			Name:        "task_delegation",
			Pattern:     `(帮我|请帮我|帮我完成).{0,50}(任务|工作|事情)`,
			AgentType:   AgentTypePlan,
			IntentType:  IntentTaskDelegation,
			Priority:    70,
			Description: "任务委托",
		},
	}

	// 编译正则表达式
	for i := range r.rules {
		r.rules[i].regex = regexp.MustCompile(r.rules[i].Pattern)
	}
}

// Route 决定使用哪种类型的 Agent
// 返回 AgentType 表示推荐的 Agent 类型
func (r *Router) Route(ctx context.Context, input string) AgentType {
	decision := r.RouteDetailed(ctx, input)
	return decision.AgentType
}

// RouteDetailed 返回详细的路由决策
func (r *Router) RouteDetailed(ctx context.Context, input string) *RouteDecision {
	// 检查缓存
	cacheKey := r.getCacheKey(input)
	if cached := r.getFromCache(cacheKey); cached != nil {
		r.logger.Debug("使用缓存的路由决策",
			zap.String("agent_type", string(cached.AgentType)),
			zap.Float64("confidence", cached.Confidence),
		)
		return cached
	}

	inputLower := strings.ToLower(input)
	var decision *RouteDecision

	// 1. 首先应用规则引擎
	decision = r.applyRules(input)

	// 2. 如果规则没有匹配，使用得分计算
	if decision == nil {
		decision = r.calculateScores(inputLower)
	}

	// 3. 记录决策历史
	r.recordDecision(decision)

	// 4. 缓存决策
	r.saveToCache(cacheKey, decision)

	r.logger.Info("路由决策完成",
		zap.String("agent_type", string(decision.AgentType)),
		zap.String("intent_type", string(decision.IntentType)),
		zap.Float64("confidence", decision.Confidence),
		zap.String("reason", decision.Reason),
	)

	return decision
}

// applyRules 应用路由规则
func (r *Router) applyRules(input string) *RouteDecision {
	// 按优先级排序规则
	sortedRules := make([]RoutingRule, len(r.rules))
	copy(sortedRules, r.rules)

	// 按优先级从高到低排序
	for i := 0; i < len(sortedRules); i++ {
		for j := i + 1; j < len(sortedRules); j++ {
			if sortedRules[j].Priority > sortedRules[i].Priority {
				sortedRules[i], sortedRules[j] = sortedRules[j], sortedRules[i]
			}
		}
	}

	// 匹配规则
	var matchedRules []string
	for _, rule := range sortedRules {
		if rule.regex != nil && rule.regex.MatchString(input) {
			matchedRules = append(matchedRules, rule.Name)

			r.logger.Debug("规则匹配",
				zap.String("rule", rule.Name),
				zap.String("pattern", rule.Pattern),
				zap.String("agent_type", string(rule.AgentType)),
			)

			return &RouteDecision{
				AgentType:    rule.AgentType,
				IntentType:   rule.IntentType,
				Confidence:   0.9,
				Reason:       rule.Description,
				MatchedRules: matchedRules,
				Timestamp:    time.Now(),
			}
		}
	}

	return nil
}

// calculateScores 计算得分并决策
func (r *Router) calculateScores(input string) *RouteDecision {
	// 计算各类型的得分
	planScore := r.calculatePlanScore(input)
	reactScore := r.calculateReActScore(input)
	chatScore := r.calculateChatScore(input)

	// 识别意图
	intent := r.identifyIntent(input)

	r.logger.Debug("路由得分",
		zap.Float64("plan", planScore),
		zap.Float64("react", reactScore),
		zap.Float64("chat", chatScore),
		zap.String("intent", string(intent)),
	)

	// 选择得分最高的类型
	maxScore := planScore
	selectedType := AgentTypePlan
	reason := "检测到复杂任务关键词"

	if reactScore > maxScore {
		maxScore = reactScore
		selectedType = AgentTypeReAct
		reason = "检测到工具调用需求"
	}
	if chatScore > maxScore {
		maxScore = chatScore
		selectedType = AgentTypeChat
		reason = "简单对话或问答"
	}

	// 计算置信度（归一化）
	totalScore := planScore + reactScore + chatScore
	confidence := maxScore / totalScore
	if totalScore == 0 {
		confidence = 0.5
	}

	return &RouteDecision{
		AgentType:  selectedType,
		IntentType: intent,
		Confidence: confidence,
		Reason:     reason,
		Timestamp:  time.Now(),
	}
}

// identifyIntent 识别意图
func (r *Router) identifyIntent(input string) IntentType {
	intentScores := make(map[IntentType]float64)

	// 计算每个意图的得分
	for intent, keywords := range r.intentKeywords {
		score := 0.0
		for _, keyword := range keywords {
			if strings.Contains(input, strings.ToLower(keyword)) {
				score += 1.0
			}
		}
		if score > 0 {
			intentScores[intent] = score
		}
	}

	// 选择得分最高的意图
	var maxIntent IntentType = IntentUnknown
	maxScore := 0.0

	for intent, score := range intentScores {
		if score > maxScore {
			maxScore = score
			maxIntent = intent
		}
	}

	return maxIntent
}

// calculatePlanScore 计算 Plan 模式得分
func (r *Router) calculatePlanScore(input string) float64 {
	score := 0.0

	// 复杂任务关键词匹配
	for _, keyword := range r.planKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			score += 0.25
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
	inputLen := utf8.RuneCountInString(input)
	if inputLen > 100 {
		score += 0.2
	} else if inputLen > 50 {
		score += 0.1
	}

	// 检测任务列表模式
	if r.hasTaskListPattern(input) {
		score += 0.3
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

	// 检测代码块模式
	if r.hasCodeBlockPattern(input) {
		score += 0.3
	}

	// 检测路径模式
	if r.hasPathPattern(input) {
		score += 0.2
	}

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
	for _, keyword := range r.chatKeywords {
		if strings.Contains(input, keyword) {
			score += 0.2
			break
		}
	}

	// 问句模式
	questionPatterns := []string{"？", "?", "吗", "呢"}
	for _, pattern := range questionPatterns {
		if strings.Contains(input, pattern) {
			score += 0.1
			break
		}
	}

	// 短输入倾向于 Chat
	inputLen := utf8.RuneCountInString(input)
	if inputLen < 20 {
		score += 0.2
	} else if inputLen < 50 {
		score += 0.1
	}

	// 如果包含工具关键词，降低 Chat 得分
	toolCount := 0
	for _, keyword := range r.toolKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			toolCount++
		}
	}
	if toolCount > 0 {
		score -= float64(toolCount) * 0.15
	}

	// 限制得分范围
	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// hasTaskListPattern 检测任务列表模式
func (r *Router) hasTaskListPattern(input string) bool {
	// 检测数字列表模式 (1. 2. 3. 或 一、二、三)
	patterns := []string{
		`[1-9][\.、]`,
		`[一二三四五六七八九十][、\.]`,
		`第[一二三四五六七八九十\d][个步骤项]`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, input)
		if matched {
			return true
		}
	}

	return false
}

// hasCodeBlockPattern 检测代码块模式
func (r *Router) hasCodeBlockPattern(input string) bool {
	// 检测代码块标记
	return strings.Contains(input, "```") ||
		strings.Contains(input, "`") ||
		regexp.MustCompile(`\b(func|def|class|import|package|var|let|const)\b`).MatchString(input)
}

// hasPathPattern 检测路径模式
func (r *Router) hasPathPattern(input string) bool {
	// 检测文件路径
	patterns := []string{
		`[a-zA-Z0-9_\-/]+\.[a-zA-Z]{1,5}`, // 文件名.扩展名
		`/[a-zA-Z0-9_\-/]+`,               // Unix 路径
		`[A-Z]:\\[a-zA-Z0-9_\-\\]+`,       // Windows 路径
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, input)
		if matched {
			return true
		}
	}

	return false
}

// 缓存相关方法

func (r *Router) getCacheKey(input string) string {
	// 使用输入的哈希作为缓存键
	if len(input) > 100 {
		input = input[:100]
	}
	return input
}

func (r *Router) getFromCache(key string) *RouteDecision {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	if decision, ok := r.decisionCache[key]; ok {
		// 检查是否过期
		if time.Since(decision.Timestamp) < r.cacheTTL {
			return decision
		}
	}
	return nil
}

func (r *Router) saveToCache(key string, decision *RouteDecision) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	// 清理过期缓存
	if len(r.decisionCache) > 1000 {
		r.cleanCache()
	}

	r.decisionCache[key] = decision
}

func (r *Router) cleanCache() {
	now := time.Now()
	for key, decision := range r.decisionCache {
		if now.Sub(decision.Timestamp) > r.cacheTTL {
			delete(r.decisionCache, key)
		}
	}
}

// 历史记录相关方法

func (r *Router) recordDecision(decision *RouteDecision) {
	r.historyMutex.Lock()
	defer r.historyMutex.Unlock()

	// 限制历史记录大小
	if len(r.decisionHistory) >= r.maxHistorySize {
		r.decisionHistory = r.decisionHistory[1:]
	}

	r.decisionHistory = append(r.decisionHistory, decision)
}

// GetDecisionHistory 获取决策历史
func (r *Router) GetDecisionHistory(limit int) []*RouteDecision {
	r.historyMutex.RLock()
	defer r.historyMutex.RUnlock()

	if limit <= 0 || limit > len(r.decisionHistory) {
		limit = len(r.decisionHistory)
	}

	// 返回最近的 N 条记录
	start := len(r.decisionHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*RouteDecision, limit)
	copy(result, r.decisionHistory[start:])

	return result
}

// GetDecisionStats 获取决策统计
func (r *Router) GetDecisionStats() map[string]interface{} {
	r.historyMutex.RLock()
	defer r.historyMutex.RUnlock()

	stats := map[string]interface{}{
		"total_decisions": len(r.decisionHistory),
		"agent_distribution": map[string]int{
			string(AgentTypeReAct): 0,
			string(AgentTypePlan):  0,
			string(AgentTypeChat):  0,
		},
		"intent_distribution": make(map[string]int),
		"avg_confidence":      0.0,
	}

	if len(r.decisionHistory) == 0 {
		return stats
	}

	var totalConfidence float64
	agentDist := stats["agent_distribution"].(map[string]int)
	intentDist := stats["intent_distribution"].(map[string]int)

	for _, decision := range r.decisionHistory {
		agentDist[string(decision.AgentType)]++
		intentDist[string(decision.IntentType)]++
		totalConfidence += decision.Confidence
	}

	stats["avg_confidence"] = totalConfidence / float64(len(r.decisionHistory))

	return stats
}

// AddRule 添加自定义路由规则
func (r *Router) AddRule(rule RoutingRule) error {
	// 编译正则表达式
	regex, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("编译规则正则表达式失败: %w", err)
	}

	rule.regex = regex
	r.rules = append(r.rules, rule)

	r.logger.Info("添加路由规则",
		zap.String("name", rule.Name),
		zap.String("pattern", rule.Pattern),
		zap.String("agent_type", string(rule.AgentType)),
	)

	return nil
}

// RouteWithConfidence 带置信度的路由决策
// 返回 AgentType 和置信度 (0-1)
func (r *Router) RouteWithConfidence(ctx context.Context, input string) (AgentType, float64) {
	decision := r.RouteDetailed(ctx, input)
	return decision.AgentType, decision.Confidence
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
