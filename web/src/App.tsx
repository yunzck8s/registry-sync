import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import AppLayout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Registries from './pages/Registries';
import Tasks from './pages/Tasks';
import Executions from './pages/Executions';
import Notifications from './pages/Notifications';

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <AppLayout>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/registries" element={<Registries />} />
            <Route path="/tasks" element={<Tasks />} />
            <Route path="/executions" element={<Executions />} />
            <Route path="/notifications" element={<Notifications />} />
          </Routes>
        </AppLayout>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
