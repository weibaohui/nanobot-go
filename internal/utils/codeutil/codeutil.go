// Package codeutil 提供 Code 生成相关的工具函数
package codeutil

import (
	"fmt"
	"log"
	"time"
)

// Config Code生成配置
type Config struct {
	MaxRetries  int           // 最大重试次数，默认5次
	InitialDelay time.Duration // 初始延迟，默认10ms
	MaxDelay    time.Duration // 最大延迟，默认1s
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:   5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}
}

// GenerateUniqueCodeWithRetry 生成唯一 Code（带重试和指数退避）
// generateFunc 生成 Code 的函数
// checker 函数用于检查 Code 是否已存在
// 使用默认配置
func GenerateUniqueCodeWithRetry(
	generateFunc func() (string, error),
	checker func(string) (bool, error),
	maxRetries int,
) (string, error) {
	return GenerateUniqueCodeWithConfig(generateFunc, checker, &Config{
		MaxRetries:   maxRetries,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	})
}

// GenerateUniqueCodeWithConfig 使用指定配置生成唯一 Code
func GenerateUniqueCodeWithConfig(
	generateFunc func() (string, error),
	checker func(string) (bool, error),
	config *Config,
) (string, error) {
	if config == nil {
		config = DefaultConfig()
	}

	delay := config.InitialDelay

	for i := 0; i < config.MaxRetries; i++ {
		code, err := generateFunc()
		if err != nil {
			return "", err
		}

		exists, err := checker(code)
		if err != nil {
			return "", err
		}

		if !exists {
			if i > 0 {
				log.Printf("[codeutil] Generated unique code after %d retries", i+1)
			}
			return code, nil
		}

		// 如果不是最后一次尝试，进行退避等待
		if i < config.MaxRetries-1 {
			log.Printf("[codeutil] Code collision detected, retrying after %v (attempt %d/%d)", delay, i+1, config.MaxRetries)
			time.Sleep(delay)
			// 指数退避：下次延迟翻倍，但不超过最大延迟
			delay *= 2
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d retries with exponential backoff", config.MaxRetries)
}

// CalculateBackoff 计算指数退避延迟
// attempt: 当前尝试次数（从0开始）
// initialDelay: 初始延迟
// maxDelay: 最大延迟
func CalculateBackoff(attempt int, initialDelay, maxDelay time.Duration) time.Duration {
	delay := initialDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > maxDelay {
			return maxDelay
		}
	}
	return delay
}
