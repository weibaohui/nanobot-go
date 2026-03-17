import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Card,
  Grid,
  Typography,
  Descriptions,
  Empty,
  Spin,
  message,
  Popconfirm,
} from 'antd';
import { EyeOutlined, StopOutlined, ReloadOutlined } from '@ant-design/icons';
import { tasksApi } from '../api';
import type { Task, TaskDetail, TaskStatus } from '../types';
import { TaskStatusLabels, TaskStatusColors } from '../types';
import type { TableColumnsType } from 'antd';

const { useBreakpoint } = Grid;
const { Title, Text } = Typography;

const Tasks: React.FC = () => {
  const screens = useBreakpoint();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState<TaskDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [stopLoading, setStopLoading] = useState<string | null>(null);

  // fetchTasks 获取任务列表
  // API: GET /api/v1/tasks
  // 返回: { items: Task[] }
  const fetchTasks = async () => {
    setLoading(true);
    try {
      const res = await tasksApi.list();
      setTasks((res.items || []) as Task[]);
    } catch (error) {
      console.error('获取任务列表失败:', error);
      message.error('获取任务列表失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTasks();
  }, []);

  // 获取任务详情
  // API: GET /api/v1/tasks/:id
  // 返回: TaskDetail
  const openDetail = async (taskId: string) => {
    setDetailLoading(true);
    setDetailVisible(true);
    try {
      const res = await tasksApi.get(taskId);
      setSelectedTask(res || null);
    } catch (error) {
      console.error('获取任务详情失败:', error);
      message.error('获取任务详情失败');
      setSelectedTask(null);
    } finally {
      setDetailLoading(false);
    }
  };

  // 停止任务
  // API: POST /api/v1/tasks/:id/stop
  const handleStopTask = async (taskId: string) => {
    setStopLoading(taskId);
    try {
      await tasksApi.stop(taskId);
      message.success('任务已停止');
      // 刷新列表
      await fetchTasks();
    } catch (error: any) {
      console.error('停止任务失败:', error);
      message.error(error?.response?.data?.error || '停止任务失败');
    } finally {
      setStopLoading(null);
    }
  };

  // 格式化时间
  const formatTime = (timeStr?: string) => {
    if (!timeStr) return '-';
    try {
      const date = new Date(timeStr);
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return timeStr;
    }
  };

  const columns: TableColumnsType<Task> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
      render: (text: string) => (
        <Text code>{text}</Text>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: screens.xs ? 80 : 100,
      render: (status: TaskStatus) => (
        <Tag color={TaskStatusColors[status]}>{TaskStatusLabels[status]}</Tag>
      ),
    },
    {
      title: '任务内容',
      dataIndex: 'work',
      key: 'work',
      ellipsis: true,
      render: (text: string) => text || '-',
    },
    {
      title: '渠道',
      dataIndex: 'channel',
      key: 'channel',
      width: screens.xs ? 80 : 100,
      render: (text: string) => text || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: screens.xs ? 140 : 170,
      render: (time: string) => formatTime(time),
    },
    {
      title: '操作',
      key: 'action',
      width: screens.xs ? 120 : 160,
      render: (_: unknown, record: Task) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => openDetail(record.id)}
          >
            详情
          </Button>
          {record.status === 'running' && (
            <Popconfirm
              title="确定停止该任务？"
              description="停止后任务将无法恢复"
              onConfirm={() => handleStopTask(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="link"
                size="small"
                danger
                icon={<StopOutlined />}
                loading={stopLoading === record.id}
              >
                停止
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title={<Title level={screens.xs ? 4 : 3} style={{ margin: 0 }}>后台任务</Title>}
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={fetchTasks}
            loading={loading}
          >
            刷新
          </Button>
        }
        styles={{ body: { padding: screens.xs ? 12 : 24 } }}
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={tasks}
          loading={loading}
          scroll={{ x: screens.xs ? 700 : undefined }}
          size={screens.xs ? 'small' : 'middle'}
          locale={{
            emptyText: '暂无任务',
          }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>

      {/* 任务详情 Modal */}
      <Modal
        title={`任务详情 - ${selectedTask?.id || ''}`}
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedTask(null);
        }}
        footer={null}
        width={screens.xs ? '100%' : 800}
        style={{ top: screens.xs ? 0 : 50 }}
      >
        {detailLoading ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Spin />
          </div>
        ) : selectedTask ? (
          <div>
            <Descriptions column={screens.xs ? 1 : 2} bordered>
              <Descriptions.Item label="任务ID">
                <Text code>{selectedTask.id}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={TaskStatusColors[selectedTask.status]}>
                  {TaskStatusLabels[selectedTask.status]}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="渠道">
                {selectedTask.channel || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="ChatID">
                {selectedTask.chat_id || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {formatTime(selectedTask.created_at)}
              </Descriptions.Item>
              <Descriptions.Item label="完成时间">
                {formatTime(selectedTask.completed_at)}
              </Descriptions.Item>
              <Descriptions.Item label="任务内容" span={screens.xs ? 1 : 2}>
                <div style={{ whiteSpace: 'pre-wrap' }}>{selectedTask.work || '-'}</div>
              </Descriptions.Item>
            </Descriptions>

            {selectedTask.result && (
              <div style={{ marginTop: 24 }}>
                <Title level={5}>执行结果</Title>
                <div
                  style={{
                    background: '#f5f5f5',
                    padding: 16,
                    borderRadius: 8,
                    fontFamily: 'monospace',
                    fontSize: '12px',
                    whiteSpace: 'pre-wrap',
                    maxHeight: '200px',
                    overflow: 'auto',
                  }}
                >
                  {selectedTask.result}
                </div>
              </div>
            )}

            {selectedTask.logs && selectedTask.logs.length > 0 && (
              <div style={{ marginTop: 24 }}>
                <Title level={5}>执行日志</Title>
                <div
                  style={{
                    background: '#1e1e1e',
                    color: '#d4d4d4',
                    padding: 16,
                    borderRadius: 8,
                    fontFamily: 'monospace',
                    fontSize: '12px',
                    whiteSpace: 'pre-wrap',
                    maxHeight: '300px',
                    overflow: 'auto',
                  }}
                >
                  {selectedTask.logs.map((log, index) => (
                    <div key={`log-${index}-${log.slice(0, 20)}`}>{log}</div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ) : (
          <Empty description="任务不存在" />
        )}
      </Modal>
    </div>
  );
};

export default Tasks;
