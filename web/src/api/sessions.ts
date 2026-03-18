import client from './client';

export const sessionsApi = {
  // 获取会话列表
  list: (params?: { user_code?: string; channel_code?: string }) => {
    const queryParams = new URLSearchParams();
    if (params?.user_code) queryParams.append('user_code', params.user_code);
    if (params?.channel_code) queryParams.append('channel_code', params.channel_code);
    const query = queryParams.toString();
    return client.get(`/sessions${query ? `?${query}` : ''}`);
  },

  // 获取会话详情
  getById: (sessionKey: string) => client.get(`/sessions/${sessionKey}`),

  // 创建会话
  create: (data: {
    user_code: string;
    channel_code: string;
    session_key: string;
    agent_code?: string;
    external_id?: string;
    metadata?: Record<string, any>;
  }) => client.post('/sessions', data),

  // 删除会话
  delete: (sessionKey: string) => client.delete(`/sessions/${sessionKey}`),

  // 更新会话活跃时间
  touch: (sessionKey: string) => client.post(`/sessions/${sessionKey}/touch`),

  // 获取会话元数据
  getMetadata: (sessionKey: string) => client.get(`/sessions/${sessionKey}/metadata`),

  // 更新会话元数据
  updateMetadata: (sessionKey: string, metadata: Record<string, any>) =>
    client.put(`/sessions/${sessionKey}/metadata`, metadata),

  // 取消正在执行的会话
  cancel: (sessionKey: string) => client.post(`/sessions/${sessionKey}/cancel`),

  // 检查会话是否活跃（正在执行中）
  checkActive: (sessionKey: string) => client.get(`/sessions/${sessionKey}/active`),
};
