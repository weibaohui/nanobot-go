import React, { useState, useEffect } from 'react';
import { Layout, Menu, Button, theme, Grid, Typography, Space } from 'antd';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  DashboardOutlined,
  RobotOutlined,
  MessageOutlined,
  KeyOutlined,
  ClockCircleOutlined,
  UserOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';

const { Header, Sider, Content } = Layout;
const { useBreakpoint } = Grid;
const { Text } = Typography;

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const screens = useBreakpoint();
  const [collapsed, setCollapsed] = useState(false);

  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  // 根据屏幕尺寸自动折叠侧边栏
  useEffect(() => {
    if (screens.xs) {
      setCollapsed(true);
    } else if (screens.sm || screens.md) {
      setCollapsed(false);
    }
  }, [screens.xs, screens.sm, screens.md]);

  const menuItems = [
    { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
    { key: '/agents', icon: <RobotOutlined />, label: 'Agent' },
    { key: '/channels', icon: <MessageOutlined />, label: '渠道' },
    { key: '/providers', icon: <KeyOutlined />, label: 'LLM' },
    { key: '/cron', icon: <ClockCircleOutlined />, label: '定时任务' },
    { key: '/users', icon: <UserOutlined />, label: '用户' },
  ];

  // 根据是否折叠显示不同的标签
  const getMenuLabel = (item: typeof menuItems[0]) => {
    if (collapsed) {
      // 折叠时只显示第一个字或保持原样
      return item.label;
    }
    return item.label;
  };

  const siderWidth = screens.xl ? 240 : screens.lg ? 220 : 200;

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        theme="light"
        width={siderWidth}
        collapsed={collapsed}
        collapsedWidth={screens.xs ? 0 : 80}
        breakpoint="lg"
        onCollapse={(value) => setCollapsed(value)}
        style={{
          position: screens.xs ? 'fixed' : 'relative',
          height: '100vh',
          zIndex: 100,
          left: 0,
          top: 0,
          boxShadow: '2px 0 8px rgba(0,0,0,0.05)',
        }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: collapsed ? '16px 8px' : '16px',
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          {collapsed ? (
            <Text strong style={{ fontSize: 20 }}>🤖</Text>
          ) : (
            <Space align="center">
              <Text strong style={{ fontSize: 20 }}>🤖</Text>
              <Text strong style={{ fontSize: 18 }}>Nanobot</Text>
            </Space>
          )}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems.map(item => ({
            ...item,
            label: getMenuLabel(item),
          }))}
          onClick={({ key }) => {
            navigate(key);
            if (screens.xs) {
              setCollapsed(true);
            }
          }}
          style={{ borderRight: 0 }}
        />
      </Sider>

      <Layout>
        <Header
          style={{
            padding: screens.xs ? '0 16px' : '0 24px',
            background: colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            boxShadow: '0 2px 8px rgba(0,0,0,0.05)',
            zIndex: 50,
          }}
        >
          <Space align="center">
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
              style={{ fontSize: 16, width: 40, height: 40 }}
            />
            <h2 style={{ margin: 0, fontSize: screens.xs ? 16 : 18 }}>管理系统</h2>
          </Space>

          <Button
            size={screens.xs ? 'small' : 'middle'}
            onClick={() => {
              localStorage.removeItem('token');
              navigate('/login');
            }}
          >
            {screens.xs ? '退出' : '退出登录'}
          </Button>
        </Header>

        <Content
          style={{
            margin: screens.xs ? '12px' : screens.sm ? '16px' : '24px 16px',
            padding: screens.xs ? 16 : screens.sm ? 20 : 24,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            minHeight: 280,
            overflow: 'auto',
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
