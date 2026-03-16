package repository

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// MCPServerRepository MCP 服务器仓库接口
type MCPServerRepository interface {
	Create(server *models.MCPServer) error
	GetByID(id uint) (*models.MCPServer, error)
	GetByCode(code string) (*models.MCPServer, error)
	List() ([]models.MCPServer, error)
	ListByStatus(status string) ([]models.MCPServer, error)
	Update(server *models.MCPServer) error
	Delete(id uint) error
	CheckCodeExists(code string) (bool, error)
}

// mcpServerRepository MCP 服务器仓库实现
type mcpServerRepository struct {
	db *gorm.DB
}

// NewMCPServerRepository 创建 MCP 服务器仓库
func NewMCPServerRepository(db *gorm.DB) MCPServerRepository {
	return &mcpServerRepository{db: db}
}

// Create 创建 MCP 服务器
func (r *mcpServerRepository) Create(server *models.MCPServer) error {
	if err := r.db.Create(server).Error; err != nil {
		return fmt.Errorf("创建 MCP 服务器失败: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取 MCP 服务器
func (r *mcpServerRepository) GetByID(id uint) (*models.MCPServer, error) {
	var server models.MCPServer
	if err := r.db.First(&server, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 MCP 服务器失败: %w", err)
	}
	return &server, nil
}

// GetByCode 根据编码获取 MCP 服务器
func (r *mcpServerRepository) GetByCode(code string) (*models.MCPServer, error) {
	var server models.MCPServer
	if err := r.db.Where("code = ?", code).First(&server).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 MCP 服务器失败: %w", err)
	}
	return &server, nil
}

// List 获取所有 MCP 服务器
func (r *mcpServerRepository) List() ([]models.MCPServer, error) {
	var servers []models.MCPServer
	if err := r.db.Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("获取 MCP 服务器列表失败: %w", err)
	}
	return servers, nil
}

// ListByStatus 根据状态获取 MCP 服务器
func (r *mcpServerRepository) ListByStatus(status string) ([]models.MCPServer, error) {
	var servers []models.MCPServer
	if err := r.db.Where("status = ?", status).Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("获取 MCP 服务器列表失败: %w", err)
	}
	return servers, nil
}

// Update 更新 MCP 服务器
func (r *mcpServerRepository) Update(server *models.MCPServer) error {
	if err := r.db.Save(server).Error; err != nil {
		return fmt.Errorf("更新 MCP 服务器失败: %w", err)
	}
	return nil
}

// Delete 删除 MCP 服务器
func (r *mcpServerRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.MCPServer{}, id).Error; err != nil {
		return fmt.Errorf("删除 MCP 服务器失败: %w", err)
	}
	return nil
}

// CheckCodeExists 检查编码是否已存在
func (r *mcpServerRepository) CheckCodeExists(code string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.MCPServer{}).Where("code = ?", code).Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查 MCP 服务器编码失败: %w", err)
	}
	return count > 0, nil
}
