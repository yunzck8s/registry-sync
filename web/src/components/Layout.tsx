import React from 'react';
import { Layout as AntLayout, Menu, theme } from 'antd';
import {
  DashboardOutlined,
  DatabaseOutlined,
  SyncOutlined,
  HistoryOutlined,
  BellOutlined,
} from '@ant-design/icons';
import { Link, useLocation } from 'react-router-dom';

const { Header, Content, Sider } = AntLayout;

interface LayoutProps {
  children: React.ReactNode;
}

const AppLayout: React.FC<LayoutProps> = ({ children }) => {
  const location = useLocation();
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  const menuItems = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: <Link to="/">仪表盘</Link>,
    },
    {
      key: '/registries',
      icon: <DatabaseOutlined />,
      label: <Link to="/registries">Registry 管理</Link>,
    },
    {
      key: '/tasks',
      icon: <SyncOutlined />,
      label: <Link to="/tasks">同步任务</Link>,
    },
    {
      key: '/executions',
      icon: <HistoryOutlined />,
      label: <Link to="/executions">执行历史</Link>,
    },
    {
      key: '/notifications',
      icon: <BellOutlined />,
      label: <Link to="/notifications">通知配置</Link>,
    },
  ];

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          background: colorBgContainer,
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <div style={{ fontSize: '20px', fontWeight: 'bold', marginRight: '50px' }}>
          Registry Sync
        </div>
      </Header>
      <AntLayout>
        <Sider
          width={200}
          style={{ background: colorBgContainer }}
        >
          <Menu
            mode="inline"
            selectedKeys={[location.pathname]}
            style={{ height: '100%', borderRight: 0 }}
            items={menuItems}
          />
        </Sider>
        <AntLayout style={{ padding: '24px' }}>
          <Content
            style={{
              padding: 24,
              margin: 0,
              minHeight: 280,
              background: colorBgContainer,
              borderRadius: '8px',
            }}
          >
            {children}
          </Content>
        </AntLayout>
      </AntLayout>
    </AntLayout>
  );
};

export default AppLayout;
