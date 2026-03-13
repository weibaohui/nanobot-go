import client from './client';
import type { ApiResponse, LoginRequest, LoginResponse, User } from '../types';

export const authApi = {
  // 登录
  login: (data: LoginRequest) =>
    client.post<any, ApiResponse<LoginResponse>>('/auth/login', data),

  // 获取当前用户信息
  me: () => client.get<any, ApiResponse<User>>('/auth/me'),

  // 修改密码
  changePassword: (id: number, oldPassword: string, newPassword: string) =>
    client.post<any, ApiResponse<void>>(`/users/${id}/change-password`, {
      old_password: oldPassword,
      new_password: newPassword,
    }),
};

// 存储 token
export const setToken = (token: string) => {
  localStorage.setItem('token', token);
};

// 获取 token
export const getToken = (): string | null => {
  return localStorage.getItem('token');
};

// 清除 token
export const clearToken = () => {
  localStorage.removeItem('token');
};

// 检查是否已登录
export const isAuthenticated = (): boolean => {
  return !!getToken();
};
