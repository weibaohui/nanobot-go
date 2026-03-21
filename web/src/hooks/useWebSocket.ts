import { useEffect, useRef, useState, useCallback } from 'react';
import type { TaskStatus } from '../types';

// WebSocket 消息类型
export type WebSocketMessageType =
  | 'ping' | 'pong' | 'message' | 'chunk' | 'error' | 'system'
  | 'task_created' | 'task_updated' | 'task_completed' | 'task_log';

export interface WebSocketMessage {
  type: WebSocketMessageType;
  payload: any;
  timestamp: number;
}

export interface ChunkPayload {
  content: string;
  session_id: string;
  is_end: boolean;
}

export interface ErrorPayload {
  code: string;
  message: string;
}

export interface SystemPayload {
  type?: string;
  session_id?: string;
  message?: string;
}

// Task 事件 Payload 类型
export interface TaskCreatedPayload {
  id: string;
  status: TaskStatus;
  work: string;
  channel?: string;
  chat_id?: string;
  created_at: string;
  created_by?: string;
}

export interface TaskUpdatedPayload {
  id: string;
  status: TaskStatus;
  result?: string;
  logs?: string[];
  updated_at: string;
}

export interface TaskCompletedPayload {
  id: string;
  status: 'finished' | 'failed' | 'stopped';
  result?: string;
  logs?: string[];
  completed_at: string;
  duration_seconds: number;
}

export interface UseWebSocketOptions {
  channelCode: string;
  token: string;
  enabled?: boolean; // 是否启用连接（默认为 true）
  onMessage?: (msg: WebSocketMessage) => void;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  // Task 事件回调
  onTaskCreated?: (payload: TaskCreatedPayload) => void;
  onTaskUpdated?: (payload: TaskUpdatedPayload) => void;
  onTaskCompleted?: (payload: TaskCompletedPayload) => void;
}

export interface UseWebSocketReturn {
  isConnected: boolean;
  isConnecting: boolean;
  sendMessage: (payload: { content: string; user_code?: string; session_id?: string }) => boolean;
  disconnect: () => void;
  connect: () => void;
}

export function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn {
  const {
    channelCode,
    token,
    enabled = true,
    onMessage,
    onError,
    onConnect,
    onDisconnect,
    onTaskCreated,
    onTaskUpdated,
    onTaskCompleted,
  } = options;
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const heartbeatTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const reconnectCountRef = useRef(0);
  const manualCloseRef = useRef(false);
  const maxReconnectCount = 5;

  const clearTimers = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current);
      heartbeatTimerRef.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    manualCloseRef.current = false;
    if (!enabled) {
      console.log('[WebSocket] 连接被禁用');
      return;
    }
    if (!channelCode || !token) {
      console.warn('[WebSocket] 缺少 channelCode 或 token');
      return;
    }

    // 如果已经连接，先断开
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setIsConnecting(true);

    // 构建 WebSocket URL
    // 使用相对路径，通过 vite 代理转发到后端
    const wsUrl = `/ws/chat?channel_code=${channelCode}&token=${token}`;

    console.log('[WebSocket] 连接到:', wsUrl);

    try {
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('[WebSocket] 连接成功');
        manualCloseRef.current = false;
        setIsConnected(true);
        setIsConnecting(false);
        reconnectCountRef.current = 0;
        onConnect?.();

        // 启动心跳
        heartbeatTimerRef.current = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'ping', timestamp: Date.now() }));
          }
        }, 30000);
      };

      ws.onmessage = (event) => {
        try {
          const msg: WebSocketMessage = JSON.parse(event.data);

          // 处理心跳响应
          if (msg.type === 'pong') {
            return;
          }

          // 处理 Task 事件
          switch (msg.type) {
            case 'task_created':
              onTaskCreated?.(msg.payload as TaskCreatedPayload);
              break;
            case 'task_updated':
              onTaskUpdated?.(msg.payload as TaskUpdatedPayload);
              break;
            case 'task_completed':
              onTaskCompleted?.(msg.payload as TaskCompletedPayload);
              break;
          }

          onMessage?.(msg);
        } catch (e) {
          console.error('[WebSocket] 解析消息失败:', e);
          onError?.(new Error('消息格式错误'));
        }
      };

      ws.onerror = (error) => {
        console.error('[WebSocket] 连接错误:', error);
        onError?.(new Error('WebSocket 连接错误'));
      };

      ws.onclose = (event) => {
        console.log('[WebSocket] 连接关闭:', event.code, event.reason);
        setIsConnected(false);
        setIsConnecting(false);
        onDisconnect?.();
        clearTimers();

        if (manualCloseRef.current) {
          manualCloseRef.current = false;
          return;
        }

        // 自动重连
        if (reconnectCountRef.current < maxReconnectCount) {
          reconnectCountRef.current++;
          const delay = Math.min(3000 * reconnectCountRef.current, 30000);
          console.log(`[WebSocket] ${delay}ms 后尝试第 ${reconnectCountRef.current} 次重连...`);
          reconnectTimerRef.current = setTimeout(connect, delay);
        } else {
          console.error('[WebSocket] 重连次数已达上限');
          onError?.(new Error('WebSocket 连接失败，请刷新页面重试'));
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('[WebSocket] 创建连接失败:', error);
      onError?.(new Error('创建 WebSocket 连接失败'));
    }
  }, [channelCode, token, enabled, onMessage, onError, onConnect, onDisconnect, clearTimers]);

  const disconnect = useCallback(() => {
    clearTimers();
    reconnectCountRef.current = maxReconnectCount; // 阻止自动重连
    manualCloseRef.current = true;
    setIsConnected(false);
    setIsConnecting(false);
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, [clearTimers]);

  const sendMessage = useCallback((payload: { content: string; user_code?: string; session_id?: string }): boolean => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      console.warn('[WebSocket] 连接未建立，无法发送消息');
      return false;
    }

    const message = {
      type: 'message',
      payload,
      timestamp: Date.now(),
    };

    try {
      wsRef.current.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error('[WebSocket] 发送消息失败:', error);
      return false;
    }
  }, []);

  // 组件挂载时连接，卸载时断开
  // 当 enabled 变为 false 时断开连接，变为 true 时连接
  useEffect(() => {
    if (enabled) {
      connect();
    } else {
      disconnect();
    }
    return () => {
      disconnect();
    };
  }, [connect, disconnect, enabled]);

  return {
    isConnected,
    isConnecting,
    sendMessage,
    disconnect,
    connect,
  };
}

export default useWebSocket;
