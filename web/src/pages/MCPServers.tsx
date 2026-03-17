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
  Grid,
  Typography,
  Descriptions,
  Tooltip,
  Collapse,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, ApiOutlined, ReloadOutlined } from '@ant-design/icons';
import { mcpServersApi } from '../api';
import type { MCPServer, CreateMCPServerRequest, MCPTransportType, MCPStatus } from '../types';
import type { TableColumnsType } from 'antd';

const { useBreakpoint } = Grid;
const { Title } = Typography;
const { TextArea } = Input;

const MCPTransportTypeOptions = [
  { value: 'stdio', label: 'stdio（标准输入输出）' },
  { value: 'http', label: 'HTTP' },
  { value: 'sse', label: 'SSE（服务器发送事件）' },
];

const MCPStatusLabels: Record<MCPStatus, { text: string; color: string }> = {
  inactive: { text: '未连接', color: 'default' },
  active: { text: '已连接', color: 'success' },
  error: { text: '错误', color: 'error' },
};

const MCPServers: React.FC = () => {
  const screens = useBreakpoint();
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [editingServer, setEditingServer] = useState<MCPServer | null>(null);
  const [selectedServer, setSelectedServer] = useState<MCPServer | null>(null);
  const [form] = Form.useForm();
  const [testingId, setTestingId] = useState<number | null>(null);
  const [refreshingId, setRefreshingId] = useState<number | null>(null);
  const [toolCounts, setToolCounts] = useState<Record<number, number>>({});

  const fetchServers = async () => {
    setLoading(true);
    try {
      const res = await mcpServersApi.list();
      const serverList = (res.items || []) as MCPServer[];
      setServers(serverList);

      // 获取每个服务器的工具数量
      const counts: Record<number, number> = {};
      await Promise.all(
        serverList.map(async (server) => {
          try {
            const toolsRes = await mcpServersApi.listTools(server.id);
            counts[server.id] = toolsRes.items?.length || 0;
          } catch {
            counts[server.id] = 0;
          }
        })
      );
      setToolCounts(counts);
    } catch (error) {
      message.error('获取 MCP Server 列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServers();
  }, []);

  const handleCreate = async (values: CreateMCPServerRequest) => {
    try {
      // 解析环境变量
      let envVars: Record<string, string> | undefined;
      if (values.env_vars && typeof values.env_vars === 'string') {
        try {
          envVars = JSON.parse(values.env_vars as string);
        } catch {
          message.error('环境变量格式错误，请使用有效的 JSON 格式');
          return;
        }
      }

      // 解析参数
      let args: string[] | undefined;
      if (values.args && typeof values.args === 'string') {
        try {
          args = JSON.parse(values.args as string);
          if (!Array.isArray(args)) {
            message.error('参数必须是 JSON 数组格式');
            return;
          }
        } catch {
          message.error('参数格式错误，请使用有效的 JSON 数组格式');
          return;
        }
      }

      const data = {
        ...values,
        env_vars: envVars,
        args,
      };

      await mcpServersApi.create(data as CreateMCPServerRequest);
      message.success('创建成功');
      setModalVisible(false);
      form.resetFields();
      fetchServers();
    } catch (error: any) {
      message.error(error?.response?.data?.error || '创建失败');
    }
  };

  const handleUpdate = async (values: CreateMCPServerRequest) => {
    if (!editingServer) return;
    try {
      // 解析环境变量
      let envVars: Record<string, string> | undefined;
      if (values.env_vars && typeof values.env_vars === 'string') {
        try {
          envVars = JSON.parse(values.env_vars as string);
        } catch {
          message.error('环境变量格式错误，请使用有效的 JSON 格式');
          return;
        }
      }

      // 解析参数
      let args: string[] | undefined;
      if (values.args && typeof values.args === 'string') {
        try {
          args = JSON.parse(values.args as string);
          if (!Array.isArray(args)) {
            message.error('参数必须是 JSON 数组格式');
            return;
          }
        } catch {
          message.error('参数格式错误，请使用有效的 JSON 数组格式');
          return;
        }
      }

      const data = {
        ...values,
        env_vars: envVars,
        args,
      };

      await mcpServersApi.update(editingServer.id, data);
      message.success('更新成功');
      setModalVisible(false);
      setEditingServer(null);
      form.resetFields();
      fetchServers();
    } catch (error: any) {
      message.error(error?.response?.data?.error || '更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await mcpServersApi.delete(id);
      message.success('删除成功');
      fetchServers();
    } catch (error: any) {
      message.error(error?.response?.data?.error || '删除失败');
    }
  };

  const handleTest = async (server: MCPServer) => {
    setTestingId(server.id);
    try {
      await mcpServersApi.test(server.id);
      message.success('连接测试成功');
      fetchServers();
    } catch (error: any) {
      message.error(error?.response?.data?.error || '连接测试失败');
    } finally {
      setTestingId(null);
    }
  };

  const handleRefresh = async (server: MCPServer) => {
    setRefreshingId(server.id);
    try {
      await mcpServersApi.refreshCapabilities(server.id);
      message.success('刷新能力成功');
      fetchServers();
    } catch (error: any) {
      message.error(error?.response?.data?.error || '刷新能力失败');
    } finally {
      setRefreshingId(null);
    }
  };

  const openDetail = (server: MCPServer) => {
    setSelectedServer(server);
    setDetailVisible(true);
  };

  const columns: TableColumnsType<MCPServer> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: screens.xs ? 50 : 60,
    },
    {
      title: '名称',
      dataIndex: 'name',
      ellipsis: true,
    },
    {
      title: '编码',
      dataIndex: 'code',
      ellipsis: true,
      width: 120,
    },
    {
      title: '传输类型',
      dataIndex: 'transport_type',
      width: screens.xs ? 80 : 150,
      render: (type: MCPTransportType) => {
        const labels: Record<MCPTransportType, string> = {
          stdio: 'stdio',
          http: 'HTTP',
          sse: 'SSE',
        };
        return <Tag>{labels[type] || type}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: screens.xs ? 70 : 100,
      render: (status: MCPStatus) => {
        const { text, color } = MCPStatusLabels[status];
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: '工具数',
      width: screens.xs ? 60 : 80,
      align: 'center',
      render: (_: any, record: MCPServer) => {
        const count = toolCounts[record.id] ?? 0;
        return <Tag color={count > 0 ? 'blue' : 'default'}>{count}</Tag>;
      },
    },
    {
      title: '操作',
      width: screens.xs ? 120 : 280,
      render: (_: any, record: MCPServer) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<EyeOutlined />}
              size="small"
              onClick={() => openDetail(record)}
            />
          </Tooltip>
          <Tooltip title="测试连接">
            <Button
              type="text"
              icon={<ApiOutlined />}
              size="small"
              loading={testingId === record.id}
              onClick={() => handleTest(record)}
            />
          </Tooltip>
          <Tooltip title="刷新能力">
            <Button
              type="text"
              icon={<ReloadOutlined />}
              size="small"
              loading={refreshingId === record.id}
              onClick={() => handleRefresh(record)}
            />
          </Tooltip>
          {!screens.xs && (
            <Button
              type="text"
              icon={<EditOutlined />}
              size="small"
              onClick={() => {
                setEditingServer(record);
                form.setFieldsValue({
                  ...record,
                  env_vars: record.env_vars ? JSON.stringify(record.env_vars, null, 2) : '',
                  args: record.args ? JSON.stringify(record.args, null, 2) : '[]',
                });
                setModalVisible(true);
              }}
            >
              编辑
            </Button>
          )}
          <Popconfirm
            title="确认删除"
            description="确定要删除这个 MCP Server 吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
          >
            <Button type="text" danger icon={<DeleteOutlined />} size="small">
              {screens.xs ? '' : '删除'}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title={<Title level={screens.xs ? 4 : 3} style={{ margin: 0 }}>MCP Server 管理</Title>}
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            size={screens.xs ? 'small' : 'middle'}
            onClick={() => {
              setEditingServer(null);
              form.resetFields();
              setModalVisible(true);
            }}
          >
            {screens.xs ? '新建' : '新建 MCP Server'}
          </Button>
        }
        styles={{ body: { padding: screens.xs ? 12 : 24 } }}
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={servers}
          loading={loading}
          scroll={{ x: screens.xs ? 500 : undefined }}
          size={screens.xs ? 'small' : 'middle'}
        />
      </Card>

      {/* 创建/编辑 Modal */}
      <Modal
        title={editingServer ? '编辑 MCP Server' : '新建 MCP Server'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingServer(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={screens.xs ? '100%' : 600}
        style={{ top: screens.xs ? 0 : 100 }}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={editingServer ? handleUpdate : handleCreate}
        >
          <Form.Item
            name="code"
            label="编码"
            rules={[{ required: true, message: '请输入编码' }, { pattern: /^[a-z0-9_-]+$/, message: '只能使用小写字母、数字、下划线和横线' }]}
          >
            <Input placeholder="如: my-mcp-server" disabled={!!editingServer} />
          </Form.Item>

          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="MCP Server 名称" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea rows={2} placeholder="MCP Server 描述" />
          </Form.Item>

          <Form.Item
            name="transport_type"
            label="传输类型"
            rules={[{ required: true, message: '请选择传输类型' }]}
            initialValue="stdio"
          >
            <Select options={MCPTransportTypeOptions} />
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prev, curr) =>
              prev.transport_type !== curr.transport_type
            }
          >
            {({ getFieldValue }) => {
              const transportType = getFieldValue('transport_type');
              if (transportType === 'stdio') {
                return (
                  <>
                    <Form.Item
                      name="command"
                      label="启动命令"
                      rules={[{ required: true, message: 'stdio 类型需要指定启动命令' }]}
                    >
                      <Input placeholder="如: npx、python、/usr/local/bin/mcp-server" />
                    </Form.Item>
                    <Form.Item
                      name="args"
                      label="参数（JSON 数组）"
                      initialValue="[]"
                    >
                      <TextArea
                        rows={3}
                        placeholder='如: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"]'
                        style={{ fontFamily: 'monospace' }}
                      />
                    </Form.Item>
                  </>
                );
              }
              return (
                <Form.Item
                  name="url"
                  label="服务 URL"
                  rules={[{ required: true, message: 'HTTP/SSE 类型需要指定服务 URL' }]}
                >
                  <Input placeholder="如: http://localhost:3000/sse" />
                </Form.Item>
              );
            }}
          </Form.Item>

          <Form.Item
            name="env_vars"
            label="环境变量（JSON 对象）"
          >
            <TextArea
              rows={3}
              placeholder='如: {"API_KEY": "xxx", "DEBUG": "true"}'
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情 Modal */}
      <Modal
        title="MCP Server 详情"
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedServer(null);
        }}
        footer={null}
        width={screens.xs ? '100%' : 700}
      >
        {selectedServer && (
          <Descriptions column={screens.xs ? 1 : 2} bordered>
            <Descriptions.Item label="ID">{selectedServer.id}</Descriptions.Item>
            <Descriptions.Item label="编码">{selectedServer.code}</Descriptions.Item>
            <Descriptions.Item label="名称" span={screens.xs ? 1 : 2}>{selectedServer.name}</Descriptions.Item>
            <Descriptions.Item label="描述" span={screens.xs ? 1 : 2}>
              {selectedServer.description || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="传输类型">
              <Tag>{selectedServer.transport_type}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={MCPStatusLabels[selectedServer.status].color}>
                {MCPStatusLabels[selectedServer.status].text}
              </Tag>
            </Descriptions.Item>
            {selectedServer.transport_type === 'stdio' ? (
              <>
                <Descriptions.Item label="启动命令" span={screens.xs ? 1 : 2}>
                  <code style={{ fontSize: '12px' }}>{selectedServer.command}</code>
                </Descriptions.Item>
                <Descriptions.Item label="参数" span={screens.xs ? 1 : 2}>
                  <pre style={{ fontSize: '12px', margin: 0 }}>
                    {JSON.stringify(selectedServer.args || [], null, 2)}
                  </pre>
                </Descriptions.Item>
              </>
            ) : (
              <Descriptions.Item label="URL" span={screens.xs ? 1 : 2}>
                <code style={{ fontSize: '12px' }}>{selectedServer.url}</code>
              </Descriptions.Item>
            )}
            {selectedServer.env_vars && Object.keys(selectedServer.env_vars).length > 0 && (
              <Descriptions.Item label="环境变量" span={screens.xs ? 1 : 2}>
                <pre style={{ fontSize: '12px', margin: 0 }}>
                  {JSON.stringify(selectedServer.env_vars, null, 2)}
                </pre>
              </Descriptions.Item>
            )}
            {selectedServer.error_message && (
              <Descriptions.Item label="错误信息" span={screens.xs ? 1 : 2}>
                <span style={{ color: 'red' }}>{selectedServer.error_message}</span>
              </Descriptions.Item>
            )}
            <Descriptions.Item label="工具数量">
              {selectedServer.capabilities?.length || 0}
            </Descriptions.Item>
            <Descriptions.Item label="最后连接时间">
              {selectedServer.last_connected_at
                ? new Date(selectedServer.last_connected_at).toLocaleString()
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间" span={screens.xs ? 1 : 2}>
              {new Date(selectedServer.created_at).toLocaleString()}
            </Descriptions.Item>
          </Descriptions>
        )}

        {selectedServer?.capabilities && selectedServer.capabilities.length > 0 && (
          <div style={{ marginTop: 24 }}>
            <Collapse
              items={[
                {
                  key: 'tools',
                  label: `可用工具 (${selectedServer.capabilities.length})`,
                  children: (
                    <Space direction="vertical" style={{ width: '100%' }}>
                      {selectedServer.capabilities.map((tool) => (
                        <Card
                          key={tool.name}
                          size="small"
                          title={tool.name}
                          style={{ width: '100%' }}
                        >
                          <p>{tool.description || '无描述'}</p>
                          {tool.input_schema && (
                            <pre style={{ fontSize: '11px', background: '#f5f5f5', padding: 8 }}>
                              {JSON.stringify(tool.input_schema, null, 2)}
                            </pre>
                          )}
                        </Card>
                      ))}
                    </Space>
                  ),
                },
              ]}
            />
          </div>
        )}
      </Modal>
    </div>
  );
};

export default MCPServers;
