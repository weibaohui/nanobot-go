import React, { useEffect, useState } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  DatePicker,
  Button,
  message,
  Space,
  Typography,
  Table,
} from 'antd';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
} from 'recharts';
import { ReloadOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import { conversationsApi } from '../api';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;
const { Title } = Typography;

// 颜色配置
const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];

interface StatsData {
  token_stats: {
    total_prompt_tokens: number;
    total_completion_tokens: number;
    total_tokens: number;
    daily_trends: {
      date: string;
      prompt_tokens: number;
      complete_tokens: number;
      total_tokens: number;
    }[];
  };
  agent_distribution: {
    code: string;
    name: string;
    count: number;
    tokens: number;
  }[];
  channel_distribution: {
    type: string;
    count: number;
  }[];
  role_distribution: {
    role: string;
    count: number;
  }[];
  session_stats: {
    total_sessions: number;
    avg_messages_per_session: number;
    avg_response_time_ms: number;
  };
}

const ConversationStats: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<StatsData | null>(null);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'),
    dayjs(),
  ]);

  const fetchStats = async () => {
    setLoading(true);
    try {
      const [start, end] = dateRange;
      const res = await conversationsApi.getStats({
        start_time: start.toISOString(),
        end_time: end.toISOString(),
      });
      setStats(res as unknown as StatsData);
    } catch (error) {
      message.error('获取统计数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  // Token 趋势图表数据
  const tokenTrendData = stats?.token_stats.daily_trends || [];

  // Agent 分布图表数据
  const agentDistData =
    stats?.agent_distribution.map((item) => ({
      name: item.name || item.code,
      count: item.count,
      tokens: item.tokens,
    })) || [];

  // Channel 分布图表数据
  const channelDistData =
    stats?.channel_distribution.map((item) => ({
      name: item.type || '未知',
      value: item.count,
    })) || [];

  // 角色分布图表数据
  const roleDistData =
    stats?.role_distribution.map((item) => ({
      name: getRoleLabel(item.role),
      value: item.count,
    })) || [];

  function getRoleLabel(role: string): string {
    const labels: Record<string, string> = {
      user: '用户',
      assistant: '助手',
      system: '系统',
      tool: '工具',
      tool_result: '工具结果',
    };
    return labels[role] || role;
  }

  const agentColumns = [
    { title: 'Agent', dataIndex: 'name', key: 'name' },
    { title: '消息数', dataIndex: 'count', key: 'count' },
    {
      title: 'Token 数',
      dataIndex: 'tokens',
      key: 'tokens',
      render: (tokens: number) => tokens.toLocaleString(),
    },
  ];

  return (
    <div>
      <Card
        title={
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => window.history.back()}>
              返回
            </Button>
            <Title level={4} style={{ margin: 0 }}>
              对话统计分析
            </Title>
          </Space>
        }
        extra={
          <Space>
            <RangePicker
              value={dateRange}
              onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
            />
            <Button type="primary" icon={<ReloadOutlined />} onClick={fetchStats} loading={loading}>
              刷新
            </Button>
          </Space>
        }
      >
        {/* 核心指标 */}
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col span={6}>
            <Card>
              <Statistic
                title="总会话数"
                value={stats?.session_stats.total_sessions || 0}
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="总 Token 数"
                value={stats?.token_stats.total_tokens || 0}
                valueStyle={{ color: '#52c41a' }}
                formatter={(value) => value.toLocaleString()}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="平均消息数/会话"
                value={stats?.session_stats.avg_messages_per_session || 0}
                precision={1}
                valueStyle={{ color: '#fa8c16' }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="平均响应时间"
                value={stats?.session_stats.avg_response_time_ms || 0}
                precision={0}
                suffix="ms"
                valueStyle={{ color: '#722ed1' }}
              />
            </Card>
          </Col>
        </Row>

        {/* Token 趋势图 */}
        <Card title="Token 消耗趋势" style={{ marginBottom: 24 }}>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={tokenTrendData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip formatter={(value) => typeof value === 'number' ? value.toLocaleString() : value} />
              <Legend />
              <Line
                type="monotone"
                dataKey="prompt_tokens"
                name="Prompt Tokens"
                stroke="#1890ff"
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="complete_tokens"
                name="Completion Tokens"
                stroke="#52c41a"
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="total_tokens"
                name="Total Tokens"
                stroke="#fa8c16"
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </Card>

        {/* Agent 分布 */}
        <Row gutter={24} style={{ marginBottom: 24 }}>
          <Col span={12}>
            <Card title="Agent 使用分布">
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={agentDistData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip />
                  <Bar dataKey="count" name="消息数" fill="#1890ff" />
                </BarChart>
              </ResponsiveContainer>
              <Table
                dataSource={agentDistData}
                columns={agentColumns}
                rowKey="name"
                pagination={false}
                size="small"
                style={{ marginTop: 16 }}
              />
            </Card>
          </Col>
          <Col span={12}>
            <Card title="Channel 来源分布">
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie
                    data={channelDistData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(props: any) => `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {channelDistData.map((_entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </Col>
        </Row>

        {/* 角色分布 */}
        <Row gutter={24}>
          <Col span={12}>
            <Card title="角色消息分布">
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie
                    data={roleDistData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(props: any) => `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {roleDistData.map((_entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </Col>
          <Col span={12}>
            <Card title="Token 使用统计">
              <Row gutter={16}>
                <Col span={8}>
                  <Statistic
                    title="Prompt Tokens"
                    value={stats?.token_stats.total_prompt_tokens || 0}
                    formatter={(value) => value.toLocaleString()}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title="Completion Tokens"
                    value={stats?.token_stats.total_completion_tokens || 0}
                    formatter={(value) => value.toLocaleString()}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title="总 Token 数"
                    value={stats?.token_stats.total_tokens || 0}
                    formatter={(value) => value.toLocaleString()}
                    valueStyle={{ color: '#1890ff', fontWeight: 'bold' }}
                  />
                </Col>
              </Row>
            </Card>
          </Col>
        </Row>
      </Card>
    </div>
  );
};

export default ConversationStats;
