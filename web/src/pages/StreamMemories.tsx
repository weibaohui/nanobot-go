import React, { useEffect, useState } from 'react';
import {
  Table,
  Card,
  Tag,
  Button,
  message,
  Descriptions,
  Modal,
  Switch,
} from 'antd';
import { EyeOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { streamMemoriesApi } from '../api';
import type { StreamMemory } from '../types';

const StreamMemories: React.FC = () => {
  const [memories, setMemories] = useState<StreamMemory[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedMemory, setSelectedMemory] = useState<StreamMemory | null>(null);
  const [showUnprocessed, setShowUnprocessed] = useState(false);

  const fetchMemories = async () => {
    setLoading(true);
    try {
      const res = await streamMemoriesApi.list();
      setMemories((res as any)?.items || []);
    } catch (error) {
      message.error('获取短期记忆失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchUnprocessed = async () => {
    setLoading(true);
    try {
      const res = await streamMemoriesApi.getUnprocessed();
      setMemories((res as any) || []);
    } catch (error) {
      message.error('获取未处理记忆失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (showUnprocessed) {
      fetchUnprocessed();
    } else {
      fetchMemories();
    }
  }, [showUnprocessed]);

  const handleMarkProcessed = async (id: number) => {
    try {
      await streamMemoriesApi.markProcessed(id);
      message.success('标记成功');
      fetchMemories();
    } catch (error) {
      message.error('标记失败');
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: 'Trace ID', dataIndex: 'trace_id', ellipsis: true },
    { title: 'Session', dataIndex: 'session_key', ellipsis: true },
    { title: 'Event Type', dataIndex: 'event_type' },
    {
      title: '内容摘要',
      dataIndex: 'content',
      ellipsis: true,
      render: (content: string) => content?.substring(0, 40) + (content?.length > 40 ? '...' : ''),
    },
    {
      title: '状态',
      dataIndex: 'processed',
      render: (processed: boolean) => (
        <Tag color={processed ? 'success' : 'warning'}>
          {processed ? '已处理' : '未处理'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      width: 180,
      render: (_: any, record: StreamMemory) => (
        <>
          <Button
            type="text"
            icon={<EyeOutlined />}
            onClick={() => {
              setSelectedMemory(record);
              setDetailVisible(true);
            }}
          >
            查看
          </Button>
          {!record.processed && (
            <Button
              type="text"
              icon={<CheckCircleOutlined />}
              onClick={() => handleMarkProcessed(record.id)}
            >
              标记处理
            </Button>
          )}
        </>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="短期记忆 (Stream Memory)"
        extra={
          <Switch
            checked={showUnprocessed}
            onChange={setShowUnprocessed}
            checkedChildren="未处理"
            unCheckedChildren="全部"
          />
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={memories}
          loading={loading}
          pagination={{ pageSize: 20 }}
        />
      </Card>

      <Modal
        title="短期记忆详情"
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedMemory(null);
        }}
        footer={null}
        width={700}
      >
        {selectedMemory && (
          <Descriptions column={1} bordered>
            <Descriptions.Item label="ID">{selectedMemory.id}</Descriptions.Item>
            <Descriptions.Item label="Trace ID">{selectedMemory.trace_id}</Descriptions.Item>
            <Descriptions.Item label="Session Key">{selectedMemory.session_key}</Descriptions.Item>
            <Descriptions.Item label="Channel Type">{selectedMemory.channel_type}</Descriptions.Item>
            <Descriptions.Item label="Event Type">{selectedMemory.event_type}</Descriptions.Item>
            <Descriptions.Item label="原始内容">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{selectedMemory.content}</pre>
            </Descriptions.Item>
            <Descriptions.Item label="AI 总结">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{selectedMemory.summary}</pre>
            </Descriptions.Item>
            <Descriptions.Item label="处理状态">
              <Tag color={selectedMemory.processed ? 'success' : 'warning'}>
                {selectedMemory.processed ? '已处理' : '未处理'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{selectedMemory.created_at}</Descriptions.Item>
            {selectedMemory.processed_at && (
              <Descriptions.Item label="处理时间">{selectedMemory.processed_at}</Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default StreamMemories;
