// Package pagination 提供分页相关的工具函数
package pagination

import "fmt"

// Params 分页参数
type Params struct {
	Page     int // 当前页码，从1开始
	PageSize int // 每页大小
}

// Result 分页结果
type Result struct {
	Page       int   // 当前页码
	PageSize   int   // 每页大小
	Total      int64 // 总记录数
	TotalPages int   // 总页数
	Offset     int   // 数据库偏移量
}

// Normalize 规范化分页参数
// 返回规范化的页码和每页大小
func Normalize(page, pageSize, defaultSize, maxSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSize
	}
	if pageSize > maxSize {
		pageSize = maxSize
	}
	return page, pageSize
}

// NormalizeDefault 使用默认参数规范化（默认20，最大1000）
func NormalizeDefault(page, pageSize int) (int, int) {
	return Normalize(page, pageSize, 20, 1000)
}

// CalculateOffset 计算数据库偏移量
func CalculateOffset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// CalculateTotalPages 计算总页数
func CalculateTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}
	return pages
}

// NewResult 创建分页结果
func NewResult(page, pageSize int, total int64) *Result {
	page, pageSize = NormalizeDefault(page, pageSize)
	return &Result{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: CalculateTotalPages(total, pageSize),
		Offset:     CalculateOffset(page, pageSize),
	}
}

// Validate 验证分页参数是否有效
func Validate(page, pageSize int) error {
	if page < 1 {
		return fmt.Errorf("page must be >= 1, got %d", page)
	}
	if pageSize < 1 {
		return fmt.Errorf("page_size must be >= 1, got %d", pageSize)
	}
	return nil
}
