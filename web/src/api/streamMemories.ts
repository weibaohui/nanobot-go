import client from './client';
import type { StreamMemory } from '../types';

export const streamMemoriesApi = {
  list: (params?: { user_code?: string; agent_code?: string }) => client.get('/stream-memories', { params }),
  getById: (id: number) => client.get(`/stream-memories/${id}`),
  create: (data: Partial<StreamMemory>) => client.post('/stream-memories', data),
  update: (id: number, data: Partial<StreamMemory>) => client.put(`/stream-memories/${id}`, data),
  delete: (id: number) => client.delete(`/stream-memories/${id}`),
  getUnprocessed: () => client.get('/stream-memories/unprocessed'),
  markProcessed: (id: number) => client.put(`/stream-memories/${id}`, { processed: true }),
  upgrade: (date: string) => client.post(`/stream-memories/upgrade?date=${date}`),
  // 从对话记录构建短期记忆
  build: (data: {
    user_code: string;
    agent_code?: string;
    date: string;
    conversation_ids: string[];
    contents: string[];
  }) => client.post('/stream-memories/build', data),
};
