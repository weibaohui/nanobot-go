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
  DatePicker,
  Space,
} from 'antd';
import { SearchOutlined, EyeOutlined, CalendarOutlined } from '@ant-design/icons';
import { longTermMemoriesApi } from '../api';
import type { LongTermMemory } from '../types';
import dayjs from 'dayjs';

const LongTermMemories: React.FC = () => {
  const [memories, setMemories] = useState<LongTermMemory[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedMemory, setSelectedMemory] = useState<LongTermMemory | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedDate, setSelectedDate] = useState<dayjs.Dayjs | null>(null);

  const fetchMemories = async () => {
    setLoading(true);
    try {
      const res = await longTermMemoriesApi.list();
      setMemories((res as any)?.items || []);
    } catch (error) {
      message.error('获取长期记忆失败');
    } finally {
      setLoading(false);
    }
  };

  const searchMemories = async () => {
    if (!searchQuery) {
      fetchMemories();
      return;
    }
    setLoading(true);
    try {
      const res = await longTermMemoriesApi.search(searchQuery);
      setMemories((res as any)?.items || []);
    } catch (error) {
      message.error('搜索失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchByDate = async () => {
    if (!selectedDate) {
      fetchMemories();
      return;
    }
    setLoading(true);
    try {
      const dateStr = selectedDate.format('YYYY-MM-DD');
      const res = await longTermMemoriesApi.getByDate(dateStr);
      const memory = (res as any)?.data || res;
      setMemories(memory ? [memory as LongTermMemory] : []);
    } catch (error) {
      message.error('获取指定日期记忆失败');
      setMemories([]);
    } finally {
      setLoading(false);
    }
  };

  const fetchRecent = async () => {
    setLoading(true);
    try {
      const res = await longTermMemoriesApi.getRecent(7);
      setMemories((res as any)?.items || []);
    } catch (error) {
      message.error('获取近期记忆失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMemories();
  }, []);

  const parseTags = (tags?: string): string[] => {
    if (!tags) return [];
    try {
      return JSON.parse(tags);
    } catch {
      return tags.split(',').map(t => t.trim());
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    {
      title: '日期',
      dataIndex: 'memory_date',
      render: (date: string) => date || '-',
    },
    {
      title: '摘要',
      dataIndex: 'summary',
      ellipsis: true,
      render: (summary: string, record: LongTermMemory) =>
        summary || record.content?.substring(0, 50) + (record.content?.length > 50 ? '...' : ''),
    },
    {
      title: '标签',
      dataIndex: 'tags',
      render: (tags: string) => (
        <Space size={4}>
          {parseTags(tags).map((tag, idx) => (
            <Tag key={idx} color="blue">{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      width: 100,
      render: (_: any, record: LongTermMemory) => (
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
      ),
    },
  ];

  return (
    <div>
      <Card
        title="长期记忆"
        extra={
          <Space>
            <DatePicker
              placeholder="选择日期"
              value={selectedDate}
              onChange={(date) => {
                setSelectedDate(date);
                if (date) {
                  fetchByDate();
                } else {
                  fetchMemories();
                }
              }}
              allowClear
            />
            <Button
              icon={<CalendarOutlined />}
              onClick={fetchRecent}
            >
              近7天
            </Button>
            <Input.Search
              placeholder="搜索记忆内容"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onSearch={searchMemories}
              style={{ width: 250 }}
              prefix={<SearchOutlined />}
            />
          </Space>
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
        title="长期记忆详情"
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
            <Descriptions.Item label="记忆日期">{selectedMemory.memory_date}</Descriptions.Item>
            <Descriptions.Item label="摘要">{selectedMemory.summary || '-'}</Descriptions.Item>
            <Descriptions.Item label="完整内容">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{selectedMemory.content}</pre>
            </Descriptions.Item>
            <Descriptions.Item label="标签">
              <Space size={4}>
                {parseTags(selectedMemory.tags).map((tag, idx) => (
                  <Tag key={idx} color="blue">{tag}</Tag>
                ))}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{selectedMemory.created_at}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{selectedMemory.updated_at}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default LongTermMemories;
