package provider

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/pkg/agent/hooks/events"
	"go.uber.org/zap"
)

// interceptToolCall 拦截工具调用，如果工具不存在则转换为技能调用
func (a *ChatModelAdapter) interceptToolCall(toolName string, argumentsJSON string) (string, string, error) {
	if toolName == "use_skill" {
		var args map[string]any
		if err := json.Unmarshal([]byte(argumentsJSON), &args); err == nil {
			if skillName, ok := args["skill_name"].(string); ok {
				skillContent := ""
				if a.skillLoader != nil {
					skillContent = a.skillLoader(skillName)
				}
				a.triggerHook(events.EventSkillCall, map[string]any{
					"skill_name":   skillName,
					"skill_length": len(skillContent),
				})
			}
		}
		return toolName, argumentsJSON, nil
	}

	if a.isRegisteredTool(toolName) {
		a.logger.Info("工具已注册，不拦截", zap.String("名称", toolName))
		return toolName, argumentsJSON, nil
	}

	if a.isKnownSkill(toolName) {
		a.logger.Info("isKnownSkill 工具转换为技能调用", zap.String("名称", toolName))
		var originalArgs map[string]any
		if err := json.Unmarshal([]byte(argumentsJSON), &originalArgs); err != nil {
			originalArgs = make(map[string]any)
		}

		skillParams := map[string]any{
			"skill_name": toolName,
			"action":     originalArgs["action"],
		}

		filteredParams := make(map[string]any)
		for k, v := range originalArgs {
			if k != "action" {
				filteredParams[k] = v
			}
		}
		if len(filteredParams) > 0 {
			skillParams["params"] = filteredParams
		}

		newArgsJSON, err := json.Marshal(skillParams)
		if err != nil {
			return toolName, argumentsJSON, err
		}

		return "use_skill", string(newArgsJSON), nil
	}

	a.logger.Info("既不是工具也不是技能，保持原样", zap.String("名称", toolName))
	return toolName, argumentsJSON, nil
}

// interceptToolCalls 拦截并转换工具调用
func (a *ChatModelAdapter) interceptToolCalls(msg *schema.Message) {
	if len(msg.ToolCalls) == 0 {
		return
	}

	for i, tc := range msg.ToolCalls {
		a.triggerHook(events.EventToolCall, map[string]any{
			"tool_name": tc.Function.Name,
			"arguments": tc.Function.Arguments,
		})

		newName, newArgs, err := a.interceptToolCall(tc.Function.Name, tc.Function.Arguments)

		if err != nil {
			continue
		}
		if newName != tc.Function.Name {
			a.triggerHook(events.EventToolIntercepted, map[string]any{
				"original_name": tc.Function.Name,
				"new_name":      newName,
				"new_args":      newArgs,
			})
			msg.ToolCalls[i].Function.Name = newName
			msg.ToolCalls[i].Function.Arguments = newArgs
		}
	}
}
