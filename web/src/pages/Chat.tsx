import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Select, message as antMessage, Spin, Avatar, Typography } from 'antd';
import { SendOutlined, UserOutlined, RobotOutlined, PlusOutlined } from '@ant-design/icons';
import { Mermaid, CodeHighlighter } from '@ant-design/x';
import XMarkdown, { type ComponentProps } from '@ant-design/x-markdown';
import Latex from '@ant-design/x-markdown/plugins/Latex';
import '@ant-design/x-markdown/themes/dark.css';
import { useWebSocket, type WebSocketMessage, type ChunkPayload, type SystemPayload } from '../hooks/useWebSocket';
import { getToken, getCurrentUser, getCurrentUserCode, usersApi, channelsApi } from '../api';
import type { User, Channel } from '../types';

const { Text } = Typography;

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  isStreaming?: boolean;
}

// 自定义代码块组件（支持 Mermaid 图表和代码高亮）
const CustomCode: React.FC<ComponentProps> = (props) => {
  const { className, children } = props;
  const lang = className?.match(/language-(\w+)/)?.[1] || '';
  if (typeof children !== 'string') return null;

  // Mermaid 图表渲染
  if (lang === 'mermaid') {
    return <Mermaid>{children}</Mermaid>;
  }

  // 代码高亮渲染
  return (
    <CodeHighlighter lang={lang}>
      {children}
    </CodeHighlighter>
  );
};

// Markdown 渲染组件
const MarkdownContent: React.FC<{ content: string }> = ({ content }) => {
  return (
    <div className="x-markdown-dark">
      <XMarkdown
        config={{ extensions: Latex() }}
        components={{ code: CustomCode }}
        paragraphTag="div"
      >
        {content}
      </XMarkdown>
    </div>
  );
};

const Chat: React.FC = () => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [socketNotice, setSocketNotice] = useState('');
  const [selectedUser, setSelectedUser] = useState<string>('');
  const [sessionId, setSessionId] = useState<string>('');
  const [selectedChannel, setSelectedChannel] = useState<string>('');
  const [users, setUsers] = useState<User[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const currentUser = getCurrentUser();
  const currentUserCode = getCurrentUserCode() || '';
  const token = getToken() || '';

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

        const usersData = (usersRes as any)?.items || [];
        const channelsData = (channelsRes as any)?.items || [];
        const wsChannels = channelsData.filter((ch: Channel) => ch.type === 'websocket');

        setUsers(usersData);
        setChannels(wsChannels);

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
        setSocketNotice(msg.payload?.message || '发生错误');
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

      if (lastMsg?.role === 'assistant' && lastMsg.isStreaming) {
        return [
          ...prev.slice(0, -1),
          { ...lastMsg, content: lastMsg.content + payload.content, isStreaming: !payload.is_end }
        ];
      } else if (lastMsg?.role === 'assistant' && !lastMsg.isStreaming) {
        if (lastMsg.content === payload.content) {
          return prev;
        }
        if (payload.content) {
          return [...prev, {
            id: Date.now().toString(),
            role: 'assistant',
            content: payload.content,
            isStreaming: !payload.is_end
          }];
        }
        return prev;
      } else {
        if (payload.content) {
          return [...prev, {
            id: Date.now().toString(),
            role: 'assistant',
            content: payload.content,
            isStreaming: !payload.is_end
          }];
        }
        return prev;
      }
    });

    if (payload.session_id && payload.session_id !== sessionId) {
      setSessionId(payload.session_id);
    }
  };

  const handleSystemMessage = (payload: SystemPayload) => {
    if (payload.session_id) {
      setSessionId(payload.session_id);
    }
  };

  const handleError = useCallback((error: Error) => {
    setSocketNotice(error.message);
  }, []);

  const isWebSocketEnabled = isAdmin ? !!selectedChannel && !!selectedUser : !!selectedChannel;

  const { isConnected, isConnecting, sendMessage, connect } = useWebSocket({
    channelCode: selectedChannel,
    token,
    enabled: isWebSocketEnabled,
    onMessage: handleMessage,
    onError: handleError,
  });

  useEffect(() => {
    if (isConnected) {
      setSocketNotice('');
    }
  }, [isConnected]);

  useEffect(() => {
    if (isWebSocketEnabled) {
      connect();
    }
  }, [selectedChannel, selectedUser, isWebSocketEnabled, connect]);

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

    const targetUserCode = isAdmin && selectedUser ? selectedUser : currentUserCode;

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: inputValue
    };
    setMessages(prev => [...prev, userMessage]);

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

  // IME 组合状态跟踪（处理中英文输入法候选词选择问题）
  const isComposingRef = useRef(false);

  const handleCompositionStart = () => {
    isComposingRef.current = true;
  };

  const handleCompositionEnd = () => {
    isComposingRef.current = false;
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // 在 IME 组合过程中不处理 Enter 键
    if (isComposingRef.current) {
      return;
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleNewChat = () => {
    setMessages([]);
    setSessionId('');
  };

  if (loading) {
    return (
      <div style={{
        height: '100vh',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        background: '#343541'
      }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{
      height: '100vh',
      display: 'flex',
      flexDirection: 'column',
      background: '#343541'
    }}>
      {/* 顶部工具栏 */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '12px 20px',
        borderBottom: '1px solid rgba(255,255,255,0.1)',
        background: '#202123'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button
            onClick={handleNewChat}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              padding: '8px 16px',
              background: 'transparent',
              border: '1px solid rgba(255,255,255,0.2)',
              borderRadius: '8px',
              color: '#fff',
              cursor: 'pointer',
              fontSize: '14px'
            }}
          >
            <PlusOutlined />
            新对话
          </button>

          <Select
            placeholder="选择渠道"
            value={selectedChannel || undefined}
            onChange={setSelectedChannel}
            style={{ width: 200 }}
            dropdownStyle={{ background: '#202123' }}
            options={channels.map(ch => ({
              label: ch.name,
              value: ch.channel_code,
            }))}
          />

          {isAdmin && (
            <Select
              placeholder="选择用户身份（必选）"
              value={selectedUser || undefined}
              onChange={setSelectedUser}
              style={{ width: 220 }}
              dropdownStyle={{ background: '#202123' }}
              popupClassName="user-select-dropdown"
              options={users.map(u => ({
                label: `${u.display_name || u.username} (${u.user_code})`,
                value: u.user_code,
              }))}
            />
          )}
        </div>

        <Text style={{ color: 'rgba(255,255,255,0.6)', fontSize: '14px' }}>
          {isConnecting ? '连接中...' : isConnected ? '已连接' : '未连接'}
        </Text>
      </div>

      {/* 提示信息 */}
      {isAdmin && !selectedUser && selectedChannel && (
        <div style={{
          padding: '12px 20px',
          background: 'rgba(59,130,246,0.1)',
          borderBottom: '1px solid rgba(59,130,246,0.2)'
        }}>
          <Text style={{ color: '#60a5fa' }}>请选择用户身份以开始对话</Text>
        </div>
      )}

      {/* 消息区域 */}
      <div style={{
        flex: 1,
        overflowY: 'auto',
        padding: '20px 0'
      }}>
        {messages.length === 0 ? (
          <div style={{
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center',
            color: 'rgba(255,255,255,0.6)'
          }}>
            <RobotOutlined style={{ fontSize: '48px', marginBottom: '16px', opacity: 0.5 }} />
            <div style={{ fontSize: '24px', fontWeight: 500, marginBottom: '8px' }}>
              AI 助手
            </div>
            <div style={{ fontSize: '14px', opacity: 0.7 }}>
              {channels.length === 0
                ? '暂无可用的 WebSocket 渠道，请先创建一个'
                : '有什么可以帮助您的吗？'}
            </div>
          </div>
        ) : (
          <div style={{ maxWidth: '768px', margin: '0 auto', padding: '0 20px' }}>
            {messages.map((msg) => (
              <div
                key={msg.id}
                style={{
                  display: 'flex',
                  flexDirection: msg.role === 'user' ? 'row-reverse' : 'row',
                  gap: '16px',
                  padding: '24px 0',
                  borderBottom: '1px solid rgba(255,255,255,0.05)'
                }}
              >
                {msg.role === 'assistant' ? (
                  <Avatar
                    size={36}
                    icon={<RobotOutlined />}
                    style={{
                      backgroundColor: '#19c37d',
                      flexShrink: 0
                    }}
                  />
                ) : (
                  <Avatar
                    size={36}
                    icon={<UserOutlined />}
                    style={{
                      backgroundColor: '#5436da',
                      flexShrink: 0
                    }}
                  />
                )}
                <div style={{ flex: 1, minWidth: 0, display: 'flex', justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start' }}>
                  <div style={{
                    color: '#fff',
                    fontSize: '14px',
                    lineHeight: '1.7',
                    maxWidth: msg.role === 'user' ? '80%' : '100%',
                    padding: msg.role === 'user' ? '12px 16px' : '0',
                    background: msg.role === 'user' ? '#5436da' : 'transparent',
                    borderRadius: msg.role === 'user' ? '16px' : '0'
                  }}>
                    {msg.role === 'user' ? (
                      <div style={{ whiteSpace: 'pre-wrap' }}>{msg.content}</div>
                    ) : (
                      <MarkdownContent content={msg.content} />
                    )}
                    {msg.isStreaming && (
                      <span style={{
                        display: 'inline-block',
                        width: '8px',
                        height: '16px',
                        backgroundColor: '#10a37f',
                        marginLeft: '4px',
                        animation: 'blink 1s infinite'
                      }} />
                    )}
                  </div>
                </div>
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* 输入区域 */}
      <div style={{
        padding: '20px',
        paddingTop: 0
      }}>
        <div style={{
          maxWidth: '768px',
          margin: '0 auto',
          position: 'relative'
        }}>
          <textarea
            ref={inputRef}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            onCompositionStart={handleCompositionStart}
            onCompositionEnd={handleCompositionEnd}
            placeholder={
              channels.length === 0
                ? '请先创建一个 WebSocket 渠道'
                : isAdmin && !selectedUser
                  ? '请先选择用户身份'
                  : '发送消息...'
            }
            disabled={!isConnected || channels.length === 0 || (isAdmin && !selectedUser)}
            style={{
              width: '100%',
              minHeight: '52px',
              maxHeight: '200px',
              padding: '12px 48px 12px 16px',
              background: '#40414f',
              border: '1px solid rgba(255,255,255,0.1)',
              borderRadius: '12px',
              color: '#fff',
              fontSize: '14px',
              lineHeight: '1.5',
              resize: 'none',
              outline: 'none',
              fontFamily: 'inherit'
            }}
            onFocus={(e) => {
              e.target.style.borderColor = 'rgba(255,255,255,0.2)';
            }}
            onBlur={(e) => {
              e.target.style.borderColor = 'rgba(255,255,255,0.1)';
            }}
          />
          <button
            onClick={handleSend}
            disabled={!inputValue.trim() || !isConnected || (isAdmin && !selectedUser)}
            style={{
              position: 'absolute',
              right: '12px',
              bottom: '12px',
              width: '32px',
              height: '32px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: inputValue.trim() && isConnected ? '#10a37f' : 'transparent',
              border: 'none',
              borderRadius: '8px',
              color: inputValue.trim() && isConnected ? '#fff' : 'rgba(255,255,255,0.3)',
              cursor: inputValue.trim() && isConnected ? 'pointer' : 'not-allowed',
              fontSize: '16px',
              transition: 'all 0.2s'
            }}
          >
            <SendOutlined />
          </button>
        </div>
        <div style={{
          textAlign: 'center',
          marginTop: '12px'
        }}>
          <Text style={{ color: 'rgba(255,255,255,0.4)', fontSize: '12px' }}>
            按 Enter 发送，Shift + Enter 换行
          </Text>
          {socketNotice && (
            <div style={{ marginTop: '6px' }}>
              <Text style={{ color: 'rgba(255,255,255,0.3)', fontSize: '11px' }}>
                {socketNotice}
              </Text>
            </div>
          )}
        </div>
      </div>

      {/* CSS 动画 */}
      <style>{`
        @keyframes blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0; }
        }
        textarea::placeholder {
          color: rgba(255,255,255,0.4);
        }
        ::-webkit-scrollbar {
          width: 8px;
        }
        ::-webkit-scrollbar-track {
          background: transparent;
        }
        ::-webkit-scrollbar-thumb {
          background: rgba(255,255,255,0.2);
          border-radius: 4px;
        }
        ::-webkit-scrollbar-thumb:hover {
          background: rgba(255,255,255,0.3);
        }
        .user-select-dropdown .ant-select-item {
          color: #fff;
        }
        .user-select-dropdown .ant-select-item-option-selected {
          background: rgba(255,255,255,0.15);
        }
      `}</style>
    </div>
  );
};

export default Chat;
