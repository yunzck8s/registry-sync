# Registry Sync - Web 前端

基于 React + TypeScript + Ant Design 的镜像同步管理界面。

## 🚀 快速开始

### 安装依赖

```bash
cd web
npm install
```

### 开发模式

```bash
npm run dev
```

访问 http://localhost:3000

API 请求会自动代理到 http://localhost:8080

### 生产构建

```bash
npm run build
```

构建输出到 `build/` 目录，可直接部署到 Web 服务器。

## 📁 项目结构

```
web/
├── src/
│   ├── api/              # API 客户端
│   │   ├── client.ts     # Axios 封装和 API 方法
│   │   └── websocket.ts  # WebSocket 客户端
│   ├── components/       # React 组件
│   │   └── Layout.tsx    # 主布局组件
│   ├── pages/            # 页面组件
│   │   ├── Dashboard.tsx      # 仪表盘
│   │   ├── Registries.tsx     # Registry 管理
│   │   ├── Tasks.tsx          # 任务管理
│   │   └── Executions.tsx     # 执行历史
│   ├── hooks/            # 自定义 Hooks
│   │   ├── useApi.ts          # API 调用 Hook
│   │   └── useWebSocket.ts    # WebSocket Hook
│   ├── types/            # TypeScript 类型定义
│   │   └── index.ts
│   ├── App.tsx           # 根组件
│   ├── main.tsx          # 入口文件
│   └── index.css         # 全局样式
├── index.html            # HTML 模板
├── vite.config.ts        # Vite 配置
├── tsconfig.json         # TypeScript 配置
└── package.json          # 依赖配置
```

## 🎨 主要功能

### 1. 仪表盘
- 📊 统计卡片（总任务数、运行中、成功、失败）
- 📈 最近执行记录列表
- 🔄 实时更新状态

### 2. Registry 管理
- ➕ 添加/编辑/删除 Registry
- 🔍 测试连接
- 📝 配置 URL、认证、QPS 限制

### 3. 同步任务管理
- ➕ 创建/编辑/删除任务
- ▶️ 立即运行任务
- ⏸️ 停止运行中的任务
- 🔘 启用/禁用任务
- ⏰ Cron 定时配置

### 4. 执行历史
- 📋 查看所有执行记录
- 📊 进度和状态展示
- 📝 查看详细日志
- 🕒 执行时间和耗时

## 🔌 API 集成

### 基础配置

API 基础地址配置在 `src/api/client.ts`：

```typescript
const API_BASE_URL = '/api/v1';
```

开发模式下，Vite 会自动将 `/api` 代理到 `http://localhost:8080`（配置在 `vite.config.ts`）。

### API 调用示例

```typescript
import { registryApi } from './api/client';

// 获取所有 Registry
const { data } = await registryApi.list();

// 创建 Registry
await registryApi.create({
  name: 'dockerhub',
  url: 'https://registry-1.docker.io',
  username: 'user',
  password: 'pass',
});
```

### WebSocket 连接

```typescript
import { wsClient } from './api/websocket';

// 连接
wsClient.connect('ws://localhost:8080/api/v1/ws');

// 订阅消息
const unsubscribe = wsClient.subscribe((message) => {
  console.log('Received:', message);
});

// 取消订阅
unsubscribe();
```

## 🛠️ 技术栈

- **框架**：React 18
- **语言**：TypeScript
- **构建工具**：Vite
- **UI 库**：Ant Design 5
- **路由**：React Router 6
- **HTTP 客户端**：Axios
- **日期处理**：Day.js
- **状态管理**：React Hooks（useState, useEffect, useCallback）

## 📝 开发规范

### 组件规范

- 使用函数组件 + Hooks
- TypeScript 严格模式
- 组件命名：PascalCase
- 文件名：与组件名一致

### API 调用规范

- 使用 `useApi` Hook 进行数据获取
- 使用 `useAsyncAction` Hook 进行操作
- 统一错误处理和提示

### 代码示例

```typescript
import { useApi, useAsyncAction } from '../hooks/useApi';
import { registryApi } from '../api/client';

const MyComponent = () => {
  // 数据获取
  const { data, loading, refetch } = useApi(() => registryApi.list(), []);

  // 异步操作
  const { loading: actionLoading, execute } = useAsyncAction();

  const handleCreate = async () => {
    const result = await execute(
      () => registryApi.create({ name: 'test' }),
      '创建成功'
    );
    if (result) {
      refetch(); // 刷新列表
    }
  };

  // ...
};
```

## 🎯 待实现功能

- [ ] 任务创建向导（分步表单）
- [ ] 实时进度条和日志滚动
- [ ] 图表可视化（执行趋势）
- [ ] 用户认证和登录
- [ ] 深色模式支持
- [ ] 响应式布局优化
- [ ] 国际化（i18n）
- [ ] 单元测试

## 🐛 常见问题

### Q: API 请求失败？
A: 确保后端服务器在 http://localhost:8080 运行，检查 CORS 配置。

### Q: WebSocket 连接失败？
A: 检查 WebSocket URL 是否正确，确保后端 WebSocket 服务已启动。

### Q: 依赖安装失败？
A: 尝试删除 `node_modules` 和 `package-lock.json`，重新运行 `npm install`。

## 📄 License

MIT License

---

**Made with ❤️ for DevOps Engineers**
