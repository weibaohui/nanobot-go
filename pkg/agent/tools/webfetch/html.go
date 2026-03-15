package webfetch

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// validateURL 验证 URL
func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("无效的 URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("只支持 http/https 协议，当前: %s", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("缺少域名")
	}

	return nil
}

// isHTML 检查内容是否为 HTML
func isHTML(body []byte) bool {
	preview := strings.ToLower(string(body[:min(256, len(body))]))
	return strings.HasPrefix(preview, "<!doctype") || strings.HasPrefix(preview, "<html")
}

// stripTags 移除 HTML 标签
func stripTags(html string) string {
	// 移除 script 标签
	re := regexp.MustCompile(`(?i)<script[\s\S]*?</script>`)
	html = re.ReplaceAllString(html, "")

	// 移除 style 标签
	re = regexp.MustCompile(`(?i)<style[\s\S]*?</style>`)
	html = re.ReplaceAllString(html, "")

	// 移除所有 HTML 标签
	re = regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(html, "")

	// 解码 HTML 实体
	text = decodeHTMLEntities(text)

	// 规范化空白
	return normalize(text)
}

// decodeHTMLEntities 解码 HTML 实体
func decodeHTMLEntities(text string) string {
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&apos;", "'")
	return text
}

// normalize 规范化空白
func normalize(text string) string {
	// 替换多个空格/制表符为单个空格
	re := regexp.MustCompile(`[ \t]+`)
	text = re.ReplaceAllString(text, " ")

	// 替换多个换行为两个换行
	re = regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// min 返回较小的整数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
