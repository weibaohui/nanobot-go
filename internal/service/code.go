package service

import (
	"strings"

	"github.com/matoous/go-nanoid/v2"
	"github.com/weibaohui/nanobot-go/internal/utils/codeutil"
)

const (
	// 各实体前缀
	UserPrefix    = "usr_"
	ChannelPrefix = "chn_"
	AgentPrefix   = "agt_"

	// Code 随机部分长度（不含前缀）
	CodeLength = 10
)

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
type codeService struct{}

// NewCodeService 创建 Code 服务
func NewCodeService() CodeService {
	return &codeService{}
}

// generateNanoID 使用 go-nanoid 生成指定长度的随机 ID
func (s *codeService) generateNanoID(length int) (string, error) {
	return gonanoid.New(length)
}

// GenerateUserCode 生成用户 Code
func (s *codeService) GenerateUserCode() (string, error) {
	id, err := s.generateNanoID(CodeLength)
	if err != nil {
		return "", err
	}
	return UserPrefix + id, nil
}

// GenerateChannelCode 生成渠道 Code
func (s *codeService) GenerateChannelCode() (string, error) {
	id, err := s.generateNanoID(CodeLength)
	if err != nil {
		return "", err
	}
	return ChannelPrefix + id, nil
}

// GenerateAgentCode 生成 Agent Code
func (s *codeService) GenerateAgentCode() (string, error) {
	id, err := s.generateNanoID(CodeLength)
	if err != nil {
		return "", err
	}
	return AgentPrefix + id, nil
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
