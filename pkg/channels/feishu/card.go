package feishu

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/weibaohui/nanobot-go/pkg/bus"
	"go.uber.org/zap"
)

// tableRegex 匹配 Markdown 表格的正则表达式
var tableRegex = regexp.MustCompile(`(?m)((?:^[ \t]*\|.+\|[ \t]*\n)(?:^[ \t]*\|[-:\s|]+\|[ \t]*\n)(?:^[ \t]*\|.+(?:\n|$))+)`)

// Send 发送消息
func (c *Channel) Send(msg *bus.OutboundMessage) error {
	if c.client == nil {
		return fmt.Errorf("飞书客户端未初始化")
	}

	// 确定 receive_id_type
	// 支持: open_id (ou_), union_id (on_), chat_id (oc_)
	receiveIDType := "open_id"
	if strings.HasPrefix(msg.ChatID, "oc_") {
		receiveIDType = "chat_id"
	} else if strings.HasPrefix(msg.ChatID, "on_") {
		receiveIDType = "union_id"
	}

	// 构建卡片消息（支持 Markdown 和表格）
	card := c.buildCard(msg.Content)
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("序列化卡片失败: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType("interactive").
			Content(string(cardJSON)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(c.ctx, req)
	if err != nil {
		return fmt.Errorf("发送飞书消息失败: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("发送飞书消息失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	c.logger.Debug("飞书消息已发送",
		zap.String("chat_id", msg.ChatID),
	)

	// 发送成功后，删除"正在处理"反应表情
	// 从 Metadata 中获取原始消息的 message_id
	if msg.Metadata != nil {
		if replyToMsgID, ok := msg.Metadata["reply_to_message_id"].(string); ok && replyToMsgID != "" {
			go c.deleteReactionFromCache(replyToMsgID)
		}
	}

	return nil
}

// buildCard 构建飞书卡片消息
func (c *Channel) buildCard(content string) map[string]interface{} {
	elements := c.buildCardElements(content)
	return map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": elements,
	}
}

// buildCardElements 构建卡片元素（支持 Markdown 和表格）
func (c *Channel) buildCardElements(content string) []interface{} {
	var elements []interface{}
	lastEnd := 0

	// 查找所有表格
	matches := tableRegex.FindAllStringIndex(content, -1)
	for _, match := range matches {
		// 表格前的 Markdown 内容
		before := strings.TrimSpace(content[lastEnd:match[0]])
		if before != "" {
			elements = append(elements, map[string]interface{}{
				"tag":     "markdown",
				"content": before,
			})
		}

		// 解析表格
		table := c.parseMarkdownTable(content[match[0]:match[1]])
		if table != nil {
			elements = append(elements, table)
		} else {
			elements = append(elements, map[string]interface{}{
				"tag":     "markdown",
				"content": content[match[0]:match[1]],
			})
		}

		lastEnd = match[1]
	}

	// 剩余内容
	remaining := strings.TrimSpace(content[lastEnd:])
	if remaining != "" {
		elements = append(elements, map[string]interface{}{
			"tag":     "markdown",
			"content": remaining,
		})
	}

	if len(elements) == 0 {
		elements = append(elements, map[string]interface{}{
			"tag":     "markdown",
			"content": content,
		})
	}

	return elements
}

// parseMarkdownTable 解析 Markdown 表格为飞书表格元素
func (c *Channel) parseMarkdownTable(tableText string) interface{} {
	lines := strings.Split(strings.TrimSpace(tableText), "\n")
	if len(lines) < 3 {
		return nil
	}

	// 过滤空行
	var validLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			validLines = append(validLines, line)
		}
	}

	if len(validLines) < 3 {
		return nil
	}

	// 解析表头
	headers := c.splitTableRow(validLines[0])
	if len(headers) == 0 {
		return nil
	}

	// 跳过分隔行（第2行）
	// 解析数据行
	var rows []map[string]interface{}
	for i := 2; i < len(validLines); i++ {
		cells := c.splitTableRow(validLines[i])
		row := make(map[string]interface{})
		for j := range headers {
			key := fmt.Sprintf("c%d", j)
			if j < len(cells) {
				row[key] = cells[j]
			} else {
				row[key] = ""
			}
		}
		rows = append(rows, row)
	}

	// 构建列定义
	var columns []map[string]interface{}
	for i, h := range headers {
		columns = append(columns, map[string]interface{}{
			"tag":          "column",
			"name":         fmt.Sprintf("c%d", i),
			"display_name": h,
			"width":        "auto",
		})
	}

	return map[string]interface{}{
		"tag":       "table",
		"page_size": len(rows) + 1,
		"columns":   columns,
		"rows":      rows,
	}
}

// splitTableRow 分割表格行
func (c *Channel) splitTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.Trim(row, "|")
	cells := strings.Split(row, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}
