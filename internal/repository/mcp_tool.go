package repository

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// MCPToolRepository MCP工具仓库接口
type MCPToolRepository interface {
	// 创建工具
	Create(tool *models.MCPToolModel) error
	// 批量创建工具
	CreateBatch(tools []models.MCPToolModel) error
	// 根据ID获取工具
	GetByID(id uint) (*models.MCPToolModel, error)
	// 获取指定MCP Server的所有工具
	ListByServerID(serverID uint) ([]models.MCPToolModel, error)
	// 删除指定MCP Server的所有工具
	DeleteByServerID(serverID uint) error
	// 更新工具
	Update(tool *models.MCPToolModel) error
}

// mcpToolRepository MCP工具仓库实现
type mcpToolRepository struct {
	db *gorm.DB
}

// NewMCPToolRepository 创建MCP工具仓库
func NewMCPToolRepository(db *gorm.DB) MCPToolRepository {
	return &mcpToolRepository{db: db}
}

// Create 创建工具
func (r *mcpToolRepository) Create(tool *models.MCPToolModel) error {
	return r.db.Create(tool).Error
}

// CreateBatch 批量创建工具
func (r *mcpToolRepository) CreateBatch(tools []models.MCPToolModel) error {
	if len(tools) == 0 {
		return nil
	}
	return r.db.Create(&tools).Error
}

// GetByID 根据ID获取工具
func (r *mcpToolRepository) GetByID(id uint) (*models.MCPToolModel, error) {
	var tool models.MCPToolModel
	if err := r.db.First(&tool, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取MCP工具失败: %w", err)
	}
	return &tool, nil
}

// ListByServerID 获取指定MCP Server的所有工具
func (r *mcpToolRepository) ListByServerID(serverID uint) ([]models.MCPToolModel, error) {
	var tools []models.MCPToolModel
	if err := r.db.Where("mcp_server_id = ?", serverID).Find(&tools).Error; err != nil {
		return nil, fmt.Errorf("获取MCP工具列表失败: %w", err)
	}
	return tools, nil
}

// DeleteByServerID 删除指定MCP Server的所有工具
func (r *mcpToolRepository) DeleteByServerID(serverID uint) error {
	return r.db.Where("mcp_server_id = ?", serverID).Delete(&models.MCPToolModel{}).Error
}

// Update 更新工具
func (r *mcpToolRepository) Update(tool *models.MCPToolModel) error {
	return r.db.Save(tool).Error
}

// MCPToolLogRepository MCP工具调用日志仓库接口
type MCPToolLogRepository interface {
	// 创建日志
	Create(log *models.MCPToolLog) error
	// 获取指定MCP Server的日志列表
	ListByServerID(serverID uint, limit int) ([]models.MCPToolLog, error)
	// 获取指定会话的日志列表
	ListBySessionKey(sessionKey string, limit int) ([]models.MCPToolLog, error)
	// 删除指定天数之前的日志
	DeleteBeforeDays(days int) error
}

// mcpToolLogRepository MCP工具调用日志仓库实现
type mcpToolLogRepository struct {
	db *gorm.DB
}

// NewMCPToolLogRepository 创建MCP工具调用日志仓库
func NewMCPToolLogRepository(db *gorm.DB) MCPToolLogRepository {
	return &mcpToolLogRepository{db: db}
}

// Create 创建日志
func (r *mcpToolLogRepository) Create(log *models.MCPToolLog) error {
	return r.db.Create(log).Error
}

// ListByServerID 获取指定MCP Server的日志列表
func (r *mcpToolLogRepository) ListByServerID(serverID uint, limit int) ([]models.MCPToolLog, error) {
	var logs []models.MCPToolLog
	query := r.db.Where("mcp_server_id = ?", serverID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("获取MCP工具日志失败: %w", err)
	}
	return logs, nil
}

// ListBySessionKey 获取指定会话的日志列表
func (r *mcpToolLogRepository) ListBySessionKey(sessionKey string, limit int) ([]models.MCPToolLog, error) {
	var logs []models.MCPToolLog
	query := r.db.Where("session_key = ?", sessionKey).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("获取MCP工具日志失败: %w", err)
	}
	return logs, nil
}

// DeleteBeforeDays 删除指定天数之前的日志
func (r *mcpToolLogRepository) DeleteBeforeDays(days int) error {
	sql := fmt.Sprintf("DELETE FROM mcp_tool_logs WHERE created_at < datetime('now', '-%d days')", days)
	return r.db.Exec(sql).Error
}
