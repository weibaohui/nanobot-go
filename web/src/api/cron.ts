import client from './client';
import type { ApiResponse, ListResponse, CronJob, CreateCronJobRequest, UpdateCronJobRequest } from '../types';

export const cronApi = {
  // 获取 Cron Job 列表
  list: (userCode?: string, channelCode?: string, page: number = 1, pageSize: number = 20) =>
    client.get<any, ApiResponse<ListResponse<CronJob>>>('/cron-jobs', {
      params: { user_code: userCode, channel_code: channelCode, offset: (page - 1) * pageSize, limit: pageSize },
    }),

  // 获取单个 Cron Job
  get: (id: number) =>
    client.get<any, ApiResponse<CronJob>>(`/cron-jobs/${id}`),

  // 创建 Cron Job
  create: (userCode: string, channelCode: string, data: CreateCronJobRequest) =>
    client.post<any, ApiResponse<CronJob>>('/cron-jobs', data, {
      params: { user_code: userCode, channel_code: channelCode },
    }),

  // 更新 Cron Job
  update: (id: number, data: UpdateCronJobRequest) =>
    client.put<any, ApiResponse<CronJob>>(`/cron-jobs/${id}`, data),

  // 删除 Cron Job
  delete: (id: number) =>
    client.delete<any, ApiResponse<void>>(`/cron-jobs/${id}`),

  // 启用
  enable: (id: number) =>
    client.post<any, ApiResponse<void>>(`/cron-jobs/${id}/enable`),

  // 禁用
  disable: (id: number) =>
    client.post<any, ApiResponse<void>>(`/cron-jobs/${id}/disable`),

  // 立即执行
  execute: (id: number) =>
    client.post<any, ApiResponse<void>>(`/cron-jobs/${id}/execute`),

  // 获取待执行任务
  getPending: () =>
    client.get<any, ApiResponse<CronJob[]>>('/cron-jobs/pending'),
};
