import React, { useEffect, useState } from 'react';
import type { ColumnsType } from 'antd/es/table';
import {
  Table,
  Card,
  Tag,
  Input,
  Button,
  message,
  Modal,
  Space,
  Popconfirm,
  Tooltip,
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  StopOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import { sessionsApi } from '../api';
import type { Session } from '../types';

const Sessions: React.FC = () => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(false);
  const [cancelling, setCancelling] = useState<string | null>(null);
  const [searchKey, setSearchKey] = useState('');
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);

  const fetchSessions = async () => {
    setLoading(true);
    try {
      const res = await sessionsApi.list();
      setSessions((res as any)?.items || []);
    } catch (error) {
      message.error('获取会话列表失败');
    } finally {
      setLoading(false);
    }
  };

  const searchSessions = async () => {
    if (!searchKey) {
      fetchSessions();
      return;
    }
    setLoading(true);
    try {
      const res = await sessionsApi.list({ user_code: searchKey });
      setSessions((res as any)?.items || []);
    } catch (error) {
      message.error('搜索失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = async (sessionKey: string) => {
    setCancelling(sessionKey);
    try {
      const res = await sessionsApi.cancel(sessionKey) as any;
      if (res?.success === false) {
        message.warning(res?.message || '取消会话失败');
      } else {
        message.success('会话已取消');
        // 刷新列表
        fetchSessions();
      }
    } catch (error: any) {
      message.error(error?.response?.data?.message || '取消会话失败');
    } finally {
      setCancelling(null);
    }
  };

  useEffect(() => {
    fetchSessions();
    // 每5秒自动刷新一次，获取最新的活跃状态
    const timer = setInterval(fetchSessions, 5000);
    return () => clearInterval(timer);
  }, []);

  const formatTime = (time?: string) => {
    if (!time) return '-';
    return new Date(time).toLocaleString();
  };

  const getStatusTag = (record: Session) => {
    // 通过会话是否有活跃处理中状态判断
    // 实际上后端不直接返回 isActive，需要通过上下文判断
    const lastActiveTime = record.last_active_at || record.updated_at;
    if (lastActiveTime) {
      const lastActive = new Date(lastActiveTime).getTime();
      const now = Date.now();
      const diffMinutes = (now - lastActive) / 1000 / 60;

      // 如果最近2分钟内有活跃，认为是活跃状态
      if (diffMinutes < 2) {
        return (
          <Tag icon={<ThunderboltOutlined />} color="processing">
            处理中
          </Tag>
        );
      }
    }

    return (
      <Tag icon={<ClockCircleOutlined />} color="default">
        空闲
      </Tag>
    );
  };

  const columns: ColumnsType<Session> = [
    {
      title: 'Session Key',
      dataIndex: 'session_key',
      ellipsis: true,
      width: 250,
      render: (key: string) => (
        <Tooltip title={key}>
          <span style={{ fontFamily: 'monospace', fontSize: '12px' }}>
            {key.length > 30 ? key.substring(0, 30) + '...' : key}
          </span>
        </Tooltip>
      ),
    },
    {
      title: '用户',
      dataIndex: 'user_code',
      width: 120,
    },
    {
      title: '渠道',
      dataIndex: 'channel_code',
      width: 120,
    },
    {
      title: 'Agent',
      dataIndex: 'agent_code',
      width: 120,
      render: (code?: string) => code || '-',
    },
    {
      title: '状态',
      width: 100,
      render: (_: any, record: Session) => getStatusTag(record),
    },
    {
      title: '最后活跃',
      dataIndex: 'last_active_at',
      width: 180,
      render: formatTime,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: formatTime,
    },
    {
      title: '操作',
      width: 180,
      fixed: 'right',
      render: (_: any, record: Session) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              setSelectedSession(record);
              setDetailVisible(true);
            }}
          >
            详情
          </Button>
          <Popconfirm
            title="确认取消会话"
            description={`确定要强制停止会话 ${record.session_key?.substring(0, 20)}... 吗？`}
            onConfirm={() => handleCancel(record.session_key)}
            okText="确认"
            cancelText="取消"
            okButtonProps={{ danger: true }}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<StopOutlined />}
              loading={cancelling === record.session_key}
            >
              停止
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="会话管理"
        extra={
          <Space>
            <Input.Search
              placeholder="输入用户代码搜索"
              value={searchKey}
              onChange={(e) => setSearchKey(e.target.value)}
              onSearch={searchSessions}
              style={{ width: 250 }}
              prefix={<SearchOutlined />}
              allowClear
            />
            <Button onClick={fetchSessions} type="primary">
              刷新
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="session_key"
          columns={columns}
          dataSource={sessions}
          loading={loading}
          pagination={{ pageSize: 20 }}
          scroll={{ x: 1200 }}
          size="small"
        />
      </Card>

      <Modal
        title="会话详情"
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedSession(null);
        }}
        footer={
          <Space>
            <Button onClick={() => setDetailVisible(false)}>关闭</Button>
            <Popconfirm
              title="确认取消会话"
              description="确定要强制停止此会话吗？"
              onConfirm={() => {
                if (selectedSession) {
                  handleCancel(selectedSession.session_key);
                  setDetailVisible(false);
                }
              }}
              okText="确认"
              cancelText="取消"
              okButtonProps={{ danger: true }}
            >
              <Button type="primary" danger icon={<StopOutlined />}>
                强制停止
              </Button>
            </Popconfirm>
          </Space>
        }
        width={700}
      >
        {selectedSession && (
          <div>
            <p><strong>Session Key:</strong></p>
            <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, fontSize: 12 }}>
              {selectedSession.session_key}
            </pre>

            <p><strong>用户信息:</strong></p>
            <p>用户代码: {selectedSession.user_code}</p>
            <p>渠道代码: {selectedSession.channel_code}</p>
            <p>Agent代码: {selectedSession.agent_code || '-'}</p>

            <p><strong>时间信息:</strong></p>
            <p>创建时间: {formatTime(selectedSession.created_at)}</p>
            <p>最后活跃: {formatTime(selectedSession.last_active_at)}</p>

            {selectedSession.external_id && (
              <>
                <p><strong>外部ID:</strong></p>
                <p>{selectedSession.external_id}</p>
              </>
            )}

            {selectedSession.metadata && (
              <>
                <p><strong>元数据:</strong></p>
                <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, fontSize: 12 }}>
                  {JSON.stringify(selectedSession.metadata, null, 2)}
                </pre>
              </>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Sessions;
