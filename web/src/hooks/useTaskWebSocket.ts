import { useState, useEffect, useRef, useCallback } from 'react';
import { getToken } from '../api/auth';

export interface TaskEvent {
  type: 'task_created' | 'task_updated' | 'task_completed' | 'task_failed' | 'task_cancelled' | 'pong';
  payload: {
    id?: string;
    status?: string;
    work?: string;
    channel_name?: string;
    created_at?: string;
    error?: string;
    [key: string]: any;
  };
}

interface UseTaskWebSocketOptions {
  onMessage?: (event: TaskEvent) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
}

// Global singleton connection manager
class TaskWebSocketManager {
  private static instance: TaskWebSocketManager;
  private ws: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private listeners: Set<(connected: boolean) => void> = new Set();
  private messageListeners: Set<(event: TaskEvent) => void> = new Set();
  private isConnecting = false;
  private connectionId = 0;

  static getInstance(): TaskWebSocketManager {
    if (!TaskWebSocketManager.instance) {
      TaskWebSocketManager.instance = new TaskWebSocketManager();
    }
    return TaskWebSocketManager.instance;
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  connect(): void {
    if (this.isConnected() || this.isConnecting) {
      console.log('[TaskWebSocketManager] Already connected or connecting');
      this.notifyListeners();
      return;
    }

    const token = getToken();
    if (!token) {
      console.error('[TaskWebSocketManager] No token available');
      return;
    }

    this.isConnecting = true;
    this.connectionId++;
    const currentConnectionId = this.connectionId;

    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/tasks?token=${token}`;
    console.log('[TaskWebSocketManager] Connecting...', { connectionId: currentConnectionId });

    try {
      const ws = new WebSocket(wsUrl);
      this.ws = ws;

      ws.onopen = () => {
        // Check if this is still the current connection
        if (this.connectionId !== currentConnectionId) {
          console.log('[TaskWebSocketManager] Connection outdated, closing');
          ws.close();
          return;
        }

        console.log('[TaskWebSocketManager] Connected');
        this.isConnecting = false;
        this.notifyListeners();

        // Start ping interval
        if (this.pingTimer) {
          clearInterval(this.pingTimer);
        }
        this.pingTimer = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify('ping'));
          }
        }, 30000);
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          console.log('[TaskWebSocketManager] Received:', data);

          if (data.type === 'pong') {
            return;
          }

          this.messageListeners.forEach((listener) => listener(data as TaskEvent));
        } catch (err) {
          console.error('[TaskWebSocketManager] Failed to parse message:', err);
        }
      };

      ws.onclose = (event) => {
        console.log('[TaskWebSocketManager] Disconnected:', event.code, event.reason);
        this.isConnecting = false;

        // Only clear ping if this is the current connection
        if (this.connectionId === currentConnectionId) {
          if (this.pingTimer) {
            clearInterval(this.pingTimer);
            this.pingTimer = null;
          }
          this.ws = null;
          this.notifyListeners();

          // Auto reconnect
          if (event.code !== 1000 && event.code !== 1001) {
            if (this.reconnectTimer) {
              clearTimeout(this.reconnectTimer);
            }
            this.reconnectTimer = setTimeout(() => {
              console.log('[TaskWebSocketManager] Reconnecting...');
              this.connect();
            }, 3000);
          }
        }
      };

      ws.onerror = (error) => {
        console.error('[TaskWebSocketManager] Error:', error);
        this.isConnecting = false;
      };
    } catch (err) {
      console.error('[TaskWebSocketManager] Failed to create connection:', err);
      this.isConnecting = false;
    }
  }

  disconnect(): void {
    console.log('[TaskWebSocketManager] Disconnect called');

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }

    if (this.ws) {
      if (this.ws.readyState === WebSocket.OPEN) {
        this.ws.close(1000, 'Manual disconnect');
      }
      this.ws = null;
    }

    // Increment connectionId to invalidate any pending connections
    this.connectionId++;
    this.notifyListeners();
  }

  subscribe(listener: (connected: boolean) => void): () => void {
    this.listeners.add(listener);
    // Immediately notify of current state
    listener(this.isConnected());
    return () => {
      this.listeners.delete(listener);
    };
  }

  subscribeToMessages(listener: (event: TaskEvent) => void): () => void {
    this.messageListeners.add(listener);
    return () => {
      this.messageListeners.delete(listener);
    };
  }

  private notifyListeners(): void {
    const connected = this.isConnected();
    console.log('[TaskWebSocketManager] Notifying listeners, connected:', connected);
    this.listeners.forEach((listener) => listener(connected));
  }
}

export function useTaskWebSocket(options: UseTaskWebSocketOptions = {}) {
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const managerRef = useRef(TaskWebSocketManager.getInstance());
  const optionsRef = useRef(options);

  // Keep options ref up to date
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  useEffect(() => {
    const manager = managerRef.current;

    // Subscribe to connection state changes
    const unsubscribe = manager.subscribe((connected) => {
      console.log('[useTaskWebSocket] Connection state changed:', connected);
      setIsConnected(connected);
      setIsConnecting(false);

      if (connected) {
        optionsRef.current.onConnect?.();
      } else {
        optionsRef.current.onDisconnect?.();
      }
    });

    // Subscribe to messages
    const unsubscribeMessages = manager.subscribeToMessages((event) => {
      optionsRef.current.onMessage?.(event);
    });

    // Start connection if not already connected
    if (!manager.isConnected()) {
      setIsConnecting(true);
      manager.connect();
    }

    return () => {
      console.log('[useTaskWebSocket] Component unmounting, unsubscribing');
      unsubscribe();
      unsubscribeMessages();
      // Note: We don't disconnect here to allow other components to use the same connection
    };
  }, []);

  const connect = useCallback(() => {
    setIsConnecting(true);
    managerRef.current.connect();
  }, []);

  const disconnect = useCallback(() => {
    managerRef.current.disconnect();
  }, []);

  return {
    isConnected,
    isConnecting,
    connect,
    disconnect,
  };
}
