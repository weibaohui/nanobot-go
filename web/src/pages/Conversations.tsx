import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Table,
  Card,
  Tag,
  Input,
  Button,
  message,
  Descriptions,
  Modal,
  Tree,
  Space,
  Typography,
  Badge,
  Divider,
  Tooltip,
  Row,
  Col,
  Statistic,
  DatePicker,
  Select,
  Form,
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  BranchesOutlined,
  MessageOutlined,
  BarChartOutlined,
  FilterOutlined,
  ClearOutlined,
} from '@ant-design/icons';
import { conversationsApi } from '../api';
import type { ConversationRecord } from '../types';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;

// 链路树节点类型
interface TraceNode {
  key: string;
  title: React.ReactNode;
  children?: TraceNode[];
  record: ConversationRecord;
  duration?: number;
}

// 聊天消息类型
interface ChatMessage {
  id: number;
  role: string;
  content: string;
  tokens: number;
  timestamp: string;
  agentName?: string;
}

const Conversations: React.FC = () => {
  const navigate = useNavigate();
  const [records, setRecords] = useState<ConversationRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [sessionKey, setSessionKey] = useState('');
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<ConversationRecord | null>(null);

  // 链路可视化状态
  const [traceVisible, setTraceVisible] = useState(false);
  const [traceRecords, setTraceRecords] = useState<ConversationRecord[]>([]);
  const [traceLoading, setTraceLoading] = useState(false);
  const [currentTraceId, setCurrentTraceId] = useState('');

  // 会话对话状态
  const [sessionVisible, setSessionVisible] = useState(false);
  const [sessionRecords, setSessionRecords] = useState<ConversationRecord[]>([]);
  const [sessionLoading, setSessionLoading] = useState(false);
  const [currentSessionKey, setCurrentSessionKey] = useState('');

  // 增强筛选状态
  const [filterVisible, setFilterVisible] = useState(false);
  const [filters, setFilters] = useState({
    dateRange: null as [dayjs.Dayjs, dayjs.Dayjs] | null,
    agentCodes: [] as string[],
    channelCodes: [] as string[],
    roles: [] as string[],
  });

  // Agent 和 Channel 选项
  const [agentOptions, setAgentOptions] = useState<{ code: string; name: string }[]>([]);
  const [channelOptions, setChannelOptions] = useState<{ code: string; name: string }[]>([]);

  const roleOptions = [
    { value: 'user', label: '用户' },
    { value: 'assistant', label: '助手' },
    { value: 'system', label: '系统' },
    { value: 'tool', label: '工具' },
    { value: 'tool_result', label: '工具结果' },
  ];

  const fetchRecords = async () => {
    setLoading(true);
    try {
      const res = await conversationsApi.list();
      setRecords((res as any)?.items || []);
      // 提取唯一的 Agent 和 Channel
      extractOptions((res as any)?.items || []);
    } catch (error) {
      message.error('获取对话记录失败');
    } finally {
      setLoading(false);
    }
  };

  const extractOptions = (items: ConversationRecord[]) => {
    const agentMap = new Map<string, string>();
    const channelMap = new Map<string, string>();

    items.forEach(item => {
      if (item.agent_code) {
        agentMap.set(item.agent_code, item.agent_name || item.agent_code);
      }
      if (item.channel_code) {
        channelMap.set(item.channel_code, item.channel_name || item.channel_code);
      }
    });

    setAgentOptions(Array.from(agentMap.entries()).map(([code, name]) => ({ code, name })));
    setChannelOptions(Array.from(channelMap.entries()).map(([code, name]) => ({ code, name })));
  };

  const searchBySession = async () => {
    if (!sessionKey) {
      fetchRecords();
      return;
    }
    setLoading(true);
    try {
      const res = await conversationsApi.getBySession(sessionKey);
      setRecords((res as any)?.items || []);
    } catch (error) {
      message.error('搜索失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchTraceRecords = async (traceId: string) => {
    setTraceLoading(true);
    try {
      const res = await conversationsApi.getByTrace(traceId);
      const items = (res as any) || [];
      setTraceRecords(items);
    } catch (error) {
      message.error('获取链路数据失败');
    } finally {
      setTraceLoading(false);
    }
  };

  const fetchSessionRecords = async (session: string) => {
    setSessionLoading(true);
    try {
      const res = await conversationsApi.getBySession(session);
      const items = (res as any)?.items || [];
      setSessionRecords(items);
    } catch (error) {
      message.error('获取会话数据失败');
    } finally {
      setSessionLoading(false);
    }
  };

  useEffect(() => {
    fetchRecords();
  }, []);

  const getRoleColor = (role?: string) => {
    const colors: Record<string, string> = {
      user: 'blue',
      assistant: 'green',
      system: 'orange',
      tool: 'purple',
      tool_result: 'cyan',
    };
    return colors[role || ''] || 'default';
  };

  const getRoleLabel = (role?: string) => {
    const labels: Record<string, string> = {
      user: '用户',
      assistant: '助手',
      system: '系统',
      tool: '工具',
      tool_result: '工具结果',
    };
    return labels[role || ''] || role;
  };

  // 构建链路树
  const buildTraceTree = (records: ConversationRecord[]): TraceNode[] => {
    const nodeMap = new Map<number, TraceNode>();
    const spanToIdMap = new Map<string, number[]>();
    const roots: TraceNode[] = [];

    // 按时间排序
    const sorted = [...records].sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );

    // 创建所有节点，使用 id 作为 key
    sorted.forEach((record, index) => {
      const nextRecord = sorted[index + 1];
      const duration = nextRecord
        ? new Date(nextRecord.timestamp).getTime() - new Date(record.timestamp).getTime()
        : 0;

      nodeMap.set(record.id, {
        key: String(record.id),
        title: (
          <Space orientation="vertical" size={0} style={{ width: '100%' }}>
            <Space>
              <Tag color={getRoleColor(record.role)}>{getRoleLabel(record.role)}</Tag>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {record.event_type}
              </Text>
              {record.total_tokens > 0 && (
                <Badge count={`${record.total_tokens} tokens`} style={{ backgroundColor: '#1890ff' }} />
              )}
              {duration > 0 && duration < 300000 && (
                <Text type="success" style={{ fontSize: 12 }}>
                  +{duration}ms
                </Text>
              )}
            </Space>
            <Text ellipsis style={{ maxWidth: 400, fontSize: 12 }}>
              {record.content?.substring(0, 100)}
              {record.content?.length > 100 ? '...' : ''}
            </Text>
          </Space>
        ),
        record,
        duration,
        children: [],
      });

      // 记录 span_id 到 id 的映射（可能有多个记录有相同的 span_id）
      if (record.span_id) {
        const ids = spanToIdMap.get(record.span_id) || [];
        ids.push(record.id);
        spanToIdMap.set(record.span_id, ids);
      }
    });

    // 建立父子关系
    sorted.forEach(record => {
      const node = nodeMap.get(record.id);
      if (!node) return;

      if (record.parent_span_id) {
        // 找到 parent_span_id 对应的节点（取第一个）
        const parentIds = spanToIdMap.get(record.parent_span_id);
        if (parentIds && parentIds.length > 0) {
          const parent = nodeMap.get(parentIds[0]);
          if (parent) {
            parent.children = parent.children || [];
            parent.children.push(node);
            return;
          }
        }
      }
      // 没有 parent 或者是根节点
      roots.push(node);
    });

    return roots;
  };

  // 计算链路统计
  const getTraceStats = (records: ConversationRecord[]) => {
    const totalTokens = records.reduce((sum, r) => sum + (r.total_tokens || 0), 0);
    const startTime = records.length > 0 ? new Date(records[0].timestamp) : null;
    const endTime = records.length > 0 ? new Date(records[records.length - 1].timestamp) : null;
    const duration = startTime && endTime ? endTime.getTime() - startTime.getTime() : 0;

    return { totalTokens, duration, count: records.length };
  };

  // 构建会话聊天消息
  const buildChatMessages = (records: ConversationRecord[]): ChatMessage[] => {
    return records
      .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())
      .map(r => ({
        id: r.id,
        role: r.role || '',
        content: r.content || '',
        tokens: r.total_tokens || 0,
        timestamp: r.timestamp,
        agentName: r.agent_name,
      }));
  };

  // 计算会话统计
  const getSessionStats = (records: ConversationRecord[]) => {
    const totalTokens = records.reduce((sum, r) => sum + (r.total_tokens || 0), 0);
    const messageCount = records.length;
    const startTime = records.length > 0 ? new Date(records[0].timestamp) : null;
    const endTime = records.length > 0 ? new Date(records[records.length - 1].timestamp) : null;
    const duration = startTime && endTime ? endTime.getTime() - startTime.getTime() : 0;

    return { totalTokens, messageCount, duration };
  };

  // 重置筛选
  const resetFilters = () => {
    setFilters({
      dateRange: null,
      agentCodes: [],
      channelCodes: [],
      roles: [],
    });
    fetchRecords();
  };

  // 应用筛选
  const applyFilters = () => {
    let filtered = [...records];

    if (filters.dateRange) {
      const [start, end] = filters.dateRange;
      filtered = filtered.filter(r => {
        const time = new Date(r.timestamp).getTime();
        return time >= start.valueOf() && time <= end.valueOf();
      });
    }

    if (filters.agentCodes.length > 0) {
      filtered = filtered.filter(r => filters.agentCodes.includes(r.agent_code || ''));
    }

    if (filters.channelCodes.length > 0) {
      filtered = filtered.filter(r => filters.channelCodes.includes(r.channel_code || ''));
    }

    if (filters.roles.length > 0) {
      filtered = filtered.filter(r => filters.roles.includes(r.role || ''));
    }

    setRecords(filtered);
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: 'Agent',
      dataIndex: 'agent_name',
      ellipsis: true,
      render: (name: string, record: ConversationRecord) => (
        <span title={record.agent_code}>{name || record.agent_code || '-'}</span>
      ),
    },
    {
      title: 'Channel',
      dataIndex: 'channel_name',
      ellipsis: true,
      render: (name: string, record: ConversationRecord) => (
        <span title={record.channel_code}>{name || record.channel_code || '-'}</span>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      width: 100,
      render: (role: string) => <Tag color={getRoleColor(role)}>{getRoleLabel(role)}</Tag>,
    },
    {
      title: '内容',
      dataIndex: 'content',
      ellipsis: true,
      render: (content: string) => content?.substring(0, 50) + (content?.length > 50 ? '...' : ''),
    },
    {
      title: 'Tokens',
      width: 80,
      render: (_: any, record: ConversationRecord) => (
        <span>{record.total_tokens || 0}</span>
      ),
    },
    {
      title: '时间',
      dataIndex: 'timestamp',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      width: 200,
      fixed: 'right' as const,
      render: (_: any, record: ConversationRecord) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => {
                setSelectedRecord(record);
                setDetailVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="查看链路">
            <Button
              type="text"
              icon={<BranchesOutlined />}
              onClick={() => {
                setCurrentTraceId(record.trace_id);
                fetchTraceRecords(record.trace_id);
                setTraceVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="查看会话">
            <Button
              type="text"
              icon={<MessageOutlined />}
              onClick={() => {
                setCurrentSessionKey(record.session_key);
                fetchSessionRecords(record.session_key);
                setSessionVisible(true);
              }}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  const traceStats = getTraceStats(traceRecords);
  const sessionStats = getSessionStats(sessionRecords);
  const chatMessages = buildChatMessages(sessionRecords);
  const traceTreeData = buildTraceTree(traceRecords);

  return (
    <div>
      <Card
        title={
          <Space>
            <span>对话记录</span>
            <Button
              type={filterVisible ? 'primary' : 'default'}
              icon={<FilterOutlined />}
              size="small"
              onClick={() => setFilterVisible(!filterVisible)}
            >
              筛选
            </Button>
          </Space>
        }
        extra={
          <Space>
            <Input.Search
              placeholder="输入 Session Key 搜索"
              value={sessionKey}
              onChange={(e) => setSessionKey(e.target.value)}
              onSearch={searchBySession}
              style={{ width: 300 }}
              prefix={<SearchOutlined />}
              allowClear
            />
            <Button icon={<BarChartOutlined />} onClick={() => navigate('/conversations/stats')}>
              统计分析
            </Button>
          </Space>
        }
      >
        {/* 筛选面板 */}
        {filterVisible && (
          <Card size="small" style={{ marginBottom: 16, background: '#f5f5f5' }}>
            <Form layout="inline">
              <Form.Item label="时间范围">
                <RangePicker
                  showTime
                  value={filters.dateRange}
                  onChange={(dates) => setFilters({ ...filters, dateRange: dates as any })}
                />
              </Form.Item>
              <Form.Item label="Agent">
                <Select
                  mode="multiple"
                  placeholder="选择Agent"
                  style={{ width: 200 }}
                  value={filters.agentCodes}
                  onChange={(values) => setFilters({ ...filters, agentCodes: values })}
                  allowClear
                >
                  {agentOptions.map(agent => (
                    <Option key={agent.code} value={agent.code}>{agent.name}</Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item label="Channel">
                <Select
                  mode="multiple"
                  placeholder="选择Channel"
                  style={{ width: 200 }}
                  value={filters.channelCodes}
                  onChange={(values) => setFilters({ ...filters, channelCodes: values })}
                  allowClear
                >
                  {channelOptions.map(ch => (
                    <Option key={ch.code} value={ch.code}>{ch.name}</Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item label="角色">
                <Select
                  mode="multiple"
                  placeholder="选择角色"
                  style={{ width: 200 }}
                  value={filters.roles}
                  onChange={(values) => setFilters({ ...filters, roles: values })}
                  allowClear
                >
                  {roleOptions.map(role => (
                    <Option key={role.value} value={role.value}>{role.label}</Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item>
                <Button type="primary" onClick={applyFilters}>
                  应用
                </Button>
                <Button icon={<ClearOutlined />} onClick={resetFilters} style={{ marginLeft: 8 }}>
                  重置
                </Button>
              </Form.Item>
            </Form>
          </Card>
        )}

        <Table
          rowKey="id"
          columns={columns}
          dataSource={records}
          loading={loading}
          pagination={{ pageSize: 20 }}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 详情弹窗 */}
      <Modal
        title="对话详情"
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedRecord(null);
        }}
        footer={null}
        width={800}
      >
        {selectedRecord && (
          <Descriptions column={1} bordered>
            <Descriptions.Item label="ID">{selectedRecord.id}</Descriptions.Item>
            <Descriptions.Item label="User Code">{selectedRecord.user_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="Agent">{selectedRecord.agent_name || selectedRecord.agent_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="Channel">{selectedRecord.channel_name || selectedRecord.channel_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="Event Type">{selectedRecord.event_type}</Descriptions.Item>
            <Descriptions.Item label="角色">
              <Tag color={getRoleColor(selectedRecord.role)}>{getRoleLabel(selectedRecord.role)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="内容">
              <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{selectedRecord.content}</pre>
            </Descriptions.Item>
            <Descriptions.Item label="Tokens">
              Prompt: {selectedRecord.prompt_tokens} / Completion: {selectedRecord.completion_tokens} / Total: {selectedRecord.total_tokens}
            </Descriptions.Item>
            <Descriptions.Item label="时间">{selectedRecord.timestamp}</Descriptions.Item>
            <Descriptions.Item label="Span ID">{selectedRecord.span_id}</Descriptions.Item>
            <Descriptions.Item label="Trace ID">{selectedRecord.trace_id}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>

      {/* 链路可视化弹窗 */}
      <Modal
        title={`对话链路 - ${currentTraceId}`}
        open={traceVisible}
        onCancel={() => {
          setTraceVisible(false);
          setTraceRecords([]);
        }}
        footer={null}
        width={900}
      >
        {traceLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : traceRecords.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>无数据</div>
        ) : (
          <div>
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Statistic title="总消息数" value={traceStats.count} />
              </Col>
              <Col span={8}>
                <Statistic title="总Token" value={traceStats.totalTokens} />
              </Col>
              <Col span={8}>
                <Statistic title="总耗时" value={`${traceStats.duration}ms`} />
              </Col>
            </Row>
            <Divider />
            <Tree
              treeData={traceTreeData}
              showLine
              defaultExpandAll
              style={{ background: '#fafafa', padding: 16, borderRadius: 8 }}
            />
          </div>
        )}
      </Modal>

      {/* 会话对话弹窗 */}
      <Modal
        title={`会话详情 - ${currentSessionKey}`}
        open={sessionVisible}
        onCancel={() => {
          setSessionVisible(false);
          setSessionRecords([]);
        }}
        footer={null}
        width={800}
      >
        {sessionLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : sessionRecords.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>无数据</div>
        ) : (
          <div>
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Statistic title="消息数" value={sessionStats.messageCount} />
              </Col>
              <Col span={8}>
                <Statistic title="总Token" value={sessionStats.totalTokens} />
              </Col>
              <Col span={8}>
                <Statistic title="时长" value={`${Math.round(sessionStats.duration / 1000)}s`} />
              </Col>
            </Row>
            <Divider />
            <div style={{ maxHeight: 500, overflowY: 'auto', padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
              {chatMessages.map((msg) => (
                <div
                  key={msg.id}
                  style={{
                    display: 'flex',
                    flexDirection: msg.role === 'user' ? 'row-reverse' : 'row',
                    marginBottom: 16,
                    alignItems: 'flex-start',
                  }}
                >
                  <div
                    style={{
                      maxWidth: '70%',
                      padding: '12px 16px',
                      borderRadius: msg.role === 'user' ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
                      background: msg.role === 'user' ? '#1890ff' : '#fff',
                      color: msg.role === 'user' ? '#fff' : '#333',
                      boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                    }}
                  >
                    <div style={{ fontSize: 12, opacity: 0.7, marginBottom: 4 }}>
                      {getRoleLabel(msg.role)} · {msg.tokens} tokens
                    </div>
                    <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                      {msg.content}
                    </div>
                    <div style={{ fontSize: 11, opacity: 0.5, marginTop: 4, textAlign: 'right' }}>
                      {dayjs(msg.timestamp).format('HH:mm:ss')}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Conversations;
