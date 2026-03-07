package observers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	hookevents "github.com/weibaohui/nanobot-go/agent/hooks/events"
	"github.com/weibaohui/nanobot-go/agent/hooks/observer"
	"github.com/weibaohui/nanobot-go/agent"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// CompressObserver 对话压缩观察器
// 在消息发送后检查会话状态，满足条件时执行压缩
type CompressObserver struct {
	*observer.BaseObserver
	cfg      *config.Config
	logger   *zap.Logger
	memory   *agent.MemoryStore
	llm      model.ChatModel
	sessions *session.Manager
}

// NewCompressObserver 创建压缩观察器
func NewCompressObserver(cfg *config.Config, logger *zap.Logger, mem *agent.MemoryStore, llm model.ChatModel, sessions *session.Manager, filter *observer.ObserverFilter) *CompressObserver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CompressObserver{
		BaseObserver: observer.NewBaseObserver("compress", filter),
		cfg:         cfg,
		logger:      logger,
		memory:      mem,
		llm:         llm,
		sessions:    sessions,
	}
}

// OnEvent 处理事件
func (co *CompressObserver) OnEvent(ctx context.Context, event hookevents.Event) error {
	// 只监听消息发送事件
	if event.GetEventType() != hookevents.EventMessageSent {
		return nil
	}

	msgSentEvent, ok := event.(*hookevents.MessageSentEvent)
	if !ok {
		return nil
	}

	// 获取会话
	sessionKey := msgSentEvent.SessionKey
	if sessionKey == "" {
		return nil
	}

	sess := co.sessions.GetOrCreate(sessionKey)
	if sess == nil {
		return nil
	}

	// 检查是否需要压缩
	if !co.shouldCompress(sess) {
		return nil
	}

	co.logger.Info("触发对话压缩",
		zap.String("session_key", sess.Key),
		zap.Int("messages", len(sess.Messages)),
	)

	if err := co.compressSession(ctx, sess); err != nil {
		co.logger.Error("对话压缩失败", zap.Error(err))
		return err
	}

	// 保存会话状态
	if err := co.sessions.Save(ctx, sess); err != nil {
		co.logger.Error("保存压缩后的会话失败", zap.Error(err))
		return fmt.Errorf("save compressed session: %w", err)
	}

	co.logger.Info("对话压缩完成",
		zap.String("session_key", sess.Key),
		zap.Int("remaining_messages", len(sess.Messages)),
	)

	return nil
}

// shouldCompress 检查是否满足压缩触发条件
func (co *CompressObserver) shouldCompress(sess *session.Session) bool {
	messageCount := len(sess.Messages)

	minMessages := co.cfg.Compress.MinMessages
	if minMessages <= 0 {
		minMessages = 20
	}

	return messageCount >= minMessages
}

// compressSession 执行压缩流程
func (co *CompressObserver) compressSession(ctx context.Context, sess *session.Session) error {
	// 1. 构建对话摘要
	dialogue := co.buildDialogueSummary(sess)

	// 2. 调用 LLM 提取关键信息
	longTerm, shortTerm, err := co.extractWithLLM(ctx, dialogue)
	if err != nil {
		return fmt.Errorf("LLM 提取失败: %w", err)
	}

	// 3. 保存到记忆文件
	if longTerm != "" {
		if err := co.memory.AppendToLongTerm(longTerm); err != nil {
			co.logger.Error("保存长期记忆失败", zap.Error(err))
		}
	}

	if shortTerm != "" {
		if err := co.memory.AppendToday(shortTerm); err != nil {
			co.logger.Error("保存短期记忆失败", zap.Error(err))
		}
	}

	// 4. 清理会话历史
	co.cleanupSession(sess)

	return nil
}

// buildDialogueSummary 构建对话摘要
func (co *CompressObserver) buildDialogueSummary(sess *session.Session) string {
	var sb strings.Builder
	sb.WriteString("# 对话历史\n\n")

	for i, msg := range sess.Messages {
		sb.WriteString(fmt.Sprintf("## 消息 %d\n", i+1))
		sb.WriteString(fmt.Sprintf("**角色**: %s\n", msg.Role))
		sb.WriteString(fmt.Sprintf("**时间**: %s\n", msg.Timestamp.Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("**内容**:\n%s\n\n", msg.Content))
	}

	return sb.String()
}

// buildCompressPrompt 构建 LLM 提示词
func (co *CompressObserver) buildCompressPrompt(dialogue string) string {
	return fmt.Sprintf(`你是一个专业的对话记忆提取专家。请分析以下对话历史，提取关键信息。

## 你的任务
1. 提取需要长期记忆的重要信息（用户偏好、重要事实、长期上下文等）
2. 提取需要短期记忆的临时信息（当前任务进度、近期讨论等）

## 输出格式
请以 JSON 格式返回：
{
  "long_term": "长期记忆内容（Markdown 格式）",
  "short_term": "短期记忆内容（Markdown 格式）"
}

## 注意事项
- 长期记忆：用户明确表达过的重要信息、偏好设置、需要跨会话保持的内容
- 短期记忆：当前正在进行的工作、临时讨论、近期对话摘要
- 保持简洁，只提取真正重要的信息
- 如果某类信息为空，对应字段返回空字符串

## 对话历史
%s
`, dialogue)
}

// extractWithLLM 调用 LLM 提取关键信息
func (co *CompressObserver) extractWithLLM(ctx context.Context, dialogue string) (string, string, error) {
	prompt := co.buildCompressPrompt(dialogue)

	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	response, err := co.llm.Generate(ctx, messages)
	if err != nil {
		return "", "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	return co.parseExtractionResult(response.Content)
}

// parseExtractionResult 解析 LLM 返回的 JSON 结果
func (co *CompressObserver) parseExtractionResult(content string) (string, string, error) {
	// 尝试提取 JSON 部分
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return "", "", fmt.Errorf("未找到有效的 JSON 内容")
	}

	jsonContent := content[jsonStart : jsonEnd+1]

	var result struct {
		LongTerm  string `json:"long_term"`
		ShortTerm string `json:"short_term"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		return "", "", fmt.Errorf("JSON 解析失败: %w", err)
	}

	return result.LongTerm, result.ShortTerm, nil
}

// cleanupSession 清理会话历史，保留最后 N 条消息
func (co *CompressObserver) cleanupSession(sess *session.Session) {
	maxHistory := co.cfg.Compress.MaxHistory
	if maxHistory <= 0 {
		maxHistory = 5
	}

	if len(sess.Messages) > maxHistory {
		sess.Messages = sess.Messages[len(sess.Messages)-maxHistory:]
	}
}

// CreateCompressLLM 创建压缩专用的 LLM 实例
func CreateCompressLLM(cfg *config.Config) (model.ChatModel, error) {
	// 如果指定了专用模型，使用专用模型；否则使用默认模型
	modelName := cfg.Compress.Model
	if modelName == "" {
		modelName = cfg.Agents.Defaults.Model
	}

	providerCfg := cfg.GetProvider(modelName)
	if providerCfg == nil || providerCfg.APIKey == "" {
		return nil, fmt.Errorf("未找到有效的 API Key")
	}

	apiBase := providerCfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	llm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  providerCfg.APIKey,
		Model:   modelName,
		BaseURL: apiBase,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 ChatModel 失败: %w", err)
	}

	return llm, nil
}