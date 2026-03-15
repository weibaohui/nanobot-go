package tools

// NamedTool 命名工具接口
type NamedTool interface {
	Name() string
}

// ContextSetter 可设置上下文的工具接口
type ContextSetter interface {
	SetContext(channel, chatID string)
}
