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
  Grid,
  Typography,
  Collapse,
  Divider,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, FileTextOutlined, ToolOutlined, ThunderboltOutlined, ApiOutlined } from '@ant-design/icons';
import { agentsApi, mcpServersApi, getCurrentUserCode } from '../api';
import type { Agent, CreateAgentRequest, MCPServer, AgentMCPBinding } from '../types';
import type { TableColumnsType } from 'antd';

// 可用技能列表
const AVAILABLE_SKILLS = [
  { value: 'cron', label: 'cron - 定时任务管理', description: '创建和管理定时任务' },
  { value: 'github', label: 'github - GitHub 操作', description: '仓库、Issue、PR 管理' },
  { value: 'skill-creator', label: 'skill-creator - 技能创建', description: '创建新技能' },
  { value: 'summarize', label: 'summarize - 文本总结', description: '总结长文本内容' },
  { value: 'tmux', label: 'tmux - 终端会话', description: '管理 tmux 会话' },
  { value: 'weather', label: 'weather - 天气查询', description: '查询天气信息' },
];

// 可用工具列表
const AVAILABLE_TOOLS = [
  { value: 'readfile', label: 'readfile - 读取文件', description: '读取文件内容' },
  { value: 'writefile', label: 'writefile - 写入文件', description: '写入文件内容' },
  { value: 'editfile', label: 'editfile - 编辑文件', description: '编辑文件内容' },
  { value: 'listdir', label: 'listdir - 列出目录', description: '列出目录内容' },
  { value: 'exec', label: 'exec - 执行命令', description: '执行 Shell 命令' },
  { value: 'websearch', label: 'websearch - 网页搜索', description: '搜索网页内容' },
  { value: 'webfetch', label: 'webfetch - 网页获取', description: '获取网页内容' },
  { value: 'message', label: 'message - 发送消息', description: '发送消息到渠道' },
  { value: 'cron', label: 'cron - 定时任务', description: '管理定时任务' },
  { value: 'askuser', label: 'askuser - 询问用户', description: '向用户提问' },
  { value: 'skill', label: 'skill - 技能调用', description: '调用技能' },
  { value: 'task_start', label: 'task_start - 启动任务', description: '启动后台任务' },
  { value: 'task_get', label: 'task_get - 获取任务', description: '获取任务状态' },
  { value: 'task_stop', label: 'task_stop - 停止任务', description: '停止后台任务' },
  { value: 'task_list', label: 'task_list - 列出任务', description: '列出所有任务' },
];

const { useBreakpoint } = Grid;
const { Title } = Typography;

// 安全解析 JSON 数组，返回数组长度或 0
const getArrayLength = (jsonStr: string | null | undefined): number => {
  if (!jsonStr || jsonStr === 'null') return 0;
  try {
    const parsed = JSON.parse(jsonStr);
    return Array.isArray(parsed) ? parsed.length : 0;
  } catch {
    return 0;
  }
};

// 安全解析 JSON 数组，返回数组或空数组
const safeParseArray = (jsonStr: string | null | undefined): any[] => {
  if (!jsonStr || jsonStr === 'null') return [];
  try {
    const parsed = JSON.parse(jsonStr);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
};

const Agents: React.FC = () => {
  const screens = useBreakpoint();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null);
  const [form] = Form.useForm();

  // MCP 绑定相关状态
  const [mcpModalVisible, setMcpModalVisible] = useState(false);
  const [mcpBindingAgent, setMcpBindingAgent] = useState<Agent | null>(null);
  const [mcpBindings, setMcpBindings] = useState<AgentMCPBinding[]>([]);
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [mcpLoading, setMcpLoading] = useState(false);
  const [mcpForm] = Form.useForm();

  const fetchAgents = async () => {
    setLoading(true);
    try {
      const userCode = getCurrentUserCode() || '';
      const res = await agentsApi.list(userCode);
      // ListResponse 直接返回 { items, total } 结构
      setAgents((res as any)?.items || []);
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
      const userCode = getCurrentUserCode() || '';
      await agentsApi.create(userCode, values);
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
      await agentsApi.setDefault(agent.user_code, agent.agent_code);
      message.success('已设为默认 Agent');
      fetchAgents();
    } catch (error) {
      message.error('设置失败');
    }
  };

  const handleToggleThinking = async (agent: Agent, enabled: boolean) => {
    try {
      await agentsApi.update(agent.id, {
        enable_thinking_process: enabled,
      } as CreateAgentRequest);
      message.success(enabled ? '已开启思考过程' : '已关闭思考过程');
      fetchAgents();
    } catch (error) {
      message.error('更新失败');
    }
  };

  const handleUpdateMaxIterations = async (agent: Agent, value: number) => {
    try {
      await agentsApi.update(agent.id, {
        max_iterations: value,
      } as CreateAgentRequest);
      message.success('已更新最大轮数');
      fetchAgents();
    } catch (error) {
      message.error('更新失败');
    }
  };

  const columns: TableColumnsType<Agent> = [
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
    ...(screens.xs ? [] : [
      {
        title: '描述',
        dataIndex: 'description',
        ellipsis: true,
      },
    ] as TableColumnsType<Agent>),
    {
      title: screens.xs ? '模型' : '模型配置',
      render: (_: any, record: Agent) => (
        <Space size="small">
          <Tag color={record.model_selection_mode === 'auto' ? 'blue' : 'green'}>
            {record.model_selection_mode === 'auto' ? '自动' : '指定'}
          </Tag>
          {record.model_selection_mode === 'specific' && !screens.xs && (
            <span>{record.model_name || record.model_id}</span>
          )}
        </Space>
      ),
    },
    {
      title: '思考',
      width: screens.xs ? 60 : 70,
      align: 'center',
      render: (_: any, record: Agent) => (
        <Switch
          size="small"
          checked={record.enable_thinking_process}
          onChange={(checked) => handleToggleThinking(record, checked)}
          checkedChildren="开"
          unCheckedChildren="关"
        />
      ),
    },
    {
      title: '轮数',
      width: screens.xs ? 70 : 90,
      align: 'center',
      render: (_: any, record: Agent) => (
        <Input
          type="number"
          size="small"
          min={1}
          max={50}
          defaultValue={record.max_iterations}
          onPressEnter={(e) => {
            const value = parseInt((e.target as HTMLInputElement).value, 10);
            if (!isNaN(value) && value !== record.max_iterations) {
              handleUpdateMaxIterations(record, value);
            }
          }}
          onBlur={(e) => {
            const value = parseInt(e.target.value, 10);
            if (!isNaN(value) && value !== record.max_iterations) {
              handleUpdateMaxIterations(record, value);
            }
          }}
          style={{ width: screens.xs ? 50 : 70, textAlign: 'center' }}
        />
      ),
    },
    {
      title: '技能',
      width: screens.xs ? 60 : 70,
      align: 'center',
      render: (_: any, record: Agent) => {
        const skillsCount = getArrayLength(record.skills_list);
        return (
          <Tag color={skillsCount === 0 ? 'default' : 'blue'}>
            {skillsCount === 0 ? '不限' : skillsCount}
          </Tag>
        );
      },
    },
    {
      title: '工具',
      width: screens.xs ? 60 : 70,
      align: 'center',
      render: (_: any, record: Agent) => {
        const toolsCount = getArrayLength(record.tools_list);
        return (
          <Tag color={toolsCount === 0 ? 'default' : 'cyan'}>
            {toolsCount === 0 ? '不限' : toolsCount}
          </Tag>
        );
      },
    },
    {
      title: '状态',
      render: (_: any, record: Agent) => (
        <Space size="small">
          {record.is_default && <Tag color="gold">默认</Tag>}
          <Tag color={record.is_active ? 'success' : 'default'}>
            {screens.xs ? '' : record.is_active ? '启用' : '禁用'}
          </Tag>
        </Space>
      ),
    },
    {
      title: '操作',
      width: screens.xs ? 100 : 300,
      render: (_: any, record: Agent) => (
        <Space
          size={[4, 4]}
          direction={screens.xs ? 'vertical' : 'horizontal'}
          wrap
        >
          <Button
            type="text"
            icon={<EditOutlined />}
            size={screens.xs ? 'small' : 'middle'}
            onClick={() => {
              setEditingAgent(record);
              form.setFieldsValue({
                ...record,
                skills_list: safeParseArray(record.skills_list),
                tools_list: safeParseArray(record.tools_list),
              });
              setModalVisible(true);
            }}
          >
            {screens.xs ? '' : '编辑'}
          </Button>
          {!screens.xs && (
            <Button
              type="text"
              icon={<ApiOutlined />}
              onClick={() => openMCPModal(record)}
            >
              MCP
            </Button>
          )}
          {!record.is_default && !screens.xs && (
            <Button type="text" onClick={() => handleSetDefault(record)}>
              默认
            </Button>
          )}
          <Popconfirm
            title="确认删除"
            description="删除后将无法恢复，是否继续？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              size={screens.xs ? 'small' : 'middle'}
            >
              {screens.xs ? '' : '删除'}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const modalWidth = screens.xs ? '100%' : screens.sm ? 600 : 800;

  // MCP 绑定相关函数
  const openMCPModal = async (agent: Agent) => {
    setMcpBindingAgent(agent);
    setMcpModalVisible(true);
    setMcpLoading(true);
    try {
      const [bindingsRes, serversRes] = await Promise.all([
        mcpServersApi.getAgentBindings(agent.id),
        mcpServersApi.list(),
      ]);
      setMcpBindings((bindingsRes.items || []) as AgentMCPBinding[]);
      setMcpServers((serversRes.items || []) as MCPServer[]);
    } catch (error) {
      message.error('获取 MCP 绑定信息失败');
    } finally {
      setMcpLoading(false);
    }
  };

  const handleCreateMcpBinding = async (values: { mcp_server_id: number }) => {
    if (!mcpBindingAgent) return;
    try {
      await mcpServersApi.createAgentBinding(mcpBindingAgent.id, {
        mcp_server_id: values.mcp_server_id,
        is_active: true,
      });
      message.success('绑定成功');
      mcpForm.resetFields();
      // 刷新绑定列表
      const res = await mcpServersApi.getAgentBindings(mcpBindingAgent.id);
      setMcpBindings((res.items || []) as AgentMCPBinding[]);
    } catch (error: any) {
      message.error(error?.response?.data?.error || '绑定失败');
    }
  };

  const handleDeleteMcpBinding = async (bindingId: number) => {
    if (!mcpBindingAgent) return;
    try {
      await mcpServersApi.deleteAgentBinding(mcpBindingAgent.id, bindingId);
      message.success('解绑成功');
      // 刷新绑定列表
      const res = await mcpServersApi.getAgentBindings(mcpBindingAgent.id);
      setMcpBindings((res.items || []) as AgentMCPBinding[]);
    } catch (error: any) {
      message.error(error?.response?.data?.error || '解绑失败');
    }
  };

  const handleToggleMcpBinding = async (binding: AgentMCPBinding) => {
    if (!mcpBindingAgent) return;
    try {
      await mcpServersApi.updateAgentBinding(mcpBindingAgent.id, binding.id, {
        is_active: !binding.is_active,
      });
      message.success(binding.is_active ? '已禁用' : '已启用');
      // 刷新绑定列表
      const res = await mcpServersApi.getAgentBindings(mcpBindingAgent.id);
      setMcpBindings((res.items || []) as AgentMCPBinding[]);
    } catch (error: any) {
      message.error(error?.response?.data?.error || '操作失败');
    }
  };

  const handleToggleAutoLoad = async (binding: AgentMCPBinding) => {
    if (!mcpBindingAgent) return;
    try {
      await mcpServersApi.updateAgentBinding(mcpBindingAgent.id, binding.id, {
        auto_load: !binding.auto_load,
      });
      message.success(!binding.auto_load ? '已设置自动加载' : '已取消自动加载');
      // 刷新绑定列表
      const res = await mcpServersApi.getAgentBindings(mcpBindingAgent.id);
      setMcpBindings((res.items || []) as AgentMCPBinding[]);
    } catch (error: any) {
      message.error(error?.response?.data?.error || '操作失败');
    }
  };

  return (
    <div>
      <Card
        title={<Title level={screens.xs ? 4 : 3} style={{ margin: 0 }}>Agent 管理</Title>}
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            size={screens.xs ? 'small' : 'middle'}
            onClick={() => {
              setEditingAgent(null);
              form.resetFields();
              setModalVisible(true);
            }}
          >
            {screens.xs ? '新建' : '新建 Agent'}
          </Button>
        }
        styles={{ body: { padding: screens.xs ? 12 : 24 } }}
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={agents}
          loading={loading}
          scroll={{ x: screens.xs ? 520 : 'max-content' }}
          size={screens.xs ? 'small' : 'middle'}
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
        width={modalWidth}
        style={{ top: screens.xs ? 0 : 100 }}
        styles={{ body: { padding: screens.xs ? 12 : 24 } }}
        destroyOnHidden
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

          <Divider>
            <ThunderboltOutlined /> 技能配置
          </Divider>

          <Form.Item
            name="skills_list"
            label="可用技能"
            initialValue={[]}
            extra="选择该 Agent 可以使用的技能"
          >
            <Select
              mode="multiple"
              placeholder="选择技能"
              options={AVAILABLE_SKILLS}
              style={{ width: '100%' }}
              allowClear
            />
          </Form.Item>

          <Divider>
            <ToolOutlined /> 工具配置
          </Divider>

          <Form.Item
            name="tools_list"
            label="可用工具"
            initialValue={[]}
            extra="选择该 Agent 可以使用的工具"
          >
            <Select
              mode="multiple"
              placeholder="选择工具"
              options={AVAILABLE_TOOLS}
              style={{ width: '100%' }}
              allowClear
            />
          </Form.Item>

          <Collapse
            ghost
            items={[
              {
                key: '1',
                label: <span><FileTextOutlined /> 配置文件编辑</span>,
                children: (
                  <>
                    <Form.Item name="identity_content" label="IDENTITY.md">
                      <Input.TextArea
                        rows={6}
                        placeholder="Agent 身份定义..."
                        style={{ fontFamily: 'monospace', fontSize: '12px' }}
                      />
                    </Form.Item>

                    <Form.Item name="soul_content" label="SOUL.md">
                      <Input.TextArea
                        rows={6}
                        placeholder="Agent 灵魂/核心定义..."
                        style={{ fontFamily: 'monospace', fontSize: '12px' }}
                      />
                    </Form.Item>

                    <Form.Item name="agents_content" label="AGENTS.md">
                      <Input.TextArea
                        rows={6}
                        placeholder="可用 Agents 定义..."
                        style={{ fontFamily: 'monospace', fontSize: '12px' }}
                      />
                    </Form.Item>

                    <Form.Item name="tools_content" label="TOOLS.md">
                      <Input.TextArea
                        rows={6}
                        placeholder="可用工具定义..."
                        style={{ fontFamily: 'monospace', fontSize: '12px' }}
                      />
                    </Form.Item>

                    <Form.Item name="user_content" label="USER.md">
                      <Input.TextArea
                        rows={6}
                        placeholder="用户信息/上下文..."
                        style={{ fontFamily: 'monospace', fontSize: '12px' }}
                      />
                    </Form.Item>
                  </>
                ),
              },
            ]}
          />
        </Form>
      </Modal>

      {/* MCP 绑定配置 Modal */}
      <Modal
        title={`配置 MCP - ${mcpBindingAgent?.name || ''}`}
        open={mcpModalVisible}
        onCancel={() => {
          setMcpModalVisible(false);
          setMcpBindingAgent(null);
          setMcpBindings([]);
          setMcpServers([]);
          mcpForm.resetFields();
        }}
        footer={null}
        width={screens.xs ? '100%' : 600}
      >
        <div style={{ marginBottom: 16 }}>
          <Title level={5}>添加 MCP Server 绑定</Title>
          <Form
            form={mcpForm}
            layout="inline"
            onFinish={handleCreateMcpBinding}
            style={{ display: 'flex', gap: 8 }}
          >
            <Form.Item
              name="mcp_server_id"
              rules={[{ required: true, message: '请选择 MCP Server' }]}
              style={{ flex: 1, marginBottom: 0 }}
            >
              <Select
                placeholder="选择 MCP Server"
                options={mcpServers
                  .filter(s => !mcpBindings.some(b => b.mcp_server_id === s.id))
                  .map(s => ({ value: s.id, label: `${s.name} (${s.code})` }))}
                style={{ width: '100%' }}
              />
            </Form.Item>
            <Button type="primary" onClick={() => mcpForm.submit()}>
              绑定
            </Button>
          </Form>
        </div>

        <Divider />

        <div>
          <Title level={5}>已绑定的 MCP Servers</Title>
          <Table
            dataSource={mcpBindings}
            rowKey="id"
            loading={mcpLoading}
            size="small"
            pagination={false}
            columns={[
              {
                title: 'MCP Server',
                render: (_, record: AgentMCPBinding) => (
                  <span>{record.mcp_server?.name || record.mcp_server_id}</span>
                ),
              },
              {
                title: '状态',
                width: 80,
                render: (_, record: AgentMCPBinding) => (
                  <Tag color={record.is_active ? 'success' : 'default'}>
                    {record.is_active ? '启用' : '禁用'}
                  </Tag>
                ),
              },
              {
                title: '自动加载',
                width: 100,
                render: (_, record: AgentMCPBinding) => (
                  <Tag color={record.auto_load ? 'blue' : 'default'}>
                    {record.auto_load ? '是' : '否'}
                  </Tag>
                ),
              },
              {
                title: '操作',
                width: 200,
                render: (_, record: AgentMCPBinding) => (
                  <Space size="small">
                    <Switch
                      size="small"
                      checked={record.is_active}
                      onChange={() => handleToggleMcpBinding(record)}
                    />
                    <Switch
                      size="small"
                      checked={record.auto_load}
                      checkedChildren="自加载"
                      unCheckedChildren="懒加载"
                      onChange={() => handleToggleAutoLoad(record)}
                    />
                    <Popconfirm
                      title="确认解绑"
                      description="确定要解绑这个 MCP Server 吗？"
                      onConfirm={() => handleDeleteMcpBinding(record.id)}
                    >
                      <Button type="text" danger size="small">
                        解绑
                      </Button>
                    </Popconfirm>
                  </Space>
                ),
              },
            ]}
          />
        </div>
      </Modal>
    </div>
  );
};

export default Agents;
