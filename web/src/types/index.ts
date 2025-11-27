// Registry 类型
export interface Registry {
  id: number;
  name: string;
  url: string;
  username: string;
  insecure: boolean;
  rate_limit: number;
  created_at: string;
  updated_at: string;
}

// 同步任务类型
export interface SyncTask {
  id: number;
  name: string;
  description?: string;
  source_registry: number;
  source_project: string;      // 新增：源项目
  source_repo: string;          // 改为可选：空=同步整个项目
  target_registry: number;
  target_project: string;       // 新增：目标项目
  target_repo: string;          // 改为可选：空=使用源仓库名
  tag_include: string[];
  tag_exclude: string[];
  tag_latest: number;
  architectures: string[];
  enabled: boolean;
  cron_expression: string;
  send_notification: boolean;
  notification_condition: 'all' | 'failed';
  notification_channel_ids: string;
  created_at: string;
  updated_at: string;
  source_registry_obj?: Registry;
  target_registry_obj?: Registry;
}

// 通知渠道类型
export interface NotificationChannel {
  id: number;
  name: string;
  type: 'wechat' | 'dingtalk';
  webhook_url: string;
  secret?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

// 执行状态
export type ExecutionStatus = 'pending' | 'running' | 'success' | 'failed' | 'canceled';

// 执行记录
export interface Execution {
  id: number;
  task_id: number;
  status: ExecutionStatus;
  start_time: string;
  end_time?: string;
  total_blobs: number;
  synced_blobs: number;
  skipped_blobs: number;
  failed_blobs: number;
  total_size: number;
  synced_size: number;
  error_message: string;
  created_at: string;
  updated_at: string;
  task?: SyncTask;
}

// 日志级别
export type LogLevel = 'info' | 'warn' | 'error' | 'debug';

// 执行日志
export interface ExecutionLog {
  id: number;
  execution_id: number;
  level: LogLevel;
  message: string;
  timestamp: string;
}

// 统计信息
export interface Stats {
  total_tasks: number;
  enabled_tasks: number;
  total_executions: number;
  running_executions: number;
  success_executions: number;
  failed_executions: number;
  total_registries: number;
}

// WebSocket 消息类型
export interface WSMessage {
  type: 'progress' | 'log';
  execution_id: number;
  data?: any;
  level?: string;
  message?: string;
}

// API 响应
export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}
