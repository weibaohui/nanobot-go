import client from './client';
import type { ApiResponse, ListResponse, MCPServer, CreateMCPServerRequest, UpdateMCPServerRequest, AgentMCPBinding, CreateAgentMCPBindingRequest, UpdateAgentMCPBindingRequest, MCPTool } from '../types';

export const mcpServersApi = {
  // 获取 MCP Server 列表
  list: () =>
    client.get<any, ListResponse<MCPServer>>('/mcp-servers'),

  // 获取单个 MCP Server
  get: (id: number) =>
    client.get<any, ApiResponse<MCPServer>>(`/mcp-servers/${id}`),

  // 创建 MCP Server
  create: (data: CreateMCPServerRequest) =>
    client.post<any, ApiResponse<MCPServer>>('/mcp-servers', data),

  // 更新 MCP Server
  update: (id: number, data: UpdateMCPServerRequest) =>
    client.put<any, ApiResponse<MCPServer>>(`/mcp-servers/${id}`, data),

  // 删除 MCP Server
  delete: (id: number) =>
    client.delete<any, ApiResponse<void>>(`/mcp-servers/${id}`),

  // 测试 MCP Server 连接
  test: (id: number) =>
    client.post<any, ApiResponse<void>>(`/mcp-servers/${id}/test`),

  // 刷新 MCP Server 能力
  refreshCapabilities: (id: number) =>
    client.post<any, ApiResponse<void>>(`/mcp-servers/${id}/refresh`),

  // 获取 MCP Server 的工具列表
  listTools: (id: number) =>
    client.get<any, ListResponse<MCPTool>>(`/mcp-servers/${id}/tools`),

  // 获取 Agent 的 MCP 绑定列表
  getAgentBindings: (agentId: number) =>
    client.get<any, ListResponse<AgentMCPBinding>>(`/agents/${agentId}/mcp-bindings`),

  // 创建 Agent MCP 绑定
  createAgentBinding: (agentId: number, data: CreateAgentMCPBindingRequest) =>
    client.post<any, ApiResponse<AgentMCPBinding>>(`/agents/${agentId}/mcp-bindings`, data),

  // 获取单个 Agent MCP 绑定
  getAgentBinding: (agentId: number, bindingId: number) =>
    client.get<any, ApiResponse<AgentMCPBinding>>(`/agents/${agentId}/mcp-bindings/${bindingId}`),

  // 更新 Agent MCP 绑定
  updateAgentBinding: (agentId: number, bindingId: number, data: UpdateAgentMCPBindingRequest) =>
    client.put<any, ApiResponse<AgentMCPBinding>>(`/agents/${agentId}/mcp-bindings/${bindingId}`, data),

  // 删除 Agent MCP 绑定
  deleteAgentBinding: (agentId: number, bindingId: number) =>
    client.delete<any, ApiResponse<void>>(`/agents/${agentId}/mcp-bindings/${bindingId}`),

  // 获取 Agent 的 MCP 工具列表
  getAgentMCPTools: (agentId: number) =>
    client.get<any, ApiResponse<MCPTool[]>>(`/agents/${agentId}/mcp-bindings/tools`),
};
