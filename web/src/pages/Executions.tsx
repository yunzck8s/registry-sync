import React, { useState, useMemo } from 'react';
import { Table, Tag, Button, Modal, Timeline, Typography, Card, Input, Select, Space, DatePicker } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, SyncOutlined, EyeOutlined, SearchOutlined } from '@ant-design/icons';
import { useApi } from '../hooks/useApi';
import { executionApi, taskApi } from '../api/client';
import type { Execution, ExecutionLog } from '../types';
import dayjs from 'dayjs';

const { Text } = Typography;
const { RangePicker } = DatePicker;

const Executions: React.FC = () => {
  const { data: executions, loading } = useApi(() => executionApi.list({ limit: 100 }), []);
  const { data: tasks } = useApi(() => taskApi.list(), []);
  const [selectedExecution, setSelectedExecution] = useState<number | null>(null);
  const [logsVisible, setLogsVisible] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | null>(null);
  const [taskFilter, setTaskFilter] = useState<number | null>(null);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null);

  const { data: logs, loading: logsLoading } = useApi(
    () => (selectedExecution ? executionApi.logs(selectedExecution) : Promise.resolve({ data: [] })),
    [selectedExecution]
  );

  // 筛选逻辑
  const filteredExecutions = useMemo(() => {
    if (!executions) return [];

    let filtered = executions;

    // 任务筛选
    if (taskFilter) {
      filtered = filtered.filter((exec) => exec.task_id === taskFilter);
    }

    // 状态筛选
    if (statusFilter) {
      filtered = filtered.filter((exec) => exec.status === statusFilter);
    }

    // 时间范围筛选
    if (dateRange && dateRange[0] && dateRange[1]) {
      filtered = filtered.filter((exec) => {
        const execTime = dayjs(exec.start_time);
        return execTime.isAfter(dateRange[0]) && execTime.isBefore(dateRange[1]);
      });
    }

    // 搜索筛选
    if (searchText) {
      const lowerSearch = searchText.toLowerCase();
      filtered = filtered.filter((exec) =>
        exec.task?.name?.toLowerCase().includes(lowerSearch) ||
        exec.error_message?.toLowerCase().includes(lowerSearch)
      );
    }

    return filtered;
  }, [executions, taskFilter, statusFilter, dateRange, searchText]);

  const getStatusTag = (status: string) => {
    const statusConfig: Record<string, { color: string; icon: React.ReactNode }> = {
      success: { color: 'success', icon: <CheckCircleOutlined /> },
      failed: { color: 'error', icon: <CloseCircleOutlined /> },
      running: { color: 'processing', icon: <SyncOutlined spin /> },
      pending: { color: 'default', icon: null },
      canceled: { color: 'warning', icon: null },
    };

    const config = statusConfig[status] || statusConfig.pending;
    return (
      <Tag color={config.color} icon={config.icon}>
        {status.toUpperCase()}
      </Tag>
    );
  };

  const handleViewLogs = (executionId: number) => {
    setSelectedExecution(executionId);
    setLogsVisible(true);
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '任务',
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
      render: (_: any, record: Execution) => {
        if (record.total_blobs === 0) return '-';
        const percent = ((record.synced_blobs / record.total_blobs) * 100).toFixed(1);
        return (
          <div>
            <div>{`${record.synced_blobs}/${record.total_blobs} Blobs`}</div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {`${percent}%`} | 失败: {record.failed_blobs} | 跳过: {record.skipped_blobs}
            </Text>
          </div>
        );
      },
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
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
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: Execution) => (
        <Button
          type="link"
          icon={<EyeOutlined />}
          onClick={() => handleViewLogs(record.id)}
        >
          查看日志
        </Button>
      ),
    },
  ];

  const getLogColor = (level: string) => {
    const colors: Record<string, string> = {
      error: 'red',
      warn: 'orange',
      info: 'blue',
      debug: 'gray',
    };
    return colors[level] || 'blue';
  };

  return (
    <div>
      <Card>
        <div style={{ marginBottom: 16 }}>
          <h2 style={{ marginBottom: 16 }}>执行历史</h2>
          <Space wrap>
            <Input
              placeholder="搜索（任务名/错误信息）"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 220 }}
              allowClear
            />
            <Select
              placeholder="任务筛选"
              value={taskFilter}
              onChange={setTaskFilter}
              style={{ width: 180 }}
              allowClear
            >
              {tasks?.map((task) => (
                <Select.Option key={task.id} value={task.id}>
                  {task.name}
                </Select.Option>
              ))}
            </Select>
            <Select
              placeholder="状态筛选"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: 130 }}
              allowClear
            >
              <Select.Option value="success">成功</Select.Option>
              <Select.Option value="failed">失败</Select.Option>
              <Select.Option value="running">运行中</Select.Option>
              <Select.Option value="pending">待运行</Select.Option>
            </Select>
            <RangePicker
              showTime
              format="YYYY-MM-DD HH:mm"
              placeholder={['开始时间', '结束时间']}
              onChange={(dates) => setDateRange(dates as any)}
            />
          </Space>
        </div>

        <Table
          dataSource={filteredExecutions}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            pageSize: 20,
            showTotal: (total) => `共 ${total} 条`,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
        />
      </Card>

      <Modal
        title="执行日志"
        open={logsVisible}
        onCancel={() => setLogsVisible(false)}
        footer={null}
        width={900}
      >
        <Timeline
          items={(logs || []).map((log: ExecutionLog) => ({
            color: getLogColor(log.level),
            children: (
              <div>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {dayjs(log.timestamp).format('HH:mm:ss')}
                </Text>
                <div style={{ marginTop: 4 }}>
                  <Tag color={getLogColor(log.level)}>{log.level.toUpperCase()}</Tag>
                  {log.message}
                </div>
              </div>
            ),
          }))}
        />
        {logsLoading && <div style={{ textAlign: 'center', padding: 20 }}>加载中...</div>}
        {!logsLoading && (!logs || logs.length === 0) && (
          <div style={{ textAlign: 'center', padding: 20, color: '#999' }}>暂无日志</div>
        )}
      </Modal>
    </div>
  );
};

export default Executions;
