package skill

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// mockSkillLoader 模拟技能加载器
func mockSkillLoader(name string) string {
	skills := map[string]string{
		"github": "# GitHub 技能\n\n这是一个 GitHub 操作技能。\n\n## 支持的操作\n- workflow: 触发工作流\n- pr: 创建 PR",
		"docker": "# Docker 技能\n\n这是一个 Docker 操作技能。",
	}
	return skills[name]
}

// TestDynamicTool_Name 测试工具名称
func TestDynamicTool_Name(t *testing.T) {
	tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)

	if tool.Name() != "github" {
		t.Errorf("Name() = %q, 期望 github", tool.Name())
	}
}

// TestDynamicTool_Info 测试工具信息
func TestDynamicTool_Info(t *testing.T) {
	t.Run("有描述", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)
		ctx := context.Background()

		info, err := tool.Info(ctx)
		if err != nil {
			t.Errorf("Info() 返回错误: %v", err)
		}

		if info.Name != "github" {
			t.Errorf("Info.Name = %q, 期望 github", info.Name)
		}

		if info.Desc != "GitHub 操作技能" {
			t.Errorf("Info.Desc = %q, 期望 GitHub 操作技能", info.Desc)
		}
	})

	t.Run("无描述使用默认值", func(t *testing.T) {
		tool := NewDynamicTool("github", "", mockSkillLoader)
		ctx := context.Background()

		info, err := tool.Info(ctx)
		if err != nil {
			t.Errorf("Info() 返回错误: %v", err)
		}

		if info.Desc == "" {
			t.Error("Info.Desc 不应该为空")
		}
	})
}

// TestDynamicTool_Run 测试执行工具
func TestDynamicTool_Run(t *testing.T) {
	t.Run("正常执行", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)
		ctx := context.Background()

		result, err := tool.Run(ctx, `{}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("带 action 和 params", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"action": "workflow", "params": {"repo": "owner/repo"}}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("技能不存在", func(t *testing.T) {
		tool := NewDynamicTool("nonexistent", "不存在的技能", mockSkillLoader)
		ctx := context.Background()

		_, err := tool.Run(ctx, `{}`)
		if err == nil {
			t.Error("Run() 应该返回错误")
		}
	})

	t.Run("无加载器", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", nil)
		ctx := context.Background()

		_, err := tool.Run(ctx, `{}`)
		if err == nil {
			t.Error("Run() 应该返回错误")
		}
	})
}

// TestRegistry_NewRegistry 测试创建注册器
func TestRegistry_NewRegistry(t *testing.T) {
	registry := NewRegistry(mockSkillLoader)

	if registry == nil {
		t.Fatal("NewRegistry 返回 nil")
	}

	if registry.loader == nil {
		t.Error("loader 不应该为 nil")
	}
}

// TestRegistry_RegisterSkill 测试注册单个技能
func TestRegistry_RegisterSkill(t *testing.T) {
	registry := NewRegistry(mockSkillLoader)

	tool := registry.RegisterSkill("github", "GitHub 操作技能")

	if tool == nil {
		t.Fatal("RegisterSkill 返回 nil")
	}

	if tool.Name() != "github" {
		t.Errorf("tool.Name() = %q, 期望 github", tool.Name())
	}

	if !registry.HasSkill("github") {
		t.Error("HasSkill 应该返回 true")
	}
}

// TestRegistry_RegisterSkills 测试批量注册技能
func TestRegistry_RegisterSkills(t *testing.T) {
	registry := NewRegistry(mockSkillLoader)

	skills := []SkillInfo{
		{Name: "github", Description: "GitHub 操作技能"},
		{Name: "docker", Description: "Docker 操作技能"},
	}
	registry.RegisterSkills(skills)

	if !registry.HasSkill("github") {
		t.Error("HasSkill(github) 应该返回 true")
	}

	if !registry.HasSkill("docker") {
		t.Error("HasSkill(docker) 应该返回 true")
	}
}

// TestRegistry_GetTool 测试获取工具
func TestRegistry_GetTool(t *testing.T) {
	registry := NewRegistry(mockSkillLoader)
	registry.RegisterSkill("github", "GitHub 操作技能")

	tool := registry.GetTool("github")
	if tool == nil {
		t.Fatal("GetTool 返回 nil")
	}

	if tool.Name() != "github" {
		t.Errorf("tool.Name() = %q, 期望 github", tool.Name())
	}
}

// TestRegistry_GetAllTools 测试获取所有工具
func TestRegistry_GetAllTools(t *testing.T) {
	registry := NewRegistry(mockSkillLoader)
	registry.RegisterSkill("github", "GitHub 操作技能")
	registry.RegisterSkill("docker", "Docker 操作技能")

	tools := registry.GetAllTools()
	if len(tools) != 2 {
		t.Errorf("GetAllTools 返回 %d 个工具, 期望 2", len(tools))
	}
}

// TestRegistry_GetSkillNames 测试获取技能名称列表
func TestRegistry_GetSkillNames(t *testing.T) {
	registry := NewRegistry(mockSkillLoader)
	registry.RegisterSkill("github", "GitHub 操作技能")
	registry.RegisterSkill("docker", "Docker 操作技能")

	names := registry.GetSkillNames()
	if len(names) != 2 {
		t.Errorf("GetSkillNames 返回 %d 个名称, 期望 2", len(names))
	}
}

// TestRegistry_HasSkill 测试检查技能是否存在
func TestRegistry_HasSkill(t *testing.T) {
	registry := NewRegistry(mockSkillLoader)

	if registry.HasSkill("github") {
		t.Error("HasSkill 应该返回 false")
	}

	registry.RegisterSkill("github", "GitHub 操作技能")

	if !registry.HasSkill("github") {
		t.Error("HasSkill 应该返回 true")
	}
}

// TestGenericSkillTool_Name 测试工具名称
func TestGenericSkillTool_Name(t *testing.T) {
	tool := NewGenericSkillTool(mockSkillLoader)

	if tool.Name() != "use_skill" {
		t.Errorf("Name() = %q, 期望 use_skill", tool.Name())
	}
}

// TestGenericSkillTool_Info 测试工具信息
func TestGenericSkillTool_Info(t *testing.T) {
	tool := NewGenericSkillTool(mockSkillLoader)
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.Name != "use_skill" {
		t.Errorf("Info.Name = %q, 期望 use_skill", info.Name)
	}
}

// TestGenericSkillTool_Run 测试执行工具
func TestGenericSkillTool_Run(t *testing.T) {
	t.Run("正常执行", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"skill_name": "github"}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("带 action 和 params", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)
		ctx := context.Background()

		result, err := tool.Run(ctx, `{"skill_name": "github", "action": "workflow", "params": {"repo": "owner/repo"}}`)
		if err != nil {
			t.Errorf("Run() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("Run() 不应该返回空结果")
		}
	})

	t.Run("空技能名称", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)
		ctx := context.Background()

		_, err := tool.Run(ctx, `{"skill_name": ""}`)
		if err == nil {
			t.Error("Run() 应该返回错误")
		}
	})

	t.Run("技能不存在", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)
		ctx := context.Background()

		_, err := tool.Run(ctx, `{"skill_name": "nonexistent"}`)
		if err == nil {
			t.Error("Run() 应该返回错误")
		}
	})

	t.Run("无加载器", func(t *testing.T) {
		tool := NewGenericSkillTool(nil)
		ctx := context.Background()

		_, err := tool.Run(ctx, `{"skill_name": "github"}`)
		if err == nil {
			t.Error("Run() 应该返回错误")
		}
	})
}

// TestInvokableRun 测试 InvokableRun 方法
func TestInvokableRun(t *testing.T) {
	t.Run("DynamicTool", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)
		ctx := context.Background()

		result, err := tool.InvokableRun(ctx, `{}`)
		if err != nil {
			t.Errorf("InvokableRun() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("InvokableRun() 不应该返回空结果")
		}
	})

	t.Run("GenericSkillTool", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)
		ctx := context.Background()

		result, err := tool.InvokableRun(ctx, `{"skill_name": "github"}`)
		if err != nil {
			t.Errorf("InvokableRun() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("InvokableRun() 不应该返回空结果")
		}
	})
}

// TestSkillInfo 测试技能信息结构
func TestSkillInfo(t *testing.T) {
	info := SkillInfo{
		Name:        "github",
		Description: "GitHub 操作技能",
	}

	if info.Name != "github" {
		t.Errorf("Name = %q, 期望 github", info.Name)
	}

	if info.Description != "GitHub 操作技能" {
		t.Errorf("Description = %q, 期望 GitHub 操作技能", info.Description)
	}
}

// TestSkillLoaderFunc 测试技能加载函数类型
func TestSkillLoaderFunc(t *testing.T) {
	var loader SkillLoaderFunc = mockSkillLoader

	result := loader("github")
	if result == "" {
		t.Error("loader 应该返回非空内容")
	}

	result = loader("nonexistent")
	if result != "" {
		t.Error("loader 应该返回空内容")
	}
}

// TestDynamicTool_executeSkill 测试执行技能内部方法
func TestDynamicTool_executeSkill(t *testing.T) {
	t.Run("无参数", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)

		result, err := tool.executeSkill("", nil)
		if err != nil {
			t.Errorf("executeSkill() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("executeSkill() 不应该返回空结果")
		}
	})

	t.Run("带参数", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)

		params := map[string]any{"repo": "owner/repo"}
		result, err := tool.executeSkill("workflow", params)
		if err != nil {
			t.Errorf("executeSkill() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("executeSkill() 不应该返回空结果")
		}
	})
}

// TestGenericSkillTool_executeSkill 测试执行技能内部方法
func TestGenericSkillTool_executeSkill(t *testing.T) {
	t.Run("无参数", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)

		result, err := tool.executeSkill("github", "", nil)
		if err != nil {
			t.Errorf("executeSkill() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("executeSkill() 不应该返回空结果")
		}
	})

	t.Run("带参数", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)

		params := map[string]any{"repo": "owner/repo"}
		result, err := tool.executeSkill("github", "workflow", params)
		if err != nil {
			t.Errorf("executeSkill() 返回错误: %v", err)
		}

		if result == "" {
			t.Error("executeSkill() 不应该返回空结果")
		}
	})
}

// TestSchemaParamsOneOf 测试 schema 参数类型
func TestSchemaParamsOneOf(t *testing.T) {
	tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)
	ctx := context.Background()

	info, err := tool.Info(ctx)
	if err != nil {
		t.Errorf("Info() 返回错误: %v", err)
	}

	if info.ParamsOneOf == nil {
		t.Error("ParamsOneOf 不应该为 nil")
	}
}

// TestToolInfoInterface 测试工具信息接口
func TestToolInfoInterface(t *testing.T) {
	t.Run("DynamicTool", func(t *testing.T) {
		tool := NewDynamicTool("github", "GitHub 操作技能", mockSkillLoader)
		ctx := context.Background()

		info, err := tool.Info(ctx)
		if err != nil {
			t.Errorf("Info() 返回错误: %v", err)
		}

		var _ *schema.ToolInfo = info
	})

	t.Run("GenericSkillTool", func(t *testing.T) {
		tool := NewGenericSkillTool(mockSkillLoader)
		ctx := context.Background()

		info, err := tool.Info(ctx)
		if err != nil {
			t.Errorf("Info() 返回错误: %v", err)
		}

		var _ *schema.ToolInfo = info
	})
}
