package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/providers"
	"go.uber.org/zap"
)

// WebSocketConfig WebSocket 渠道配置
type WebSocketConfig struct {
	// Addr 监听地址，如 ":8088"
	Addr string
	// Path WebSocket 路径，如 "/ws"
	Path string
	// AllowFrom 允许的用户 ID 列表（为空表示允许所有）
	AllowFrom []string
	// EnableStreaming 是否启用流式输出（打字机效果）
	EnableStreaming bool
}

// WebSocketChannel WebSocket 渠道
type WebSocketChannel struct {
	*BaseChannel
	config    *WebSocketConfig
	server    *http.Server
	upgrader  websocket.Upgrader
	clients   map[string]*websocket.Conn // chatID -> conn
	clientsMu sync.RWMutex
	logger    *zap.Logger

	// 用于流式处理
	provider providers.LLMProvider
	model    string
}

// SetProvider 设置 LLM 提供者（用于流式处理）
func (c *WebSocketChannel) SetProvider(provider providers.LLMProvider, model string) {
	c.provider = provider
	c.model = model
}

// NewWebSocketChannel 创建 WebSocket 渠道
func NewWebSocketChannel(config *WebSocketConfig, bus *bus.MessageBus, logger *zap.Logger) *WebSocketChannel {
	if config == nil {
		config = &WebSocketConfig{}
	}
	if config.Addr == "" {
		config.Addr = ":8088"
	}
	if config.Path == "" {
		config.Path = "/ws"
	}

	return &WebSocketChannel{
		BaseChannel: NewBaseChannel("websocket", bus),
		config:      config,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
		clients: make(map[string]*websocket.Conn),
		logger:  logger,
	}
}

// Name 返回渠道名称
func (c *WebSocketChannel) Name() string {
	return "websocket"
}

// Start 启动 WebSocket 服务
func (c *WebSocketChannel) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// WebSocket 端点
	mux.HandleFunc(c.config.Path, c.handleWebSocket)

	// 静态页面
	mux.HandleFunc("/", c.handleIndex)

	// 订阅出站消息（用于非流式响应）
	c.SubscribeOutbound(ctx, func(msg *bus.OutboundMessage) {
		if msg.Channel != "websocket" {
			return
		}
		c.sendToClient(msg.ChatID, msg.Content)
	})

	// 订阅流式消息（用于打字机效果）
	c.bus.SubscribeStream("websocket", func(chunk *bus.StreamChunk) error {
		return c.sendStreamChunk(chunk.ChatID, chunk)
	})

	c.server = &http.Server{
		Addr:    c.config.Addr,
		Handler: mux,
	}

	c.logger.Info("WebSocket 渠道启动",
		zap.String("addr", c.config.Addr),
		zap.String("path", c.config.Path),
		zap.Bool("streaming", true),
	)

	// 在后台启动服务器
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.logger.Error("WebSocket 服务器错误", zap.Error(err))
		}
	}()

	return nil
}

// Stop 停止 WebSocket 服务
func (c *WebSocketChannel) Stop() {
	if c.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.server.Shutdown(ctx)
	}

	// 关闭所有客户端连接
	c.clientsMu.Lock()
	for _, conn := range c.clients {
		conn.Close()
	}
	c.clients = make(map[string]*websocket.Conn)
	c.clientsMu.Unlock()

	c.logger.Info("WebSocket 渠道已停止")
}

// handleWebSocket 处理 WebSocket 连接
func (c *WebSocketChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Error("WebSocket 升级失败", zap.Error(err))
		return
	}
	defer conn.Close()

	// 生成 chatID
	chatID := generateChatID(r)

	// 注册客户端
	c.clientsMu.Lock()
	c.clients[chatID] = conn
	c.clientsMu.Unlock()

	c.logger.Info("WebSocket 客户端连接",
		zap.String("chat_id", chatID),
		zap.String("remote_addr", r.RemoteAddr),
	)

	// 清理连接
	defer func() {
		c.clientsMu.Lock()
		delete(c.clients, chatID)
		c.clientsMu.Unlock()
		c.logger.Info("WebSocket 客户端断开", zap.String("chat_id", chatID))
	}()

	// 读取消息循环
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket 读取错误", zap.Error(err))
			}
			break
		}

		// 解析消息
		var msg struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			c.logger.Error("解析消息失败", zap.Error(err))
			continue
		}

		if msg.Content == "" {
			continue
		}

		// 检查用户权限
		if len(c.config.AllowFrom) > 0 {
			allowed := false
			for _, id := range c.config.AllowFrom {
				if id == chatID || id == "*" {
					allowed = true
					break
				}
			}
			if !allowed {
				c.sendToClient(chatID, "抱歉，您没有权限使用此服务。")
				continue
			}
		}

		c.logger.Info("收到 WebSocket 消息",
			zap.String("chat_id", chatID),
			zap.String("content", truncate(msg.Content, 100)),
		)

		// 发布入站消息
		c.PublishInbound(&bus.InboundMessage{
			Channel:   "websocket",
			SenderID:  chatID,
			ChatID:    chatID,
			Content:   msg.Content,
			Timestamp: time.Now(),
		})
	}
}

// handleIndex 处理首页请求
func (c *WebSocketChannel) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

// sendToClient 发送消息给客户端
func (c *WebSocketChannel) sendToClient(chatID, content string) {
	c.clientsMu.RLock()
	conn, ok := c.clients[chatID]
	c.clientsMu.RUnlock()

	if !ok {
		return
	}

	msg := struct {
		Type    string `json:"type"`
		Content string `json:"content"`
		Time    string `json:"time"`
	}{
		Type:    "message",
		Content: content,
		Time:    time.Now().Format("15:04:05"),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.logger.Error("发送消息失败", zap.Error(err))
	}
}

// sendStreamChunk 发送流式片段给客户端（打字机效果）
func (c *WebSocketChannel) sendStreamChunk(chatID string, chunk *bus.StreamChunk) error {
	c.clientsMu.RLock()
	conn, ok := c.clients[chatID]
	c.clientsMu.RUnlock()

	if !ok {
		return nil
	}

	msg := struct {
		Type  string `json:"type"`
		Delta string `json:"delta"`
		Text  string `json:"text"`
		Time  string `json:"time"`
		Done  bool   `json:"done"`
	}{
		Type:  "stream",
		Delta: chunk.Delta,
		Text:  chunk.Content,
		Time:  time.Now().Format("15:04:05"),
		Done:  chunk.Done,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("序列化流式消息失败", zap.Error(err))
		return err
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.logger.Error("发送流式消息失败", zap.Error(err))
		return err
	}

	return nil
}

// StreamToClient 流式发送消息给客户端（打字机效果）
func (c *WebSocketChannel) StreamToClient(chatID string, ch <-chan string) {
	c.clientsMu.RLock()
	conn, ok := c.clients[chatID]
	c.clientsMu.RUnlock()

	if !ok {
		return
	}

	var fullContent string
	for delta := range ch {
		fullContent += delta

		msg := struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
			Text  string `json:"text"`
			Time  string `json:"time"`
			Done  bool   `json:"done"`
		}{
			Type:  "stream",
			Delta: delta,
			Text:  fullContent,
			Time:  time.Now().Format("15:04:05"),
			Done:  false,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			c.logger.Error("序列化流式消息失败", zap.Error(err))
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			c.logger.Error("发送流式消息失败", zap.Error(err))
			return
		}
	}

	// 发送完成消息
	doneMsg := struct {
		Type    string `json:"type"`
		Content string `json:"content"`
		Time    string `json:"time"`
		Done    bool   `json:"done"`
	}{
		Type:    "stream",
		Content: fullContent,
		Time:    time.Now().Format("15:04:05"),
		Done:    true,
	}

	data, _ := json.Marshal(doneMsg)
	conn.WriteMessage(websocket.TextMessage, data)
}

// generateChatID 生成 chatID
func generateChatID(r *http.Request) string {
	return fmt.Sprintf("ws_%s", r.RemoteAddr)
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// indexHTML 是聊天页面的 HTML（带打字机效果）
var indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🐈 nanobot - AI 助手</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/github-markdown-css@5.5.1/github-markdown.min.css">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/styles/github.min.css">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .chat-container {
            width: 100%;
            max-width: 800px;
            height: 90vh;
            background: #fff;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        .chat-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .chat-header .logo {
            font-size: 32px;
        }
        .chat-header h1 {
            font-size: 20px;
            font-weight: 600;
        }
        .chat-header .status {
            margin-left: auto;
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
        }
        .chat-header .status-dot {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            background: #4ade80;
        }
        .chat-header .status-dot.disconnected {
            background: #f87171;
        }
        .chat-messages {
            flex: 1;
            overflow-y: auto;
            padding: 20px;
            background: #f8fafc;
        }
        .message {
            margin-bottom: 16px;
            display: flex;
            flex-direction: column;
        }
        .message.user {
            align-items: flex-end;
        }
        .message.assistant {
            align-items: flex-start;
        }
        .message-bubble {
            max-width: 80%;
            padding: 12px 18px;
            border-radius: 18px;
            line-height: 1.6;
            word-wrap: break-word;
            white-space: pre-wrap;
        }
        .message.user .message-bubble {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border-bottom-right-radius: 4px;
        }
        .message.assistant .message-bubble {
            background: white;
            color: #1f2937;
            border-bottom-left-radius: 4px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .message.assistant .message-bubble.markdown-body {
            white-space: normal;
            color: #1f2937;
            background: white;
        }
        .message.assistant .message-bubble.markdown-body pre {
            overflow: auto;
        }
        .message.assistant .message-bubble.markdown-body code {
            white-space: pre;
        }
        .message-time {
            font-size: 11px;
            color: #9ca3af;
            margin-top: 4px;
            padding: 0 4px;
        }
        .cursor {
            display: inline-block;
            width: 8px;
            height: 18px;
            background: #667eea;
            animation: blink 1s infinite;
            vertical-align: text-bottom;
            margin-left: 2px;
        }
        @keyframes blink {
            0%, 50% { opacity: 1; }
            51%, 100% { opacity: 0; }
        }
        .chat-input-container {
            padding: 20px;
            background: white;
            border-top: 1px solid #e5e7eb;
        }
        .chat-input-wrapper {
            display: flex;
            gap: 12px;
            align-items: flex-end;
        }
        .chat-input {
            flex: 1;
            padding: 14px 18px;
            border: 2px solid #e5e7eb;
            border-radius: 24px;
            font-size: 15px;
            outline: none;
            transition: border-color 0.2s;
            resize: none;
            max-height: 120px;
            font-family: inherit;
        }
        .chat-input:focus {
            border-color: #667eea;
        }
        .send-button {
            width: 50px;
            height: 50px;
            border: none;
            border-radius: 50%;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .send-button:hover {
            transform: scale(1.05);
            box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
        }
        .send-button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
            transform: none;
        }
        .send-button svg {
            width: 24px;
            height: 24px;
        }
        .typing-indicator {
            display: none;
            align-items: center;
            gap: 4px;
            padding: 12px 18px;
            background: white;
            border-radius: 18px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            margin-bottom: 16px;
        }
        .typing-indicator.show {
            display: flex;
        }
        .typing-indicator span {
            width: 8px;
            height: 8px;
            background: #9ca3af;
            border-radius: 50%;
            animation: typing 1.4s infinite;
        }
        .typing-indicator span:nth-child(2) {
            animation-delay: 0.2s;
        }
        .typing-indicator span:nth-child(3) {
            animation-delay: 0.4s;
        }
        @keyframes typing {
            0%, 60%, 100% { transform: translateY(0); }
            30% { transform: translateY(-8px); }
        }
        .welcome-message {
            text-align: center;
            padding: 40px 20px;
            color: #6b7280;
        }
        .welcome-message h2 {
            font-size: 24px;
            margin-bottom: 10px;
            color: #1f2937;
        }
        .welcome-message p {
            font-size: 14px;
            line-height: 1.6;
        }
        .welcome-message .tips {
            margin-top: 20px;
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            justify-content: center;
        }
        .welcome-message .tip {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 13px;
            cursor: pointer;
            transition: transform 0.2s;
        }
        .welcome-message .tip:hover {
            transform: scale(1.05);
        }
        @media (max-width: 600px) {
            body {
                padding: 0;
            }
            .chat-container {
                height: 100vh;
                border-radius: 0;
            }
            .chat-input {
                font-size: 16px;
            }
        }
    </style>
</head>
<body>
    <div class="chat-container">
        <div class="chat-header">
            <span class="logo">🐈</span>
            <h1>nanobot</h1>
            <div class="status">
                <span class="status-dot" id="statusDot"></span>
                <span id="statusText">连接中...</span>
            </div>
        </div>
        <div class="chat-messages" id="chatMessages">
            <div class="welcome-message">
                <h2>🐾 欢迎使用 nanobot</h2>
                <p>我是一个 AI 助手，可以帮助你完成各种任务。<br>支持打字机效果实时输出！</p>
                <div class="tips">
                    <span class="tip" onclick="sendTip('你好')">👋 打个招呼</span>
                    <span class="tip" onclick="sendTip('帮我规划一次旅行')">🗺️ 规划任务</span>
                    <span class="tip" onclick="sendTip('帮我写一段代码')">💻 写代码</span>
                </div>
            </div>
        </div>
        <div class="typing-indicator" id="typingIndicator">
            <span></span><span></span><span></span>
        </div>
        <div class="chat-input-container">
            <div class="chat-input-wrapper">
                <textarea
                    class="chat-input"
                    id="chatInput"
                    placeholder="输入消息..."
                    rows="1"
                ></textarea>
                <button class="send-button" id="sendButton" onclick="sendMessage()">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/>
                    </svg>
                </button>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/marked@12.0.2/marked.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/lib/highlight.min.js"></script>
    <script>
        let ws = null;
        let connected = false;
        let streamingMessage = null;
        let streamingContent = '';
        let isComposing = false; // 输入法组合状态
        const messagesDiv = document.getElementById('chatMessages');
        const chatInput = document.getElementById('chatInput');
        const sendButton = document.getElementById('sendButton');
        const statusDot = document.getElementById('statusDot');
        const statusText = document.getElementById('statusText');
        const typingIndicator = document.getElementById('typingIndicator');

        marked.setOptions({
            highlight: function(code, lang) {
                if (lang && hljs.getLanguage(lang)) {
                    return hljs.highlight(code, { language: lang }).value;
                }
                return hljs.highlightAuto(code).value;
            },
            breaks: true,
            gfm: true
        });

        // 监听输入法组合事件
        chatInput.addEventListener('compositionstart', function() {
            isComposing = true;
        });
        chatInput.addEventListener('compositionend', function() {
            isComposing = false;
        });

        // 使用 addEventListener 监听键盘事件
        chatInput.addEventListener('keydown', function(event) {
            handleKeyDown(event);
        });

        // 自动调整输入框高度
        chatInput.addEventListener('input', function() {
            this.style.height = 'auto';
            this.style.height = Math.min(this.scrollHeight, 120) + 'px';
        });

        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = protocol + '//' + window.location.host + '/ws';

            ws = new WebSocket(wsUrl);

            ws.onopen = function() {
                connected = true;
                statusDot.classList.remove('disconnected');
                statusText.textContent = '已连接';
                console.log('WebSocket 已连接');
            };

            ws.onclose = function() {
                connected = false;
                statusDot.classList.add('disconnected');
                statusText.textContent = '已断开';
                console.log('WebSocket 已断开');

                // 5秒后重连
                setTimeout(connect, 5000);
            };

            ws.onerror = function(error) {
                console.error('WebSocket 错误:', error);
            };

            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);

                if (data.type === 'stream') {
                    // 真正的流式消息（打字机效果）
                    handleStreamMessage(data);
                } else if (data.type === 'message') {
                    // 完整消息 - 使用前端打字机效果
                    typewriterMessage('assistant', data.content, data.time);
                }
            };
        }

        function handleStreamMessage(data) {
            // 如果是新消息开始，创建消息气泡
            if (!streamingMessage) {
                hideTyping();
                streamingMessage = createMessageBubble('assistant');
                streamingContent = '';
            }

            // 追加内容
            if (data.delta) {
                streamingContent += data.delta;
                updateMessageContent(streamingMessage, streamingContent);
            }

            // 如果消息完成
            if (data.done) {
                // 移除光标，添加时间戳
                finishMessage(streamingMessage, data.time || new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }));
                streamingMessage = null;
                streamingContent = '';
            }
        }

        function createMessageBubble(role) {
            // 移除欢迎消息
            const welcome = document.querySelector('.welcome-message');
            if (welcome) {
                welcome.remove();
            }

            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + role;

            const bubbleDiv = document.createElement('div');
            bubbleDiv.className = role === 'assistant' ? 'message-bubble markdown-body' : 'message-bubble';

            // 添加闪烁光标
            const cursor = document.createElement('span');
            cursor.className = 'cursor';
            bubbleDiv.appendChild(cursor);

            messageDiv.appendChild(bubbleDiv);
            messagesDiv.appendChild(messageDiv);

            // 滚动到底部
            messagesDiv.scrollTop = messagesDiv.scrollHeight;

            return { div: messageDiv, bubble: bubbleDiv };
        }

        function renderMarkdown(content) {
            return marked.parse(content || '');
        }

        function setBubbleContent(bubble, content) {
            if (bubble.classList.contains('markdown-body')) {
                bubble.innerHTML = renderMarkdown(content);
            } else {
                bubble.textContent = content;
            }
        }

        function updateMessageContent(msgObj, content) {
            // 保留光标
            const cursor = msgObj.bubble.querySelector('.cursor');
            setBubbleContent(msgObj.bubble, content);
            if (cursor) {
                msgObj.bubble.appendChild(cursor);
            }

            // 滚动到底部
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function finishMessage(msgObj, time) {
            // 移除光标
            const cursor = msgObj.bubble.querySelector('.cursor');
            if (cursor) {
                cursor.remove();
            }

            // 添加时间戳
            const timeDiv = document.createElement('div');
            timeDiv.className = 'message-time';
            timeDiv.textContent = time;
            msgObj.div.appendChild(timeDiv);
        }

        function sendMessage() {
            const content = chatInput.value.trim();
            if (!content || !connected) return;

            addMessage('user', content);
            chatInput.value = '';
            chatInput.style.height = 'auto';

            ws.send(JSON.stringify({ content: content }));
            showTyping();
        }

        function sendTip(text) {
            chatInput.value = text;
            sendMessage();
        }

        function addMessage(role, content, time) {
            // 移除欢迎消息
            const welcome = document.querySelector('.welcome-message');
            if (welcome) {
                welcome.remove();
            }

            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + role;

            const bubbleDiv = document.createElement('div');
            bubbleDiv.className = role === 'assistant' ? 'message-bubble markdown-body' : 'message-bubble';
            setBubbleContent(bubbleDiv, content);

            const timeDiv = document.createElement('div');
            timeDiv.className = 'message-time';
            timeDiv.textContent = time || new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });

            messageDiv.appendChild(bubbleDiv);
            messageDiv.appendChild(timeDiv);
            messagesDiv.appendChild(messageDiv);

            // 滚动到底部
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        // 打字机效果显示消息
        function typewriterMessage(role, content, time) {
            hideTyping();

            // 移除欢迎消息
            const welcome = document.querySelector('.welcome-message');
            if (welcome) {
                welcome.remove();
            }

            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + role;

            const bubbleDiv = document.createElement('div');
            bubbleDiv.className = role === 'assistant' ? 'message-bubble markdown-body' : 'message-bubble';

            // 添加闪烁光标
            const cursor = document.createElement('span');
            cursor.className = 'cursor';
            bubbleDiv.appendChild(cursor);

            const timeDiv = document.createElement('div');
            timeDiv.className = 'message-time';
            timeDiv.textContent = time || new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });

            messageDiv.appendChild(bubbleDiv);
            messageDiv.appendChild(timeDiv);
            messagesDiv.appendChild(messageDiv);

            // 打字机效果
            let index = 0;
            const speed = 20; // 每个字符的延迟（毫秒）

            function type() {
                if (index < content.length) {
                    bubbleDiv.textContent = content.substring(0, index + 1);
                    bubbleDiv.appendChild(cursor);
                    index++;
                    messagesDiv.scrollTop = messagesDiv.scrollHeight;
                    setTimeout(type, speed);
                } else {
                    // 完成，移除光标
                    cursor.remove();
                    if (role === 'assistant') {
                        setBubbleContent(bubbleDiv, content);
                    }
                }
            }

            type();
        }

        function showTyping() {
            typingIndicator.classList.add('show');
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function hideTyping() {
            typingIndicator.classList.remove('show');
        }

        function handleKeyDown(event) {
            // 如果正在输入法组合中（如选中文本），不处理回车
            // 检测方式：
            // 1. 自定义 isComposing 标记（compositionstart/end 事件）
            // 2. 原生 event.isComposing 属性
            // 3. keyCode === 229（IME 激活时的特殊码）
            if (isComposing || event.isComposing || event.keyCode === 229) {
                return;
            }
            if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                sendMessage();
            }
        }

        // 启动连接
        connect();

        // 聚焦输入框
        chatInput.focus();
    </script>
</body>
</html>
`
