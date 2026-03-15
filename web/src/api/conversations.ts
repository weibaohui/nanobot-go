import client from './client';
import type { ConversationRecord } from '../types';

interface StatsParams {
  start_time?: string;
  end_time?: string;
  agent_codes?: string;
  channel_codes?: string;
  roles?: string;
}

export const conversationsApi = {
  list: () => client.get('/conversations'),
  getById: (id: number) => client.get(`/conversations/${id}`),
  create: (data: Partial<ConversationRecord>) => client.post('/conversations', data),
  update: (id: number, data: Partial<ConversationRecord>) => client.put(`/conversations/${id}`, data),
  delete: (id: number) => client.delete(`/conversations/${id}`),
  getBySession: (sessionKey: string) => client.get(`/conversations/session/${sessionKey}`),
  getByTrace: (traceID: string) => client.get(`/conversations/trace/${traceID}`),
  getByUserAndDate: (userCode: string, date: string) => client.get(`/conversations/user/${userCode}/date/${date}`),
  getStats: (params: StatsParams) => {
    const queryParams = new URLSearchParams();
    if (params.start_time) queryParams.append('start_time', params.start_time);
    if (params.end_time) queryParams.append('end_time', params.end_time);
    if (params.agent_codes) queryParams.append('agent_codes', params.agent_codes);
    if (params.channel_codes) queryParams.append('channel_codes', params.channel_codes);
    if (params.roles) queryParams.append('roles', params.roles);
    return client.get(`/conversations/stats?${queryParams.toString()}`);
  },
};
