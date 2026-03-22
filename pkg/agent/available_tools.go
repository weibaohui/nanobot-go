package agent

// ToolInfo 工具信息
type ToolInfo struct {
	Value       string
	Label       string
	Description string
}

// AvailableTools 返回内置工具列表
func AvailableTools() []ToolInfo {
	return []ToolInfo{
		{Value: "readfile", Label: "readfile - 读取文件", Description: "读取文件内容"},
		{Value: "writefile", Label: "writefile - 写入文件", Description: "写入文件内容"},
		{Value: "editfile", Label: "editfile - 编辑文件", Description: "编辑文件内容"},
		{Value: "listdir", Label: "listdir - 列出目录", Description: "列出目录内容"},
		{Value: "exec", Label: "exec - 执行命令", Description: "执行 Shell 命令"},
		{Value: "websearch", Label: "websearch - 网页搜索", Description: "搜索网页内容"},
		{Value: "webfetch", Label: "webfetch - 网页获取", Description: "获取网页内容"},
		{Value: "cron", Label: "cron - 定时任务", Description: "管理定时任务"},
		{Value: "askuser", Label: "askuser - 询问用户", Description: "向用户提问"},
		{Value: "skill", Label: "skill - 技能调用", Description: "调用技能"},
	}
}
