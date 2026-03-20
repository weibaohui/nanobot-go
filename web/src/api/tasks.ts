import client from './client';
import type { ListResponse, Task, TaskDetail } from '../types';

export const tasksApi = {
  // 获取任务列表（支持筛选参数）
  list: (params?: string) =>
    client.get<any, ListResponse<Task>>(`/tasks${params ? `?${params}` : ''}`),

  // 获取单个任务详情
  get: (id: string) =>
    client.get<any, TaskDetail>(`/tasks/${id}`),

  // 创建任务
  create: (data: { work: string }) =>
    client.post<any, { message: string; data: Task }>('/tasks', data),

  // 停止任务
  stop: (id: string) =>
    client.post<any, { message: string; data: Task }>(`/tasks/${id}/stop`),

  // 重试任务
  retry: (id: string) =>
    client.post<any, { message: string; data: Task }>(`/tasks/${id}/retry`),
};
