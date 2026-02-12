package providers

// ProviderSpec 提供商规格定义
type ProviderSpec struct {
	Name              string
	Keywords          []string
	EnvKey            string
	DisplayName       string
	LitellmPrefix     string
	SkipPrefixes      []string
	EnvExtras         []EnvExtra
	IsGateway         bool
	IsLocal           bool
	DetectByKeyPrefix string
	DetectByBaseKeyword string
	DefaultAPIBase    string
	StripModelPrefix  bool
	ModelOverrides    []ModelOverride
}

// EnvExtra 环境变量额外配置
type EnvExtra struct {
	Name  string
	Value string
}

// ModelOverride 模型参数覆盖
type ModelOverride struct {
	Pattern   string
	Overrides map[string]any
}

// Label 返回提供商显示标签
func (s *ProviderSpec) Label() string {
	if s.DisplayName != "" {
		return s.DisplayName
	}
	return s.Name
}

// Providers 提供商列表（按优先级排序）
var Providers = []ProviderSpec{
	// === 网关（通过 api_key/api_base 检测，而非模型名称）===

	{
		Name:              "openrouter",
		Keywords:          []string{"openrouter"},
		EnvKey:            "OPENROUTER_API_KEY",
		DisplayName:       "OpenRouter",
		LitellmPrefix:     "openrouter",
		IsGateway:         true,
		DetectByKeyPrefix: "sk-or-",
		DetectByBaseKeyword: "openrouter",
		DefaultAPIBase:    "https://openrouter.ai/api/v1",
	},

	{
		Name:              "aihubmix",
		Keywords:          []string{"aihubmix"},
		EnvKey:            "OPENAI_API_KEY",
		DisplayName:       "AiHubMix",
		LitellmPrefix:     "openai",
		IsGateway:         true,
		DetectByBaseKeyword: "aihubmix",
		DefaultAPIBase:    "https://aihubmix.com/v1",
		StripModelPrefix:  true,
	},

	// === 标准提供商（通过模型名称关键词匹配）===

	{
		Name:          "anthropic",
		Keywords:      []string{"anthropic", "claude"},
		EnvKey:        "ANTHROPIC_API_KEY",
		DisplayName:   "Anthropic",
	},

	{
		Name:        "openai",
		Keywords:    []string{"openai", "gpt"},
		EnvKey:      "OPENAI_API_KEY",
		DisplayName: "OpenAI",
	},

	{
		Name:          "deepseek",
		Keywords:      []string{"deepseek"},
		EnvKey:        "DEEPSEEK_API_KEY",
		DisplayName:   "DeepSeek",
		LitellmPrefix: "deepseek",
		SkipPrefixes:  []string{"deepseek/"},
	},

	{
		Name:          "gemini",
		Keywords:      []string{"gemini"},
		EnvKey:        "GEMINI_API_KEY",
		DisplayName:   "Gemini",
		LitellmPrefix: "gemini",
		SkipPrefixes:  []string{"gemini/"},
	},

	{
		Name:          "zhipu",
		Keywords:      []string{"zhipu", "glm", "zai"},
		EnvKey:        "ZAI_API_KEY",
		DisplayName:   "Zhipu AI",
		LitellmPrefix: "zai",
		SkipPrefixes:  []string{"zhipu/", "zai/", "openrouter/", "hosted_vllm/"},
		EnvExtras:     []EnvExtra{{Name: "ZHIPUAI_API_KEY", Value: "{api_key}"}},
	},

	{
		Name:          "dashscope",
		Keywords:      []string{"qwen", "dashscope"},
		EnvKey:        "DASHSCOPE_API_KEY",
		DisplayName:   "DashScope",
		LitellmPrefix: "dashscope",
		SkipPrefixes:  []string{"dashscope/", "openrouter/"},
	},

	{
		Name:          "moonshot",
		Keywords:      []string{"moonshot", "kimi"},
		EnvKey:        "MOONSHOT_API_KEY",
		DisplayName:   "Moonshot",
		LitellmPrefix: "moonshot",
		SkipPrefixes:  []string{"moonshot/", "openrouter/"},
		DefaultAPIBase: "https://api.moonshot.ai/v1",
		ModelOverrides: []ModelOverride{
			{Pattern: "kimi-k2.5", Overrides: map[string]any{"temperature": 1.0}},
		},
	},

	{
		Name:          "minimax",
		Keywords:      []string{"minimax"},
		EnvKey:        "MINIMAX_API_KEY",
		DisplayName:   "MiniMax",
		LitellmPrefix: "minimax",
		SkipPrefixes:  []string{"minimax/", "openrouter/"},
		DefaultAPIBase: "https://api.minimax.io/v1",
	},

	// === 本地部署 ===

	{
		Name:          "vllm",
		Keywords:      []string{"vllm"},
		EnvKey:        "HOSTED_VLLM_API_KEY",
		DisplayName:   "vLLM/Local",
		LitellmPrefix: "hosted_vllm",
		IsLocal:       true,
	},

	// === 辅助（非主要 LLM 提供商）===

	{
		Name:          "groq",
		Keywords:      []string{"groq"},
		EnvKey:        "GROQ_API_KEY",
		DisplayName:   "Groq",
		LitellmPrefix: "groq",
		SkipPrefixes:  []string{"groq/"},
	},

	// === 国内云平台 ===

	{
		Name:           "siliconflow",
		Keywords:       []string{"siliconflow"},
		EnvKey:         "SILICONFLOW_API_KEY",
		DisplayName:    "SiliconFlow",
		LitellmPrefix:  "openai",
		SkipPrefixes:   []string{"siliconflow/"},
		DefaultAPIBase: "https://api.siliconflow.cn/v1",
	},
}

// FindByModel 通过模型名称关键词查找提供商
func FindByModel(model string) *ProviderSpec {
	modelLower := model
	for _, spec := range Providers {
		if spec.IsGateway || spec.IsLocal {
			continue
		}
		for _, kw := range spec.Keywords {
			if contains(modelLower, kw) {
				return &spec
			}
		}
	}
	return nil
}

// FindGateway 检测网关/本地提供商
func FindGateway(providerName, apiKey, apiBase string) *ProviderSpec {
	// 1. 通过配置名称直接匹配
	if providerName != "" {
		if spec := FindByName(providerName); spec != nil && (spec.IsGateway || spec.IsLocal) {
			return spec
		}
	}

	// 2. 通过 API key 前缀检测
	for _, spec := range Providers {
		if spec.DetectByKeyPrefix != "" && apiKey != "" && hasPrefix(apiKey, spec.DetectByKeyPrefix) {
			return &spec
		}
		if spec.DetectByBaseKeyword != "" && apiBase != "" && contains(apiBase, spec.DetectByBaseKeyword) {
			return &spec
		}
	}

	return nil
}

// FindByName 通过名称查找提供商
func FindByName(name string) *ProviderSpec {
	for _, spec := range Providers {
		if spec.Name == name {
			return &spec
		}
	}
	return nil
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return len(sLower) >= len(substrLower) && indexOf(sLower, substrLower) >= 0
}

// hasPrefix 检查字符串是否有指定前缀
func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		sc := s[i]
		pc := prefix[i]
		if sc >= 'A' && sc <= 'Z' {
			sc += 32
		}
		if pc >= 'A' && pc <= 'Z' {
			pc += 32
		}
		if sc != pc {
			return false
		}
	}
	return true
}

// toLower 转换为小写
func toLower(s string) string {
	result := make([]byte, len(s))
	for i, c := range []byte(s) {
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// indexOf 查找子串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
