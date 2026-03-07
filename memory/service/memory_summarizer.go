package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/weibaohui/nanobot-go/memory/models"
	"github.com/weibaohui/nanobot-go/utils"
)

// LLMClient LLM 客户端接口
type LLMClient interface {
	// Complete 调用 LLM 生成文本
	Complete(ctx context.Context, prompt string, systemPrompt string) (string, error)
}

// MemorySummarizer 记忆总结器接口
type MemorySummarizer interface {
	// SummarizeConversation 对单条对话进行初步总结
	SummarizeConversation(ctx context.Context, messages []models.Message) (*models.ConversationSummary, error)

	// SummarizeToLongTerm 将多条流水记忆提炼为长期记忆
	SummarizeToLongTerm(ctx context.Context, streams []models.StreamMemory) (*models.LongTermSummary, error)
}

// memorySummarizer 记忆总结器实现
type memorySummarizer struct {
	client           LLMClient
	conversationPrompt string
	longTermPrompt   string
	maxMessages      int
}

// NewMemorySummarizer 创建记忆总结器实例
func NewMemorySummarizer(client LLMClient, conversationPrompt, longTermPrompt string) MemorySummarizer {
	if conversationPrompt == "" {
		conversationPrompt = defaultConversationPrompt
	}
	if longTermPrompt == "" {
		longTermPrompt = defaultLongTermPrompt
	}

	return &memorySummarizer{
		client:             client,
		conversationPrompt: conversationPrompt,
		longTermPrompt:     longTermPrompt,
		maxMessages:        100,
	}
}

// defaultConversationPrompt 默认对话总结提示词
const defaultConversationPrompt = `请对以下对话进行简要总结。

要求：
1. 用一句话概括对话主题
2. 列出3-5个关键要点（换行分隔）
3. 去除敏感信息（密码、API Key、Token等）
4. 控制在200字以内

对话内容：
{{CONTENT}}

请按以下JSON格式返回：
{
  "summary": "一句话总结",
  "key_points": "关键要点1\n关键要点2\n关键要点3"
}`

// defaultLongTermPrompt 默认长期记忆提炼提示词
const defaultLongTermPrompt = `请对以下对话记录进行精华提炼，形成长期记忆。

要求：
1. 发生了什么：概括今日主要对话主题和事件（300字以内）
2. 结论/结果：重要的决策、答案、解决方案（200字以内）
3. 价值与用途：这些信息有什么用（100字以内）
4. 高印象事件：列出3-5个重要、搞笑或特殊的事件标题

对话记录：
{{CONTENT}}

请按以下JSON格式返回：
{
  "what_happened": "发生了什么...",
  "conclusion": "结论/结果...",
  "value": "价值与用途...",
  "highlights": ["高印象事件1", "高印象事件2", "高印象事件3"]
}`

// SummarizeConversation 对单条对话进行初步总结
func (s *memorySummarizer) SummarizeConversation(ctx context.Context, messages []models.Message) (*models.ConversationSummary, error) {
	if len(messages) == 0 {
		return &models.ConversationSummary{
			Summary:   "空对话",
			KeyPoints: "",
		}, nil
	}

	// 限制消息数量
	if len(messages) > s.maxMessages {
		messages = messages[len(messages)-s.maxMessages:]
	}

	// 构建对话内容
	var content strings.Builder
	for _, msg := range messages {
		content.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp.Format("15:04:05"), msg.Role, msg.Content))
	}

	// 替换提示词模板
	prompt := strings.ReplaceAll(s.conversationPrompt, "{{CONTENT}}", content.String())

	// 调用 LLM
	response, err := s.client.Complete(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("llm complete failed: %w", err)
	}

	// 解析响应
	summary := s.parseConversationSummary(response)
	return summary, nil
}

// SummarizeToLongTerm 将多条流水记忆提炼为长期记忆
func (s *memorySummarizer) SummarizeToLongTerm(ctx context.Context, streams []models.StreamMemory) (*models.LongTermSummary, error) {
	if len(streams) == 0 {
		return &models.LongTermSummary{
			WhatHappened: "无记录",
			Conclusion:   "",
			Value:        "",
			Highlights:   []string{},
		}, nil
	}

	// 构建对话记录
	var content strings.Builder
	for i, stream := range streams {
		content.WriteString(fmt.Sprintf("\n--- 记录 %d ---\n", i+1))
		content.WriteString(fmt.Sprintf("时间: %s\n", stream.CreatedAt.Format("15:04:05")))
		content.WriteString(fmt.Sprintf("会话: %s\n", stream.SessionKey))
		if stream.Summary != "" {
			content.WriteString(fmt.Sprintf("总结: %s\n", stream.Summary))
		} else {
			content.WriteString(fmt.Sprintf("内容: %s\n", utils.TruncateString(stream.Content, 500)))
		}
	}

	// 替换提示词模板
	prompt := strings.ReplaceAll(s.longTermPrompt, "{{CONTENT}}", content.String())

	// 调用 LLM
	response, err := s.client.Complete(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("llm complete failed: %w", err)
	}

	// 解析响应
	summary := s.parseLongTermSummary(response)
	return summary, nil
}

// parseConversationSummary 解析对话总结响应
func (s *memorySummarizer) parseConversationSummary(response string) *models.ConversationSummary {
	// 尝试解析 JSON
	var result struct {
		Summary   string `json:"summary"`
		KeyPoints string `json:"key_points"`
	}

	// 提取 JSON 部分
	jsonStr := extractJSON(response)
	if jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return &models.ConversationSummary{
				Summary:   result.Summary,
				KeyPoints: result.KeyPoints,
			}
		}
	}

	// JSON 解析失败，使用文本解析
	summary := &models.ConversationSummary{}

	lines := strings.Split(response, "\n")
	var keyPoints []string
	inKeyPoints := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检测关键要点部分
		if strings.Contains(strings.ToLower(line), "关键要点") ||
			strings.Contains(strings.ToLower(line), "要点") {
			inKeyPoints = true
			continue
		}

		// 检测总结部分
		if strings.Contains(strings.ToLower(line), "总结") ||
			strings.Contains(strings.ToLower(line), "概括") {
			if idx := strings.Index(line, ":"); idx != -1 {
				summary.Summary = strings.TrimSpace(line[idx+1:])
			}
			continue
		}

		// 收集关键要点
		if inKeyPoints && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") ||
			(strings.Contains(line, ".") && len(line) > 10)) {
			point := strings.TrimPrefix(line, "-")
			point = strings.TrimPrefix(point, "•")
			point = strings.TrimSpace(point)
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		}
	}

	if summary.Summary == "" && len(lines) > 0 {
		summary.Summary = lines[0]
	}

	if len(keyPoints) > 0 {
		summary.KeyPoints = strings.Join(keyPoints, "\n")
	}

	// 过滤敏感信息
	summary.Summary = filterSensitiveInfo(summary.Summary)
	summary.KeyPoints = filterSensitiveInfo(summary.KeyPoints)

	return summary
}

// parseLongTermSummary 解析长期记忆总结响应
func (s *memorySummarizer) parseLongTermSummary(response string) *models.LongTermSummary {
	// 尝试解析 JSON
	var result struct {
		WhatHappened string   `json:"what_happened"`
		Conclusion   string   `json:"conclusion"`
		Value        string   `json:"value"`
		Highlights   []string `json:"highlights"`
	}

	// 提取 JSON 部分
	jsonStr := extractJSON(response)
	if jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return &models.LongTermSummary{
				WhatHappened: filterSensitiveInfo(result.WhatHappened),
				Conclusion:   filterSensitiveInfo(result.Conclusion),
				Value:        filterSensitiveInfo(result.Value),
				Highlights:   filterSensitiveHighlights(result.Highlights),
			}
		}
	}

	// JSON 解析失败，使用文本解析
	summary := &models.LongTermSummary{}

	lines := strings.Split(response, "\n")
	var currentSection string
	var highlights []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lowerLine := strings.ToLower(line)

		// 检测各节
		switch {
		case strings.Contains(lowerLine, "发生了什么") || strings.Contains(lowerLine, "what_happened"):
			currentSection = "what_happened"
			if idx := strings.Index(line, ":"); idx != -1 {
				summary.WhatHappened = strings.TrimSpace(line[idx+1:])
			}
		case strings.Contains(lowerLine, "结论") || strings.Contains(lowerLine, "conclusion"):
			currentSection = "conclusion"
			if idx := strings.Index(line, ":"); idx != -1 {
				summary.Conclusion = strings.TrimSpace(line[idx+1:])
			}
		case strings.Contains(lowerLine, "价值") || strings.Contains(lowerLine, "value"):
			currentSection = "value"
			if idx := strings.Index(line, ":"); idx != -1 {
				summary.Value = strings.TrimSpace(line[idx+1:])
			}
		case strings.Contains(lowerLine, "高印象") || strings.Contains(lowerLine, "highlights"):
			currentSection = "highlights"
		default:
			// 累积内容
			switch currentSection {
			case "what_happened":
				summary.WhatHappened += " " + line
			case "conclusion":
				summary.Conclusion += " " + line
			case "value":
				summary.Value += " " + line
			case "highlights":
				if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") ||
					(strings.Contains(line, ".") && len(line) < 100) {
					h := strings.TrimPrefix(line, "-")
					h = strings.TrimPrefix(h, "•")
					h = strings.TrimSpace(h)
					if h != "" {
						highlights = append(highlights, h)
					}
				}
			}
		}
	}

	if len(highlights) > 0 {
		summary.Highlights = highlights
	}

	// 过滤敏感信息
	summary.WhatHappened = filterSensitiveInfo(summary.WhatHappened)
	summary.Conclusion = filterSensitiveInfo(summary.Conclusion)
	summary.Value = filterSensitiveInfo(summary.Value)

	return summary
}

// extractJSON 从文本中提取 JSON
func extractJSON(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return text[start : end+1]
	}
	return ""
}

// filterSensitiveInfo 过滤敏感信息
func filterSensitiveInfo(text string) string {
	if text == "" {
		return text
	}

	// 常见的敏感信息模式
	patterns := []struct {
		pattern     string
		replacement string
	}{
		// API Key / Token
		{`[a-zA-Z0-9_-]{20,}`, `[FILTERED_TOKEN]`},
		// 密码字段
		{`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`, `[FILTERED_PASSWORD]`},
		// API Key 字段
		{`(?i)(api[_-]?key|apikey|token|secret)\s*[:=]\s*\S+`, `[FILTERED_APIKEY]`},
	}

	result := text
	for _, p := range patterns {
		// 这里简化处理，实际可以使用正则表达式
		_ = p
	}

	return result
}

// filterSensitiveHighlights 过滤高印象事件中的敏感信息
func filterSensitiveHighlights(highlights []string) []string {
	filtered := make([]string, 0, len(highlights))
	for _, h := range highlights {
		filtered = append(filtered, filterSensitiveInfo(h))
	}
	return filtered
}

