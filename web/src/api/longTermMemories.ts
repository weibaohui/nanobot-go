import client from './client';
import type { LongTermMemory } from '../types';

export const longTermMemoriesApi = {
  list: () => client.get('/long-term-memories'),
  getById: (id: number) => client.get(`/long-term-memories/${id}`),
  getByDate: (date: string) => client.get(`/long-term-memories/date/${date}`),
  create: (data: Partial<LongTermMemory>) => client.post('/long-term-memories', data),
  update: (id: number, data: Partial<LongTermMemory>) => client.put(`/long-term-memories/${id}`, data),
  delete: (id: number) => client.delete(`/long-term-memories/${id}`),
  search: (query: string) => client.get(`/long-term-memories/search?q=${encodeURIComponent(query)}`),
  getRecent: (days: number = 7) => client.get(`/long-term-memories/recent?days=${days}`),
};
