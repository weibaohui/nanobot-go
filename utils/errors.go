package utils

import (
	"errors"
	"fmt"
)

// 常用哨兵错误
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrTimeout      = errors.New("operation timeout")
	ErrCanceled     = errors.New("operation canceled")
)

// WrapError 包装错误并添加上下文信息
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

// WrapErrorWithField 包装错误并添加字段信息
func WrapErrorWithField(op string, field string, value any, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s=%v: %w", op, field, value, err)
}

// IsNotFoundError 检查是否为未找到错误
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsCanceledError 检查是否为取消错误
func IsCanceledError(err error) bool {
	return errors.Is(err, ErrCanceled)
}
