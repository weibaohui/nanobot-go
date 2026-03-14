import client from './client';
import type { ApiResponse, ListResponse, User, CreateUserRequest, UpdateUserRequest } from '../types';

export const usersApi = {
  // 获取用户列表
  list: (page: number = 1, pageSize: number = 20) =>
    client.get<any, ApiResponse<ListResponse<User>>>('/users', {
      params: { offset: (page - 1) * pageSize, limit: pageSize },
    }),

  // 获取单个用户
  get: (id: number) =>
    client.get<any, ApiResponse<User>>(`/users/${id}`),

  // 根据 Code 获取用户
  getByCode: (code: string) =>
    client.get<any, ApiResponse<User>>(`/users/code/${code}`),

  // 创建用户
  create: (data: CreateUserRequest) =>
    client.post<any, ApiResponse<User>>('/users', data),

  // 更新用户
  update: (id: number, data: UpdateUserRequest) =>
    client.put<any, ApiResponse<User>>(`/users/${id}`, data),

  // 删除用户
  delete: (id: number) =>
    client.delete<any, ApiResponse<void>>(`/users/${id}`),

  // 修改密码
  changePassword: (id: number, oldPassword: string, newPassword: string) =>
    client.post<any, ApiResponse<void>>(`/users/${id}/change-password`, {
      old_password: oldPassword,
      new_password: newPassword,
    }),
};
