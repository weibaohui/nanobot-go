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
  Switch,
  message,
  Popconfirm,
  Card,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { channelsApi, agentsApi, getCurrentUserCode } from '../api';
import type { Channel, CreateChannelRequest, Agent, ChannelType } from '../types';
import { ChannelTypeLabels } from '../types';

const Channels: React.FC = () => {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);
  const [form] = Form.useForm();

  const fetchData = async () => {
    setLoading(true);
    try {
      const userCode = getCurrentUserCode() || '';
      const [channelsRes, agentsRes] = await Promise.all([
        channelsApi.list(userCode),
        agentsApi.list(userCode),
      ]);
      // channelsApi.list 和 agentsApi.list 返回 ListResponse { items, total }
      // 但 client 响应拦截器返回 response.data，所以需要调整访问路径
      // 列表 API 返回的是 ListResponse，需要访问 .items
      // 单个 API 返回的是直接对象，需要访问 .data
      setChannels((channelsRes as any)?.items || []);
      setAgents((agentsRes as any)?.items || []);
    } catch (error) {
      message.error('获取数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleCreate = async (values: CreateChannelRequest) => {
    try {
      const userCode = getCurrentUserCode() || '';
      await channelsApi.create(userCode, values);
      message.success('创建成功');
      setModalVisible(false);
      form.resetFields();
      fetchData();
    } catch (error) {
      message.error('创建失败');
    }
  };

  const handleUpdate = async (values: CreateChannelRequest) => {
    if (!editingChannel) return;
    try {
      await channelsApi.update(editingChannel.id, values);
      message.success('更新成功');
      setModalVisible(false);
      setEditingChannel(null);
      form.resetFields();
      fetchData();
    } catch (error) {
      message.error('更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await channelsApi.delete(id);
      message.success('删除成功');
      fetchData();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const getConfigFields = (type: ChannelType) => {
    switch (type) {
      case 'feishu':
        return (
          <>
            <Form.Item name={['config', 'app_id']} label="App ID" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item name={['config', 'app_secret']} label="App Secret" rules={[{ required: true }]}>
              <Input.Password />
            </Form.Item>
            <Form.Item name={['config', 'encrypt_key']} label="Encrypt Key">
              <Input />
            </Form.Item>
            <Form.Item name={['config', 'verification_token']} label="Verification Token">
              <Input />
            </Form.Item>
          </>
        );
      case 'dingtalk':
        return (
          <>
            <Form.Item name={['config', 'client_id']} label="Client ID" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item name={['config', 'client_secret']} label="Client Secret" rules={[{ required: true }]}>
              <Input.Password />
            </Form.Item>
          </>
        );
      case 'matrix':
        return (
          <>
            <Form.Item name={['config', 'homeserver']} label="Homeserver" rules={[{ required: true }]}>
              <Input placeholder="https://matrix.org" />
            </Form.Item>
            <Form.Item name={['config', 'user_id']} label="User ID" rules={[{ required: true }]}>
              <Input placeholder="@user:matrix.org" />
            </Form.Item>
            <Form.Item name={['config', 'token']} label="Token" rules={[{ required: true }]}>
              <Input.Password />
            </Form.Item>
          </>
        );
      case 'websocket':
        return (
          <>
            <Form.Item name={['config', 'addr']} label="监听地址" initialValue=":8080">
              <Input />
            </Form.Item>
            <Form.Item name={['config', 'path']} label="路径" initialValue="/ws">
              <Input />
            </Form.Item>
          </>
        );
      default:
        return null;
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '名称',
      dataIndex: 'name',
    },
    {
      title: '类型',
      dataIndex: 'type',
      render: (type: ChannelType) => <Tag>{ChannelTypeLabels[type] || type}</Tag>,
    },
    {
      title: '绑定 Agent',
      render: (_: any, record: Channel) => {
        const agent = agents.find(a => a.agent_code === record.agent_code);
        return agent ? agent.name : <span style={{ color: '#999' }}>未绑定</span>;
      },
    },
    {
      title: '状态',
      render: (_: any, record: Channel) => (
        <Tag color={record.is_active ? 'success' : 'default'}>
          {record.is_active ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      width: 200,
      render: (_: any, record: Channel) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingChannel(record);
              const config = record.config ? JSON.parse(record.config) : {};
              const allowFrom = record.allow_from ? JSON.parse(record.allow_from) : [];
              form.setFieldsValue({
                ...record,
                config,
                allow_from: allowFrom.join('\n'),
              });
              setModalVisible(true);
            }}
          >
            编辑
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
        title="渠道管理"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingChannel(null);
              form.resetFields();
              setModalVisible(true);
            }}
          >
            新建渠道
          </Button>
        }
      >
        <Table rowKey="id" columns={columns} dataSource={channels} loading={loading} />
      </Card>

      <Modal
        title={editingChannel ? '编辑渠道' : '新建渠道'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingChannel(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            const data: CreateChannelRequest = {
              ...values,
              allow_from: values.allow_from ? values.allow_from.split('\n').filter(Boolean) : [],
            };
            editingChannel ? handleUpdate(data) : handleCreate(data);
          }}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="渠道名称" />
          </Form.Item>

          <Form.Item name="type" label="类型" rules={[{ required: true }]}>
            <Select placeholder="选择渠道类型">
              {Object.entries(ChannelTypeLabels).map(([key, label]) => (
                <Select.Option key={key} value={key}>{label}</Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="agent_code" label="绑定 Agent">
            <Select placeholder="选择要绑定的 Agent" allowClear>
              {agents.map(agent => (
                <Select.Option key={agent.agent_code} value={agent.agent_code}>{agent.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="allow_from" label="白名单用户">
            <Input.TextArea
              rows={3}
              placeholder="每行一个用户ID，留空表示允许所有用户"
            />
          </Form.Item>

          <Form.Item noStyle shouldUpdate={(prev, curr) => prev.type !== curr.type}>
            {({ getFieldValue }) => getConfigFields(getFieldValue('type'))}
          </Form.Item>

          <Form.Item name="is_active" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Channels;
