import React, { useState, useEffect } from 'react';
import { Layout, Menu, Button, theme, Grid, Typography, Space, Avatar, Dropdown } from 'antd';
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
  CommentOutlined,
  TeamOutlined,
  ApiOutlined,
  LogoutOutlined,
  ToolOutlined,
  UnorderedListOutlined,
  WechatOutlined,
} from '@ant-design/icons';
import { authApi, clearToken } from '../api';
import type { User } from '../types';

const { Header, Sider, Content } = Layout;
const { useBreakpoint } = Grid;
const { Text } = Typography;

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const screens = useBreakpoint();
  const [collapsed, setCollapsed] = useState(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);

  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  // 获取当前用户信息
  useEffect(() => {
    const fetchCurrentUser = async () => {
      try {
        const res = await authApi.me() as any;
        if (res && res.id) {
          setCurrentUser(res);
        }
      } catch (error) {
        console.error('获取用户信息失败:', error);
      }
    };
    fetchCurrentUser();
  }, []);

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
    { key: '/chat', icon: <WechatOutlined />, label: 'AI 对话' },
    { key: '/agents', icon: <RobotOutlined />, label: 'Agent' },
    { key: '/channels', icon: <MessageOutlined />, label: '渠道' },
    { key: '/providers', icon: <KeyOutlined />, label: 'LLM' },
    { key: '/mcp-servers', icon: <ApiOutlined />, label: 'MCP Server' },
    { key: '/skills', icon: <ToolOutlined />, label: '技能' },
    { key: '/cron', icon: <ClockCircleOutlined />, label: '定时任务' },
    { key: '/tasks', icon: <UnorderedListOutlined />, label: '后台任务' },
    { key: '/conversations', icon: <CommentOutlined />, label: '对话记录' },
    { key: '/sessions', icon: <TeamOutlined />, label: '会话管理' },
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

          <Dropdown
            menu={{
              items: [
                {
                  key: 'logout',
                  icon: <LogoutOutlined />,
                  label: '退出登录',
                  onClick: () => {
                    clearToken();
                    navigate('/login');
                  },
                },
              ],
            }}
            placement="bottomRight"
          >
            <Space style={{ cursor: 'pointer' }}>
              <Avatar icon={<UserOutlined />} />
              {!screens.xs && (
                <Typography.Text>{currentUser?.display_name || currentUser?.username || '用户'}</Typography.Text>
              )}
            </Space>
          </Dropdown>
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
