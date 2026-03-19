// 用户类型
export interface User {
  id: number;
  user_code: string;
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
  agent_code: string;
  user_code: string;
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
  mcp_list?: string;
  model_selection_mode: 'auto' | 'specific';
  model_id?: string;
  model_name?: string;
  max_tokens: number;
  temperature: number;
  max_iterations: number;
  is_active: boolean;
  is_default: boolean;
  enable_thinking_process: boolean;
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
  mcp_list?: string[];
  is_default?: boolean;
  enable_thinking_process?: boolean;
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
  channel_code: string;
  user_code: string;
  agent_code?: string;
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
  agent_code?: string;
}

export interface UpdateChannelRequest {
  name?: string;
  config?: Record<string, any>;
  allow_from?: string[];
  is_active?: boolean;
  agent_code?: string;
}

// LLM Provider 类型
export interface LLMProvider {
  id: number;
  user_code: string;
  provider_key: string;
  provider_name?: string;
  api_base?: string;
  extra_headers?: string;
  supported_models?: string;
  embedding_models?: string;
  default_embedding_model?: string;
  default_model?: string;
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
  default_model?: string;
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

// Embedding Model 类型
export interface EmbeddingModelInfo {
  id: string;
  name: string;
  dimensions: number;
}

// Cron Job 类型
export interface CronJob {
  id: number;
  user_code: string;
  channel_code: string;
  name: string;
  description?: string;
  cron_expression: string;
  timezone: string;
  prompt: string;
  model_selection_mode: 'auto' | 'specific';
  model_id?: string;
  model_name?: string;
  target_channel_code?: string;
  target_user_code?: string;
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
  channel_code: string;
  cron_expression: string;
  timezone?: string;
  prompt: string;
  model_selection_mode?: 'auto' | 'specific';
  model_id?: string;
  model_name?: string;
  target_channel_code?: string;
  target_user_code?: string;
}

export interface UpdateCronJobRequest extends Partial<CreateCronJobRequest> {
  is_active?: boolean;
}

// 认证类型
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
  expires_at: number;
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
  parent_span_id?: string;
  session_key: string;
  event_type: string;
  role?: string;
  content: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  user_code?: string;
  agent_code?: string;
  channel_code?: string;
  channel_type?: string;
  agent_name?: string;
  channel_name?: string;
  timestamp: string;
}

// Session 类型
export interface Session {
  id: number;
  session_key: string;
  user_code: string;
  channel_code: string;
  agent_code?: string;
  external_id?: string;
  metadata?: Record<string, any>;
  last_active_at?: string;
  created_at: string;
  updated_at: string;
}

// Skill 技能类型
export interface Skill {
  name: string;
  description: string;
  source: string;
}

export interface SkillDetail extends Skill {
  content: string;
  bound_agents: Agent[];
}

// MCP Server 传输类型
export type MCPTransportType = 'stdio' | 'http' | 'sse';

export const MCPTransportTypeLabels: Record<MCPTransportType, string> = {
  stdio: 'stdio（标准输入输出）',
  http: 'HTTP',
  sse: 'SSE（服务器发送事件）',
};

// MCP Server 状态
export type MCPStatus = 'inactive' | 'active' | 'error';

export const MCPStatusLabels: Record<MCPStatus, string> = {
  inactive: '未连接',
  active: '已连接',
  error: '错误',
};

// MCP Tool 定义
export interface MCPTool {
  name: string;
  description?: string;
  input_schema?: Record<string, any>;
}

// MCP Server 类型
export interface MCPServer {
  id: number;
  code: string;
  name: string;
  description?: string;
  transport_type: MCPTransportType;
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
  status: MCPStatus;
  error_message?: string;
  capabilities?: MCPTool[];
  last_connected_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateMCPServerRequest {
  code: string;
  name: string;
  description?: string;
  transport_type: MCPTransportType;
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
}

export interface UpdateMCPServerRequest extends Partial<CreateMCPServerRequest> {}

// Agent MCP 绑定
export interface AgentMCPBinding {
  id: number;
  agent_id: number;
  mcp_server_id: number;
  mcp_server?: MCPServer;
  enabled_tools?: string[];
  is_active: boolean;
  auto_load: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAgentMCPBindingRequest {
  mcp_server_id: number;
  enabled_tools?: string[];
  is_active?: boolean;
  auto_load?: boolean;
}

export interface UpdateAgentMCPBindingRequest {
  enabled_tools?: string[];
  is_active?: boolean;
  auto_load?: boolean;
}

// Task 后台任务类型
export type TaskStatus = 'pending' | 'running' | 'finished' | 'failed' | 'stopped';

export const TaskStatusLabels: Record<TaskStatus, string> = {
  pending: '等待中',
  running: '运行中',
  finished: '已完成',
  failed: '失败',
  stopped: '已停止',
};

export const TaskStatusColors: Record<TaskStatus, string> = {
  pending: 'default',
  running: 'processing',
  finished: 'success',
  failed: 'error',
  stopped: 'warning',
};

export interface Task {
  id: string;
  status: TaskStatus;
  work: string;
  channel?: string;
  chat_id?: string;
  created_at: string;
  completed_at?: string;
  result?: string;
}

export interface TaskDetail extends Task {
  logs: string[];
}


// MCP Server Tool 列表项 (用于展示)
export interface MCPToolItem {
  id: number;
  mcp_server_id: number;
  name: string;
  description: string;
  input_schema?: Record<string, any> | null;
  created_at?: string;
  updated_at?: string;
}

// MCP Tool 调用日志
export interface MCPToolLog {
  id: number;
  session_key: string;
  mcp_server_id: number;
  tool_name: string;
  parameters?: Record<string, any>;
  result?: string;
  error_message?: string;
  execute_time: number;
  created_at: string;
}
