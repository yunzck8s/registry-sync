import React, { useState, useMemo } from 'react';
import { Table, Button, Modal, Form, Input, Switch, Space, Popconfirm, Card, Select, Tag } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined, SearchOutlined } from '@ant-design/icons';
import { useApi, useAsyncAction } from '../hooks/useApi';
import { notificationApi } from '../api/client';
import type { NotificationChannel } from '../types';

const Notifications: React.FC = () => {
  const { data: channels, loading, refetch } = useApi(() => notificationApi.list(), []);
  const { loading: actionLoading, execute } = useAsyncAction();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null);
  const [searchText, setSearchText] = useState('');
  const [form] = Form.useForm();

  // 搜索过滤
  const filteredChannels = useMemo(() => {
    if (!searchText) return channels;
    const lowerSearch = searchText.toLowerCase();
    return channels?.filter((ch) =>
      ch.name.toLowerCase().includes(lowerSearch) ||
      ch.webhook_url.toLowerCase().includes(lowerSearch)
    );
  }, [channels, searchText]);

  const handleCreate = () => {
    setEditingChannel(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: NotificationChannel) => {
    setEditingChannel(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    const success = await execute(
      () => notificationApi.delete(id),
      '通知渠道删除成功'
    );
    if (success) refetch();
  };

  const handleTest = async (id: number) => {
    await execute(() => notificationApi.test(id), '测试通知已发送，请检查您的通知渠道');
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingChannel) {
        await execute(
          () => notificationApi.update(editingChannel.id, values),
          '通知渠道更新成功'
        );
      } else {
        await execute(() => notificationApi.create(values), '通知渠道创建成功');
      }
      setModalVisible(false);
      refetch();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => {
        const typeConfig: Record<string, { color: string; label: string }> = {
          wechat: { color: 'green', label: '企业微信' },
          dingtalk: { color: 'blue', label: '钉钉' },
        };
        const config = typeConfig[type] || { color: 'default', label: type };
        return <Tag color={config.color}>{config.label}</Tag>;
      },
    },
    {
      title: 'Webhook URL',
      dataIndex: 'webhook_url',
      key: 'webhook_url',
      ellipsis: true,
      render: (url: string) => (
        <span title={url}>{url.length > 50 ? url.substring(0, 50) + '...' : url}</span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'success' : 'default'}>
          {enabled ? '已启用' : '已禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: NotificationChannel) => (
        <Space>
          <Button
            type="link"
            icon={<CheckCircleOutlined />}
            onClick={() => handleTest(record.id)}
          >
            测试
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此通知渠道吗？"
            description="删除后，使用此渠道的任务将无法发送通知"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ margin: 0 }}>通知配置</h2>
          <Space>
            <Input
              placeholder="搜索通知渠道（名称/URL）"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 250 }}
              allowClear
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              添加通知渠道
            </Button>
          </Space>
        </div>

        <Table
          dataSource={filteredChannels || []}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            showTotal: (total) => `共 ${total} 条`,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
        />
      </Card>

      <Modal
        title={editingChannel ? '编辑通知渠道' : '添加通知渠道'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={actionLoading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="渠道名称"
            rules={[{ required: true, message: '请输入渠道名称' }]}
          >
            <Input placeholder="例如: 运维群通知" />
          </Form.Item>

          <Form.Item
            name="type"
            label="渠道类型"
            rules={[{ required: true, message: '请选择渠道类型' }]}
          >
            <Select placeholder="选择通知渠道类型">
              <Select.Option value="wechat">企业微信</Select.Option>
              <Select.Option value="dingtalk">钉钉（暂未启用签名）</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="webhook_url"
            label="Webhook URL"
            rules={[
              { required: true, message: '请输入 Webhook URL' },
              { type: 'url', message: '请输入有效的 URL' },
            ]}
            extra="企业微信群机器人 Webhook 地址或钉钉群机器人 Webhook 地址"
          >
            <Input placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
          </Form.Item>

          <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Notifications;
