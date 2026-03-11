import React from 'react';
import { Layout, Menu, Button, theme } from 'antd';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  DashboardOutlined,
  RobotOutlined,
  MessageOutlined,
  KeyOutlined,
  ClockCircleOutlined,
  UserOutlined,
} from '@ant-design/icons';

const { Header, Sider, Content } = Layout;

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  const menuItems = [
    { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
    { key: '/agents', icon: <RobotOutlined />, label: 'Agent 管理' },
    { key: '/channels', icon: <MessageOutlined />, label: '渠道管理' },
    { key: '/providers', icon: <KeyOutlined />, label: 'LLM 提供商' },
    { key: '/cron', icon: <ClockCircleOutlined />, label: '定时任务' },
    { key: '/users', icon: <UserOutlined />, label: '用户管理' },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider theme="light" width={200}>
        <div style={{ height: 64, padding: '16px', fontSize: 18, fontWeight: 'bold', textAlign: 'center' }}>
          Nanobot
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 24px', background: colorBgContainer, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ margin: 0 }}>管理系统</h2>
          <Button
            onClick={() => {
              localStorage.removeItem('token');
              navigate('/login');
            }}
          >
            退出登录
          </Button>
        </Header>
        <Content style={{ margin: '24px 16px', padding: 24, background: colorBgContainer, borderRadius: borderRadiusLG }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
