import client from './client';
import type { ApiResponse, ListResponse, LLMProvider, CreateProviderRequest, UpdateProviderRequest } from '../types';

export const providersApi = {
  // 获取 Provider 列表
  list: (userId?: number, page: number = 1, pageSize: number = 20) =>
    client.get<any, ApiResponse<ListResponse<LLMProvider>>>('/providers', {
      params: { user_id: userId, offset: (page - 1) * pageSize, limit: pageSize },
    }),

  // 获取单个 Provider
  get: (id: number) =>
    client.get<any, ApiResponse<LLMProvider>>(`/providers/${id}`),

  // 创建 Provider
  create: (userId: number, data: CreateProviderRequest) =>
    client.post<any, ApiResponse<LLMProvider>>('/providers', { ...data, user_id: userId }),

  // 更新 Provider
  update: (id: number, data: UpdateProviderRequest) =>
    client.put<any, ApiResponse<LLMProvider>>(`/providers/${id}`, data),

  // 删除 Provider
  delete: (id: number) =>
    client.delete<any, ApiResponse<void>>(`/providers/${id}`),

  // 获取默认 Provider
  getDefault: (userId: number) =>
    client.get<any, ApiResponse<LLMProvider>>(`/users/${userId}/default-provider`),

  // 设置默认 Provider
  setDefault: (userId: number, providerId: number) =>
    client.post<any, ApiResponse<void>>(`/users/${userId}/default-provider`, { provider_id: providerId }),

  // 测试连接
  testConnection: (id: number) =>
    client.post<any, ApiResponse<{ success: boolean; message?: string }>>(`/providers/${id}/test`),
};
