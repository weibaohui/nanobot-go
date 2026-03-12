import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import MainLayout from './layouts/MainLayout';
import Dashboard from './pages/Dashboard';
import Agents from './pages/Agents';
import Channels from './pages/Channels';
import Providers from './pages/Providers';
import CronJobs from './pages/CronJobs';
import Users from './pages/Users';
import Conversations from './pages/Conversations';
import StreamMemories from './pages/StreamMemories';
import LongTermMemories from './pages/LongTermMemories';

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<MainLayout />}>
            <Route index element={<Dashboard />} />
            <Route path="agents" element={<Agents />} />
            <Route path="channels" element={<Channels />} />
            <Route path="providers" element={<Providers />} />
            <Route path="cron" element={<CronJobs />} />
            <Route path="users" element={<Users />} />
            <Route path="conversations" element={<Conversations />} />
            <Route path="stream-memories" element={<StreamMemories />} />
            <Route path="long-term-memories" element={<LongTermMemories />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
