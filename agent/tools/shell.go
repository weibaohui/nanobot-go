package tools

import (
	"context"
	"fmt"
	"os/exec"
)

// ExecTool 执行命令工具
type ExecTool struct {
	Timeout             int
	WorkingDir          string
	RestrictToWorkspace bool
}

func (t *ExecTool) Name() string        { return "exec" }
func (t *ExecTool) Description() string { return "执行 shell 命令" }
func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string", "description": "要执行的命令"},
		},
		"required": []string{"command"},
	}
}
func (t *ExecTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	command, _ := params["command"].(string)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = t.WorkingDir
	output, err := cmd.CombinedOutput()
	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\n错误: %s", err)
	}
	if len(result) > 10000 {
		result = result[:10000] + "...(已截断)"
	}
	return result, nil
}
func (t *ExecTool) ToSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}
