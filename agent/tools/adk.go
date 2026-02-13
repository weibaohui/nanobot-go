package tools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func toolInfoFromTool(t Tool) (*schema.ToolInfo, error) {
	params := t.Parameters()
	paramsOneOf := convertToParamsOneOf(params)
	return &schema.ToolInfo{
		Name:        t.Name(),
		Desc:        t.Description(),
		ParamsOneOf: paramsOneOf,
	}, nil
}

func runToolFromJSON(ctx context.Context, t Tool, argumentsInJSON string) (string, error) {
	var params map[string]any
	if argumentsInJSON != "" && argumentsInJSON != "{}" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", err
		}
	}
	return t.Execute(ctx, params)
}

func convertToParamsOneOf(params map[string]any) *schema.ParamsOneOf {
	if params == nil {
		return nil
	}

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

	paramInfos := make(map[string]*schema.ParameterInfo)
	for name, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		info := &schema.ParameterInfo{}
		if typ, ok := propMap["type"].(string); ok {
			info.Type = schema.DataType(typ)
		}
		if desc, ok := propMap["description"].(string); ok {
			info.Desc = desc
		}
		for _, r := range required {
			if r == name {
				info.Required = true
				break
			}
		}
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

func (t *ReadFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *ReadFileTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *ReadFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *WriteFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *WriteFileTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *WriteFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *EditFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *EditFileTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *EditFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *ListDirTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *ListDirTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *ListDirTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *ExecTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *ExecTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *ExecTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *WebSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *WebSearchTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *WebSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *WebFetchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *WebFetchTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *WebFetchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *SpawnTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *SpawnTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *SpawnTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *CronTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *CronTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *CronTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

func (t *MessageTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolInfoFromTool(t)
}

func (t *MessageTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return runToolFromJSON(ctx, t, argumentsInJSON)
}

func (t *MessageTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}
