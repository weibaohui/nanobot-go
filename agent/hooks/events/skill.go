package events

// SkillLookupEvent 查找技能事件
type SkillLookupEvent struct {
	*BaseEvent
	SkillName string `json:"skill_name"` // 技能名称
	Found     bool   `json:"found"`      // 是否找到
	Source    string `json:"source"`     // 来源 (workspace/builtin)
	Path      string `json:"path"`       // 技能文件路径
}

// NewSkillLookupEvent 创建查找技能事件
func NewSkillLookupEvent(traceID, spanID, parentSpanID string, skillName string, found bool, source, path string) *SkillLookupEvent {
	return &SkillLookupEvent{
		BaseEvent: NewBaseEvent(traceID, spanID, parentSpanID, EventSkillLookup),
		SkillName: skillName,
		Found:     found,
		Source:    source,
		Path:      path,
	}
}

// SkillUsedEvent 使用技能事件
type SkillUsedEvent struct {
	*BaseEvent
	SkillName   string `json:"skill_name"`   // 技能名称
	SkillLength int    `json:"skill_length"` // 技能内容长度
}

// NewSkillUsedEvent 创建使用技能事件
func NewSkillUsedEvent(traceID, spanID, parentSpanID string, skillName string, skillLength int) *SkillUsedEvent {
	return &SkillUsedEvent{
		BaseEvent:   NewBaseEvent(traceID, spanID, parentSpanID, EventSkillUsed),
		SkillName:   skillName,
		SkillLength: skillLength,
	}
}
