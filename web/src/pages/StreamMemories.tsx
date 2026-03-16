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
  DatePicker,
  Space,
  Popconfirm,
} from 'antd';
import { EyeOutlined, CheckCircleOutlined, RocketOutlined, DeleteOutlined } from '@ant-design/icons';
import { streamMemoriesApi } from '../api';
import type { StreamMemory } from '../types';
import dayjs from 'dayjs';

const StreamMemories: React.FC = () => {
  const [memories, setMemories] = useState<StreamMemory[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedMemory, setSelectedMemory] = useState<StreamMemory | null>(null);
  const [showUnprocessed, setShowUnprocessed] = useState(false);
  const [upgradeModalVisible, setUpgradeModalVisible] = useState(false);
  const [upgradeDate, setUpgradeDate] = useState(dayjs());
  const [upgrading, setUpgrading] = useState(false);

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

  const handleDelete = async (id: number) => {
    try {
      await streamMemoriesApi.delete(id);
      message.success('删除成功');
      fetchMemories();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleUpgrade = async () => {
    setUpgrading(true);
    try {
      const dateStr = upgradeDate.format('YYYY-MM-DD');
      const res = await streamMemoriesApi.upgrade(dateStr);
      message.success(res.data?.message || '记忆升级成功');
      setUpgradeModalVisible(false);
      fetchMemories();
    } catch (error: any) {
      message.error(error?.response?.data?.error || '记忆升级失败');
    } finally {
      setUpgrading(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    {
      title: '用户',
      dataIndex: 'user_code',
      ellipsis: true,
      render: (user_code: string) => user_code || '-',
    },
    {
      title: 'Agent',
      dataIndex: 'agent_code',
      ellipsis: true,
      width: 120,
      render: (agent_code: string) => agent_code || '-',
    },
    {
      title: '日期',
      dataIndex: 'date',
      width: 120,
      render: (date: string) => date || '-',
    },
    {
      title: '摘要',
      dataIndex: 'summary',
      ellipsis: true,
      render: (summary: string) => summary?.substring(0, 50) + (summary?.length > 50 ? '...' : '') || '-',
    },
    {
      title: '状态',
      dataIndex: 'processed',
      width: 100,
      render: (processed: boolean) => (
        <Tag color={processed ? 'success' : 'warning'}>
          {processed ? '已处理' : '未处理'}
        </Tag>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      width: 220,
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
          <Popconfirm
            title="确认删除"
            description="确定要删除这条短期记忆吗？此操作不可恢复。"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="短期记忆 (Stream Memory)"
        extra={
          <Space>
            <Button
              type="primary"
              icon={<RocketOutlined />}
              onClick={() => setUpgradeModalVisible(true)}
            >
              升级记忆
            </Button>
            <Switch
              checked={showUnprocessed}
              onChange={setShowUnprocessed}
              checkedChildren="未处理"
              unCheckedChildren="全部"
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
            <Descriptions.Item label="用户编码">{selectedMemory.user_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="Agent编码">{selectedMemory.agent_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="日期">{selectedMemory.date || '-'}</Descriptions.Item>
            <Descriptions.Item label="AI 总结">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{selectedMemory.summary || '无'}</pre>
            </Descriptions.Item>
            <Descriptions.Item label="处理状态">
              <Tag color={selectedMemory.processed ? 'success' : 'warning'}>
                {selectedMemory.processed ? '已处理' : '未处理'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{selectedMemory.created_at}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{selectedMemory.updated_at}</Descriptions.Item>
            {selectedMemory.processed_at && (
              <Descriptions.Item label="处理时间">{selectedMemory.processed_at}</Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Modal>

      <Modal
        title="升级短期记忆到长期记忆"
        open={upgradeModalVisible}
        onOk={handleUpgrade}
        onCancel={() => setUpgradeModalVisible(false)}
        confirmLoading={upgrading}
        okText="开始升级"
        cancelText="取消"
      >
        <p>选择要升级的日期，系统将把所有未处理的短期记忆汇总为长期记忆。</p>
        <Space orientation="vertical" style={{ marginTop: 16 }}>
          <span>选择日期：</span>
          <DatePicker
            value={upgradeDate}
            onChange={(date) => date && setUpgradeDate(date)}
            format="YYYY-MM-DD"
          />
        </Space>
      </Modal>
    </div>
  );
};

export default StreamMemories;
