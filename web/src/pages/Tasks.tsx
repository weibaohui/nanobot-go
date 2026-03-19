import React, { useEffect, useState, useCallback, useMemo } from 'react';
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
  Input,
  Select,
  Badge,
  notification,
  Tooltip,
} from 'antd';
import {
  EyeOutlined,
  StopOutlined,
  ReloadOutlined,
  PlusOutlined,
  RedoOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  PauseCircleOutlined,
  HourglassOutlined,
} from '@ant-design/icons';
import { useTaskWebSocket } from '../hooks/useTaskWebSocket';
import { tasksApi } from '../api';
import type { Task, TaskDetail, TaskStatus } from '../types';
import { TaskStatusLabels, TaskStatusColors } from '../types';
import type { TableColumnsType } from 'antd';

const { useBreakpoint } = Grid;
const { Title, Text } = Typography;
const { Search } = Input;
const { Option } = Select;

// 任务状态选项
const statusOptions = [
  { value: 'pending', label: '等待中', color: 'default' },
  { value: 'running', label: '运行中', color: 'processing' },
  { value: 'finished', label: '已完成', color: 'success' },
  { value: 'failed', label: '失败', color: 'error' },
  { value: 'stopped', label: '已停止', color: 'warning' },
];

// 时间范围选项
const timeRangeOptions = [
  { value: 'all', label: '全部时间' },
  { value: 'today', label: '今天' },
  { value: 'week', label: '最近7天' },
  { value: 'month', label: '最近30天' },
];

const Tasks: React.FC = () => {
  const screens = useBreakpoint();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState<TaskDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [stopLoading, setStopLoading] = useState<string | null>(null);
  const [retryLoading, setRetryLoading] = useState<string | null>(null);
  const [newTaskWork, setNewTaskWork] = useState('');
  const [creating, setCreating] = useState(false);

  // 筛选状态
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<TaskStatus[]>([]);
  const [timeRange, setTimeRange] = useState<string>('all');

  // Task WebSocket 连接 - 独立的 WebSocket 端点，不需要 channel
  const { isConnected, isConnecting: wsConnecting } = useTaskWebSocket({
    onMessage: (event) => {
      console.log('[Task] WebSocket 消息:', event);

      switch (event.type) {
        case 'task_created':
          if (event.payload.id) {
            const newTask: Task = {
              id: event.payload.id,
              status: (event.payload.status as TaskStatus) || 'pending',
              work: event.payload.work || '',
              channel: event.payload.channel_name || '-',
              chat_id: '',
              created_at: event.payload.created_at || new Date().toISOString(),
            };
            setTasks((prev) => [newTask, ...prev]);
            message.info(`新任务 ${event.payload.id} 已创建`);
          }
          break;

        case 'task_updated':
          if (event.payload.id) {
            setTasks((prev) =>
              prev.map((t) =>
                t.id === event.payload.id
                  ? { ...t, status: (event.payload.status as TaskStatus) || t.status }
                  : t
              )
            );
          }
          break;

        case 'task_completed':
        case 'task_failed':
        case 'task_cancelled':
          if (event.payload.id) {
            setTasks((prev) =>
              prev.map((t) =>
                t.id === event.payload.id
                  ? { ...t, status: (event.payload.status as TaskStatus) || t.status }
                  : t
              )
            );
            // 显示通知
            const statusText =
              event.payload.status === 'finished'
                ? '已完成'
                : event.payload.status === 'failed'
                ? '失败'
                : '已停止';
            notification.info({
              message: `任务 ${event.payload.id} ${statusText}`,
              description: event.payload.error || '任务执行结束',
              placement: 'topRight',
              duration: 5,
            });
          }
          break;
      }
    },
    onConnect: () => {
      console.log('[Task] WebSocket 已连接');
    },
    onDisconnect: () => {
      console.log('[Task] WebSocket 已断开');
    },
  });

  // 获取任务列表
  const fetchTasks = useCallback(async () => {
    setLoading(true);
    try {
      // 构建查询参数
      const params = new URLSearchParams();
      if (statusFilter.length > 0) {
        params.append('status', statusFilter.join(','));
      }
      if (keyword) {
        params.append('keyword', keyword);
      }
      if (timeRange !== 'all') {
        const now = new Date();
        let since: Date;
        switch (timeRange) {
          case 'today':
            since = new Date(now.setHours(0, 0, 0, 0));
            break;
          case 'week':
            since = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
            break;
          case 'month':
            since = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
            break;
          default:
            since = new Date(0);
        }
        params.append('since', since.toISOString());
      }

      const res = await tasksApi.list(params.toString());
      setTasks((res.items || []) as Task[]);
    } catch (error) {
      console.error('获取任务列表失败:', error);
      message.error('获取任务列表失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }, [statusFilter, keyword, timeRange]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  // 获取任务详情
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
  const handleStopTask = async (taskId: string) => {
    setStopLoading(taskId);
    try {
      await tasksApi.stop(taskId);
      message.success('任务已停止');
      await fetchTasks();
    } catch (error: any) {
      console.error('停止任务失败:', error);
      message.error(error?.response?.data?.error || '停止任务失败');
    } finally {
      setStopLoading(null);
    }
  };

  // 重试任务
  const handleRetryTask = async (taskId: string) => {
    setRetryLoading(taskId);
    try {
      const res = await tasksApi.retry(taskId);
      message.success(`任务已重试，新任务ID: ${res.data.id}`);
      await fetchTasks();
    } catch (error: any) {
      console.error('重试任务失败:', error);
      message.error(error?.response?.data?.error || '重试任务失败');
    } finally {
      setRetryLoading(null);
    }
  };

  // 创建任务
  const handleCreateTask = async () => {
    if (!newTaskWork.trim()) {
      message.error('请输入任务内容');
      return;
    }
    setCreating(true);
    try {
      const res = await tasksApi.create({ work: newTaskWork.trim() });
      message.success(`任务已创建，任务ID: ${res.data.id}`);
      setNewTaskWork('');
      setCreateModalVisible(false);
      await fetchTasks();
    } catch (error: any) {
      console.error('创建任务失败:', error);
      message.error(error?.response?.data?.error || '创建任务失败');
    } finally {
      setCreating(false);
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

  // 获取状态图标
  const getStatusIcon = (status: TaskStatus) => {
    switch (status) {
      case 'running':
        return <ClockCircleOutlined spin />;
      case 'finished':
        return <CheckCircleOutlined />;
      case 'failed':
        return <CloseCircleOutlined />;
      case 'stopped':
        return <PauseCircleOutlined />;
      case 'pending':
        return <HourglassOutlined />;
      default:
        return null;
    }
  };

  // 表格列定义
  const columns: TableColumnsType<Task> = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        key: 'id',
        width: 80,
        render: (text: string) => <Text code>{text}</Text>,
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: screens.xs ? 100 : 120,
        render: (status: TaskStatus) => (
          <Tag
            color={TaskStatusColors[status]}
            icon={getStatusIcon(status)}
            style={{ display: 'flex', alignItems: 'center', gap: 4 }}
          >
            {TaskStatusLabels[status]}
          </Tag>
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
        width: screens.xs ? 120 : 180,
        fixed: 'right',
        render: (_: unknown, record: Task) => (
          <Space size="small">
            <Tooltip title="查看详情">
              <Button
                type="link"
                size="small"
                icon={<EyeOutlined />}
                onClick={() => openDetail(record.id)}
              />
            </Tooltip>
            {record.status === 'running' && (
              <Tooltip title="停止任务">
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
                  />
                </Popconfirm>
              </Tooltip>
            )}
            {(record.status === 'failed' || record.status === 'stopped') && (
              <Tooltip title="重试任务">
                <Button
                  type="link"
                  size="small"
                  icon={<RedoOutlined />}
                  loading={retryLoading === record.id}
                  onClick={() => handleRetryTask(record.id)}
                />
              </Tooltip>
            )}
          </Space>
        ),
      },
    ],
    [screens.xs, stopLoading, retryLoading]
  );

  // 筛选后的任务列表
  const filteredTasks = useMemo(() => {
    return tasks.filter((task) => {
      // 关键词筛选
      if (keyword && !task.work.toLowerCase().includes(keyword.toLowerCase())) {
        return false;
      }
      // 状态筛选
      if (statusFilter.length > 0 && !statusFilter.includes(task.status)) {
        return false;
      }
      return true;
    });
  }, [tasks, keyword, statusFilter]);

  return (
    <div>
      <Card
        title={
          <Space>
            <Title level={screens.xs ? 4 : 3} style={{ margin: 0 }}>
              后台任务
            </Title>
            <Badge
              count={tasks.filter((t) => t.status === 'running').length}
              showZero
              color="blue"
            />
            {wsConnecting ? (
              <Badge status="processing" text="连接中" />
            ) : isConnected ? (
              <Badge status="success" text="实时" />
            ) : (
              <Badge status="default" text="离线" />
            )}
          </Space>
        }
        extra={
          <Space>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalVisible(true)}
            >
              新建任务
            </Button>
            <Button
              icon={<ReloadOutlined />}
              onClick={fetchTasks}
              loading={loading}
            >
              刷新
            </Button>
          </Space>
        }
        styles={{ body: { padding: screens.xs ? 12 : 24 } }}
      >
        {/* 筛选栏 */}
        <Space wrap style={{ marginBottom: 16 }}>
          <Search
            placeholder="搜索任务内容"
            allowClear
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onSearch={fetchTasks}
            style={{ width: screens.xs ? 200 : 300 }}
          />
          <Select
            mode="multiple"
            placeholder="状态筛选"
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: screens.xs ? 200 : 300 }}
            allowClear
            maxTagCount="responsive"
          >
            {statusOptions.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                <Tag color={opt.color}>{opt.label}</Tag>
              </Option>
            ))}
          </Select>
          <Select
            placeholder="时间范围"
            value={timeRange}
            onChange={setTimeRange}
            style={{ width: 120 }}
          >
            {timeRangeOptions.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Space>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={filteredTasks}
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
                <div style={{ whiteSpace: 'pre-wrap' }}>
                  {selectedTask.work || '-'}
                </div>
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

      {/* 创建任务 Modal */}
      <Modal
        title="新建任务"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          setNewTaskWork('');
        }}
        onOk={handleCreateTask}
        confirmLoading={creating}
        okText="创建"
        cancelText="取消"
      >
        <div style={{ marginTop: 16 }}>
          <Typography.Text>任务内容</Typography.Text>
          <Input.TextArea
            rows={4}
            placeholder="请输入任务内容描述..."
            value={newTaskWork}
            onChange={(e) => setNewTaskWork(e.target.value)}
            style={{ marginTop: 8 }}
          />
        </div>
      </Modal>
    </div>
  );
};

export default Tasks;
