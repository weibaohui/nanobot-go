package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
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

	// 删除关联的工具
	if err := s.mcpToolRepo.DeleteByServerID(id); err != nil {
		return fmt.Errorf("删除关联工具失败: %w", err)
	}

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

	// 尝试连接 MCP 服务器
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mcpClient, err := s.createMCPClient(server)
	if err != nil {
		return fmt.Errorf("创建 MCP 客户端失败: %w", err)
	}
	defer mcpClient.Close()

	// 启动传输层
	if err := mcpClient.Start(ctx); err != nil {
		_ = s.UpdateServerStatus(id, "error", fmt.Sprintf("启动失败: %v", err))
		return fmt.Errorf("MCP 服务器启动失败: %w", err)
	}

	// 执行 MCP 初始化握手
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "nanobot-mcp-client",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		_ = s.UpdateServerStatus(id, "error", fmt.Sprintf("初始化失败: %v", err))
		return fmt.Errorf("MCP 服务器初始化失败: %w", err)
	}

	_ = s.UpdateServerStatus(id, "active", "")
	return nil
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

	// 创建 MCP 客户端
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mcpClient, err := s.createMCPClient(server)
	if err != nil {
		_ = s.UpdateServerStatus(id, "error", fmt.Sprintf("创建客户端失败: %v", err))
		return fmt.Errorf("创建 MCP 客户端失败: %w", err)
	}
	defer mcpClient.Close()

	// 启动传输层
	if err := mcpClient.Start(ctx); err != nil {
		_ = s.UpdateServerStatus(id, "error", fmt.Sprintf("启动失败: %v", err))
		return fmt.Errorf("MCP 服务器启动失败: %w", err)
	}

	// 执行 MCP 初始化握手
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "nanobot-mcp-client",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		_ = s.UpdateServerStatus(id, "error", fmt.Sprintf("初始化失败: %v", err))
		return fmt.Errorf("MCP 服务器初始化失败: %w", err)
	}

	// 获取工具列表
	toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		_ = s.UpdateServerStatus(id, "error", fmt.Sprintf("获取工具列表失败: %v", err))
		return fmt.Errorf("获取工具列表失败: %w", err)
	}

	// 清空旧工具
	if err := s.mcpToolRepo.DeleteByServerID(id); err != nil {
		return fmt.Errorf("清空旧工具失败: %w", err)
	}

	// 保存工具到数据库
	var capabilities []models.MCPTool
	for _, tool := range toolsResult.Tools {
		// 构建 input_schema
		var inputSchema map[string]interface{}
		if tool.InputSchema.Properties != nil {
			schemaData, _ := json.Marshal(tool.InputSchema)
			_ = json.Unmarshal(schemaData, &inputSchema)
		}

		// 添加到 capabilities
		capabilities = append(capabilities, models.MCPTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: inputSchema,
		})

		// 保存到 mcp_tools 表
		mcpTool := &models.MCPToolModel{
			MCPServerID: id,
			Name:        tool.Name,
			Description: tool.Description,
		}
		if inputSchema != nil {
			schemaJSON, _ := json.Marshal(inputSchema)
			mcpTool.InputSchema = schemaJSON
		}

		if err := s.mcpToolRepo.Create(mcpTool); err != nil {
			return fmt.Errorf("保存工具失败: %w", err)
		}
	}

	// 更新服务器 capabilities 字段（向后兼容）
	if err := server.SetCapabilities(capabilities); err != nil {
		return fmt.Errorf("设置能力列表失败: %w", err)
	}

	server.UpdatedAt = time.Now()
	if err := s.mcpServerRepo.Update(server); err != nil {
		return err
	}

	_ = s.UpdateServerStatus(id, "active", "")
	return nil
}

// createMCPClient 根据服务器配置创建 MCP 客户端
func (s *service) createMCPClient(server *models.MCPServer) (*client.Client, error) {
	switch server.TransportType {
	case "stdio":
		return s.createStdioClient(server)
	case "http", "sse":
		return s.createSSEClient(server)
	default:
		return nil, fmt.Errorf("不支持的传输类型: %s", server.TransportType)
	}
}

// createStdioClient 创建 stdio 类型 MCP 客户端
func (s *service) createStdioClient(server *models.MCPServer) (*client.Client, error) {
	if server.Command == "" {
		return nil, fmt.Errorf("stdio 类型需要指定启动命令")
	}

	args := server.GetArgs()
	envVars := server.GetEnvVars()

	// 构建环境变量字符串数组
	var env []string
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return client.NewStdioMCPClient(server.Command, env, args...)
}

// createSSEClient 创建 SSE 类型 MCP 客户端（也用于 HTTP）
func (s *service) createSSEClient(server *models.MCPServer) (*client.Client, error) {
	if server.URL == "" {
		return nil, fmt.Errorf("%s 类型需要指定服务 URL", server.TransportType)
	}

	return client.NewSSEMCPClient(server.URL)
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

// ExecuteTool 执行 MCP 工具
func (s *service) ExecuteTool(serverID uint, toolName string, params map[string]interface{}) (string, error) {
	server, err := s.mcpServerRepo.GetByID(serverID)
	if err != nil {
		return "", fmt.Errorf("获取 MCP 服务器失败: %w", err)
	}
	if server == nil {
		return "", fmt.Errorf("MCP 服务器不存在")
	}

	if server.Status != "active" {
		return "", fmt.Errorf("MCP 服务器未激活，当前状态: %s", server.Status)
	}

	// 创建 MCP 客户端
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mcpClient, err := s.createMCPClient(server)
	if err != nil {
		return "", fmt.Errorf("创建 MCP 客户端失败: %w", err)
	}
	defer mcpClient.Close()

	// 启动传输层
	if err := mcpClient.Start(ctx); err != nil {
		return "", fmt.Errorf("MCP 服务器启动失败: %w", err)
	}

	// 执行 MCP 初始化握手
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "nanobot-mcp-client",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		return "", fmt.Errorf("MCP 服务器初始化失败: %w", err)
	}

	// 调用工具
	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: params,
		},
	})
	if err != nil {
		return "", fmt.Errorf("调用 MCP 工具失败: %w", err)
	}

	// 处理结果
	if result.IsError {
		return "", fmt.Errorf("MCP 工具执行错误: %s", result.Content)
	}

	// 将结果转换为 JSON 字符串
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}
