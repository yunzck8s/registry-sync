import React, { useState, useMemo } from 'react';
import { Table, Button, Modal, Form, Input, Switch, InputNumber, Space, Popconfirm, Card } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined, SearchOutlined } from '@ant-design/icons';
import { useApi, useAsyncAction } from '../hooks/useApi';
import { registryApi } from '../api/client';
import type { Registry } from '../types';

const Registries: React.FC = () => {
  const { data: registries, loading, refetch } = useApi(() => registryApi.list(), []);
  const { loading: actionLoading, execute } = useAsyncAction();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRegistry, setEditingRegistry] = useState<Registry | null>(null);
  const [searchText, setSearchText] = useState('');
  const [form] = Form.useForm();

  // 搜索过滤
  const filteredRegistries = useMemo(() => {
    if (!searchText) return registries;
    const lowerSearch = searchText.toLowerCase();
    return registries?.filter((reg) =>
      reg.name.toLowerCase().includes(lowerSearch) ||
      reg.url.toLowerCase().includes(lowerSearch) ||
      reg.username?.toLowerCase().includes(lowerSearch)
    );
  }, [registries, searchText]);

  const handleCreate = () => {
    setEditingRegistry(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: Registry) => {
    setEditingRegistry(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    const success = await execute(
      () => registryApi.delete(id),
      'Registry 删除成功'
    );
    if (success) refetch();
  };

  const handleTest = async (id: number) => {
    await execute(() => registryApi.test(id), '连接测试成功');
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingRegistry) {
        await execute(
          () => registryApi.update(editingRegistry.id, values),
          'Registry 更新成功'
        );
      } else {
        await execute(() => registryApi.create(values), 'Registry 创建成功');
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
      title: 'URL',
      dataIndex: 'url',
      key: 'url',
    },
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: 'QPS 限制',
      dataIndex: 'rate_limit',
      key: 'rate_limit',
      render: (limit: number) => limit || '无限制',
    },
    {
      title: '不安全连接',
      dataIndex: 'insecure',
      key: 'insecure',
      render: (insecure: boolean) => (insecure ? '是' : '否'),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: Registry) => (
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
            title="确定要删除此 Registry 吗？"
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
          <h2 style={{ margin: 0 }}>Registry 管理</h2>
          <Space>
            <Input
              placeholder="搜索 Registry（名称/URL/用户名）"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 300 }}
              allowClear
            />
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              添加 Registry
            </Button>
          </Space>
        </div>

        <Table
          dataSource={filteredRegistries || []}
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
        title={editingRegistry ? '编辑 Registry' : '添加 Registry'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={actionLoading}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="例如: dockerhub" />
          </Form.Item>

          <Form.Item
            name="url"
            label="URL"
            rules={[{ required: true, message: '请输入 URL' }]}
          >
            <Input placeholder="https://registry-1.docker.io" />
          </Form.Item>

          <Form.Item name="username" label="用户名">
            <Input placeholder="用户名" />
          </Form.Item>

          <Form.Item name="password" label="密码">
            <Input.Password placeholder="密码" />
          </Form.Item>

          <Form.Item name="rate_limit" label="QPS 限制">
            <InputNumber min={0} placeholder="0 表示无限制" style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="insecure" label="允许不安全连接" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Registries;
