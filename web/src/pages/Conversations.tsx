import React, { useEffect, useState } from 'react';
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
  const [_currentSessionKey, _setCurrentSessionKey] = useState('');

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

  // 整理为记忆状态

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

  const fetchSessionRecordsByTrace = async (_session: string, traceId: string) => {
    setSessionLoading(true);
    try {
      // 直接使用 trace API 获取该 trace 的所有消息
      const res = await conversationsApi.getByTrace(traceId);
      const items = (res as any) || [];
      setSessionRecords(items);
    } catch (error) {
      message.error('获取对话数据失败');
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

    const eventPriority: Record<string, number> = {
      llm_call_end: 10,
      tool_completed: 20,
    };
    const rolePriority: Record<string, number> = {
      tool: 10,
      tool_result: 20,
    };
    const compareByOrder = (a: ConversationRecord, b: ConversationRecord) => {
      const timeDiff = new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime();
      if (timeDiff !== 0) return timeDiff;
      const eventDiff = (eventPriority[a.event_type || ''] || 1000) - (eventPriority[b.event_type || ''] || 1000);
      if (eventDiff !== 0) return eventDiff;
      const roleDiff = (rolePriority[a.role || ''] || 1000) - (rolePriority[b.role || ''] || 1000);
      if (roleDiff !== 0) return roleDiff;
      return a.id - b.id;
    };

    // 按时间排序（与“查看对话”一致）
    const sorted = [...records].sort(
      compareByOrder
    );
    const indexById = new Map<number, number>();
    sorted.forEach((record, index) => {
      indexById.set(record.id, index);
    });

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
    });
    const roots = sorted
      .map(record => nodeMap.get(record.id))
      .filter((node): node is TraceNode => !!node);

    const detachNode = (targetId: number) => {
      const rootIndex = roots.findIndex(node => node.record.id === targetId);
      if (rootIndex >= 0) {
        roots.splice(rootIndex, 1);
      }
      nodeMap.forEach(node => {
        if (!node.children || node.children.length === 0) return;
        node.children = node.children.filter(child => child.record.id !== targetId);
      });
    };

    sorted.forEach((record, index) => {
      if (record.role !== 'tool_result') return;
      const resultNode = nodeMap.get(record.id);
      if (!resultNode) return;

      let targetToolRecord: ConversationRecord | undefined;
      for (let i = index - 1; i >= 0; i -= 1) {
        const candidate = sorted[i];
        if (candidate.role !== 'tool') continue;
        if (record.parent_span_id && candidate.span_id === record.parent_span_id) {
          targetToolRecord = candidate;
          break;
        }
        if (record.span_id && candidate.span_id === record.span_id) {
          targetToolRecord = candidate;
          break;
        }
      }

      if (!targetToolRecord) {
        for (let i = index - 1; i >= 0; i -= 1) {
          if (sorted[i].role === 'tool') {
            targetToolRecord = sorted[i];
            break;
          }
        }
      }

      if (!targetToolRecord) return;
      const toolNode = nodeMap.get(targetToolRecord.id);
      if (!toolNode) return;
      const toolIndex = indexById.get(toolNode.record.id);
      const resultIndex = indexById.get(resultNode.record.id);
      if (toolIndex === undefined || resultIndex === undefined || toolIndex >= resultIndex) return;

      detachNode(resultNode.record.id);
      toolNode.children = toolNode.children || [];
      if (!toolNode.children.some(child => child.record.id === resultNode.record.id)) {
        toolNode.children.push(resultNode);
      }
    });

    const sortTreeNodes = (nodes: TraceNode[]) => {
      nodes.sort((a, b) => compareByOrder(a.record, b.record));
      nodes.forEach(node => {
        if (node.children && node.children.length > 0) {
          sortTreeNodes(node.children);
        }
      });
    };

    sortTreeNodes(roots);
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
          <Tooltip title="查看对话">
            <Button
              type="text"
              icon={<MessageOutlined />}
              onClick={() => {
                _setCurrentSessionKey(record.session_key);
                setCurrentTraceId(record.trace_id);
                fetchSessionRecordsByTrace(record.session_key, record.trace_id);
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
        title={`对话详情 - ${currentTraceId?.slice(0, 8) || ''}...`}
        open={sessionVisible}
        onCancel={() => {
          setSessionVisible(false);
          setSessionRecords([]);
        }}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button onClick={() => setSessionVisible(false)}>关闭</Button>
          </div>
        }
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
