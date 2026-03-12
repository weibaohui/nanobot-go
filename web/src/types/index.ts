// 用户类型
export interface User {
  id: number;
  username: string;
  email?: string;
  display_name?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateUserRequest {
  username: string;
  email?: string;
  password: string;
  display_name?: string;
}

export interface UpdateUserRequest {
  email?: string;
  display_name?: string;
  is_active?: boolean;
}

// Agent 类型
export interface Agent {
  id: number;
  user_id: number;
  name: string;
  description?: string;
  identity_content?: string;
  soul_content?: string;
  agents_content?: string;
  user_content?: string;
  tools_content?: string;
  memory_content?: string;
  memory_summary?: string;
  skills_list?: string;
  tools_list?: string;
  model_selection_mode: 'auto' | 'specific';
  model_id?: string;
  model_name?: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  is_active: boolean;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAgentRequest {
  name: string;
  description?: string;
  identity_content?: string;
  soul_content?: string;
  agents_content?: string;
  user_content?: string;
  tools_content?: string;
  model_selection_mode?: 'auto' | 'specific';
  model_id?: string;
  model_name?: string;
  max_tokens?: number;
  temperature?: number;
  max_iterations?: number;
  skills_list?: string[];
  tools_list?: string[];
  is_default?: boolean;
}

export interface UpdateAgentRequest extends Partial<CreateAgentRequest> {
  is_active?: boolean;
}

// Channel 类型
export type ChannelType = 'feishu' | 'dingtalk' | 'matrix' | 'websocket';

export const ChannelTypeLabels: Record<ChannelType, string> = {
  feishu: '飞书',
  dingtalk: '钉钉',
  matrix: 'Matrix',
  websocket: 'WebSocket',
};

export interface Channel {
  id: number;
  user_id: number;
  agent_id?: number;
  name: string;
  type: ChannelType;
  is_active: boolean;
  allow_from?: string;
  config?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateChannelRequest {
  name: string;
  type: ChannelType;
  config?: Record<string, any>;
  allow_from?: string[];
  agent_id?: number;
}

export interface UpdateChannelRequest {
  name?: string;
  config?: Record<string, any>;
  allow_from?: string[];
  is_active?: boolean;
  agent_id?: number;
}

// LLM Provider 类型
export interface LLMProvider {
  id: number;
  user_id: number;
  provider_key: string;
  provider_name?: string;
  api_base?: string;
  extra_headers?: string;
  supported_models?: string;
  is_default: boolean;
  priority: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateProviderRequest {
  provider_key: string;
  provider_name?: string;
  api_key?: string;
  api_base?: string;
  extra_headers?: Record<string, string>;
  supported_models?: ModelInfo[];
  is_default?: boolean;
  priority?: number;
}

export interface UpdateProviderRequest extends Partial<CreateProviderRequest> {
  is_active?: boolean;
}

export interface ModelInfo {
  id: string;
  name: string;
  max_tokens?: number;
}

// Cron Job 类型
export interface CronJob {
  id: number;
  user_id: number;
  channel_id: number;
  name: string;
  description?: string;
  cron_expression: string;
  timezone: string;
  prompt: string;
  model_selection_mode: 'auto' | 'specific';
  model_id?: string;
  model_name?: string;
  target_channel_id?: number;
  target_user_id?: string;
  is_active: boolean;
  last_run_at?: string;
  last_run_status?: 'success' | 'failed' | 'running';
  last_run_result?: string;
  next_run_at?: string;
  run_count: number;
  fail_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCronJobRequest {
  name: string;
  description?: string;
  channel_id: number;
  cron_expression: string;
  timezone?: string;
  prompt: string;
  model_selection_mode?: 'auto' | 'specific';
  model_id?: string;
  model_name?: string;
  target_channel_id?: number;
  target_user_id?: string;
}

export interface UpdateCronJobRequest extends Partial<CreateCronJobRequest> {
  is_active?: boolean;
}

// API 响应类型
export interface ApiResponse<T> {
  code: number;
  message?: string;
  data: T;
}

export interface ListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// Conversation Record 类型
export interface ConversationRecord {
  id: number;
  trace_id: string;
  span_id: string;
  session_key: string;
  event_type: string;
  role?: string;
  content: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  user_id?: string;
  agent_id?: string;
  channel_id?: string;
  timestamp: string;
}

// Stream Memory 类型
export interface StreamMemory {
  id: number;
  trace_id: string;
  session_key: string;
  channel_type: string;
  event_type: string;
  content: string;
  summary?: string;
  processed: boolean;
  processed_at?: string;
  created_at: string;
}

// Long-term Memory 类型
export interface LongTermMemory {
  id: number;
  memory_date: string;
  content: string;
  summary?: string;
  tags?: string;
  created_at: string;
  updated_at: string;
}
