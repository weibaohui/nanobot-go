import client from './client';
import type { ConversationRecord } from '../types';

export const conversationsApi = {
  list: () => client.get('/conversations'),
  getById: (id: number) => client.get(`/conversations/${id}`),
  create: (data: Partial<ConversationRecord>) => client.post('/conversations', data),
  update: (id: number, data: Partial<ConversationRecord>) => client.put(`/conversations/${id}`, data),
  delete: (id: number) => client.delete(`/conversations/${id}`),
  getBySession: (sessionKey: string) => client.get(`/conversations/session/${sessionKey}`),
  getByTrace: (traceID: string) => client.get(`/conversations/trace/${traceID}`),
};
