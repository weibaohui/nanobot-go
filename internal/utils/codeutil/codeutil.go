// Package codeutil 提供 Code 生成相关的工具函数
package codeutil

import "fmt"

// GenerateUniqueCodeWithRetry 生成唯一 Code（带重试）
// generateFunc 生成 Code 的函数
// checker 函数用于检查 Code 是否已存在
func GenerateUniqueCodeWithRetry(
	generateFunc func() (string, error),
	checker func(string) (bool, error),
	maxRetries int,
) (string, error) {
	for i := 0; i < maxRetries; i++ {
		code, err := generateFunc()
		if err != nil {
			return "", err
		}

		exists, err := checker(code)
		if err != nil {
			return "", err
		}

		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique code after %d retries", maxRetries)
}
