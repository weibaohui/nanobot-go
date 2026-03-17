import client from './client';
import type { ListResponse, Task, TaskDetail } from '../types';

export const tasksApi = {
  // 获取所有任务列表
  list: () =>
    client.get<any, ListResponse<Task>>('/tasks'),

  // 获取单个任务详情
  get: (id: string) =>
    client.get<any, TaskDetail>(`/tasks/${id}`),

  // 停止任务
  stop: (id: string) =>
    client.post<any, { message: string; data: Task }>(`/tasks/${id}/stop`),
};
