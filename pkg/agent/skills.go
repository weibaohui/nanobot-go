package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SkillsLoader 技能加载器
type SkillsLoader struct {
	workspace       string
	workspaceSkills string
	builtinSkills   string
}

// NewSkillsLoader 创建技能加载器
func NewSkillsLoader(workspace string) *SkillsLoader {
	return &SkillsLoader{
		workspace:       workspace,
		workspaceSkills: filepath.Join(workspace, "skills"),
		builtinSkills:   detectBuiltinSkillsDir(),
	}
}

// SkillInfo 技能信息
type SkillInfo struct {
	Name   string
	Path   string
	Source string // "workspace" 或 "builtin"
}

// ListSkills 列出所有可用技能
func (s *SkillsLoader) ListSkills(filterUnavailable bool) []SkillInfo {
	var skills []SkillInfo

	// 工作区技能（最高优先级）
	if dir, err := os.ReadDir(s.workspaceSkills); err == nil {
		for _, entry := range dir {
			if entry.IsDir() {
				skillFile := filepath.Join(s.workspaceSkills, entry.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					skills = append(skills, SkillInfo{
						Name:   entry.Name(),
						Path:   skillFile,
						Source: "workspace",
					})
				}
			}
		}
	}

	// 内置技能
	if s.builtinSkills != "" {
		if dir, err := os.ReadDir(s.builtinSkills); err == nil {
			for _, entry := range dir {
				if entry.IsDir() {
					skillFile := filepath.Join(s.builtinSkills, entry.Name(), "SKILL.md")
					if _, err := os.Stat(skillFile); err == nil {
						// 检查是否已存在
						exists := false
						for _, sk := range skills {
							if sk.Name == entry.Name() {
								exists = true
								break
							}
						}
						if !exists {
							skills = append(skills, SkillInfo{
								Name:   entry.Name(),
								Path:   skillFile,
								Source: "builtin",
							})
						}
					}
				}
			}
		}
	}

	// 过滤不可用技能
	if filterUnavailable {
		var filtered []SkillInfo
		for _, sk := range skills {
			if s.checkRequirements(sk.Name) {
				filtered = append(filtered, sk)
			}
		}
		return filtered
	}

	return skills
}

// LoadSkill 加载技能内容
func (s *SkillsLoader) LoadSkill(name string) string {
	// 先检查工作区
	workspaceSkill := filepath.Join(s.workspaceSkills, name, "SKILL.md")
	if data, err := os.ReadFile(workspaceSkill); err == nil {
		return string(data)
	}

	// 检查内置
	if s.builtinSkills != "" {
		builtinSkill := filepath.Join(s.builtinSkills, name, "SKILL.md")
		if data, err := os.ReadFile(builtinSkill); err == nil {
			return string(data)
		}
	}

	return ""
}

// LoadSkillsForContext 加载指定技能用于上下文
func (s *SkillsLoader) LoadSkillsForContext(skillNames []string) string {
	var parts []string
	for _, name := range skillNames {
		content := s.LoadSkill(name)
		if content != "" {
			content = s.stripFrontmatter(content)
			parts = append(parts, "### 技能: "+name+"\n\n"+content)
		}
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// BuildSkillsSummary 构建技能摘要
func (s *SkillsLoader) BuildSkillsSummary() string {
	allSkills := s.ListSkills(false)
	if len(allSkills) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "<skills>")
	for _, sk := range allSkills {
		available := s.checkRequirements(sk.Name)
		lines = append(lines, "  <skill available=\""+boolStr(available)+"\">")
		lines = append(lines, "    <name>"+escapeXML(sk.Name)+"</name>")
		lines = append(lines, "    <description>"+escapeXML(s.getSkillDescription(sk.Name))+"</description>")
		lines = append(lines, "    <location>"+sk.Path+"</location>")

		if !available {
			missing := s.getMissingRequirements(sk.Name)
			if missing != "" {
				lines = append(lines, "    <requires>"+escapeXML(missing)+"</requires>")
			}
		}
		lines = append(lines, "  </skill>")
	}
	lines = append(lines, "</skills>")

	return strings.Join(lines, "\n")
}

// getSkillDescription 获取技能描述
func (s *SkillsLoader) getSkillDescription(name string) string {
	meta := s.GetSkillMetadata(name)
	if desc, ok := meta["description"]; ok {
		return desc
	}
	return name
}

// stripFrontmatter 移除 YAML 前言
func (s *SkillsLoader) stripFrontmatter(content string) string {
	if strings.HasPrefix(content, "---") {
		re := regexp.MustCompile(`(?s)^---\n.*?\n---\n`)
		content = re.ReplaceAllString(content, "")
	}
	return strings.TrimSpace(content)
}

// checkRequirements 检查技能需求是否满足
func (s *SkillsLoader) checkRequirements(name string) bool {
	meta := s.GetSkillMetadata(name)
	if meta == nil {
		return true
	}

	// 检查二进制文件需求
	if bins, ok := meta["requires_bins"]; ok {
		for _, bin := range strings.Split(bins, ",") {
			bin = strings.TrimSpace(bin)
			if bin != "" && !s.hasBinary(bin) {
				return false
			}
		}
	}

	// 检查环境变量需求
	if envs, ok := meta["requires_env"]; ok {
		for _, env := range strings.Split(envs, ",") {
			env = strings.TrimSpace(env)
			if env != "" && os.Getenv(env) == "" {
				return false
			}
		}
	}

	return true
}

// getMissingRequirements 获取缺失的需求
func (s *SkillsLoader) getMissingRequirements(name string) string {
	meta := s.GetSkillMetadata(name)
	if meta == nil {
		return ""
	}

	var missing []string

	if bins, ok := meta["requires_bins"]; ok {
		for _, bin := range strings.Split(bins, ",") {
			bin = strings.TrimSpace(bin)
			if bin != "" && !s.hasBinary(bin) {
				missing = append(missing, "CLI: "+bin)
			}
		}
	}

	if envs, ok := meta["requires_env"]; ok {
		for _, env := range strings.Split(envs, ",") {
			env = strings.TrimSpace(env)
			if env != "" && os.Getenv(env) == "" {
				missing = append(missing, "ENV: "+env)
			}
		}
	}

	return strings.Join(missing, ", ")
}

// hasBinary 检查二进制文件是否存在
func (s *SkillsLoader) hasBinary(name string) bool {
	// 检查 PATH
	path := os.Getenv("PATH")
	for _, dir := range strings.Split(path, string(os.PathListSeparator)) {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// GetSkillMetadata 获取技能元数据
func (s *SkillsLoader) GetSkillMetadata(name string) map[string]string {
	content := s.LoadSkill(name)
	if content == "" {
		return nil
	}

	if !strings.HasPrefix(content, "---") {
		return nil
	}

	// 解析 YAML 前言
	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil
	}

	metadata := make(map[string]string)
	for _, line := range strings.Split(match[1], "\n") {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"'")
			metadata[key] = value
		}
	}

	return metadata
}

// GetAlwaysSkills 获取标记为 always=true 的技能
func (s *SkillsLoader) GetAlwaysSkills() []string {
	var result []string
	for _, sk := range s.ListSkills(true) {
		meta := s.GetSkillMetadata(sk.Name)
		if always, ok := meta["always"]; ok && always == "true" {
			result = append(result, sk.Name)
		}
	}
	return result
}

// 辅助函数
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// detectBuiltinSkillsDir 解析内置技能目录
func detectBuiltinSkillsDir() string {
	if env := strings.TrimSpace(os.Getenv("NANOBOT_SKILLS_DIR")); env != "" {
		return env
	}
	if dir := findSkillsDirFromCWD(); dir != "" {
		return dir
	}
	if dir := findSkillsDirFromExecutable(); dir != "" {
		return dir
	}
	return ""
}

// findSkillsDirFromCWD 从当前工作目录查找 skills
func findSkillsDirFromCWD() string {
	if cwd, err := os.Getwd(); err == nil {
		dir := filepath.Join(cwd, "skills")
		if isDir(dir) {
			return dir
		}
	}
	return ""
}

// findSkillsDirFromExecutable 从可执行文件目录查找 skills
func findSkillsDirFromExecutable() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "skills")
		if isDir(dir) {
			return dir
		}
	}
	return ""
}

// isDir 判断路径是否为目录
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
