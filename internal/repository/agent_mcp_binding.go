package repository

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/gorm"
)

// AgentMCPBindingRepository Agent MCP 绑定仓库接口
type AgentMCPBindingRepository interface {
	Create(binding *models.AgentMCPBinding) error
	GetByID(id uint) (*models.AgentMCPBinding, error)
	GetByAgentID(agentID uint) ([]models.AgentMCPBinding, error)
	GetByAgentAndMCPServer(agentID, mcpServerID uint) (*models.AgentMCPBinding, error)
	GetByAgentCode(agentCode string) ([]models.AgentMCPBinding, error)
	Update(binding *models.AgentMCPBinding) error
	Delete(id uint) error
	DeleteByAgentAndMCPServer(agentID, mcpServerID uint) error
	CheckExists(agentID, mcpServerID uint) (bool, error)
}

// agentMCPBindingRepository Agent MCP 绑定仓库实现
type agentMCPBindingRepository struct {
	db *gorm.DB
}

// NewAgentMCPBindingRepository 创建 Agent MCP 绑定仓库
func NewAgentMCPBindingRepository(db *gorm.DB) AgentMCPBindingRepository {
	return &agentMCPBindingRepository{db: db}
}

// Create 创建绑定
func (r *agentMCPBindingRepository) Create(binding *models.AgentMCPBinding) error {
	if err := r.db.Create(binding).Error; err != nil {
		return fmt.Errorf("创建 Agent MCP 绑定失败: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取绑定
func (r *agentMCPBindingRepository) GetByID(id uint) (*models.AgentMCPBinding, error) {
	var binding models.AgentMCPBinding
	if err := r.db.Preload("Agent").Preload("MCPServer").First(&binding, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 Agent MCP 绑定失败: %w", err)
	}
	return &binding, nil
}

// GetByAgentID 获取 Agent 的所有绑定
func (r *agentMCPBindingRepository) GetByAgentID(agentID uint) ([]models.AgentMCPBinding, error) {
	var bindings []models.AgentMCPBinding
	if err := r.db.Preload("MCPServer").Where("agent_id = ?", agentID).Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("获取 Agent MCP 绑定列表失败: %w", err)
	}
	return bindings, nil
}

// GetByAgentAndMCPServer 获取指定 Agent 和 MCP 服务器的绑定
func (r *agentMCPBindingRepository) GetByAgentAndMCPServer(agentID, mcpServerID uint) (*models.AgentMCPBinding, error) {
	var binding models.AgentMCPBinding
	if err := r.db.Where("agent_id = ? AND mcp_server_id = ?", agentID, mcpServerID).First(&binding).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 Agent MCP 绑定失败: %w", err)
	}
	return &binding, nil
}

// GetByAgentCode 根据 AgentCode 获取所有绑定
func (r *agentMCPBindingRepository) GetByAgentCode(agentCode string) ([]models.AgentMCPBinding, error) {
	var bindings []models.AgentMCPBinding
	if err := r.db.
		Joins("JOIN agents ON agents.id = agent_mcp_bindings.agent_id").
		Preload("MCPServer").
		Where("agents.agent_code = ?", agentCode).
		Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("获取 Agent MCP 绑定列表失败: %w", err)
	}
	return bindings, nil
}

// Update 更新绑定
func (r *agentMCPBindingRepository) Update(binding *models.AgentMCPBinding) error {
	if err := r.db.Save(binding).Error; err != nil {
		return fmt.Errorf("更新 Agent MCP 绑定失败: %w", err)
	}
	return nil
}

// Delete 删除绑定
func (r *agentMCPBindingRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.AgentMCPBinding{}, id).Error; err != nil {
		return fmt.Errorf("删除 Agent MCP 绑定失败: %w", err)
	}
	return nil
}

// DeleteByAgentAndMCPServer 删除指定 Agent 和 MCP 服务器的绑定
func (r *agentMCPBindingRepository) DeleteByAgentAndMCPServer(agentID, mcpServerID uint) error {
	if err := r.db.Where("agent_id = ? AND mcp_server_id = ?", agentID, mcpServerID).
		Delete(&models.AgentMCPBinding{}).Error; err != nil {
		return fmt.Errorf("删除 Agent MCP 绑定失败: %w", err)
	}
	return nil
}

// CheckExists 检查绑定是否已存在
func (r *agentMCPBindingRepository) CheckExists(agentID, mcpServerID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.AgentMCPBinding{}).
		Where("agent_id = ? AND mcp_server_id = ?", agentID, mcpServerID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查 Agent MCP 绑定失败: %w", err)
	}
	return count > 0, nil
}
