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
import { agentsApi } from '../api';
import type { Agent, CreateAgentRequest } from '../types';

const Agents: React.FC = () => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null);
  const [form] = Form.useForm();

  const fetchAgents = async () => {
    setLoading(true);
    try {
      const res = await agentsApi.list();
      setAgents(res.data?.items || []);
    } catch (error) {
      message.error('获取 Agent 列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAgents();
  }, []);

  const handleCreate = async (values: CreateAgentRequest) => {
    try {
      await agentsApi.create(1, values);
      message.success('创建成功');
      setModalVisible(false);
      form.resetFields();
      fetchAgents();
    } catch (error) {
      message.error('创建失败');
    }
  };

  const handleUpdate = async (values: CreateAgentRequest) => {
    if (!editingAgent) return;
    try {
      await agentsApi.update(editingAgent.id, values);
      message.success('更新成功');
      setModalVisible(false);
      setEditingAgent(null);
      form.resetFields();
      fetchAgents();
    } catch (error) {
      message.error('更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await agentsApi.delete(id);
      message.success('删除成功');
      fetchAgents();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleSetDefault = async (agent: Agent) => {
    try {
      await agentsApi.setDefault(agent.user_id, agent.id);
      message.success('已设为默认 Agent');
      fetchAgents();
    } catch (error) {
      message.error('设置失败');
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
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
    },
    {
      title: '模型配置',
      render: (_: any, record: Agent) => (
        <Space>
          <Tag color={record.model_selection_mode === 'auto' ? 'blue' : 'green'}>
            {record.model_selection_mode === 'auto' ? '自动' : '指定'}
          </Tag>
          {record.model_selection_mode === 'specific' && (
            <span>{record.model_name || record.model_id}</span>
          )}
        </Space>
      ),
    },
    {
      title: '状态',
      render: (_: any, record: Agent) => (
        <Space>
          {record.is_default && <Tag color="gold">默认</Tag>}
          <Tag color={record.is_active ? 'success' : 'default'}>
            {record.is_active ? '启用' : '禁用'}
          </Tag>
        </Space>
      ),
    },
    {
      title: '操作',
      width: 250,
      render: (_: any, record: Agent) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingAgent(record);
              form.setFieldsValue({
                ...record,
                skills_list: record.skills_list ? JSON.parse(record.skills_list) : [],
                tools_list: record.tools_list ? JSON.parse(record.tools_list) : [],
              });
              setModalVisible(true);
            }}
          >
            编辑
          </Button>
          {!record.is_default && (
            <Button type="text" onClick={() => handleSetDefault(record)}>
              设为默认
            </Button>
          )}
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
        title="Agent 管理"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingAgent(null);
              form.resetFields();
              setModalVisible(true);
            }}
          >
            新建 Agent
          </Button>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={agents}
          loading={loading}
        />
      </Card>

      <Modal
        title={editingAgent ? '编辑 Agent' : '新建 Agent'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingAgent(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={editingAgent ? handleUpdate : handleCreate}
        >
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="Agent 名称" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="Agent 描述" />
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

          <Form.Item name="max_tokens" label="Max Tokens" initialValue={4096}>
            <Input type="number" />
          </Form.Item>

          <Form.Item name="temperature" label="Temperature" initialValue={0.7}>
            <Input type="number" step={0.1} min={0} max={2} />
          </Form.Item>

          <Form.Item name="max_iterations" label="Max Iterations" initialValue={15}>
            <Input type="number" />
          </Form.Item>

          <Form.Item name="is_default" valuePropName="checked" initialValue={false}>
            <Switch checkedChildren="默认" unCheckedChildren="非默认" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Agents;
