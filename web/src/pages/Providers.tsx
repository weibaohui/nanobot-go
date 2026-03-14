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
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, DatabaseOutlined, StarOutlined } from '@ant-design/icons';
import { providersApi, getCurrentUserCode } from '../api';
import type { LLMProvider, CreateProviderRequest, ModelInfo, EmbeddingModelInfo } from '../types';

const Providers: React.FC = () => {
  const [providers, setProviders] = useState<LLMProvider[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProvider, setEditingProvider] = useState<LLMProvider | null>(null);
  const [models, setModels] = useState<ModelInfo[]>([]);
  const [modelInput, setModelInput] = useState({ id: '', name: '', maxTokens: 8192 });
  const [defaultModel, setDefaultModel] = useState('');
  const [form] = Form.useForm();

  // 嵌入模型配置相关状态
  const [embeddingModalVisible, setEmbeddingModalVisible] = useState(false);
  const [embeddingModels, setEmbeddingModels] = useState<EmbeddingModelInfo[]>([]);
  const [defaultEmbeddingModel, setDefaultEmbeddingModel] = useState('');
  const [embeddingProvider, setEmbeddingProvider] = useState<LLMProvider | null>(null);
  const [embeddingInput, setEmbeddingInput] = useState({ id: '', name: '', dimensions: 1536 });
  const [embeddingLoading, setEmbeddingLoading] = useState(false);

  const fetchProviders = async () => {
    setLoading(true);
    try {
      const userCode = getCurrentUserCode() || '';
      const res = await providersApi.list(userCode);
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
      const userCode = getCurrentUserCode() || '';
      await providersApi.create(userCode, {
        ...values,
        supported_models: models,
        default_model: defaultModel,
      });
      message.success('创建成功');
      setModalVisible(false);
      form.resetFields();
      setModels([]);
      setDefaultModel('');
      fetchProviders();
    } catch (error) {
      message.error('创建失败');
    }
  };

  const handleUpdate = async (values: CreateProviderRequest) => {
    if (!editingProvider) return;
    try {
      await providersApi.update(editingProvider.id, {
        ...values,
        supported_models: models,
        default_model: defaultModel,
      });
      message.success('更新成功');
      setModalVisible(false);
      setEditingProvider(null);
      form.resetFields();
      setModels([]);
      setDefaultModel('');
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
      await providersApi.setDefault(provider.user_code, provider.id);
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
    if (models.some(m => m.id === modelInput.id)) {
      message.warning('模型 ID 已存在');
      return;
    }
    setModels([...models, { id: modelInput.id, name: modelInput.name, max_tokens: modelInput.maxTokens }]);
    setModelInput({ id: '', name: '', maxTokens: 8192 });
  };

  const handleRemoveModel = (index: number) => {
    const model = models[index];
    if (model.id === defaultModel) {
      setDefaultModel('');
    }
    setModels(models.filter((_, i) => i !== index));
  };

  const handleSetDefaultModel = (modelId: string) => {
    setDefaultModel(modelId);
    message.success('已设为默认模型');
  };

  // 嵌入模型配置相关方法
  const openEmbeddingModal = async (provider: LLMProvider) => {
    setEmbeddingProvider(provider);
    setEmbeddingModalVisible(true);
    setEmbeddingLoading(true);
    try {
      const res = await providersApi.getEmbeddingModels(provider.id);
      const data = res as any;
      setEmbeddingModels(data?.embedding_models || []);
      setDefaultEmbeddingModel(data?.default_embedding_model || '');
    } catch (error) {
      message.error('获取嵌入模型配置失败');
    } finally {
      setEmbeddingLoading(false);
    }
  };

  const handleAddEmbeddingModel = () => {
    if (!embeddingInput.id || !embeddingInput.name || !embeddingInput.dimensions) {
      message.warning('请输入模型 ID、名称和维度');
      return;
    }
    if (embeddingModels.some(m => m.id === embeddingInput.id)) {
      message.warning('模型 ID 已存在');
      return;
    }
    setEmbeddingModels([...embeddingModels, { ...embeddingInput }]);
    setEmbeddingInput({ id: '', name: '', dimensions: 1536 });
  };

  const handleRemoveEmbeddingModel = (index: number) => {
    const model = embeddingModels[index];
    if (model.id === defaultEmbeddingModel) {
      setDefaultEmbeddingModel('');
    }
    setEmbeddingModels(embeddingModels.filter((_, i) => i !== index));
  };

  const handleSetDefaultEmbeddingModel = (modelId: string) => {
    setDefaultEmbeddingModel(modelId);
    message.success('已设为默认嵌入模型');
  };

  const handleSaveEmbeddingModels = async () => {
    if (!embeddingProvider) return;
    try {
      await providersApi.updateEmbeddingModels(embeddingProvider.id, {
        embedding_models: embeddingModels,
        default_embedding_model: defaultEmbeddingModel,
      });
      message.success('嵌入模型配置保存成功');
      setEmbeddingModalVisible(false);
      fetchProviders();
    } catch (error) {
      message.error('保存失败');
    }
  };

  const openModal = (provider?: LLMProvider) => {
    if (provider) {
      setEditingProvider(provider);
      const supportedModels = provider.supported_models ? JSON.parse(provider.supported_models) : [];
      setModels(supportedModels);
      setDefaultModel(provider.default_model || '');
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
      setDefaultModel('');
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
      title: '嵌入模型',
      render: (_: any, record: LLMProvider) => {
        const hasModels = record.embedding_models && record.embedding_models !== '[]' && record.embedding_models !== '';
        const count = hasModels ? JSON.parse(record.embedding_models!).length : 0;
        return (
          <Tag color={hasModels ? 'success' : 'default'}>
            {hasModels ? `${count} 个模型` : '未配置'}
          </Tag>
        );
      },
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
      width: 320,
      render: (_: any, record: LLMProvider) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => openModal(record)}>
            编辑
          </Button>
          <Button type="text" icon={<DatabaseOutlined />} onClick={() => openEmbeddingModal(record)}>
            嵌入模型
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

      {/* 嵌入模型配置弹窗 */}
      <Modal
        title={`配置嵌入模型 - ${embeddingProvider?.provider_name || ''}`}
        open={embeddingModalVisible}
        onCancel={() => setEmbeddingModalVisible(false)}
        footer={[
          <Button key="cancel" onClick={() => setEmbeddingModalVisible(false)}>
            取消
          </Button>,
          <Button key="save" type="primary" onClick={handleSaveEmbeddingModels}>
            保存
          </Button>,
        ]}
        width={800}
      >
        <div style={{ marginBottom: 16 }}>
          <h4>嵌入模型列表</h4>
          <Table
            size="small"
            rowKey="id"
            loading={embeddingLoading}
            dataSource={embeddingModels}
            columns={[
              { title: '模型ID', dataIndex: 'id', ellipsis: true },
              { title: '名称', dataIndex: 'name' },
              { title: '维度', dataIndex: 'dimensions', width: 100 },
              {
                title: '默认',
                width: 80,
                render: (_: any, record: EmbeddingModelInfo) =>
                  record.id === defaultEmbeddingModel ? (
                    <Tag color="gold">默认</Tag>
                  ) : null,
              },
              {
                title: '操作',
                width: 180,
                render: (_: any, record: EmbeddingModelInfo, index: number) => (
                  <Space size="small">
                    {record.id !== defaultEmbeddingModel && (
                      <Button
                        type="text"
                        size="small"
                        icon={<StarOutlined />}
                        onClick={() => handleSetDefaultEmbeddingModel(record.id)}
                      >
                        设为默认
                      </Button>
                    )}
                    <Button
                      type="text"
                      size="small"
                      danger
                      onClick={() => handleRemoveEmbeddingModel(index)}
                    >
                      删除
                    </Button>
                  </Space>
                ),
              },
            ]}
            pagination={false}
            locale={{ emptyText: '暂无嵌入模型' }}
          />
        </div>

        <div style={{ marginTop: 24, padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
          <h4>添加新模型</h4>
          <Space style={{ marginTop: 8 }}>
            <Input
              placeholder="模型ID (如: text-embedding-3-small)"
              value={embeddingInput.id}
              onChange={(e) => setEmbeddingInput({ ...embeddingInput, id: e.target.value })}
              style={{ width: 220 }}
            />
            <Input
              placeholder="模型名称"
              value={embeddingInput.name}
              onChange={(e) => setEmbeddingInput({ ...embeddingInput, name: e.target.value })}
              style={{ width: 180 }}
            />
            <InputNumber
              placeholder="维度"
              value={embeddingInput.dimensions}
              onChange={(value) => setEmbeddingInput({ ...embeddingInput, dimensions: value || 1536 })}
              style={{ width: 100 }}
              min={1}
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddEmbeddingModel}>
              添加
            </Button>
          </Space>
        </div>
      </Modal>

      <Modal
        title={editingProvider ? '编辑提供商' : '新建提供商'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingProvider(null);
          setModels([]);
          setDefaultModel('');
          setModelInput({ id: '', name: '', maxTokens: 8192 });
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
            <Table
              size="small"
              rowKey="id"
              dataSource={models}
              columns={[
                { title: '模型ID', dataIndex: 'id', ellipsis: true },
                { title: '名称', dataIndex: 'name' },
                { title: '最大Token', dataIndex: 'max_tokens', width: 100 },
                {
                  title: '默认',
                  width: 80,
                  render: (_: any, record: ModelInfo) =>
                    record.id === defaultModel ? (
                      <Tag color="gold">默认</Tag>
                    ) : null,
                },
                {
                  title: '操作',
                  width: 180,
                  render: (_: any, record: ModelInfo, index: number) => (
                    <Space size="small">
                      {record.id !== defaultModel && (
                        <Button
                          type="text"
                          size="small"
                          icon={<StarOutlined />}
                          onClick={() => handleSetDefaultModel(record.id)}
                        >
                          设为默认
                        </Button>
                      )}
                      <Button
                        type="text"
                        size="small"
                        danger
                        onClick={() => handleRemoveModel(index)}
                      >
                        删除
                      </Button>
                    </Space>
                  ),
                },
              ]}
              pagination={false}
              locale={{ emptyText: '暂无模型' }}
              style={{ marginBottom: 16 }}
            />

            <div style={{ padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
              <h4 style={{ marginTop: 0 }}>添加新模型</h4>
              <Space style={{ marginTop: 8 }}>
                <Input
                  placeholder="模型ID (如: claude-3-opus)"
                  value={modelInput.id}
                  onChange={(e) => setModelInput({ ...modelInput, id: e.target.value })}
                  style={{ width: 220 }}
                />
                <Input
                  placeholder="模型名称"
                  value={modelInput.name}
                  onChange={(e) => setModelInput({ ...modelInput, name: e.target.value })}
                  style={{ width: 180 }}
                />
                <InputNumber
                  placeholder="最大Token"
                  value={modelInput.maxTokens}
                  onChange={(value) => setModelInput({ ...modelInput, maxTokens: value || 8192 })}
                  style={{ width: 120 }}
                  min={1}
                />
                <Button type="primary" icon={<PlusOutlined />} onClick={handleAddModel}>
                  添加
                </Button>
              </Space>
            </div>
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