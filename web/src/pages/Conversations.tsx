import React, { useEffect, useState } from 'react';
import {
  Table,
  Card,
  Tag,
  Input,
  Button,
  message,
  Descriptions,
  Modal,
} from 'antd';
import { SearchOutlined, EyeOutlined } from '@ant-design/icons';
import { conversationsApi } from '../api';
import type { ConversationRecord } from '../types';

const Conversations: React.FC = () => {
  const [records, setRecords] = useState<ConversationRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [sessionKey, setSessionKey] = useState('');
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<ConversationRecord | null>(null);

  const fetchRecords = async () => {
    setLoading(true);
    try {
      const res = await conversationsApi.list();
      setRecords((res as any)?.items || []);
    } catch (error) {
      message.error('获取对话记录失败');
    } finally {
      setLoading(false);
    }
  };

  const searchBySession = async () => {
    if (!sessionKey) {
      fetchRecords();
      return;
    }
    setLoading(true);
    try {
      const res = await conversationsApi.getBySession(sessionKey);
      setRecords((res as any)?.items || []);
    } catch (error) {
      message.error('搜索失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRecords();
  }, []);

  const getRoleColor = (role?: string) => {
    const colors: Record<string, string> = {
      user: 'blue',
      assistant: 'green',
      system: 'orange',
      tool: 'purple',
    };
    return colors[role || ''] || 'default';
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: 'Agent Code',
      dataIndex: 'agent_code',
      ellipsis: true,
      render: (code: string) => code || '-',
    },
    {
      title: 'Channel Code',
      dataIndex: 'channel_code',
      ellipsis: true,
      render: (code: string) => code || '-',
    },
    {
      title: '角色',
      dataIndex: 'role',
      render: (role: string) => <Tag color={getRoleColor(role)}>{role}</Tag>,
    },
    {
      title: '内容',
      dataIndex: 'content',
      ellipsis: true,
      render: (content: string) => content?.substring(0, 50) + (content?.length > 50 ? '...' : ''),
    },
    {
      title: 'Tokens',
      render: (_: any, record: ConversationRecord) => (
        <span>{record.total_tokens || 0}</span>
      ),
    },
    {
      title: '时间',
      dataIndex: 'timestamp',
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: 'Trace ID',
      dataIndex: 'trace_id',
      ellipsis: true,
    },
    {
      title: '操作',
      width: 100,
      render: (_: any, record: ConversationRecord) => (
        <Button
          type="text"
          icon={<EyeOutlined />}
          onClick={() => {
            setSelectedRecord(record);
            setDetailVisible(true);
          }}
        >
          查看
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="对话记录"
        extra={
          <Input.Search
            placeholder="输入 Session Key 搜索"
            value={sessionKey}
            onChange={(e) => setSessionKey(e.target.value)}
            onSearch={searchBySession}
            style={{ width: 300 }}
            prefix={<SearchOutlined />}
          />
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={records}
          loading={loading}
          pagination={{ pageSize: 20 }}
        />
      </Card>

      <Modal
        title="对话详情"
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedRecord(null);
        }}
        footer={null}
        width={800}
      >
        {selectedRecord && (
          <Descriptions column={1} bordered>
            <Descriptions.Item label="ID">{selectedRecord.id}</Descriptions.Item>
            <Descriptions.Item label="User Code">{selectedRecord.user_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="Agent Code">{selectedRecord.agent_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="Channel Code">{selectedRecord.channel_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="Event Type">{selectedRecord.event_type}</Descriptions.Item>
            <Descriptions.Item label="角色">
              <Tag color={getRoleColor(selectedRecord.role)}>{selectedRecord.role}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="内容">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{selectedRecord.content}</pre>
            </Descriptions.Item>
            <Descriptions.Item label="Tokens">
              Prompt: {selectedRecord.prompt_tokens} / Completion: {selectedRecord.completion_tokens} / Total: {selectedRecord.total_tokens}
            </Descriptions.Item>
            <Descriptions.Item label="时间">{selectedRecord.timestamp}</Descriptions.Item>
            <Descriptions.Item label="Span ID">{selectedRecord.span_id}</Descriptions.Item>
            <Descriptions.Item label="Trace ID">{selectedRecord.trace_id}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default Conversations;
