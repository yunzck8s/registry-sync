import React from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Spin, Progress, Typography } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useApi } from '../hooks/useApi';
import { statsApi, executionApi } from '../api/client';
import type { Execution } from '../types';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const { Text, Title } = Typography;

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const { data: stats, loading: statsLoading } = useApi(() => statsApi.get(), []);
  const { data: executions, loading: executionsLoading } = useApi(
    () => executionApi.list({ limit: 10 }),
    []
  );

  const getStatusTag = (status: string) => {
    const statusConfig: Record<string, { color: string; icon: React.ReactNode }> = {
      success: { color: 'success', icon: <CheckCircleOutlined /> },
      failed: { color: 'error', icon: <CloseCircleOutlined /> },
      running: { color: 'processing', icon: <SyncOutlined spin /> },
      pending: { color: 'default', icon: <ClockCircleOutlined /> },
      canceled: { color: 'warning', icon: <CloseCircleOutlined /> },
    };

    const config = statusConfig[status] || statusConfig.pending;
    return (
      <Tag color={config.color} icon={config.icon}>
        {status.toUpperCase()}
      </Tag>
    );
  };

  const columns = [
    {
      title: '任务名称',
      dataIndex: ['task', 'name'],
      key: 'task_name',
      render: (name: string) => name || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => getStatusTag(status),
    },
    {
      title: '进度',
      key: 'progress',
      width: 200,
      render: (_: any, record: Execution) => {
        if (record.total_blobs === 0) return '-';
        const percent = Math.floor((record.synced_blobs / record.total_blobs) * 100);
        return (
          <div>
            <Progress
              percent={percent}
              size="small"
              status={record.status === 'failed' ? 'exception' : record.status === 'success' ? 'success' : 'active'}
            />
            <Text type="secondary" style={{ fontSize: 11 }}>
              {record.synced_blobs}/{record.total_blobs} Blobs
            </Text>
          </div>
        );
      },
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      render: (time: string) => dayjs(time).fromNow(),
    },
    {
      title: '耗时',
      key: 'duration',
      render: (_: any, record: Execution) => {
        const start = dayjs(record.start_time);
        const end = record.end_time ? dayjs(record.end_time) : dayjs();
        const duration = end.diff(start, 'second');

        if (duration < 60) return `${duration}秒`;
        if (duration < 3600) return `${Math.floor(duration / 60)}分钟`;
        return `${Math.floor(duration / 3600)}小时`;
      },
    },
  ];

  // 计算成功率
  const totalExecutions = (stats?.success_executions || 0) + (stats?.failed_executions || 0);
  const successRate = totalExecutions > 0
    ? ((stats?.success_executions || 0) / totalExecutions * 100).toFixed(1)
    : '0';

  // 可点击的统计卡片组件
  const StatCard: React.FC<{
    title: string;
    value: number;
    icon: React.ReactNode;
    color: string;
    bgColor: string;
    onClick?: () => void;
    suffix?: string;
  }> = ({ title, value, icon, color, bgColor, onClick, suffix }) => (
    <Card
      hoverable={!!onClick}
      onClick={onClick}
      style={{
        borderRadius: 8,
        background: `linear-gradient(135deg, ${bgColor}15 0%, ${bgColor}05 100%)`,
        border: `1px solid ${bgColor}30`,
        cursor: onClick ? 'pointer' : 'default',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div style={{ flex: 1 }}>
          <Text type="secondary" style={{ fontSize: 14 }}>
            {title}
          </Text>
          <div style={{ marginTop: 8, display: 'flex', alignItems: 'baseline' }}>
            <Title level={2} style={{ margin: 0, color, fontSize: 32, fontWeight: 600 }}>
              {value}
            </Title>
            {suffix && (
              <Text type="secondary" style={{ marginLeft: 4, fontSize: 14 }}>
                {suffix}
              </Text>
            )}
          </div>
        </div>
        <div
          style={{
            fontSize: 32,
            color,
            background: `${color}15`,
            borderRadius: 8,
            padding: '12px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {icon}
        </div>
      </div>
      {onClick && (
        <div style={{ marginTop: 12, color: color, fontSize: 12, fontWeight: 500 }}>
          查看详情 <RightOutlined style={{ fontSize: 10 }} />
        </div>
      )}
    </Card>
  );

  if (statsLoading || executionsLoading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2} style={{ margin: 0 }}>仪表盘</Title>
        <Text type="secondary">实时监控您的镜像同步状态</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="总任务数"
            value={stats?.total_tasks || 0}
            icon={<DatabaseOutlined />}
            color="#1890ff"
            bgColor="#1890ff"
            onClick={() => navigate('/tasks')}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="运行中"
            value={stats?.running_executions || 0}
            icon={<SyncOutlined spin />}
            color="#faad14"
            bgColor="#faad14"
            onClick={() => navigate('/executions')}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="成功执行"
            value={stats?.success_executions || 0}
            icon={<CheckCircleOutlined />}
            color="#52c41a"
            bgColor="#52c41a"
            onClick={() => navigate('/executions')}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="失败执行"
            value={stats?.failed_executions || 0}
            icon={<CloseCircleOutlined />}
            color="#ff4d4f"
            bgColor="#ff4d4f"
            onClick={() => navigate('/executions')}
          />
        </Col>
      </Row>

      {/* 第二行统计 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={8}>
          <Card style={{ borderRadius: 8 }}>
            <Statistic
              title="成功率"
              value={Number(successRate)}
              precision={1}
              suffix="%"
              valueStyle={{ color: Number(successRate) >= 80 ? '#52c41a' : '#faad14' }}
              prefix={Number(successRate) >= 80 ? <ArrowUpOutlined /> : <ArrowDownOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card style={{ borderRadius: 8 }}>
            <Statistic
              title="总执行次数"
              value={totalExecutions}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card style={{ borderRadius: 8 }}>
            <Statistic
              title="活跃Registry"
              value={stats?.total_registries || 0}
              valueStyle={{ color: '#722ed1' }}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 最近执行表格 */}
      <Card
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>最近执行</span>
            <a onClick={() => navigate('/executions')} style={{ fontSize: 14, fontWeight: 'normal' }}>
              查看全部 <RightOutlined />
            </a>
          </div>
        }
        style={{ borderRadius: 8 }}
      >
        <Table
          dataSource={executions || []}
          columns={columns}
          rowKey="id"
          pagination={false}
          size="middle"
        />
      </Card>
    </div>
  );
};

export default Dashboard;
