import client from './client';
import type { ListResponse, Skill, SkillDetail } from '../types';

export const skillsApi = {
  // 获取所有技能列表
  list: () =>
    client.get<any, ListResponse<Skill>>('/skills'),

  // 获取单个技能详情
  get: (name: string) =>
    client.get<any, SkillDetail>(`/skills/${name}`),
};
