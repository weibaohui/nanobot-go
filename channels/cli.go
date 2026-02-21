package channels

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/weibaohui/nanobot-go/bus"
	"go.uber.org/zap"
)

// CLIChannel 命令行渠道
type CLIChannel struct {
	*BaseChannel
	logger   *zap.Logger
	chatID   string
	stopChan chan struct{}
}

// NewCLIChannel 创建 CLI 渠道
func NewCLIChannel(messageBus *bus.MessageBus, chatID string, logger *zap.Logger) *CLIChannel {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CLIChannel{
		BaseChannel: NewBaseChannel("cli", messageBus),
		logger:      logger,
		chatID:      chatID,
		stopChan:    make(chan struct{}),
	}
}

// Start 启动 CLI 渠道
func (c *CLIChannel) Start(ctx context.Context) error {
	c.logger.Info("CLI 渠道已启动")

	// 订阅出站消息
	c.SubscribeOutbound(ctx, func(msg *bus.OutboundMessage) {
		fmt.Println("\n" + msg.Content)
		fmt.Print("\n> ")
	})

	// 启动输入循环
	go c.inputLoop(ctx)

	return nil
}

// Stop 停止 CLI 渠道
func (c *CLIChannel) Stop() {
	close(c.stopChan)
	c.logger.Info("CLI 渠道已停止")
}

// inputLoop 输入循环
func (c *CLIChannel) inputLoop(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("> ")
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		text := strings.TrimSpace(line)
		if text == "" {
			fmt.Print("> ")
			continue
		}

		// 处理命令
		if strings.HasPrefix(text, "/") {
			c.handleCommand(text)
			fmt.Print("> ")
			continue
		}

		// 发布入站消息
		msg := bus.NewInboundMessage("cli", "user", c.chatID, text)
		c.PublishInbound(msg)
	}
}

// handleCommand 处理命令
func (c *CLIChannel) handleCommand(cmd string) {
	switch cmd {
	case "/exit", "/quit":
		fmt.Println("再见!")
		os.Exit(0)
	case "/help":
		fmt.Println(`可用命令:
  /help    显示帮助
  /exit    退出程序
  /clear   清空会话
  /status  显示状态`)
	case "/clear":
		fmt.Println("会话已清空")
	case "/status":
		fmt.Println("状态: 运行中")
	default:
		fmt.Printf("未知命令: %s\n", cmd)
	}
}
