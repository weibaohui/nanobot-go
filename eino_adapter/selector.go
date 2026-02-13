package eino_adapter

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// ModeSelector determines whether to use plan-execute mode based on task complexity
type ModeSelector struct {
	// ComplexKeywords indicate complex tasks that need planning
	ComplexKeywords []string
	// ActionKeywords are action verbs that may indicate multi-step tasks
	ActionKeywords []string
	// MinInputLength threshold for input length to trigger plan mode
	MinInputLength int
	// MinActionCount minimum action keywords to trigger plan mode
	MinActionCount int
}

// NewModeSelector creates a new ModeSelector with default settings
func NewModeSelector() *ModeSelector {
	return &ModeSelector{
		ComplexKeywords: []string{
			"规划", "帮我", "完成以下任务", "帮我完成", "请帮我",
			"计划", "安排", "组织", "设计", "实现",
			"plan", "help me", "schedule", "organize", "design", "implement",
			"step by step", "一步步", "分步骤",
		},
		ActionKeywords: []string{
			"查询", "搜索", "预订", "推荐", "创建", "删除", "修改",
			"分析", "总结", "比较", "计算", "生成", "编写", "读取",
			"query", "search", "book", "recommend", "create", "delete", "modify",
			"analyze", "summarize", "compare", "calculate", "generate", "write", "read",
		},
		MinInputLength: 50,
		MinActionCount: 2,
	}
}

// ShouldUsePlanMode determines if the input should use plan-execute mode
func (s *ModeSelector) ShouldUsePlanMode(input string) bool {
	input = strings.ToLower(input)

	// Check for complex task keywords
	for _, keyword := range s.ComplexKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			return true
		}
	}

	// Count action keywords
	actionCount := 0
	for _, keyword := range s.ActionKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			actionCount++
		}
	}
	if actionCount >= s.MinActionCount {
		return true
	}

	// Check input length (only for non-trivial inputs)
	if utf8.RuneCountInString(input) > s.MinInputLength {
		// For longer inputs, check if they contain multiple sentences or clauses
		// which might indicate a complex task
		sentenceCount := countSentences(input)
		if sentenceCount >= 2 && actionCount >= 1 {
			return true
		}
	}

	return false
}

// countSentences counts the number of sentences in the input
func countSentences(input string) int {
	// Match sentence-ending punctuation followed by space or end of string
	re := regexp.MustCompile(`[。！？.!?]\s*|$`)
	matches := re.FindAllStringIndex(input, -1)
	if len(matches) <= 1 {
		return 1
	}
	return len(matches) - 1
}

// ShouldUsePlanModeWithThreshold allows customizing the decision threshold
func (s *ModeSelector) ShouldUsePlanModeWithThreshold(input string, actionThreshold, lengthThreshold int) bool {
	input = strings.ToLower(input)

	// Check for complex task keywords
	for _, keyword := range s.ComplexKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			return true
		}
	}

	// Count action keywords
	actionCount := 0
	for _, keyword := range s.ActionKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			actionCount++
		}
	}
	if actionCount >= actionThreshold {
		return true
	}

	// Check input length
	if utf8.RuneCountInString(input) > lengthThreshold && actionCount >= 1 {
		return true
	}

	return false
}
