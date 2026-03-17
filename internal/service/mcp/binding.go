package mcp

import (
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// CreateAgentBinding 创建 Agent MCP 绑定
func (s *service) CreateAgentBinding(agentID uint, req CreateAgentMCPBindingRequest) (*models.AgentMCPBinding, error) {
	// 检查 Agent 是否存在
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("Agent 不存在")
	}

	// 检查 MCP 服务器是否存在
	server, err := s.mcpServerRepo.GetByID(req.MCPServerID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("MCP 服务器不存在")
	}

	// 检查绑定是否已存在
	exists, err := s.agentMCPBindingRepo.CheckExists(agentID, req.MCPServerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("Agent 已绑定该 MCP 服务器")
	}

	binding := &models.AgentMCPBinding{
		AgentID:     agentID,
		MCPServerID: req.MCPServerID,
		IsActive:    true,
		AutoLoad:    false,
	}

	// 设置启用的工具
	if len(req.EnabledTools) > 0 {
		if err := binding.SetEnabledTools(req.EnabledTools); err != nil {
			return nil, fmt.Errorf("设置启用工具失败: %w", err)
		}
	}

	if req.IsActive != nil {
		binding.IsActive = *req.IsActive
	}

	if req.AutoLoad != nil {
		binding.AutoLoad = *req.AutoLoad
	}

	if err := s.agentMCPBindingRepo.Create(binding); err != nil {
		return nil, err
	}

	return binding, nil
}

// GetAgentBindings 获取 Agent 的所有 MCP 绑定
func (s *service) GetAgentBindings(agentID uint) ([]models.AgentMCPBinding, error) {
	return s.agentMCPBindingRepo.GetByAgentID(agentID)
}

// GetAgentBindingByID 根据 ID 获取绑定
func (s *service) GetAgentBindingByID(bindingID uint) (*models.AgentMCPBinding, error) {
	return s.agentMCPBindingRepo.GetByID(bindingID)
}

// UpdateAgentBinding 更新 Agent MCP 绑定
func (s *service) UpdateAgentBinding(bindingID uint, req UpdateAgentMCPBindingRequest) (*models.AgentMCPBinding, error) {
	binding, err := s.agentMCPBindingRepo.GetByID(bindingID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, fmt.Errorf("绑定不存在")
	}

	// 更新字段
	if req.EnabledTools != nil {
		if len(req.EnabledTools) == 0 {
			binding.EnabledTools = "" // 全部启用
		} else {
			if err := binding.SetEnabledTools(req.EnabledTools); err != nil {
				return nil, fmt.Errorf("设置启用工具失败: %w", err)
			}
		}
	}
	if req.IsActive != nil {
		binding.IsActive = *req.IsActive
	}
	if req.AutoLoad != nil {
		binding.AutoLoad = *req.AutoLoad
	}

	binding.UpdatedAt = time.Now()
	if err := s.agentMCPBindingRepo.Update(binding); err != nil {
		return nil, err
	}

	return binding, nil
}

// DeleteAgentBinding 删除 Agent MCP 绑定
func (s *service) DeleteAgentBinding(bindingID uint) error {
	binding, err := s.agentMCPBindingRepo.GetByID(bindingID)
	if err != nil {
		return err
	}
	if binding == nil {
		return fmt.Errorf("绑定不存在")
	}

	return s.agentMCPBindingRepo.Delete(bindingID)
}

// DeleteAgentBindingByServer 根据 MCP 服务器删除绑定
func (s *service) DeleteAgentBindingByServer(agentID, mcpServerID uint) error {
	return s.agentMCPBindingRepo.DeleteByAgentAndMCPServer(agentID, mcpServerID)
}

// GetAgentMCPTools 获取 Agent 可用的 MCP 工具
func (s *service) GetAgentMCPTools(agentCode string) ([]AgentMCPToolInfo, error) {
	// 获取 Agent
	agent, err := s.agentRepo.GetByAgentCode(agentCode)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("Agent 不存在")
	}

	// 获取 Agent 的所有绑定
	bindings, err := s.agentMCPBindingRepo.GetByAgentID(agent.ID)
	if err != nil {
		return nil, err
	}

	var tools []AgentMCPToolInfo
	for _, binding := range bindings {
		if !binding.IsActive {
			continue
		}

		// 获取 MCP 服务器详情
		server, err := s.mcpServerRepo.GetByID(binding.MCPServerID)
		if err != nil {
			continue
		}
		if server == nil || server.Status != "active" {
			continue
		}

		// 获取服务器的能力列表
		capabilities := server.GetCapabilities()
		enabledTools := binding.GetEnabledTools()

		for _, tool := range capabilities {
			// 检查工具是否启用
			if enabledTools != nil && !contains(enabledTools, tool.Name) {
				continue
			}

			tools = append(tools, AgentMCPToolInfo{
				MCPServerCode: server.Code,
				MCPServerName: server.Name,
				Tool:          tool,
			})
		}
	}

	return tools, nil
}

// GetAgentMCPServersWithBinding 获取 Agent 绑定的 MCP Servers（包含绑定配置）
func (s *service) GetAgentMCPServersWithBinding(agentCode string) ([]AgentMCPServerInfo, error) {
	// 获取 Agent
	agent, err := s.agentRepo.GetByAgentCode(agentCode)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("Agent 不存在")
	}

	// 获取 Agent 的所有绑定
	bindings, err := s.agentMCPBindingRepo.GetByAgentID(agent.ID)
	if err != nil {
		return nil, err
	}

	var result []AgentMCPServerInfo
	for _, binding := range bindings {
		// 获取 MCP Server 详情
		server, err := s.mcpServerRepo.GetByID(binding.MCPServerID)
		if err != nil {
			// 记录错误但继续处理其他绑定，避免因为一个 Server 的错误导致整个列表失败
			// 这里可以考虑添加日志记录
			continue
		}
		if server == nil {
			continue
		}

		result = append(result, AgentMCPServerInfo{
			MCPServer: server,
			Binding:   &binding,
		})
	}

	return result, nil
}

// contains 检查字符串数组是否包含指定元素
func contains(arr []string, str string) bool {
	for _, s := range arr {
		if s == str {
			return true
		}
	}
	return false
}