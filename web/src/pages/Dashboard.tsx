import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic } from 'antd';
import {
  RobotOutlined,
  MessageOutlined,
  KeyOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import { agentsApi, channelsApi, providersApi, cronApi } from '../api';

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState({
    agents: 0,
    channels: 0,
    providers: 0,
    cronJobs: 0,
  });

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const [agentsRes, channelsRes, providersRes, cronRes] = await Promise.all([
          agentsApi.list(),
          channelsApi.list(),
          providersApi.list(),
          cronApi.list(),
        ]);

        setStats({
          agents: agentsRes.data?.total || 0,
          channels: channelsRes.data?.total || 0,
          providers: providersRes.data?.total || 0,
          cronJobs: cronRes.data?.total || 0,
        });
      } catch (error) {
        console.error('获取统计数据失败:', error);
      }
    };

    fetchStats();
  }, []);

  return (
    <div>
      <h1>仪表盘</h1>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="Agent 数量"
              value={stats.agents}
              prefix={<RobotOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="渠道数量"
              value={stats.channels}
              prefix={<MessageOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="LLM 提供商"
              value={stats.providers}
              prefix={<KeyOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="定时任务"
              value={stats.cronJobs}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
