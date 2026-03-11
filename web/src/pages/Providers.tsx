import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  InputNumber,
  Switch,
  message,
  Popconfirm,
  Card,
  List,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckOutlined } from '@ant-design/icons';
import { providersApi } from '../api';
import type { LLMProvider, CreateProviderRequest, ModelInfo } from '../types';

const Providers: React.FC = () => {
  const [providers, setProviders] = useState<LLMProvider[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProvider, setEditingProvider] = useState<LLMProvider | null>(null);
  const [models, setModels] = useState<ModelInfo[]>([]);
  const [modelInput, setModelInput] = useState({ id: '', name: '' });
  const [form] = Form.useForm();

  const fetchProviders = async () => {
    setLoading(true);
    try {
      const res = await providersApi.list();
      // providersApi.list 返回 ListResponse { items, total }
      setProviders((res as any)?.items || []);
    } catch (error) {
      message.error('获取 Provider 列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProviders();
  }, []);

  const handleCreate = async (values: CreateProviderRequest) => {
    try {
      await providersApi.create(1, { ...values, supported_models: models });
      message.success('创建成功');
      setModalVisible(false);
      form.resetFields();
      setModels([]);
      fetchProviders();
    } catch (error) {
      message.error('创建失败');
    }
  };

  const handleUpdate = async (values: CreateProviderRequest) => {
    if (!editingProvider) return;
    try {
      await providersApi.update(editingProvider.id, { ...values, supported_models: models });
      message.success('更新成功');
      setModalVisible(false);
      setEditingProvider(null);
      form.resetFields();
      setModels([]);
      fetchProviders();
    } catch (error) {
      message.error('更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await providersApi.delete(id);
      message.success('删除成功');
      fetchProviders();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleSetDefault = async (provider: LLMProvider) => {
    try {
      await providersApi.setDefault(provider.user_id, provider.id);
      message.success('已设为默认 Provider');
      fetchProviders();
    } catch (error) {
      message.error('设置失败');
    }
  };

  const handleAddModel = () => {
    if (!modelInput.id || !modelInput.name) {
      message.warning('请输入模型 ID 和名称');
      return;
    }
    setModels([...models, { ...modelInput }]);
    setModelInput({ id: '', name: '' });
  };

  const handleRemoveModel = (index: number) => {
    setModels(models.filter((_, i) => i !== index));
  };

  const openModal = (provider?: LLMProvider) => {
    if (provider) {
      setEditingProvider(provider);
      const supportedModels = provider.supported_models ? JSON.parse(provider.supported_models) : [];
      setModels(supportedModels);
      form.setFieldsValue({
        provider_key: provider.provider_key,
        provider_name: provider.provider_name,
        api_base: provider.api_base,
        priority: provider.priority,
        is_default: provider.is_default,
        is_active: provider.is_active,
      });
    } else {
      setEditingProvider(null);
      setModels([]);
      form.resetFields();
    }
    setModalVisible(true);
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '标识', dataIndex: 'provider_key' },
    { title: '名称', dataIndex: 'provider_name' },
    { title: 'API Base', dataIndex: 'api_base', ellipsis: true },
    {
      title: '优先级',
      dataIndex: 'priority',
      render: (priority: number) => <Tag>{priority}</Tag>,
    },
    {
      title: '状态',
      render: (_: any, record: LLMProvider) => (
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
      render: (_: any, record: LLMProvider) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => openModal(record)}>
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
        title="LLM 提供商管理"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
            新建提供商
          </Button>
        }
      >
        <Table rowKey="id" columns={columns} dataSource={providers} loading={loading} />
      </Card>

      <Modal
        title={editingProvider ? '编辑提供商' : '新建提供商'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingProvider(null);
          setModels([]);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={700}
      >
        <Form form={form} layout="vertical" onFinish={editingProvider ? handleUpdate : handleCreate}>
          <Form.Item
            name="provider_key"
            label="提供商标识"
            rules={[{ required: true, message: '请输入标识' }]}
          >
            <Input placeholder="如: anthropic, openai, siliconflow" disabled={!!editingProvider} />
          </Form.Item>

          <Form.Item name="provider_name" label="显示名称">
            <Input placeholder="如: Anthropic, OpenAI" />
          </Form.Item>

          <Form.Item name="api_key" label="API Key">
            <Input.Password placeholder="sk-..." />
          </Form.Item>

          <Form.Item name="api_base" label="API Base URL">
            <Input placeholder="https://api.example.com/v1" />
          </Form.Item>

          <Form.Item name="priority" label="优先级" initialValue={0}>
            <InputNumber min={0} style={{ width: '100%' }} placeholder="数值越大优先级越高" />
          </Form.Item>

          <Form.Item label="支持的模型">
            <Space style={{ marginBottom: 16 }}>
              <Input
                placeholder="模型 ID"
                value={modelInput.id}
                onChange={(e) => setModelInput({ ...modelInput, id: e.target.value })}
              />
              <Input
                placeholder="模型名称"
                value={modelInput.name}
                onChange={(e) => setModelInput({ ...modelInput, name: e.target.value })}
              />
              <Button icon={<CheckOutlined />} onClick={handleAddModel}>
                添加
              </Button>
            </Space>
            <List
              size="small"
              bordered
              dataSource={models}
              renderItem={(item, index) => (
                <List.Item
                  actions={[
                    <Button type="link" danger onClick={() => handleRemoveModel(index)}>
                      删除
                    </Button>,
                  ]}
                >
                  {item.name} ({item.id})
                </List.Item>
              )}
            />
          </Form.Item>

          <Form.Item name="is_default" valuePropName="checked" initialValue={false}>
            <Switch checkedChildren="默认" unCheckedChildren="非默认" />
          </Form.Item>

          <Form.Item name="is_active" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Providers;