package webfetch

import (
	"fmt"
	"regexp"
	"strings"
)

// toMarkdown 将 HTML 转换为 Markdown
func toMarkdown(html string) string {
	// 转换链接: <a href="url">text</a> -> [text](url)
	re := regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)</a>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			return fmt.Sprintf("[%s](%s)", stripTags(submatches[2]), submatches[1])
		}
		return match
	})

	// 转换标题: <h1>text</h1> -> # text
	re = regexp.MustCompile(`(?i)<h([1-6])[^>]*>([\s\S]*?)</h[1-6]>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			level := submatches[1]
			text := stripTags(submatches[2])
			return fmt.Sprintf("\n%s %s\n", strings.Repeat("#", parseInt(level)), text)
		}
		return match
	})

	// 转换列表项: <li>text</li> -> - text
	re = regexp.MustCompile(`(?i)<li[^>]*>([\s\S]*?)</li>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("\n- %s", stripTags(submatches[1]))
		}
		return match
	})

	// 转换段落/块元素
	re = regexp.MustCompile(`(?i)</(p|div|section|article)>`)
	html = re.ReplaceAllString(html, "\n\n")

	// 转换换行
	re = regexp.MustCompile(`(?i)<(br|hr)\s*/?>`)
	html = re.ReplaceAllString(html, "\n")

	// 转换粗体
	re = regexp.MustCompile(`(?i)<(?:strong|b)[^>]*>([\s\S]*?)</(?:strong|b)>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("**%s**", stripTags(submatches[1]))
		}
		return match
	})

	// 转换斜体
	re = regexp.MustCompile(`(?i)<(?:em|i)[^>]*>([\s\S]*?)</(?:em|i)>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("*%s*", stripTags(submatches[1]))
		}
		return match
	})

	// 转换代码块
	re = regexp.MustCompile(`(?i)<code[^>]*>([\s\S]*?)</code>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("`%s`", submatches[1])
		}
		return match
	})

	// 转换预格式化块
	re = regexp.MustCompile(`(?i)<pre[^>]*>([\s\S]*?)</pre>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			return fmt.Sprintf("\n```\n%s\n```\n", stripTags(submatches[1]))
		}
		return match
	})

	// 移除剩余标签并规范化
	return normalize(stripTags(html))
}

// parseInt 解析整数
func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}
