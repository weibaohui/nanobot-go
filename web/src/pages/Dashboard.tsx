import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Grid, DatePicker, Button, Space, Typography, Table } from 'antd';
import {
  RobotOutlined,
  MessageOutlined,
  KeyOutlined,
  ClockCircleOutlined,
  BarChartOutlined,
  TeamOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
import dayjs from 'dayjs';
import { agentsApi, channelsApi, providersApi, cronApi, getCurrentUserCode, conversationsApi } from '../api';

const { useBreakpoint } = Grid;
const { RangePicker } = DatePicker;
const { Title } = Typography;

// 颜色配置
const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];

interface ConversationStats {
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

const Dashboard: React.FC = () => {
  const screens = useBreakpoint();
  const [stats, setStats] = useState({
    agents: 0,
    channels: 0,
    providers: 0,
    cronJobs: 0,
  });

  // 对话统计状态
  const [convStats, setConvStats] = useState<ConversationStats | null>(null);
  const [convLoading, setConvLoading] = useState(false);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'),
    dayjs(),
  ]);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const userCode = getCurrentUserCode() || '';
        const [agentsRes, channelsRes, providersRes, cronRes] = await Promise.all([
          agentsApi.list(userCode),
          channelsApi.list(userCode),
          providersApi.list(userCode),
          cronApi.list(userCode),
        ]);

        setStats({
          agents: (agentsRes as any)?.total || 0,
          channels: (channelsRes as any)?.total || 0,
          providers: (providersRes as any)?.total || 0,
          cronJobs: (cronRes as any)?.total || 0,
        });
      } catch (error) {
        console.error('获取统计数据失败:', error);
      }
    };

    fetchStats();
  }, []);

  // 获取对话统计
  const fetchConversationStats = async () => {
    setConvLoading(true);
    try {
      const [start, end] = dateRange;
      const res = await conversationsApi.getStats({
        start_time: start.toISOString(),
        end_time: end.toISOString(),
      });
      setConvStats(res as unknown as ConversationStats);
    } catch (error) {
      console.error('获取对话统计失败:', error);
    } finally {
      setConvLoading(false);
    }
  };

  useEffect(() => {
    fetchConversationStats();
  }, []);

  const gutter: [number, number] = screens.xs ? [8, 8] : [16, 16];

  // Token 趋势图表数据
  const tokenTrendData = convStats?.token_stats.daily_trends || [];

  // Agent 分布图表数据
  const agentDistData =
    convStats?.agent_distribution.map((item) => ({
      name: item.name || item.code,
      value: item.count,
      count: item.count,
      tokens: item.tokens,
    })) || [];

  // Agent 分布表格列定义
  const agentColumns = [
    { title: 'Agent', dataIndex: 'name', key: 'name' },
    { title: '消息数', dataIndex: 'count', key: 'count' },
    {
      title: 'Token 数',
      dataIndex: 'tokens',
      key: 'tokens',
      render: (tokens: number) => tokens?.toLocaleString() || 0,
    },
  ];

  // Channel 分布图表数据
  const channelDistData =
    convStats?.channel_distribution.map((item) => ({
      name: item.type || '未知',
      value: item.count,
    })) || [];

  return (
    <div>
      <h1 style={{ marginBottom: 24 }}>仪表盘</h1>

      {/* 基础统计 */}
      <Row gutter={gutter} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} md={12} lg={8} xl={6}>
          <Card>
            <Statistic
              title="Agent 数量"
              value={stats.agents}
              prefix={<RobotOutlined />}
              valueStyle={{ fontSize: screens.xs ? 24 : 32 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={12} lg={8} xl={6}>
          <Card>
            <Statistic
              title="渠道数量"
              value={stats.channels}
              prefix={<MessageOutlined />}
              valueStyle={{ fontSize: screens.xs ? 24 : 32 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={12} lg={8} xl={6}>
          <Card>
            <Statistic
              title="LLM 提供商"
              value={stats.providers}
              prefix={<KeyOutlined />}
              valueStyle={{ fontSize: screens.xs ? 24 : 32 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={12} lg={8} xl={6}>
          <Card>
            <Statistic
              title="定时任务"
              value={stats.cronJobs}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ fontSize: screens.xs ? 24 : 32 }}
            />
          </Card>
        </Col>
      </Row>

      {/* 对话统计标题和筛选 */}
      <Card
        title={
          <Space>
            <BarChartOutlined />
            <Title level={4} style={{ margin: 0 }}>
              对话统计
            </Title>
          </Space>
        }
        extra={
          <Space>
            <RangePicker
              value={dateRange}
              onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
              style={{ width: 220 }}
            />
            <Button type="primary" onClick={fetchConversationStats} loading={convLoading}>
              刷新
            </Button>
          </Space>
        }
        style={{ marginBottom: 24 }}
      >
        {/* 对话核心指标 */}
        <Row gutter={gutter} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#f6ffed' }}>
              <Statistic
                title="总会话数"
                value={convStats?.session_stats.total_sessions || 0}
                prefix={<TeamOutlined />}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#e6f7ff' }}>
              <Statistic
                title="总 Token 数"
                value={convStats?.token_stats.total_tokens || 0}
                prefix={<ThunderboltOutlined />}
                valueStyle={{ color: '#1890ff' }}
                formatter={(value) => value.toLocaleString()}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#fff7e6' }}>
              <Statistic
                title="平均消息数/会话"
                value={convStats?.session_stats.avg_messages_per_session || 0}
                precision={1}
                valueStyle={{ color: '#fa8c16' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={6}>
            <Card bordered={false} style={{ background: '#f9f0ff' }}>
              <Statistic
                title="平均响应时间"
                value={convStats?.session_stats.avg_response_time_ms || 0}
                precision={0}
                suffix="ms"
                valueStyle={{ color: '#722ed1' }}
              />
            </Card>
          </Col>
        </Row>

        {/* Token 消耗趋势 */}
        <Card title="Token 消耗趋势" style={{ marginBottom: 16 }}>
          <ResponsiveContainer width="100%" height={250}>
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

        {/* 分布图表 */}
        <Row gutter={gutter}>
          <Col xs={24} md={12}>
            <Card title="Agent 使用分布">
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={agentDistData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(props: any) => `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`}
                    outerRadius={70}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {agentDistData.map((_entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
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
          <Col xs={24} md={12}>
            <Card title="Channel 来源分布">
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={channelDistData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(props: any) => `${props.name || ''}: ${((props.percent || 0) * 100).toFixed(0)}%`}
                    outerRadius={70}
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
      </Card>
    </div>
  );
};

export default Dashboard;
