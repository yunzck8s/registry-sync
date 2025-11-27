import axios from 'axios';
import type {
  Registry,
  SyncTask,
  Execution,
  ExecutionLog,
  Stats,
  NotificationChannel,
} from '../types';

const API_BASE_URL = '/api/v1';

const client = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Registry API
export const registryApi = {
  list: () => client.get<Registry[]>('/registries'),
  get: (id: number) => client.get<Registry>(`/registries/${id}`),
  create: (data: Partial<Registry>) => client.post<Registry>('/registries', data),
  update: (id: number, data: Partial<Registry>) =>
    client.put<Registry>(`/registries/${id}`, data),
  delete: (id: number) => client.delete(`/registries/${id}`),
  test: (id: number) => client.post(`/registries/${id}/test`),
  listProjects: (id: number) => client.get<string[]>(`/registries/${id}/projects`),
  listRepositories: (id: number, project: string) =>
    client.get<string[]>(`/registries/${id}/projects/${project}/repositories`),
};

// Task API
export const taskApi = {
  list: () => client.get<SyncTask[]>('/tasks'),
  get: (id: number) => client.get<SyncTask>(`/tasks/${id}`),
  create: (data: Partial<SyncTask>) => client.post<SyncTask>('/tasks', data),
  update: (id: number, data: Partial<SyncTask>) =>
    client.put<SyncTask>(`/tasks/${id}`, data),
  delete: (id: number) => client.delete(`/tasks/${id}`),
  run: (id: number) => client.post(`/tasks/${id}/run`),
  stop: (id: number) => client.post(`/tasks/${id}/stop`),
};

// Execution API
export const executionApi = {
  list: (params?: { limit?: number; task_id?: number }) =>
    client.get<Execution[]>('/executions', { params }),
  get: (id: number) => client.get<Execution>(`/executions/${id}`),
  logs: (id: number, limit = 1000) =>
    client.get<ExecutionLog[]>(`/executions/${id}/logs`, {
      params: { limit },
    }),
};

// Stats API
export const statsApi = {
  get: () => client.get<Stats>('/stats'),
};

// Notification API
export const notificationApi = {
  list: () => client.get<NotificationChannel[]>('/notifications'),
  get: (id: number) => client.get<NotificationChannel>(`/notifications/${id}`),
  create: (data: Partial<NotificationChannel>) =>
    client.post<NotificationChannel>('/notifications', data),
  update: (id: number, data: Partial<NotificationChannel>) =>
    client.put<NotificationChannel>(`/notifications/${id}`, data),
  delete: (id: number) => client.delete(`/notifications/${id}`),
  test: (id: number) => client.post(`/notifications/${id}/test`),
};

export default client;
