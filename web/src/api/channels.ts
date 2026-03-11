import client from './client';
import type { ApiResponse, ListResponse, Channel, CreateChannelRequest, UpdateChannelRequest } from '../types';

export const channelsApi = {
  // 获取 Channel 列表
  list: (userId?: number, page: number = 1, pageSize: number = 20) =>
    client.get<any, ApiResponse<ListResponse<Channel>>>('/channels', {
      params: { user_id: userId, offset: (page - 1) * pageSize, limit: pageSize },
    }),

  // 获取单个 Channel
  get: (id: number) =>
    client.get<any, ApiResponse<Channel>>(`/channels/${id}`),

  // 创建 Channel
  create: (userId: number, data: CreateChannelRequest) =>
    client.post<any, ApiResponse<Channel>>('/channels', data, {
      params: { user_id: userId },
    }),

  // 更新 Channel
  update: (id: number, data: UpdateChannelRequest) =>
    client.put<any, ApiResponse<Channel>>(`/channels/${id}`, data),

  // 删除 Channel
  delete: (id: number) =>
    client.delete<any, ApiResponse<void>>(`/channels/${id}`),

  // 绑定 Agent
  bindAgent: (channelId: number, agentId: number) =>
    client.post<any, ApiResponse<void>>(`/channels/${channelId}/bind-agent`, { agent_id: agentId }),

  // 解绑 Agent
  unbindAgent: (channelId: number) =>
    client.post<any, ApiResponse<void>>(`/channels/${channelId}/unbind-agent`),

  // 获取 Channel 配置
  getConfig: (id: number) =>
    client.get<any, ApiResponse<Record<string, any>>>(`/channels/${id}/config`),

  // 更新 Channel 配置
  updateConfig: (id: number, config: Record<string, any>) =>
    client.put<any, ApiResponse<void>>(`/channels/${id}/config`, config),

  // 获取白名单
  getAllowList: (id: number) =>
    client.get<any, ApiResponse<string[]>>(`/channels/${id}/allow-list`),

  // 设置白名单
  setAllowList: (id: number, allowList: string[]) =>
    client.put<any, ApiResponse<void>>(`/channels/${id}/allow-list`, { allow_list: allowList }),
};
