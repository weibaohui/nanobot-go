import client from './client';
import type { ApiResponse, ListResponse, LLMProvider, CreateProviderRequest, UpdateProviderRequest, EmbeddingModelInfo } from '../types';

export const providersApi = {
  // 获取 Provider 列表
  list: (userCode?: string, page: number = 1, pageSize: number = 20) =>
    client.get<any, ApiResponse<ListResponse<LLMProvider>>>('/providers', {
      params: { user_code: userCode, offset: (page - 1) * pageSize, limit: pageSize },
    }),

  // 获取单个 Provider
  get: (id: number) =>
    client.get<any, ApiResponse<LLMProvider>>(`/providers/${id}`),

  // 创建 Provider
  create: (userCode: string, data: CreateProviderRequest) =>
    client.post<any, ApiResponse<LLMProvider>>('/providers', data, {
      params: { user_code: userCode },
    }),

  // 更新 Provider
  update: (id: number, data: UpdateProviderRequest) =>
    client.put<any, ApiResponse<LLMProvider>>(`/providers/${id}`, data),

  // 删除 Provider
  delete: (id: number) =>
    client.delete<any, ApiResponse<void>>(`/providers/${id}`),

  // 获取默认 Provider
  getDefault: (userCode: string) =>
    client.get<any, ApiResponse<LLMProvider>>(`/users/${userCode}/default-provider`),

  // 设置默认 Provider
  setDefault: (userCode: string, providerId: number) =>
    client.post<any, ApiResponse<void>>(`/users/${userCode}/default-provider`, { provider_id: providerId }),

  // 测试连接
  testConnection: (id: number) =>
    client.post<any, ApiResponse<{ success: boolean; message?: string }>>(`/providers/${id}/test`),

  // 获取嵌入模型配置
  getEmbeddingModels: (id: number) =>
    client.get<any, ApiResponse<{
      embedding_models: EmbeddingModelInfo[];
      default_embedding_model: string;
      has_embedding_models: boolean;
    }>>(`/providers/${id}/embedding`),

  // 更新嵌入模型配置
  updateEmbeddingModels: (id: number, data: {
    embedding_models: EmbeddingModelInfo[];
    default_embedding_model: string;
  }) =>
    client.put<any, ApiResponse<void>>(`/providers/${id}/embedding`, data),
};
