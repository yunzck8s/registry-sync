import React, { useState, useEffect, useMemo } from 'react';
import { Table, Button, Space, Tag, Switch, Popconfirm, Modal, Form, Input, Select, InputNumber, Checkbox, Card, Alert } from 'antd';
import { PlayCircleOutlined, StopOutlined, EditOutlined, DeleteOutlined, PlusOutlined, SearchOutlined, FilterOutlined } from '@ant-design/icons';
import { useApi, useAsyncAction } from '../hooks/useApi';
import { taskApi, registryApi, notificationApi } from '../api/client';
import type { SyncTask } from '../types';

const Tasks: React.FC = () => {
  const { data: tasks, loading, refetch } = useApi(() => taskApi.list(), []);
  const { data: registries } = useApi(() => registryApi.list(), []);
  const { data: notifications } = useApi(() => notificationApi.list(), []);
  const { loading: actionLoading, execute } = useAsyncAction();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<SyncTask | null>(null);
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | null>(null);
  const [form] = Form.useForm();

  // 源配置状态
  const [sourceProjects, setSourceProjects] = useState<string[]>([]);
  const [sourceRepos, setSourceRepos] = useState<string[]>([]);
  const [sourceProjectsLoading, setSourceProjectsLoading] = useState(false);
  const [sourceReposLoading, setSourceReposLoading] = useState(false);
  const [syncAllSourceRepos, setSyncAllSourceRepos] = useState(false);

  // 目标配置状态
  const [targetProjects, setTargetProjects] = useState<string[]>([]);
  const [targetRepos, setTargetRepos] = useState<string[]>([]);
  const [targetProjectsLoading, setTargetProjectsLoading] = useState(false);
  const [targetReposLoading, setTargetReposLoading] = useState(false);

  const handleCreate = () => {
    setEditingTask(null);
    form.resetFields();
    setSourceProjects([]);
    setSourceRepos([]);
    setTargetProjects([]);
    setTargetRepos([]);
    setSyncAllSourceRepos(false);
    setModalVisible(true);
  };

  const handleEdit = (record: SyncTask) => {
    setEditingTask(record);

    // Parse notification channel IDs
    let notificationChannelIds: number[] = [];
    if (record.notification_channel_ids) {
      try {
        notificationChannelIds = JSON.parse(record.notification_channel_ids);
      } catch (e) {
        console.error('Failed to parse notification_channel_ids:', e);
      }
    }

    form.setFieldsValue({
      ...record,
      tag_include: record.tag_include?.join(','),
      tag_exclude: record.tag_exclude?.join(','),
      notification_channel_ids: notificationChannelIds,
    });
    setSyncAllSourceRepos(!record.source_repo);
    setModalVisible(true);

    // 加载项目和仓库列表
    if (record.source_registry) {
      loadSourceProjects(record.source_registry);
      if (record.source_project) {
        loadSourceRepositories(record.source_registry, record.source_project);
      }
    }
    if (record.target_registry) {
      loadTargetProjects(record.target_registry);
      if (record.target_project) {
        loadTargetRepositories(record.target_registry, record.target_project);
      }
    }
  };

  // 加载源项目列表
  const loadSourceProjects = async (registryId: number) => {
    setSourceProjectsLoading(true);
    try {
      const response = await registryApi.listProjects(registryId);
      const projects = response.data || [];
      console.log(`✅ 加载源项目列表: ${projects.length} 个项目`, projects);
      setSourceProjects(projects);
    } catch (error) {
      console.error('Failed to load source projects:', error);
      setSourceProjects([]);
    } finally {
      setSourceProjectsLoading(false);
    }
  };

  // 加载源仓库列表
  const loadSourceRepositories = async (registryId: number, project: string) => {
    setSourceReposLoading(true);
    try {
      const response = await registryApi.listRepositories(registryId, project);
      const repos = response.data || [];
      console.log(`✅ 加载源仓库列表 (${project}): ${repos.length} 个仓库`, repos);
      setSourceRepos(repos);
    } catch (error) {
      console.error('Failed to load source repositories:', error);
      setSourceRepos([]);
    } finally {
      setSourceReposLoading(false);
    }
  };

  // 加载目标项目列表
  const loadTargetProjects = async (registryId: number) => {
    setTargetProjectsLoading(true);
    try {
      const response = await registryApi.listProjects(registryId);
      setTargetProjects(response.data || []);
    } catch (error) {
      console.error('Failed to load target projects:', error);
      setTargetProjects([]);
    } finally {
      setTargetProjectsLoading(false);
    }
  };

  // 加载目标仓库列表
  const loadTargetRepositories = async (registryId: number, project: string) => {
    setTargetReposLoading(true);
    try {
      const response = await registryApi.listRepositories(registryId, project);
      setTargetRepos(response.data || []);
    } catch (error) {
      console.error('Failed to load target repositories:', error);
      setTargetRepos([]);
    } finally {
      setTargetReposLoading(false);
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();

      // 处理数组字段和特殊逻辑
      const taskData = {
        ...values,
        source_repo: syncAllSourceRepos ? '' : (values.source_repo || ''),
        target_project: Array.isArray(values.target_project)
          ? values.target_project[0]
          : values.target_project,  // tags mode 返回数组，需要转换为字符串
        tag_include: values.tag_include ? values.tag_include.split(',').map((s: string) => s.trim()).filter(Boolean) : [],
        tag_exclude: values.tag_exclude ? values.tag_exclude.split(',').map((s: string) => s.trim()).filter(Boolean) : [],
        architectures: values.architectures || ['amd64'],
        tag_latest: values.tag_latest || 0,
        enabled: values.enabled !== false,
        send_notification: values.send_notification || false,
        notification_condition: values.notification_condition || 'all',
        notification_channel_ids: JSON.stringify(values.notification_channel_ids || []),
      };

      if (editingTask) {
        await execute(
          () => taskApi.update(editingTask.id, taskData),
          '任务更新成功'
        );
      } else {
        await execute(() => taskApi.create(taskData), '任务创建成功');
      }
      setModalVisible(false);
      refetch();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  const handleRun = async (id: number) => {
    const success = await execute(() => taskApi.run(id), '任务已开始执行');
    if (success) refetch();
  };

  const handleStop = async (id: number) => {
    const success = await execute(() => taskApi.stop(id), '任务已停止');
    if (success) refetch();
  };

  const handleToggle = async (id: number, enabled: boolean) => {
    const success = await execute(
      () => taskApi.update(id, { enabled }),
      enabled ? '任务已启用' : '任务已禁用'
    );
    if (success) refetch();
  };

  const handleDelete = async (id: number) => {
    const success = await execute(() => taskApi.delete(id), '任务已删除');
    if (success) refetch();
  };

  // 搜索和筛选
  const filteredTasks = useMemo(() => {
    if (!tasks) return [];

    let filtered = tasks;

    // 搜索过滤
    if (searchText) {
      const lowerSearch = searchText.toLowerCase();
      filtered = filtered.filter((task) =>
        task.name.toLowerCase().includes(lowerSearch) ||
        task.description?.toLowerCase().includes(lowerSearch) ||
        task.source_project?.toLowerCase().includes(lowerSearch) ||
        task.target_project?.toLowerCase().includes(lowerSearch)
      );
    }

    // 状态过滤
    if (statusFilter) {
      if (statusFilter === 'enabled') {
        filtered = filtered.filter((task) => task.enabled);
      } else if (statusFilter === 'disabled') {
        filtered = filtered.filter((task) => !task.enabled);
      }
    }

    return filtered;
  }, [tasks, searchText, statusFilter]);

  const columns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '源',
      key: 'source',
      render: (_: any, record: SyncTask) => (
        <span>
          {record.source_registry_obj?.name || `Registry ${record.source_registry}`} /
          {record.source_project}{record.source_repo ? `/${record.source_repo}` : ' (整个项目)'}
        </span>
      ),
    },
    {
      title: '目标',
      key: 'target',
      render: (_: any, record: SyncTask) => (
        <span>
          {record.target_registry_obj?.name || `Registry ${record.target_registry}`} /
          {record.target_project}{record.target_repo ? `/${record.target_repo}` : ''}
        </span>
      ),
    },
    {
      title: '架构',
      dataIndex: 'architectures',
      key: 'architectures',
      render: (archs: string[]) => (
        <>
          {archs?.map((arch) => (
            <Tag key={arch}>{arch}</Tag>
          ))}
        </>
      ),
    },
    {
      title: 'Cron',
      dataIndex: 'cron_expression',
      key: 'cron_expression',
      render: (cron: string) => cron || '-',
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: SyncTask) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleToggle(record.id, checked)}
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: SyncTask) => (
        <Space>
          <Button
            type="link"
            icon={<PlayCircleOutlined />}
            onClick={() => handleRun(record.id)}
          >
            运行
          </Button>
          <Button
            type="link"
            icon={<StopOutlined />}
            onClick={() => handleStop(record.id)}
          >
            停止
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此任务吗？"
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
          <h2 style={{ margin: 0 }}>同步任务</h2>
          <Space>
            <Input
              placeholder="搜索任务（名称/描述/项目）"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 250 }}
              allowClear
            />
            <Select
              placeholder="状态筛选"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: 130 }}
              allowClear
            >
              <Select.Option value="enabled">已启用</Select.Option>
              <Select.Option value="disabled">已禁用</Select.Option>
            </Select>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              创建任务
            </Button>
          </Space>
        </div>

        <Table
          dataSource={filteredTasks}
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
        title={editingTask ? '编辑任务' : '创建任务'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={actionLoading}
        width={700}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="例如: nginx-sync" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="任务描述（可选）" rows={2} />
          </Form.Item>

          <div style={{ border: '1px solid #d9d9d9', borderRadius: 4, padding: 16, marginBottom: 16 }}>
            <h3 style={{ marginTop: 0 }}>源配置</h3>

            <Form.Item
              name="source_registry"
              label="源 Registry"
              rules={[{ required: true, message: '请选择源 Registry' }]}
            >
              <Select
                placeholder="选择源 Registry"
                onChange={(value) => {
                  loadSourceProjects(value);
                  setSourceRepos([]);
                  form.setFieldsValue({ source_project: undefined, source_repo: undefined });
                }}
              >
                {registries?.map((reg) => (
                  <Select.Option key={reg.id} value={reg.id}>
                    {reg.name} ({reg.url})
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="source_project"
              label="项目"
              rules={[{ required: true, message: '请选择项目' }]}
            >
              <Select
                showSearch
                placeholder="选择项目"
                loading={sourceProjectsLoading}
                optionFilterProp="children"
                filterOption={(input, option) =>
                  (option?.children as string).toLowerCase().includes(input.toLowerCase())
                }
                virtual={true}
                onChange={(value) => {
                  const registryId = form.getFieldValue('source_registry');
                  if (registryId) {
                    loadSourceRepositories(registryId, value);
                  }
                  form.setFieldsValue({ source_repo: undefined });
                }}
              >
                {sourceProjects.map((project) => (
                  <Select.Option key={project} value={project}>
                    {project}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item label=" " colon={false}>
              <Checkbox
                checked={syncAllSourceRepos}
                onChange={(e) => {
                  setSyncAllSourceRepos(e.target.checked);
                  if (e.target.checked) {
                    form.setFieldsValue({ source_repo: undefined });
                  }
                }}
              >
                同步整个项目的所有仓库
              </Checkbox>
            </Form.Item>

            {!syncAllSourceRepos && (
              <Form.Item
                name="source_repo"
                label="仓库（可选）"
              >
                <Select
                  showSearch
                  placeholder="选择仓库（留空=同步整个项目）"
                  loading={sourceReposLoading}
                  allowClear
                  optionFilterProp="children"
                  filterOption={(input, option) =>
                    (option?.children as string).toLowerCase().includes(input.toLowerCase())
                  }
                  virtual={true}
                >
                  {sourceRepos.map((repo) => (
                    <Select.Option key={repo} value={repo}>
                      {repo}
                    </Select.Option>
                  ))}
                </Select>
              </Form.Item>
            )}
          </div>

          <div style={{ border: '1px solid #d9d9d9', borderRadius: 4, padding: 16, marginBottom: 16 }}>
            <h3 style={{ marginTop: 0 }}>目标配置</h3>

            <Form.Item
              name="target_registry"
              label="目标 Registry"
              rules={[{ required: true, message: '请选择目标 Registry' }]}
            >
              <Select
                placeholder="选择目标 Registry"
                onChange={(value) => {
                  loadTargetProjects(value);
                  setTargetRepos([]);
                  form.setFieldsValue({ target_project: undefined, target_repo: undefined });
                }}
              >
                {registries?.map((reg) => (
                  <Select.Option key={reg.id} value={reg.id}>
                    {reg.name} ({reg.url})
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="target_project"
              label="项目"
              rules={[{ required: true, message: '请输入目标项目' }]}
            >
              <Select
                mode="tags"
                placeholder="选择已有项目或输入新项目名"
                loading={targetProjectsLoading}
                optionFilterProp="children"
                filterOption={(input, option) =>
                  (option?.children as string).toLowerCase().includes(input.toLowerCase())
                }
                virtual={true}
                onChange={(value) => {
                  const registryId = form.getFieldValue('target_registry');
                  if (registryId && typeof value === 'string') {
                    loadTargetRepositories(registryId, value);
                  }
                }}
                maxTagCount={1}
              >
                {targetProjects.map((project) => (
                  <Select.Option key={project} value={project}>
                    {project}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="target_repo"
              label="仓库（可选）"
              extra="留空则自动使用源仓库名"
            >
              <Input placeholder="自定义仓库名（可选）" />
            </Form.Item>
          </div>

          <Form.Item name="tag_include" label="包含 Tag（正则，逗号分隔）">
            <Input placeholder='例如: ^1\\.2[0-9]\\.*,latest' />
          </Form.Item>

          <Form.Item name="tag_exclude" label="排除 Tag（正则，逗号分隔）">
            <Input placeholder="例如: .*-alpine,.*-debug" />
          </Form.Item>

          <Form.Item name="tag_latest" label="保留最新 N 个">
            <InputNumber min={0} placeholder="0 表示不限制" style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="architectures" label="架构">
            <Select mode="multiple" placeholder="选择架构">
              <Select.Option value="amd64">amd64</Select.Option>
              <Select.Option value="arm64">arm64</Select.Option>
              <Select.Option value="arm/v7">arm/v7</Select.Option>
              <Select.Option value="386">386</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item name="cron_expression" label="Cron 表达式（定时任务）">
            <Input placeholder='例如: 0 2 * * * (每天凌晨2点)' />
          </Form.Item>

          <div style={{ border: '1px solid #d9d9d9', borderRadius: 4, padding: 16, marginBottom: 16 }}>
            <h3 style={{ marginTop: 0 }}>通知配置</h3>

            <Form.Item name="send_notification" label="启用通知" valuePropName="checked" initialValue={false}>
              <Switch />
            </Form.Item>

            <Form.Item shouldUpdate={(prev, curr) => prev.send_notification !== curr.send_notification}>
              {({ getFieldValue }) =>
                getFieldValue('send_notification') ? (
                  <>
                    <Form.Item name="notification_condition" label="通知条件" initialValue="all">
                      <Select>
                        <Select.Option value="all">全部发送（成功+失败）</Select.Option>
                        <Select.Option value="failed">仅失败时发送</Select.Option>
                      </Select>
                    </Form.Item>

                    <Form.Item
                      name="notification_channel_ids"
                      label="通知渠道"
                      extra="选择要接收通知的渠道（可多选）"
                    >
                      <Checkbox.Group style={{ width: '100%' }}>
                        <Space direction="vertical" style={{ width: '100%' }}>
                          {notifications?.filter(n => n.enabled).map((channel) => (
                            <Checkbox key={channel.id} value={channel.id}>
                              {channel.name} ({channel.type === 'wechat' ? '企业微信' : '钉钉'})
                            </Checkbox>
                          ))}
                          {(!notifications || notifications.filter(n => n.enabled).length === 0) && (
                            <span style={{ color: '#999' }}>
                              暂无可用的通知渠道，请先在"通知配置"页面添加
                            </span>
                          )}
                        </Space>
                      </Checkbox.Group>
                    </Form.Item>

                    {form.getFieldValue('cron_expression') && (
                      <Alert
                        message="定时任务通知提示"
                        description="此任务为定时任务，将按 cron 表达式自动执行。如果频率较高（如每小时执行），可能会产生大量通知消息。建议仅在重要任务上启用通知。"
                        type="warning"
                        showIcon
                        style={{ marginTop: 8 }}
                      />
                    )}
                  </>
                ) : null
              }
            </Form.Item>
          </div>

          <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Tasks;
