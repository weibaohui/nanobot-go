package mcp

import (
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// CreateServer 创建 MCP 服务器
func (s *service) CreateServer(req CreateMCPServerRequest) (*models.MCPServer, error) {
	// 检查编码是否已存在
	exists, err := s.mcpServerRepo.CheckCodeExists(req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("MCP 服务器编码已存在: %s", req.Code)
	}

	// 验证传输类型配置
	if err := validateTransportConfig(req.TransportType, req.Command, req.URL); err != nil {
		return nil, err
	}

	server := &models.MCPServer{
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		TransportType: req.TransportType,
		Command:       req.Command,
		URL:           req.URL,
		Status:        "inactive",
	}

	// 设置参数
	if len(req.Args) > 0 {
		if err := server.SetArgs(req.Args); err != nil {
			return nil, fmt.Errorf("设置参数失败: %w", err)
		}
	}

	// 设置环境变量
	if len(req.EnvVars) > 0 {
		if err := server.SetEnvVars(req.EnvVars); err != nil {
			return nil, fmt.Errorf("设置环境变量失败: %w", err)
		}
	}

	if err := s.mcpServerRepo.Create(server); err != nil {
		return nil, err
	}

	return server, nil
}

// GetServer 获取 MCP 服务器
func (s *service) GetServer(id uint) (*models.MCPServer, error) {
	return s.mcpServerRepo.GetByID(id)
}

// GetServerByCode 根据编码获取 MCP 服务器
func (s *service) GetServerByCode(code string) (*models.MCPServer, error) {
	return s.mcpServerRepo.GetByCode(code)
}

// ListServers 获取所有 MCP 服务器
func (s *service) ListServers() ([]models.MCPServer, error) {
	return s.mcpServerRepo.List()
}

// ListServersByStatus 根据状态获取 MCP 服务器
func (s *service) ListServersByStatus(status string) ([]models.MCPServer, error) {
	return s.mcpServerRepo.ListByStatus(status)
}

// UpdateServer 更新 MCP 服务器
func (s *service) UpdateServer(id uint, req UpdateMCPServerRequest) (*models.MCPServer, error) {
	server, err := s.mcpServerRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("MCP 服务器不存在")
	}

	// 更新字段
	if req.Name != "" {
		server.Name = req.Name
	}
	if req.Description != "" {
		server.Description = req.Description
	}
	if req.TransportType != "" {
		// 验证传输类型配置
		command := req.Command
		if command == "" {
			command = server.Command
		}
		url := req.URL
		if url == "" {
			url = server.URL
		}
		if err := validateTransportConfig(req.TransportType, command, url); err != nil {
			return nil, err
		}
		server.TransportType = req.TransportType
	}
	if req.Command != "" {
		server.Command = req.Command
	}
	if req.URL != "" {
		server.URL = req.URL
	}
	if len(req.Args) > 0 {
		if err := server.SetArgs(req.Args); err != nil {
			return nil, fmt.Errorf("设置参数失败: %w", err)
		}
	}
	if len(req.EnvVars) > 0 {
		if err := server.SetEnvVars(req.EnvVars); err != nil {
			return nil, fmt.Errorf("设置环境变量失败: %w", err)
		}
	}

	server.UpdatedAt = time.Now()
	if err := s.mcpServerRepo.Update(server); err != nil {
		return nil, err
	}

	return server, nil
}

// DeleteServer 删除 MCP 服务器
func (s *service) DeleteServer(id uint) error {
	// 检查是否存在
	server, err := s.mcpServerRepo.GetByID(id)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("MCP 服务器不存在")
	}

	// TODO: 断开连接（如果已连接）

	return s.mcpServerRepo.Delete(id)
}

// UpdateServerStatus 更新 MCP 服务器状态
func (s *service) UpdateServerStatus(id uint, status string, errorMsg string) error {
	server, err := s.mcpServerRepo.GetByID(id)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("MCP 服务器不存在")
	}

	server.Status = status
	server.ErrorMessage = errorMsg

	if status == "active" {
		now := time.Now()
		server.LastConnectedAt = &now
		server.ErrorMessage = ""
	}

	server.UpdatedAt = time.Now()
	return s.mcpServerRepo.Update(server)
}

// TestServer 测试 MCP 服务器连接
func (s *service) TestServer(id uint) error {
	server, err := s.mcpServerRepo.GetByID(id)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("MCP 服务器不存在")
	}

	// TODO: 实现实际的 MCP 连接测试
	// 这里先模拟成功
	return s.UpdateServerStatus(id, "active", "")
}

// RefreshCapabilities 刷新 MCP 服务器能力
func (s *service) RefreshCapabilities(id uint) error {
	server, err := s.mcpServerRepo.GetByID(id)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("MCP 服务器不存在")
	}

	// TODO: 实现实际的 MCP 能力获取
	// 这里先模拟获取成功
	capabilities := []models.MCPTool{
		{
			Name:        "example_tool",
			Description: "示例工具",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{
						"type":        "string",
						"description": "参数1",
					},
				},
			},
		},
	}

	if err := server.SetCapabilities(capabilities); err != nil {
		return fmt.Errorf("设置能力列表失败: %w", err)
	}

	server.UpdatedAt = time.Now()
	return s.mcpServerRepo.Update(server)
}

// validateTransportConfig 验证传输类型配置
func validateTransportConfig(transportType, command, url string) error {
	switch transportType {
	case "stdio":
		if command == "" {
			return fmt.Errorf("stdio 类型需要指定启动命令")
		}
	case "http", "sse":
		if url == "" {
			return fmt.Errorf("%s 类型需要指定服务 URL", transportType)
		}
	default:
		return fmt.Errorf("不支持的传输类型: %s", transportType)
	}
	return nil
}
