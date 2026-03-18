import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Card, Select, Space, message as antMessage, Alert, Spin } from 'antd';
import { Bubble } from '@ant-design/x';
import { Sender } from '@ant-design/x';
import { Welcome } from '@ant-design/x';
import { useWebSocket, type WebSocketMessage, type ChunkPayload, type SystemPayload } from '../hooks/useWebSocket';
import { getToken, getCurrentUser, getCurrentUserCode, usersApi, channelsApi } from '../api';
import type { User, Channel } from '../types';
import { MessageOutlined } from '@ant-design/icons';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  isStreaming?: boolean;
}

const Chat: React.FC = () => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [selectedUser, setSelectedUser] = useState<string>('');
  const [sessionId, setSessionId] = useState<string>('');
  const [selectedChannel, setSelectedChannel] = useState<string>('');
  const [users, setUsers] = useState<User[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [isConnecting, setIsConnecting] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const currentUser = getCurrentUser();
  const currentUserCode = getCurrentUserCode() || '';
  const token = getToken() || '';

  // 判断是否为管理员（简化判断：第一个创建的用户或特定用户名为 admin）
  const isAdmin = currentUser?.username === 'admin';

  // 获取可用用户列表和渠道列表
  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const [usersRes, channelsRes] = await Promise.all([
          usersApi.list(),
          channelsApi.list(currentUserCode),
        ]);

        // 处理响应数据
        const usersData = (usersRes as any)?.items || [];
        const channelsData = (channelsRes as any)?.items || [];

        // 过滤出 WebSocket 类型的渠道
        const wsChannels = channelsData.filter((ch: Channel) => ch.type === 'websocket');

        setUsers(usersData);
        setChannels(wsChannels);

        // 默认选择第一个 WebSocket 渠道
        if (wsChannels.length > 0 && !selectedChannel) {
          setSelectedChannel(wsChannels[0].channel_code);
        }
      } catch (error) {
        antMessage.error('获取数据失败');
        console.error(error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [currentUserCode]);

  // 自动滚动到底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // 处理收到的消息
  const handleMessage = useCallback((msg: WebSocketMessage) => {
    switch (msg.type) {
      case 'chunk':
        handleChunk(msg.payload as ChunkPayload);
        break;
      case 'error':
        antMessage.error(msg.payload?.message || '发生错误');
        break;
      case 'system':
        handleSystemMessage(msg.payload as SystemPayload);
        break;
    }
  }, []);

  // 处理流式响应片段
  const handleChunk = (payload: ChunkPayload) => {
    setMessages(prev => {
      const lastMsg = prev[prev.length - 1];

      // 如果是结束标记
      if (payload.is_end) {
        if (lastMsg?.role === 'assistant' && lastMsg.isStreaming) {
          return [
            ...prev.slice(0, -1),
            { ...lastMsg, isStreaming: false }
          ];
        }
        return prev;
      }

      // 追加到现有消息或创建新消息
      if (lastMsg?.role === 'assistant' && lastMsg.isStreaming) {
        // 追加到现有消息
        return [
          ...prev.slice(0, -1),
          { ...lastMsg, content: lastMsg.content + payload.content }
        ];
      } else {
        // 新消息开始
        return [...prev, {
          id: Date.now().toString(),
          role: 'assistant',
          content: payload.content,
          isStreaming: true
        }];
      }
    });

    // 更新会话 ID
    if (payload.session_id && payload.session_id !== sessionId) {
      setSessionId(payload.session_id);
    }
  };

  // 处理系统消息
  const handleSystemMessage = (payload: SystemPayload) => {
    if (payload.type === 'connected') {
      antMessage.success('已连接到 AI 助手');
    }
    if (payload.session_id) {
      setSessionId(payload.session_id);
    }
  };

  // 处理连接错误
  const handleError = useCallback((error: Error) => {
    setIsConnecting(false);
    antMessage.error(error.message);
  }, []);

  // 处理连接成功
  const handleConnect = useCallback(() => {
    setIsConnecting(false);
  }, []);

  // 处理连接断开
  const handleDisconnect = useCallback(() => {
    setIsConnecting(false);
  }, []);

  // WebSocket 连接
  const { isConnected, sendMessage, connect } = useWebSocket({
    channelCode: selectedChannel,
    token,
    onMessage: handleMessage,
    onError: handleError,
    onConnect: handleConnect,
    onDisconnect: handleDisconnect,
  });

  // 当渠道改变时重新连接
  useEffect(() => {
    if (selectedChannel) {
      setIsConnecting(true);
      connect();
    }
  }, [selectedChannel, connect]);

  // 发送消息
  const handleSend = () => {
    if (!inputValue.trim()) return;
    if (!isConnected) {
      antMessage.warning('连接未建立，请稍后再试');
      return;
    }
    if (!selectedChannel) {
      antMessage.warning('请先选择一个渠道');
      return;
    }

    // 确定用户身份
    const targetUserCode = isAdmin && selectedUser ? selectedUser : currentUserCode;

    // 添加到本地消息列表
    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: inputValue
    };
    setMessages(prev => [...prev, userMessage]);

    // 发送 WebSocket 消息
    const success = sendMessage({
      content: inputValue,
      user_code: targetUserCode,
      session_id: sessionId
    });

    if (success) {
      setInputValue('');
    } else {
      antMessage.error('发送失败，请检查连接状态');
    }
  };

  if (loading) {
    return (
      <Card style={{ height: 'calc(100vh - 180px)', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
        <Spin size="large" tip="加载中..." />
      </Card>
    );
  }

  return (
    <Card
      title={
        <Space>
          <MessageOutlined />
          <span>AI 对话</span>
        </Space>
      }
      style={{ height: 'calc(100vh - 180px)' }}
      bodyStyle={{ height: 'calc(100% - 57px)', padding: '16px' }}
    >
      <Space direction="vertical" style={{ width: '100%', height: '100%' }} size="middle">
        {/* 渠道选择器 */}
        <Select
          placeholder="选择对话渠道"
          value={selectedChannel || undefined}
          onChange={setSelectedChannel}
          style={{ width: '100%' }}
          options={channels.map(ch => ({
            label: ch.name,
            value: ch.channel_code,
          }))}
        />

        {/* 用户选择器 - 仅管理员可见 */}
        {isAdmin && (
          <Select
            placeholder="选择用户身份（可选，默认为当前用户）"
            value={selectedUser || undefined}
            onChange={setSelectedUser}
            allowClear
            style={{ width: '100%' }}
            options={users.map(u => ({
              label: `${u.display_name || u.username} (${u.user_code})`,
              value: u.user_code,
            }))}
          />
        )}

        {/* 连接状态 */}
        {!isConnected && selectedChannel && (
          <Alert
            message="连接断开，正在尝试重连..."
            type="warning"
            showIcon
            banner
          />
        )}

        {/* 消息列表区域 */}
        <div
          style={{
            flex: 1,
            overflowY: 'auto',
            padding: '8px',
            backgroundColor: '#f5f5f5',
            borderRadius: '8px',
            minHeight: '300px'
          }}
        >
          {messages.length === 0 ? (
            <Welcome
              title="AI 助手"
              description={
                <Space direction="vertical" align="center">
                  <span>有什么可以帮助您的吗？</span>
                  {channels.length === 0 && (
                    <span style={{ color: '#ff4d4f' }}>
                      暂无可用的 WebSocket 渠道，请先创建一个
                    </span>
                  )}
                </Space>
              }
              style={{ marginTop: '40px' }}
            />
          ) : (
            <Space direction="vertical" style={{ width: '100%' }}>
              {messages.map((msg) => (
                <Bubble
                  key={msg.id}
                  placement={msg.role === 'user' ? 'end' : 'start'}
                  content={msg.content}
                  loading={msg.isStreaming}
                />
              ))}
              <div ref={messagesEndRef} />
            </Space>
          )}
        </div>

        {/* 输入框 */}
        <Sender
          value={inputValue}
          onChange={setInputValue}
          onSubmit={handleSend}
          placeholder={channels.length === 0 ? "请先创建一个 WebSocket 渠道" : "输入消息..."}
          disabled={!isConnected || channels.length === 0}
          loading={isConnecting}
        />
      </Space>
    </Card>
  );
};

export default Chat;
