import client from './client';
import type { StreamMemory } from '../types';

export const streamMemoriesApi = {
  list: () => client.get('/stream-memories'),
  getById: (id: number) => client.get(`/stream-memories/${id}`),
  create: (data: Partial<StreamMemory>) => client.post('/stream-memories', data),
  update: (id: number, data: Partial<StreamMemory>) => client.put(`/stream-memories/${id}`, data),
  delete: (id: number) => client.delete(`/stream-memories/${id}`),
  getUnprocessed: () => client.get('/stream-memories/unprocessed'),
  markProcessed: (id: number) => client.put(`/stream-memories/${id}`, { processed: true }),
};
