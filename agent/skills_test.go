package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewSkillsLoader 测试创建技能加载器
func TestNewSkillsLoader(t *testing.T) {
	loader := NewSkillsLoader("/tmp/test_workspace")
	if loader == nil {
		t.Fatal("NewSkillsLoader 返回 nil")
	}

	if loader.workspace != "/tmp/test_workspace" {
		t.Errorf("workspace = %q, 期望 /tmp/test_workspace", loader.workspace)
	}
}

// TestSkillsLoader_ListSkills 测试列出技能
func TestSkillsLoader_ListSkills(t *testing.T) {
	t.Run("列出工作区技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		testSkillDir := filepath.Join(skillsDir, "test-skill")
		os.MkdirAll(testSkillDir, 0755)

		skillContent := "---\ndescription: 测试技能\n---\n技能内容"
		os.WriteFile(filepath.Join(testSkillDir, "SKILL.md"), []byte(skillContent), 0644)

		loader := NewSkillsLoader(tmpDir)
		skills := loader.ListSkills(false)

		if len(skills) != 1 {
			t.Fatalf("ListSkills 返回 %d 个技能, 期望 1", len(skills))
		}

		if skills[0].Name != "test-skill" {
			t.Errorf("技能名称 = %q, 期望 test-skill", skills[0].Name)
		}

		if skills[0].Source != "workspace" {
			t.Errorf("技能来源 = %q, 期望 workspace", skills[0].Source)
		}
	})

	t.Run("过滤不可用技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")

		// 创建一个需要不存在二进制的技能
		skillDir := filepath.Join(skillsDir, "requires-missing-bin")
		os.MkdirAll(skillDir, 0755)
		skillContent := "---\ndescription: 需要缺失二进制\nrequires_bins: nonexistent_binary_xyz\n---\n技能内容"
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644)

		loader := NewSkillsLoader(tmpDir)

		allSkills := loader.ListSkills(false)
		if len(allSkills) != 1 {
			t.Errorf("不过滤时应该返回 1 个技能, 实际返回 %d", len(allSkills))
		}

		filteredSkills := loader.ListSkills(true)
		if len(filteredSkills) != 0 {
			t.Errorf("过滤后应该返回 0 个技能, 实际返回 %d", len(filteredSkills))
		}
	})

	t.Run("空技能目录", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewSkillsLoader(tmpDir)

		skills := loader.ListSkills(false)
		if len(skills) != 0 {
			t.Errorf("ListSkills 返回 %d 个技能, 期望 0", len(skills))
		}
	})
}

// TestSkillsLoader_LoadSkill 测试加载技能内容
func TestSkillsLoader_LoadSkill(t *testing.T) {
	t.Run("加载存在的技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		skillDir := filepath.Join(skillsDir, "my-skill")
		os.MkdirAll(skillDir, 0755)

		content := "---\ndescription: 我的技能\n---\n技能内容"
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

		loader := NewSkillsLoader(tmpDir)
		result := loader.LoadSkill("my-skill")

		if result != content {
			t.Errorf("LoadSkill() = %q, 期望 %q", result, content)
		}
	})

	t.Run("加载不存在的技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewSkillsLoader(tmpDir)

		result := loader.LoadSkill("nonexistent")
		if result != "" {
			t.Errorf("LoadSkill() = %q, 期望空字符串", result)
		}
	})
}

// TestSkillsLoader_GetSkillMetadata 测试获取技能元数据
func TestSkillsLoader_GetSkillMetadata(t *testing.T) {
	t.Run("解析有效元数据", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		skillDir := filepath.Join(skillsDir, "meta-skill")
		os.MkdirAll(skillDir, 0755)

		content := `---
description: 技能描述
requires_bins: git, docker
requires_env: API_KEY
always: true
---
技能内容`
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

		loader := NewSkillsLoader(tmpDir)
		meta := loader.GetSkillMetadata("meta-skill")

		if meta == nil {
			t.Fatal("GetSkillMetadata 返回 nil")
		}

		if meta["description"] != "技能描述" {
			t.Errorf("description = %q, 期望 技能描述", meta["description"])
		}

		if meta["requires_bins"] != "git, docker" {
			t.Errorf("requires_bins = %q, 期望 'git, docker'", meta["requires_bins"])
		}

		if meta["always"] != "true" {
			t.Errorf("always = %q, 期望 true", meta["always"])
		}
	})

	t.Run("无前言的技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		skillDir := filepath.Join(skillsDir, "no-meta")
		os.MkdirAll(skillDir, 0755)

		content := "没有前言的技能内容"
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

		loader := NewSkillsLoader(tmpDir)
		meta := loader.GetSkillMetadata("no-meta")

		if meta != nil {
			t.Errorf("GetSkillMetadata 应该返回 nil, 实际返回 %v", meta)
		}
	})

	t.Run("不存在的技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewSkillsLoader(tmpDir)

		meta := loader.GetSkillMetadata("nonexistent")
		if meta != nil {
			t.Errorf("GetSkillMetadata 应该返回 nil, 实际返回 %v", meta)
		}
	})
}

// TestSkillsLoader_BuildSkillsSummary 测试构建技能摘要
func TestSkillsLoader_BuildSkillsSummary(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	skillDir := filepath.Join(skillsDir, "summary-skill")
	os.MkdirAll(skillDir, 0755)

	content := "---\ndescription: 摘要测试技能\n---\n技能内容"
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

	loader := NewSkillsLoader(tmpDir)
	summary := loader.BuildSkillsSummary()

	if summary == "" {
		t.Fatal("BuildSkillsSummary 返回空字符串")
	}

	if summary[:8] != "<skills>" {
		t.Errorf("摘要应该以 <skills> 开头")
	}
}

// TestSkillsLoader_LoadSkillsForContext 测试加载技能用于上下文
func TestSkillsLoader_LoadSkillsForContext(t *testing.T) {
	t.Run("加载多个技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")

		for _, name := range []string{"skill1", "skill2"} {
			skillDir := filepath.Join(skillsDir, name)
			os.MkdirAll(skillDir, 0755)
			content := "---\ndescription: " + name + "\n---\n" + name + " 内容"
			os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
		}

		loader := NewSkillsLoader(tmpDir)
		result := loader.LoadSkillsForContext([]string{"skill1", "skill2"})

		if result == "" {
			t.Fatal("LoadSkillsForContext 返回空字符串")
		}

		if len(result) < 20 {
			t.Errorf("内容太短: %q", result)
		}
	})

	t.Run("加载不存在的技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewSkillsLoader(tmpDir)

		result := loader.LoadSkillsForContext([]string{"nonexistent"})
		if result != "" {
			t.Errorf("LoadSkillsForContext = %q, 期望空字符串", result)
		}
	})
}

// TestSkillsLoader_GetAlwaysSkills 测试获取始终加载的技能
func TestSkillsLoader_GetAlwaysSkills(t *testing.T) {
	t.Run("有 always 技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")

		// 创建 always=true 的技能
		alwaysDir := filepath.Join(skillsDir, "always-skill")
		os.MkdirAll(alwaysDir, 0755)
		alwaysContent := "---\ndescription: 始终加载\nalways: true\n---\n内容"
		os.WriteFile(filepath.Join(alwaysDir, "SKILL.md"), []byte(alwaysContent), 0644)

		// 创建 always=false 的技能
		normalDir := filepath.Join(skillsDir, "normal-skill")
		os.MkdirAll(normalDir, 0755)
		normalContent := "---\ndescription: 普通技能\nalways: false\n---\n内容"
		os.WriteFile(filepath.Join(normalDir, "SKILL.md"), []byte(normalContent), 0644)

		loader := NewSkillsLoader(tmpDir)
		alwaysSkills := loader.GetAlwaysSkills()

		if len(alwaysSkills) != 1 {
			t.Errorf("GetAlwaysSkills 返回 %d 个技能, 期望 1", len(alwaysSkills))
		}

		if len(alwaysSkills) > 0 && alwaysSkills[0] != "always-skill" {
			t.Errorf("技能名称 = %q, 期望 always-skill", alwaysSkills[0])
		}
	})

	t.Run("无 always 技能", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := NewSkillsLoader(tmpDir)

		alwaysSkills := loader.GetAlwaysSkills()
		if len(alwaysSkills) != 0 {
			t.Errorf("GetAlwaysSkills 返回 %d 个技能, 期望 0", len(alwaysSkills))
		}
	})
}

// TestStripFrontmatter 测试移除 YAML 前言
func TestStripFrontmatter(t *testing.T) {
	loader := &SkillsLoader{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "有前言",
			input:    "---\nkey: value\n---\n内容",
			expected: "内容",
		},
		{
			name:     "无前言",
			input:    "直接内容",
			expected: "直接内容",
		},
		{
			name:     "多行前言",
			input:    "---\nkey1: value1\nkey2: value2\n---\n实际内容",
			expected: "实际内容",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.stripFrontmatter(tt.input)
			if result != tt.expected {
				t.Errorf("stripFrontmatter() = %q, 期望 %q", result, tt.expected)
			}
		})
	}
}

// TestBoolStr 测试布尔值转字符串
func TestBoolStr(t *testing.T) {
	if boolStr(true) != "true" {
		t.Error("boolStr(true) 应该返回 'true'")
	}

	if boolStr(false) != "false" {
		t.Error("boolStr(false) 应该返回 'false'")
	}
}

// TestEscapeXML 测试 XML 转义
func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"a<b", "a&lt;b"},
		{"a>b", "a&gt;b"},
		{"a&b", "a&amp;b"},
		{"<tag>", "&lt;tag&gt;"},
	}

	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q) = %q, 期望 %q", tt.input, result, tt.expected)
		}
	}
}

// TestIsDir 测试目录判断
func TestIsDir(t *testing.T) {
	t.Run("存在的目录", func(t *testing.T) {
		tmpDir := t.TempDir()
		if !isDir(tmpDir) {
			t.Errorf("isDir(%q) 应该返回 true", tmpDir)
		}
	})

	t.Run("存在的文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(filePath, []byte("test"), 0644)

		if isDir(filePath) {
			t.Errorf("isDir(%q) 应该返回 false", filePath)
		}
	})

	t.Run("不存在的路径", func(t *testing.T) {
		if isDir("/nonexistent/path") {
			t.Error("isDir 对不存在的路径应该返回 false")
		}
	})
}
