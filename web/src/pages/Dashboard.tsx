import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Grid } from 'antd';
import {
  RobotOutlined,
  MessageOutlined,
  KeyOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import { agentsApi, channelsApi, providersApi, cronApi, getCurrentUserCode } from '../api';

const { useBreakpoint } = Grid;

const Dashboard: React.FC = () => {
  const screens = useBreakpoint();
  const [stats, setStats] = useState({
    agents: 0,
    channels: 0,
    providers: 0,
    cronJobs: 0,
  });

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

  const gutter: [number, number] = screens.xs ? [8, 8] : [16, 16];

  return (
    <div>
      <h1 style={{ marginBottom: 24 }}>仪表盘</h1>
      <Row gutter={gutter}>
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
    </div>
  );
};

export default Dashboard;
