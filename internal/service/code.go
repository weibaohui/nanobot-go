package service

import (
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/weibaohui/nanobot-go/internal/utils/codeutil"
)

const (
	// CodeAlphabet 去除易混淆字符（0, O, 1, I, l）
	CodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// 各实体前缀
	UserPrefix    = "usr_"
	ChannelPrefix = "chn_"
	AgentPrefix   = "agt_"

	// Code 随机部分长度（不含前缀）
	CodeLength = 10
)

// CodeGenerator Code 生成器
type CodeGenerator struct {
	alphabet []rune
	length   int
}

// NewCodeGenerator 创建 Code 生成器
func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{
		alphabet: []rune(CodeAlphabet),
		length:   CodeLength,
	}
}

// Generate 生成随机 Code
func (g *CodeGenerator) Generate() (string, error) {
	var sb strings.Builder
	alphabetLen := big.NewInt(int64(len(g.alphabet)))

	for i := 0; i < g.length; i++ {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		sb.WriteRune(g.alphabet[idx.Int64()])
	}

	return sb.String(), nil
}

// GenerateWithPrefix 带前缀生成 Code
func (g *CodeGenerator) GenerateWithPrefix(prefix string) (string, error) {
	code, err := g.Generate()
	if err != nil {
		return "", err
	}
	return prefix + code, nil
}

// CodeService Code 生成服务接口
type CodeService interface {
	// GenerateUserCode 生成用户 Code
	GenerateUserCode() (string, error)
	// GenerateChannelCode 生成渠道 Code
	GenerateChannelCode() (string, error)
	// GenerateAgentCode 生成 Agent Code
	GenerateAgentCode() (string, error)
	// ValidateCode 验证 Code 格式是否合法
	ValidateCode(code string, prefix string) bool
}

// codeService Code 服务实现
type codeService struct {
	generator *CodeGenerator
}

// NewCodeService 创建 Code 服务
func NewCodeService() CodeService {
	return &codeService{
		generator: NewCodeGenerator(),
	}
}

// GenerateUserCode 生成用户 Code
func (s *codeService) GenerateUserCode() (string, error) {
	return s.generator.GenerateWithPrefix(UserPrefix)
}

// GenerateChannelCode 生成渠道 Code
func (s *codeService) GenerateChannelCode() (string, error) {
	return s.generator.GenerateWithPrefix(ChannelPrefix)
}

// GenerateAgentCode 生成 Agent Code
func (s *codeService) GenerateAgentCode() (string, error) {
	return s.generator.GenerateWithPrefix(AgentPrefix)
}

// ValidateCode 验证 Code 格式是否合法
func (s *codeService) ValidateCode(code string, prefix string) bool {
	if code == "" {
		return false
	}
	// 检查前缀
	if !strings.HasPrefix(code, prefix) {
		return false
	}
	// 检查长度
	if len(code) != len(prefix)+CodeLength {
		return false
	}
	// 检查字符是否都在合法字符集中
	alphabetSet := make(map[rune]bool)
	for _, r := range CodeAlphabet {
		alphabetSet[r] = true
	}

	randomPart := code[len(prefix):]
	for _, r := range randomPart {
		if !alphabetSet[r] {
			return false
		}
	}
	return true
}

// GenerateUniqueCodeWithRetry 生成唯一 Code（带重试）
// 这是一个包装函数，实际实现在 codeutil 包中
func GenerateUniqueCodeWithRetry(
	generateFunc func() (string, error),
	checker func(string) (bool, error),
	maxRetries int,
) (string, error) {
	return codeutil.GenerateUniqueCodeWithRetry(generateFunc, checker, maxRetries)
}
