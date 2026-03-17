package mcp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
	"gorm.io/datatypes"
)

// ListTools 获取 MCP Server 的工具列表
func (s *service) ListTools(serverID uint) ([]models.MCPToolModel, error) {
	// 检查 MCP Server 是否存在
	server, err := s.mcpServerRepo.GetByID(serverID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("MCP 服务器不存在")
	}

	return s.mcpToolRepo.ListByServerID(serverID)
}

// GetTool 获取单个工具
func (s *service) GetTool(toolID uint) (*models.MCPToolModel, error) {
	return s.mcpToolRepo.GetByID(toolID)
}

// ListToolLogs 获取 MCP Server 的调用日志
func (s *service) ListToolLogs(serverID uint, limit int) ([]models.MCPToolLog, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.mcpToolLogRepo.ListByServerID(serverID, limit)
}

// LogToolExecution 记录工具调用日志
func (s *service) LogToolExecution(sessionKey string, serverID uint, toolName string, params interface{}, result string, errMsg string, executeTimeMs int64) error {
	var paramsJSON datatypes.JSON
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			paramsJSON = nil
		} else {
			paramsJSON = data
		}
	}

	log := &models.MCPToolLog{
		SessionKey:   sessionKey,
		MCPServerID:  serverID,
		ToolName:     toolName,
		Parameters:   paramsJSON,
		Result:       result,
		ErrorMessage: errMsg,
		ExecuteTime:  uint(executeTimeMs),
		CreatedAt:    time.Now(),
	}

	return s.mcpToolLogRepo.Create(log)
}

// RefreshCapabilitiesWithStorage 刷新 MCP 服务器能力并存储到 mcp_tools 表
// 这个方法会删除旧工具，重新插入新工具
func (s *service) RefreshCapabilitiesWithStorage(id uint) error {
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

	// 1. 删除旧工具
	if err := s.mcpToolRepo.DeleteByServerID(id); err != nil {
		return fmt.Errorf("删除旧工具失败: %w", err)
	}

	// 2. 创建新工具记录
	for _, cap := range capabilities {
		// 将 map 转换为 JSON
		inputSchemaJSON, _ := json.Marshal(cap.InputSchema)
		tool := &models.MCPToolModel{
			MCPServerID: id,
			Name:        cap.Name,
			Description: cap.Description,
			InputSchema: inputSchemaJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.mcpToolRepo.Create(tool); err != nil {
			// 记录错误但继续
			continue
		}
	}

	// 3. 同时更新 capabilities 字段（兼容旧逻辑）
	if err := server.SetCapabilities(capabilities); err != nil {
		return fmt.Errorf("设置能力列表失败: %w", err)
	}

	// 4. 更新服务器状态
	server.UpdatedAt = time.Now()
	return s.mcpServerRepo.Update(server)
}
