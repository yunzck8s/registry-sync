# Registry Sync

企业级容器镜像同步工具，基于 Registry API V2 实现，支持跨云、跨 Registry 的镜像同步。

## ✨ 核心特性

- 🚀 **直接操作 Registry API V2** - 无需 Docker Daemon，直接搬运 Blob 和 Manifest
- ⚡ **高性能** - 并发控制 + 增量同步 + 智能重试（指数退避）
- 🎯 **智能过滤** - 正则表达式匹配、黑名单过滤、保留最新 N 个版本
- 🌐 **多架构支持** - 自动识别 Manifest List，支持 amd64/arm64 等多架构
- 🔄 **定时同步** - Cron 表达式定时任务
- 📊 **Web 管理界面** - React + Ant Design 现代化 UI
- 💾 **数据持久化** - SQLite 存储任务配置和执行历史
- 🔌 **实时推送** - WebSocket 实时同步进度和日志

## 🚀 快速开始

### 方式 1：命令行工具（单次同步）

```bash
# 1. 构建
go build -o registry-sync cmd/cli/main.go

# 2. 配置
cp configs/sync.example.yaml configs/sync.yaml
# 编辑 configs/sync.yaml，配置源和目标 Registry

# 3. 运行
./registry-sync --config configs/sync.yaml
```

### 方式 2：Web 管理平台（推荐）

```bash
# 1. 启动后端服务
go build -o registry-sync-server cmd/server/main.go
./registry-sync-server
# 服务运行在 http://localhost:8080

# 2. 启动前端（开发模式）
cd web
npm install
npm run dev
# 前端运行在 http://localhost:3000

# 访问 http://localhost:3000 使用 Web 界面
```

### 生产部署

```bash
# 1. 构建前端
cd web && npm run build

# 2. 启动服务器（自动提供前端静态文件）
./registry-sync-server --port 8080

# 访问 http://localhost:8080
```

## 📝 配置示例

```yaml
version: "1.0"

# 全局设置
global:
  concurrency: 5          # 并发传输 Blob 的数量
  retry:
    max_attempts: 5       # 失败操作的最大重试次数
    initial_interval: 1s
    max_interval: 30s
  timeout: 10m

# Registry 定义
registries:
  dockerhub:
    url: https://registry-1.docker.io
    username: ${DOCKERHUB_USER}
    password: ${DOCKERHUB_PASSWORD}

  harbor:
    url: https://harbor.example.com
    username: admin
    password: ${HARBOR_PASSWORD}
    ratelimit:
      qps: 50

# 同步规则
sync_rules:
  - name: "nginx-sync"
    source:
      registry: dockerhub
      repository: library/nginx
    target:
      registry: harbor
      repository: prod/nginx
    tags:
      include: ["^1\\.2[0-9]\\.*"]    # 只同步 1.2x 版本
      exclude: [".*-alpine"]           # 排除 alpine 变体
      latest: 10                        # 只保留最新 10 个
    architectures: ["amd64", "arm64"]
    enabled: true
    cron_expression: "0 2 * * *"       # 每天凌晨 2 点执行
```

## 🎨 Web 界面功能

### 仪表盘
- 统计卡片（总任务数、运行中、成功、失败）
- 最近执行记录列表
- 实时状态更新

### Registry 管理
- 添加/编辑/删除 Registry
- 测试连接
- 配置认证和 QPS 限制

### 任务管理
- 创建/编辑/删除同步任务
- 立即运行/停止任务
- 启用/禁用任务
- Cron 定时配置

### 执行历史
- 查看所有执行记录
- 进度和状态展示
- 查看详细日志
- 执行时间和耗时统计

## 📡 API 文档

### Registry 管理
```bash
POST   /api/v1/registries          # 创建 Registry
GET    /api/v1/registries          # 列出所有 Registry
GET    /api/v1/registries/:id      # 获取单个 Registry
PUT    /api/v1/registries/:id      # 更新 Registry
DELETE /api/v1/registries/:id      # 删除 Registry
POST   /api/v1/registries/:id/test # 测试连接
```

### 任务管理
```bash
POST   /api/v1/tasks               # 创建任务
GET    /api/v1/tasks               # 列出所有任务
GET    /api/v1/tasks/:id           # 获取任务详情
PUT    /api/v1/tasks/:id           # 更新任务
DELETE /api/v1/tasks/:id           # 删除任务
POST   /api/v1/tasks/:id/run       # 立即运行
POST   /api/v1/tasks/:id/stop      # 停止任务
```

### 执行历史
```bash
GET    /api/v1/executions          # 列出执行记录
GET    /api/v1/executions/:id      # 获取执行详情
GET    /api/v1/executions/:id/logs # 获取执行日志
GET    /api/v1/stats               # 获取统计信息
```

### WebSocket
```bash
WS     /api/v1/ws                  # 实时进度推送
```

## 🛠️ 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin (HTTP) + Gorilla WebSocket
- **数据库**: SQLite + GORM
- **调度**: robfig/cron

### 前端
- **框架**: React 18 + TypeScript
- **UI**: Ant Design 5
- **构建**: Vite
- **HTTP**: Axios
- **路由**: React Router 6

## 📂 项目结构

```
registry-sync/
├── cmd/
│   ├── cli/main.go              # CLI 工具入口
│   └── server/main.go           # Web 服务器入口
├── pkg/                         # 核心逻辑（CLI 和 Server 共用）
│   ├── config/                  # 配置管理
│   ├── registry/                # Registry API V2 客户端
│   ├── sync/                    # 同步引擎（Worker Pool + 智能重试）
│   ├── filter/                  # Tag 过滤器
│   └── ratelimit/               # 限流器
├── internal/                    # Web 服务专用
│   ├── api/handlers/            # REST API 处理器
│   ├── db/models/               # 数据模型
│   ├── db/store/                # 数据访问层
│   ├── scheduler/               # 任务调度器
│   └── websocket/               # WebSocket Hub
├── web/                         # React 前端
│   ├── src/
│   │   ├── api/                 # API 客户端
│   │   ├── components/          # React 组件
│   │   ├── pages/               # 页面组件
│   │   ├── hooks/               # 自定义 Hooks
│   │   └── types/               # TypeScript 类型
│   └── package.json
├── configs/
│   └── sync.example.yaml        # 配置示例
└── README.md
```

## 🔧 开发

### 构建

```bash
# CLI 工具
make build

# Web 服务器
make server

# 或使用 Makefile
make help
```

### 运行测试

```bash
go test -v ./...
```

### 前端开发

```bash
cd web
npm install
npm run dev      # 开发模式
npm run build    # 生产构建
```

## 🎯 使用场景

- **跨云迁移**: 从 Docker Hub 同步到阿里云 ACR / 腾讯云 TCR
- **内网镜像**: 从公网同步到内网 Harbor
- **灾备同步**: 定期同步到备份 Registry
- **多架构**: 同步 amd64 和 arm64 镜像到生产环境
- **版本控制**: 只保留稳定版本，过滤测试版本

## 📊 性能优化

- **并发传输**: 默认 5 个 worker，可根据带宽调整
- **增量同步**: 只传输新增或变更的 Blob
- **Rate Limiting**: 避免触发云厂商限流
- **跨仓库 Mount**: 如果目标 Registry 支持，直接引用已有 Blob
- **流式传输**: 不占用本地磁盘空间

## 🐛 常见问题

### Q: 如何加速同步？
A: 增加 `global.concurrency` 值（建议 5-10），确保网络带宽充足

### Q: 同步失败怎么办？
A: 检查：1) Registry 凭据是否正确 2) 网络是否通畅 3) 是否触发了 Rate Limit

### Q: 如何只同步特定版本？
A: 使用 `tags.include` 正则表达式精确匹配版本号

### Q: 支持 Docker Hub 的限流吗？
A: 支持，配置 `ratelimit.qps` 限制请求速率

## 📄 文档

- [Web Server API 文档](WEB_SERVER.md)
- [前端开发文档](web/README.md)
- [项目结构说明](PROJECT_STRUCTURE.md)

## 📅 Roadmap

- [x] Phase 1: CLI + 配置文件
- [x] Phase 2: RESTful API + 数据持久化 + Cron 调度 + WebSocket
- [x] Phase 3: React 前端界面（基础版本）
- [ ] Phase 4: 任务创建向导 + 实时进度优化
- [ ] Phase 5: 用户认证和 RBAC
- [ ] Phase 6: Webhook 通知 + Swagger 文档
- [ ] Phase 7: Docker 镜像和 K8s 部署

## 📄 License

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**Made with ❤️ for DevOps Engineers**
