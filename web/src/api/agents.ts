import client from './client';
import type { ApiResponse, ListResponse, Agent, CreateAgentRequest, UpdateAgentRequest } from '../types';

export const agentsApi = {
  // 获取 Agent 列表
  list: (userCode?: string, page: number = 1, pageSize: number = 20) =>
    client.get<any, ApiResponse<ListResponse<Agent>>>('/agents', {
      params: { user_code: userCode, offset: (page - 1) * pageSize, limit: pageSize },
    }),

  // 获取单个 Agent
  get: (id: number) =>
    client.get<any, ApiResponse<Agent>>(`/agents/${id}`),

  // 根据 Code 获取 Agent
  getByCode: (code: string) =>
    client.get<any, ApiResponse<Agent>>(`/agents/code/${code}`),

  // 创建 Agent
  create: (userCode: string, data: CreateAgentRequest) =>
    client.post<any, ApiResponse<Agent>>('/agents', data, {
      params: { user_code: userCode },
    }),

  // 更新 Agent
  update: (id: number, data: UpdateAgentRequest) =>
    client.put<any, ApiResponse<Agent>>(`/agents/${id}`, data),

  // 删除 Agent
  delete: (id: number) =>
    client.delete<any, ApiResponse<void>>(`/agents/${id}`),

  // 获取默认 Agent
  getDefault: (userCode: string) =>
    client.get<any, ApiResponse<Agent>>(`/users/${userCode}/default-agent`),

  // 设置默认 Agent
  setDefault: (userCode: string, agentCode: string) =>
    client.post<any, ApiResponse<void>>(`/users/${userCode}/default-agent`, { agent_code: agentCode }),

  // 获取 Agent 配置
  getConfig: (id: number) =>
    client.get<any, ApiResponse<any>>(`/agents/${id}/config`),

  // 更新 Agent 配置
  updateConfig: (id: number, config: any) =>
    client.put<any, ApiResponse<void>>(`/agents/${id}/config`, config),

  // 获取模型配置
  getModelConfig: (id: number) =>
    client.get<any, ApiResponse<any>>(`/agents/${id}/model-config`),

  // 更新模型配置
  updateModelConfig: (id: number, config: { selection_mode: 'auto' | 'specific'; model_id?: string; model_name?: string; max_tokens?: number; temperature?: number }) =>
    client.put<any, ApiResponse<Agent>>(`/agents/${id}/model-config`, config),
};
