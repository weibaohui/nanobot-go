import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Card,
  Descriptions,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
} from '@ant-design/icons';
import { cronApi, channelsApi, getCurrentUserCode } from '../api';
import type { CronJob, CreateCronJobRequest, Channel } from '../types';

const CronJobs: React.FC = () => {
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [editingJob, setEditingJob] = useState<CronJob | null>(null);
  const [selectedJob, setSelectedJob] = useState<CronJob | null>(null);
  const [form] = Form.useForm();

  const fetchData = async () => {
    setLoading(true);
    try {
      const userCode = getCurrentUserCode() || '';
      const [jobsRes, channelsRes] = await Promise.all([
        cronApi.list(userCode),
        channelsApi.list(userCode),
      ]);
      // 列表 API 返回 ListResponse { items, total }
      setJobs((jobsRes as any)?.items || []);
      setChannels((channelsRes as any)?.items || []);
    } catch (error) {
      message.error('获取数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleCreate = async (values: CreateCronJobRequest) => {
    try {
      const userCode = getCurrentUserCode() || '';
      await cronApi.create(userCode, values.channel_code, values);
      message.success('创建成功');
      setModalVisible(false);
      form.resetFields();
      fetchData();
    } catch (error) {
      message.error('创建失败');
    }
  };

  const handleUpdate = async (values: CreateCronJobRequest) => {
    if (!editingJob) return;
    try {
      await cronApi.update(editingJob.id, values);
      message.success('更新成功');
      setModalVisible(false);
      setEditingJob(null);
      form.resetFields();
      fetchData();
    } catch (error) {
      message.error('更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await cronApi.delete(id);
      message.success('删除成功');
      fetchData();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleToggleStatus = async (job: CronJob) => {
    try {
      if (job.is_active) {
        await cronApi.disable(job.id);
        message.success('已禁用');
      } else {
        await cronApi.enable(job.id);
        message.success('已启用');
      }
      fetchData();
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleExecute = async (id: number) => {
    try {
      await cronApi.execute(id);
      message.success('已开始执行');
    } catch (error) {
      message.error('执行失败');
    }
  };

  const getStatusTag = (status?: string) => {
    const colors: Record<string, string> = {
      success: 'success',
      failed: 'error',
      running: 'processing',
    };
    const labels: Record<string, string> = {
      success: '成功',
      failed: '失败',
      running: '执行中',
    };
    return status ? <Tag color={colors[status]}>{labels[status]}</Tag> : null;
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '名称', dataIndex: 'name' },
    {
      title: '来源渠道',
      render: (_: any, record: CronJob) => {
        const channel = channels.find((c) => c.channel_code === record.channel_code);
        return channel?.name || record.channel_code;
      },
    },
    { title: 'Cron 表达式', dataIndex: 'cron_expression', render: (expr: string) => <code>{expr}</code> },
    {
      title: '模型配置',
      render: (_: any, record: CronJob) => (
        <Space>
          <Tag color={record.model_selection_mode === 'auto' ? 'blue' : 'green'}>
            {record.model_selection_mode === 'auto' ? '自动' : '指定'}
          </Tag>
          {record.model_selection_mode === 'specific' && record.model_name}
        </Space>
      ),
    },
    {
      title: '状态',
      render: (_: any, record: CronJob) => (
        <Space>
          <Tag color={record.is_active ? 'success' : 'default'}>
            {record.is_active ? '启用' : '禁用'}
          </Tag>
          {record.last_run_status && getStatusTag(record.last_run_status)}
        </Space>
      ),
    },
    {
      title: '执行统计',
      render: (_: any, record: CronJob) => (
        <span>
          成功: {record.run_count - record.fail_count} / 失败: {record.fail_count}
        </span>
      ),
    },
    {
      title: '操作',
      width: 280,
      render: (_: any, record: CronJob) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingJob(record);
              form.setFieldsValue({
                ...record,
              });
              setModalVisible(true);
            }}
          >
            编辑
          </Button>
          <Button
            type="text"
            icon={record.is_active ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
            onClick={() => handleToggleStatus(record)}
          >
            {record.is_active ? '禁用' : '启用'}
          </Button>
          <Button type="text" onClick={() => handleExecute(record.id)}>
            立即执行
          </Button>
          <Button
            type="text"
            onClick={() => {
              setSelectedJob(record);
              setDetailVisible(true);
            }}
          >
            详情
          </Button>
          <Popconfirm
            title="确认删除"
            description="删除后将无法恢复，是否继续？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="定时任务管理"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingJob(null);
              form.resetFields();
              setModalVisible(true);
            }}
          >
            新建任务
          </Button>
        }
      >
        <Table rowKey="id" columns={columns} dataSource={jobs} loading={loading} />
      </Card>

      <Modal
        title={editingJob ? '编辑定时任务' : '新建定时任务'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingJob(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={700}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={editingJob ? handleUpdate : handleCreate}
        >
          <Form.Item name="name" label="任务名称" rules={[{ required: true }]}>
            <Input placeholder="如：每日早报" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="任务描述" />
          </Form.Item>

          <Form.Item
            name="channel_id"
            label="来源渠道"
            rules={[{ required: true, message: '请选择来源渠道' }]}
          >
            <Select placeholder="选择来源渠道">
              {channels.map((channel) => (
                <Select.Option key={channel.id} value={channel.id}>
                  {channel.name} ({channel.type})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="cron_expression"
            label="Cron 表达式"
            rules={[{ required: true, message: '请输入 Cron 表达式' }]}
          >
            <Input placeholder="如：0 9 * * * (每天早上9点)" />
          </Form.Item>

          <Form.Item name="timezone" label="时区" initialValue="Asia/Shanghai">
            <Input placeholder="Asia/Shanghai" />
          </Form.Item>

          <Form.Item
            name="prompt"
            label="执行提示词"
            rules={[{ required: true, message: '请输入提示词' }]}
          >
            <Input.TextArea rows={4} placeholder="发送给 LLM 的提示词内容" />
          </Form.Item>

          <Form.Item
            name="model_selection_mode"
            label="模型选择模式"
            initialValue="auto"
          >
            <Select>
              <Select.Option value="auto">自动选择</Select.Option>
              <Select.Option value="specific">指定模型</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prev, curr) =>
              prev.model_selection_mode !== curr.model_selection_mode
            }
          >
            {({ getFieldValue }) =>
              getFieldValue('model_selection_mode') === 'specific' ? (
                <>
                  <Form.Item
                    name="model_id"
                    label="模型 ID"
                    rules={[{ required: true, message: '请输入模型 ID' }]}
                  >
                    <Input placeholder="如: claude-opus-4" />
                  </Form.Item>
                  <Form.Item name="model_name" label="模型名称">
                    <Input placeholder="如: Claude Opus 4" />
                  </Form.Item>
                </>
              ) : null
            }
          </Form.Item>

          <Form.Item name="target_channel_id" label="目标渠道">
            <Select placeholder="选择输出目标渠道（可选，默认原渠道）" allowClear>
              {channels.map((channel) => (
                <Select.Option key={channel.id} value={channel.id}>
                  {channel.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="target_user_id" label="目标用户 ID">
            <Input placeholder="定向推送的用户 ID（可选）" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="任务详情"
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedJob(null);
        }}
        footer={null}
        width={600}
      >
        {selectedJob && (
          <Descriptions column={1} bordered>
            <Descriptions.Item label="ID">{selectedJob.id}</Descriptions.Item>
            <Descriptions.Item label="名称">{selectedJob.name}</Descriptions.Item>
            <Descriptions.Item label="描述">{selectedJob.description}</Descriptions.Item>
            <Descriptions.Item label="Cron 表达式">
              <code>{selectedJob.cron_expression}</code>
            </Descriptions.Item>
            <Descriptions.Item label="时区">{selectedJob.timezone}</Descriptions.Item>
            <Descriptions.Item label="提示词">
              <pre style={{ whiteSpace: 'pre-wrap' }}>{selectedJob.prompt}</pre>
            </Descriptions.Item>
            <Descriptions.Item label="模型配置">
              {selectedJob.model_selection_mode === 'auto'
                ? '自动选择'
                : `${selectedJob.model_name} (${selectedJob.model_id})`}
            </Descriptions.Item>
            <Descriptions.Item label="最后执行">
              {selectedJob.last_run_at || '未执行'}
              {selectedJob.last_run_status && ` (${selectedJob.last_run_status})`}
            </Descriptions.Item>
            <Descriptions.Item label="下次执行">{selectedJob.next_run_at || '-'}</Descriptions.Item>
            <Descriptions.Item label="执行统计">
              总计: {selectedJob.run_count} / 失败: {selectedJob.fail_count}
            </Descriptions.Item>
            {selectedJob.last_run_result && (
              <Descriptions.Item label="最后结果">
                <pre style={{ whiteSpace: 'pre-wrap' }}>{selectedJob.last_run_result}</pre>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default CronJobs;
