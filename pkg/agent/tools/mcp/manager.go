// Package mcp 提供会话级 MCP 工具管理功能
// 支持渐进式加载：默认只展示 Server 名称，通过 use_mcp 工具按需加载具体工具
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	mcpsvc "github.com/weibaohui/nanobot-go/internal/service/mcp"
	"go.uber.org/zap"
)

// LoadedServer 已加载的 MCP Server 信息
type LoadedServer struct {
	Code        string
	Name        string
	Description string
	Tools       []tool.InvokableTool
	LoadedAt    time.Time
}

// Manager 会话级 MCP 工具管理器
type Manager struct {
	// 已加载的 MCP Servers
	loadedServers map[string]*LoadedServer
	mu            sync.RWMutex

	// MCP Service
	mcpService mcpsvc.Service

	// 日志
	logger *zap.Logger
}

// NewManager 创建 MCP 管理器
func NewManager(mcpService mcpsvc.Service, logger *zap.Logger) *Manager {
	return &Manager{
		loadedServers: make(map[string]*LoadedServer),
		mcpService:    mcpService,
		logger:        logger,
	}
}

// LoadServer 加载指定 MCP Server 的工具
// 幂等操作：如果 Server 已加载，直接返回已加载的信息
func (m *Manager) LoadServer(serverCode string) (*LoadedServer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已加载
	if loaded, ok := m.loadedServers[serverCode]; ok {
		m.logger.Debug("MCP Server 已加载，返回缓存", zap.String("server_code", serverCode))
		return loaded, nil
	}

	// 获取 Server 详情
	server, err := m.mcpService.GetServerByCode(serverCode)
	if err != nil {
		return nil, fmt.Errorf("获取 MCP Server 失败: %w", err)
	}
	if server == nil {
		return nil, fmt.Errorf("MCP Server '%s' 不存在", serverCode)
	}

	// 检查 Server 状态
	if server.Status != "active" {
		return nil, fmt.Errorf("MCP Server '%s' 未激活，当前状态: %s", serverCode, server.Status)
	}

	// 获取 Server 的工具列表
	capabilities := server.GetCapabilities()
	if len(capabilities) == 0 {
		// 允许加载无工具的 Server（支持动态工具场景），仅记录警告
		m.logger.Warn("MCP Server 没有可用工具",
			zap.String("server_code", serverCode))
	}

	// 创建工具实例
	var tools []tool.InvokableTool
	for _, cap := range capabilities {
		t := NewMCPTool(server.ID, server.Code, cap.Name, cap.Description, cap.InputSchema, m.mcpService, m.logger)
		tools = append(tools, t)
	}

	// 保存已加载信息
	loaded := &LoadedServer{
		Code:        server.Code,
		Name:        server.Name,
		Description: server.Description,
		Tools:       tools,
		LoadedAt:    time.Now(),
	}
	m.loadedServers[serverCode] = loaded

	m.logger.Info("MCP Server 加载成功",
		zap.String("server_code", serverCode),
		zap.String("server_name", server.Name),
		zap.Int("tool_count", len(tools)),
	)

	return loaded, nil
}

// IsLoaded 检查指定 Server 是否已加载
func (m *Manager) IsLoaded(serverCode string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.loadedServers[serverCode]
	return ok
}

// GetLoadedTools 获取所有已加载的 MCP 工具
func (m *Manager) GetLoadedTools() []tool.BaseTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []tool.BaseTool
	for _, server := range m.loadedServers {
		for _, t := range server.Tools {
			result = append(result, t)
		}
	}
	return result
}

// GetLoadedServer 获取指定已加载的 Server
func (m *Manager) GetLoadedServer(serverCode string) *LoadedServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadedServers[serverCode]
}

// GetLoadedServers 获取所有已加载的 Servers
func (m *Manager) GetLoadedServers() []*LoadedServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*LoadedServer
	for _, server := range m.loadedServers {
		result = append(result, server)
	}
	return result
}

// GetLoadedServerCodes 获取所有已加载的 Server codes
func (m *Manager) GetLoadedServerCodes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var codes []string
	for code := range m.loadedServers {
		codes = append(codes, code)
	}
	return codes
}

// UnloadServer 卸载指定 Server（从已加载列表中移除）
func (m *Manager) UnloadServer(serverCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.loadedServers, serverCode)
	m.logger.Debug("MCP Server 已卸载", zap.String("server_code", serverCode))
}

// Clear 清空所有已加载的 Servers
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadedServers = make(map[string]*LoadedServer)
	m.logger.Debug("所有 MCP Server 已清空")
}

// GetServerInfo 获取 MCP Server 信息（不加载工具）
func (m *Manager) GetServerInfo(serverCode string) (*ServerInfo, error) {
	server, err := m.mcpService.GetServerByCode(serverCode)
	if err != nil {
		return nil, fmt.Errorf("获取 MCP Server 失败: %w", err)
	}
	if server == nil {
		return nil, fmt.Errorf("MCP Server '%s' 不存在", serverCode)
	}

	return &ServerInfo{
		Code:        server.Code,
		Name:        server.Name,
		Description: server.Description,
		Status:      server.Status,
		ToolCount:   len(server.GetCapabilities()),
	}, nil
}

// ServerInfo MCP Server 基本信息（用于展示，不加载工具）
type ServerInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	ToolCount   int    `json:"tool_count"`
}

// MCPToolInfo 工具信息
type MCPToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ExecuteTool 执行 MCP 工具
// 如果 Server 未加载，会自动加载
func (m *Manager) ExecuteTool(ctx context.Context, serverCode, toolName string, params map[string]interface{}) (string, error) {
	// 规范化 nil params 为空对象，避免序列化为 "null"
	if params == nil {
		params = map[string]interface{}{}
	}

	// 检查 Server 是否已加载
	server := m.GetLoadedServer(serverCode)
	if server == nil {
		// 自动加载 Server
		m.logger.Info("MCP Server 未加载，正在自动加载", zap.String("server_code", serverCode))
		var err error
		server, err = m.LoadServer(serverCode)
		if err != nil {
			return "", fmt.Errorf("自动加载 MCP Server '%s' 失败: %w", serverCode, err)
		}
		m.logger.Info("MCP Server 自动加载成功", zap.String("server_code", serverCode))
	}

	// 查找工具
	for _, t := range server.Tools {
		if mcpTool, ok := t.(*MCPTool); ok && mcpTool.toolName == toolName {
			paramsJSON, err := json.Marshal(params)
			if err != nil {
				return "", fmt.Errorf("序列化参数失败: %w", err)
			}
			return mcpTool.InvokableRun(ctx, string(paramsJSON))
		}
	}

	return "", fmt.Errorf("工具 '%s' 在 MCP Server '%s' 中不存在", toolName, serverCode)
}
