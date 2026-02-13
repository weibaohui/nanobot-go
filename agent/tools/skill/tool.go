package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/nanobot-go/agent/tools/common"
)

// SkillLoaderFunc 技能加载函数类型
type SkillLoaderFunc func(name string) string

// DynamicTool 动态技能工具（每个技能一个实例）
type DynamicTool struct {
	name        string
	description string
	loadSkill   SkillLoaderFunc
}

// NewDynamicTool 创建动态技能工具
func NewDynamicTool(name, description string, loader SkillLoaderFunc) *DynamicTool {
	return &DynamicTool{
		name:        name,
		description: description,
		loadSkill:   loader,
	}
}

// Name 返回工具名称
func (t *DynamicTool) Name() string {
	return t.name
}

// Info 返回工具信息
func (t *DynamicTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	desc := t.description
	if desc == "" {
		desc = fmt.Sprintf("使用 %s 技能执行相关操作", t.name)
	}

	return &schema.ToolInfo{
		Name: t.name,
		Desc: desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.DataType("string"),
				Desc:     "要执行的操作（可选），由技能定义",
				Required: false,
			},
			"params": {
				Type:     schema.DataType("object"),
				Desc:     "操作参数（可选）",
				Required: false,
			},
		}),
	}, nil
}

// Run 执行工具逻辑
func (t *DynamicTool) Run(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	if err := common.DecodeArgs(argumentsInJSON, &args); err != nil {
		return "", err
	}

	return t.executeSkill(args.Action, args.Params)
}

// InvokableRun 可直接调用的执行入口
func (t *DynamicTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.Run(ctx, argumentsInJSON, opts...)
}

// executeSkill 执行技能
func (t *DynamicTool) executeSkill(action string, params map[string]any) (string, error) {
	if t.loadSkill == nil {
		return "", fmt.Errorf("技能加载器未配置")
	}

	// 加载技能内容
	content := t.loadSkill(t.name)
	if content == "" {
		return "", fmt.Errorf("技能 '%s' 不存在", t.name)
	}

	// 构建结果
	var result strings.Builder
	fmt.Fprintf(&result, "# 技能: %s\n\n", t.name)
	result.WriteString(content)

	// 如果有指定 action，提示如何使用
	if action != "" {
		fmt.Fprintf(&result, "\n\n---\n\n## 执行操作: %s\n", action)
		if len(params) > 0 {
			paramsJSON, _ := json.MarshalIndent(params, "", "  ")
			fmt.Fprintf(&result, "\n参数:\n```json\n%s\n```\n", string(paramsJSON))
		}
		result.WriteString("\n请根据技能说明执行此操作。")
	}

	return result.String(), nil
}

// SkillInfo 技能信息（用于注册）
type SkillInfo struct {
	Name        string
	Description string
}

// Registry 技能工具注册器
type Registry struct {
	loader    SkillLoaderFunc
	skills    []SkillInfo
	tools     map[string]*DynamicTool
}

// NewRegistry 创建技能注册器
func NewRegistry(loader SkillLoaderFunc) *Registry {
	return &Registry{
		loader: loader,
		tools:  make(map[string]*DynamicTool),
	}
}

// RegisterSkill 注册单个技能
func (r *Registry) RegisterSkill(name, description string) *DynamicTool {
	tool := NewDynamicTool(name, description, r.loader)
	r.tools[name] = tool
	r.skills = append(r.skills, SkillInfo{Name: name, Description: description})
	return tool
}

// RegisterSkills 批量注册技能
func (r *Registry) RegisterSkills(skills []SkillInfo) {
	for _, sk := range skills {
		r.RegisterSkill(sk.Name, sk.Description)
	}
}

// GetTool 获取技能工具
func (r *Registry) GetTool(name string) *DynamicTool {
	return r.tools[name]
}

// GetAllTools 获取所有技能工具
func (r *Registry) GetAllTools() []*DynamicTool {
	var tools []*DynamicTool
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetSkillNames 获取所有技能名称
func (r *Registry) GetSkillNames() []string {
	var names []string
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// HasSkill 检查技能是否存在
func (r *Registry) HasSkill(name string) bool {
	_, ok := r.tools[name]
	return ok
}
