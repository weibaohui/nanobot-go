import React, { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { ConfigProvider, Spin } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import MainLayout from './layouts/MainLayout';
import Dashboard from './pages/Dashboard';
import Agents from './pages/Agents';
import Channels from './pages/Channels';
import Providers from './pages/Providers';
import CronJobs from './pages/CronJobs';
import Users from './pages/Users';
import Conversations from './pages/Conversations';
import ConversationStats from './pages/ConversationStats';
import Sessions from './pages/Sessions';
import StreamMemories from './pages/StreamMemories';
import LongTermMemories from './pages/LongTermMemories';
import MCPServers from './pages/MCPServers';
import Login from './pages/Login';
import { isAuthenticated, authApi, setCurrentUser, getCurrentUserCode } from './api';

// 路由守卫组件
const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const location = useLocation();
  const authenticated = isAuthenticated();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // 如果已登录但没有用户信息，获取当前用户信息
    if (authenticated && !getCurrentUserCode()) {
      authApi.me().then((res: any) => {
        if (res.data) {
          setCurrentUser(res.data);
        }
      }).catch(() => {
        // 获取失败不处理，后续请求会因为没有 user_code 而失败
      }).finally(() => {
        setLoading(false);
      });
    } else {
      setLoading(false);
    }
  }, [authenticated]);

  if (!authenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (loading) {
    return (
      <div style={{ height: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  return <>{children}</>;
};

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          {/* 公开路由 */}
          <Route path="/login" element={<Login />} />

          {/* 需要认证的路由 */}
          <Route path="/" element={
            <PrivateRoute>
              <MainLayout />
            </PrivateRoute>
          }>
            <Route index element={<Dashboard />} />
            <Route path="agents" element={<Agents />} />
            <Route path="channels" element={<Channels />} />
            <Route path="providers" element={<Providers />} />
            <Route path="cron" element={<CronJobs />} />
            <Route path="users" element={<Users />} />
            <Route path="conversations" element={<Conversations />} />
            <Route path="conversations/stats" element={<ConversationStats />} />
            <Route path="sessions" element={<Sessions />} />
            <Route path="stream-memories" element={<StreamMemories />} />
            <Route path="long-term-memories" element={<LongTermMemories />} />
            <Route path="mcp-servers" element={<MCPServers />} />
          </Route>

          {/* 未匹配路由重定向 */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
