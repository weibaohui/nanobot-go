package eino_adapter

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	agenttools "github.com/weibaohui/nanobot-go/agent/tools"
)

// ToolAdapter adapts nanobot-go's Tool interface to eino's tool.BaseTool interface
type ToolAdapter struct {
	tool agenttools.Tool
}

// NewToolAdapter creates a new adapter that wraps nanobot-go's Tool
func NewToolAdapter(t agenttools.Tool) *ToolAdapter {
	return &ToolAdapter{tool: t}
}

// Info returns the tool information
func (a *ToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params := a.tool.Parameters()

	// Convert parameters to eino's ParamsOneOf format
	paramsOneOf := convertToParamsOneOf(params)

	return &schema.ToolInfo{
		Name:        a.tool.Name(),
		Desc:        a.tool.Description(),
		ParamsOneOf: paramsOneOf,
	}, nil
}

// Run executes the tool with the given arguments
func (a *ToolAdapter) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Parse the JSON arguments
	var params map[string]any
	if argumentsInJSON != "" && argumentsInJSON != "{}" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", err
		}
	}

	// Execute the tool
	return a.tool.Execute(ctx, params)
}

// InvokableRun executes the tool directly with JSON arguments
func (a *ToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return a.Run(ctx, argumentsInJSON, opts...)
}

// convertToParamsOneOf converts nanobot-go parameters to eino's ParamsOneOf format
func convertToParamsOneOf(params map[string]any) *schema.ParamsOneOf {
	if params == nil {
		return nil
	}

	// Extract properties and required fields
	props, ok := params["properties"].(map[string]any)
	if !ok {
		return nil
	}

	required := make([]string, 0)
	if req, ok := params["required"].([]string); ok {
		required = req
	} else if reqAny, ok := params["required"].([]any); ok {
		for _, r := range reqAny {
			if rs, ok := r.(string); ok {
				required = append(required, rs)
			}
		}
	}

	// Build parameter info map
	paramInfos := make(map[string]*schema.ParameterInfo)
	for name, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		info := &schema.ParameterInfo{}

		// Set type
		if typ, ok := propMap["type"].(string); ok {
			info.Type = schema.DataType(typ)
		}

		// Set description
		if desc, ok := propMap["description"].(string); ok {
			info.Desc = desc
		}

		// Check if required
		for _, r := range required {
			if r == name {
				info.Required = true
				break
			}
		}

		// Handle array items
		if items, ok := propMap["items"].(map[string]any); ok {
			itemInfo := &schema.ParameterInfo{}
			if typ, ok := items["type"].(string); ok {
				itemInfo.Type = schema.DataType(typ)
			}
			if desc, ok := items["description"].(string); ok {
				itemInfo.Desc = desc
			}
			info.ElemInfo = itemInfo
		}

		// Handle enum - convert []any to []string
		if enumAny, ok := propMap["enum"].([]any); ok {
			enumStr := make([]string, 0, len(enumAny))
			for _, e := range enumAny {
				if s, ok := e.(string); ok {
					enumStr = append(enumStr, s)
				}
			}
			info.Enum = enumStr
		}

		paramInfos[name] = info
	}

	return schema.NewParamsOneOfByParams(paramInfos)
}

// ConvertTools converts a slice of nanobot-go tools to eino tools
func ConvertTools(tools []agenttools.Tool) []tool.BaseTool {
	result := make([]tool.BaseTool, len(tools))
	for i, t := range tools {
		result[i] = NewToolAdapter(t)
	}
	return result
}

// ConvertToolsFromRegistry converts all tools from a registry to eino tools
func ConvertToolsFromRegistry(registry *agenttools.Registry) []tool.BaseTool {
	definitions := registry.GetDefinitions()
	result := make([]tool.BaseTool, 0, len(definitions))

	for _, def := range definitions {
		name, ok := def["name"].(string)
		if !ok {
			continue
		}

		// Get the tool from the registry
		t := registry.Get(name)
		if t == nil {
			continue
		}

		result = append(result, NewToolAdapter(t))
	}

	return result
}
