# Registry Sync - Web Server

Web 界面和 RESTful API 服务，用于管理镜像同步任务。

## 🚀 快速开始

### 启动服务器

```bash
# 编译服务器
go build -o registry-sync-server cmd/server/main.go

# 或使用 Makefile
make server

# 启动服务器（默认端口 8080）
./registry-sync-server

# 自定义端口和数据库路径
./registry-sync-server --port 3000 --db /path/to/database.db
```

### 访问

- **Web UI**: http://localhost:8080
- **API 文档**: http://localhost:8080/api/v1/health
- **WebSocket**: ws://localhost:8080/api/v1/ws

## 📡 API 文档

### 1. Registry 管理

#### 创建 Registry
```bash
POST /api/v1/registries
Content-Type: application/json

{
  "name": "dockerhub",
  "url": "https://registry-1.docker.io",
  "username": "myuser",
  "password": "mypass",
  "insecure": false,
  "rate_limit": 50
}
```

#### 列出所有 Registry
```bash
GET /api/v1/registries
```

#### 获取单个 Registry
```bash
GET /api/v1/registries/:id
```

#### 更新 Registry
```bash
PUT /api/v1/registries/:id
Content-Type: application/json

{
  "name": "dockerhub",
  "url": "https://registry-1.docker.io",
  "rate_limit": 100
}
```

#### 删除 Registry
```bash
DELETE /api/v1/registries/:id
```

#### 测试 Registry 连接
```bash
POST /api/v1/registries/:id/test
```

### 2. 同步任务管理

#### 创建同步任务
```bash
POST /api/v1/tasks
Content-Type: application/json

{
  "name": "nginx-sync",
  "description": "同步 nginx 镜像",
  "source_registry": 1,
  "source_repo": "library/nginx",
  "target_registry": 2,
  "target_repo": "prod/nginx",
  "tag_include": ["^1\\.2[0-9]\\.*"],
  "tag_exclude": [".*-alpine"],
  "tag_latest": 10,
  "architectures": ["amd64", "arm64"],
  "enabled": true,
  "cron_expression": "0 2 * * *"
}
```

#### 列出所有任务
```bash
GET /api/v1/tasks
```

#### 获取单个任务
```bash
GET /api/v1/tasks/:id
```

#### 更新任务
```bash
PUT /api/v1/tasks/:id
Content-Type: application/json

{
  "enabled": false,
  "cron_expression": "0 3 * * *"
}
```

#### 删除任务
```bash
DELETE /api/v1/tasks/:id
```

#### 立即运行任务
```bash
POST /api/v1/tasks/:id/run
```

#### 停止正在运行的任务
```bash
POST /api/v1/tasks/:id/stop
```

### 3. 执行历史

#### 列出所有执行记录
```bash
GET /api/v1/executions?limit=50

# 按任务过滤
GET /api/v1/executions?task_id=1&limit=20
```

#### 获取单个执行记录
```bash
GET /api/v1/executions/:id
```

#### 获取执行日志
```bash
GET /api/v1/executions/:id/logs?limit=1000
```

### 4. 统计信息

#### 获取系统统计
```bash
GET /api/v1/stats
```

响应示例：
```json
{
  "total_tasks": 12,
  "enabled_tasks": 8,
  "total_executions": 156,
  "running_executions": 2,
  "success_executions": 142,
  "failed_executions": 12
}
```

### 5. WebSocket 实时更新

连接到 WebSocket 接收实时进度更新：

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  if (data.type === 'progress') {
    console.log('Progress:', data.execution_id, data.data);
  } else if (data.type === 'log') {
    console.log('Log:', data.level, data.message);
  }
};
```

## 🗄️ 数据库

使用 SQLite 作为默认数据库，自动创建以下表：

- `registries` - Registry 配置
- `sync_tasks` - 同步任务
- `executions` - 执行记录
- `execution_logs` - 执行日志

数据库文件默认为 `registry-sync.db`，可通过 `--db` 参数指定路径。

## ⏰ 定时任务

使用 Cron 表达式配置定时任务：

```
# 每天凌晨 2 点执行
0 2 * * *

# 每小时执行
0 * * * *

# 每周一上午 9 点执行
0 9 * * 1

# 每 30 分钟执行
*/30 * * * *
```

## 🔧 配置示例

### 完整工作流程

#### 1. 创建 Registry

```bash
# 创建 Docker Hub
curl -X POST http://localhost:8080/api/v1/registries \
  -H "Content-Type: application/json" \
  -d '{
    "name": "dockerhub",
    "url": "https://registry-1.docker.io",
    "username": "myuser",
    "password": "mypass"
  }'

# 创建 Harbor
curl -X POST http://localhost:8080/api/v1/registries \
  -H "Content-Type: application/json" \
  -d '{
    "name": "harbor",
    "url": "https://harbor.example.com",
    "username": "admin",
    "password": "Harbor12345",
    "rate_limit": 50
  }'
```

#### 2. 创建同步任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx-sync",
    "source_registry": 1,
    "source_repo": "library/nginx",
    "target_registry": 2,
    "target_repo": "prod/nginx",
    "tag_include": ["^1\\.2[0-9]\\.*"],
    "tag_exclude": [".*-alpine"],
    "tag_latest": 10,
    "architectures": ["amd64", "arm64"],
    "enabled": true,
    "cron_expression": "0 2 * * *"
  }'
```

#### 3. 立即运行任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/run
```

#### 4. 查看执行历史

```bash
# 列出所有执行
curl http://localhost:8080/api/v1/executions

# 查看特定执行的日志
curl http://localhost:8080/api/v1/executions/1/logs
```

## 🎨 前端开发（待实现）

前端使用 React + TypeScript + Ant Design：

```bash
cd web
npm install
npm start  # 开发模式
npm build  # 生产构建
```

### 主要页面

1. **仪表盘** - 统计概览、最近执行
2. **Registry 管理** - CRUD Registry 配置
3. **任务管理** - 创建、编辑、运行同步任务
4. **执行历史** - 查看执行记录和日志
5. **实时监控** - WebSocket 实时进度

## 🔐 安全建议

1. **生产环境使用 HTTPS**
2. **添加用户认证**（待实现）
3. **限制 CORS 来源**
4. **加密数据库中的密码**
5. **使用环境变量管理敏感信息**

## 📊 架构说明

```
Web Server
├── cmd/server/          # 服务器入口
├── internal/
│   ├── api/             # API 处理器
│   │   ├── handlers/    # HTTP 处理器
│   │   └── middleware/  # 中间件
│   ├── db/              # 数据库
│   │   ├── models/      # 数据模型
│   │   └── store/       # 数据访问层
│   ├── scheduler/       # 任务调度器
│   └── websocket/       # WebSocket 支持
└── web/                 # 前端代码（React）
```

## 🐛 故障排查

### 数据库锁定
```bash
# 如果遇到 "database is locked" 错误
# 确保只有一个服务器实例运行
pkill registry-sync-server
rm registry-sync.db-journal
```

### WebSocket 连接失败
```bash
# 检查防火墙设置
# 确保 WebSocket 请求头正确
# 浏览器开发者工具 -> Network -> WS
```

### 任务不执行
```bash
# 检查任务是否启用
curl http://localhost:8080/api/v1/tasks/1

# 检查 Cron 表达式是否正确
# 使用在线工具验证：https://crontab.guru/
```

## 📝 开发计划

- [x] RESTful API
- [x] SQLite 数据持久化
- [x] Cron 定时任务
- [x] WebSocket 实时推送
- [ ] React 前端界面
- [ ] 用户认证和授权
- [ ] Webhook 通知
- [ ] API 文档（Swagger）
- [ ] Docker 镜像
- [ ] Kubernetes Deployment

---

**Made with ❤️ for DevOps Engineers**
