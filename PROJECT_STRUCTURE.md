# Registry Sync - 项目结构

完整的项目文件结构和说明。

## 📁 目录结构

```
registry-sync/
├── cmd/                          # 应用程序入口
│   ├── cli/                      # CLI 工具
│   │   └── main.go              # CLI 主程序
│   └── server/                   # Web 服务器
│       └── main.go              # 服务器主程序
│
├── pkg/                          # 公共库（CLI 和 Server 共用）
│   ├── config/                   # 配置管理
│   │   └── config.go            # YAML 配置解析
│   ├── registry/                 # Registry API V2 客户端
│   │   ├── client.go            # HTTP 客户端和认证
│   │   ├── manifest.go          # Manifest 操作
│   │   └── blob.go              # Blob 上传下载
│   ├── sync/                     # 同步引擎
│   │   ├── engine.go            # 主流程控制
│   │   ├── worker.go            # Worker Pool（并发控制）
│   │   └── retry.go             # 智能重试（指数退避）
│   ├── filter/                   # Tag 过滤器
│   │   └── filter.go            # 正则匹配、黑白名单
│   └── ratelimit/                # 限流器
│       └── limiter.go           # Token Bucket 算法
│
├── internal/                     # 私有库（仅 Server 使用）
│   ├── api/                      # RESTful API
│   │   ├── handlers/            # HTTP 处理器
│   │   │   ├── registry.go      # Registry CRUD
│   │   │   ├── task.go          # Task CRUD
│   │   │   └── execution.go     # Execution 查询
│   │   └── middleware/          # 中间件
│   │       └── cors.go          # CORS 处理
│   ├── db/                       # 数据库
│   │   ├── models/              # 数据模型
│   │   │   ├── registry.go      # Registry 模型
│   │   │   ├── task.go          # SyncTask 模型
│   │   │   └── execution.go     # Execution 模型
│   │   └── store/               # 数据访问层
│   │       └── store.go         # GORM 封装
│   ├── scheduler/                # 任务调度器
│   │   └── scheduler.go         # Cron 调度 + 后台执行
│   └── websocket/                # WebSocket
│       └── hub.go               # WebSocket Hub（实时推送）
│
├── web/                          # 前端应用
│   ├── src/
│   │   ├── api/                  # API 客户端
│   │   │   ├── client.ts        # Axios 封装
│   │   │   └── websocket.ts     # WebSocket 客户端
│   │   ├── components/          # React 组件
│   │   │   └── Layout.tsx       # 主布局
│   │   ├── pages/                # 页面组件
│   │   │   ├── Dashboard.tsx    # 仪表盘
│   │   │   ├── Registries.tsx   # Registry 管理
│   │   │   ├── Tasks.tsx        # 任务管理
│   │   │   └── Executions.tsx   # 执行历史
│   │   ├── hooks/                # 自定义 Hooks
│   │   │   ├── useApi.ts        # API 调用 Hook
│   │   │   └── useWebSocket.ts  # WebSocket Hook
│   │   ├── types/                # TypeScript 类型
│   │   │   └── index.ts         # 类型定义
│   │   ├── App.tsx               # 根组件
│   │   ├── main.tsx              # 入口文件
│   │   └── index.css             # 全局样式
│   ├── index.html                # HTML 模板
│   ├── vite.config.ts            # Vite 配置
│   ├── tsconfig.json             # TypeScript 配置
│   ├── package.json              # 依赖配置
│   └── README.md                 # 前端文档
│
├── configs/                      # 配置文件
│   └── sync.example.yaml        # 配置示例
│
├── go.mod                        # Go 依赖
├── go.sum                        # Go 依赖锁定
├── Makefile                      # 构建脚本
├── README.md                     # 主文档
├── WEB_SERVER.md                 # API 文档
├── PROJECT_STRUCTURE.md          # 本文件
└── .gitignore                    # Git 忽略规则
```

## 🔧 构建产物

编译后生成的二进制文件：

```
registry-sync            # CLI 工具（9.2MB）
registry-sync-server     # Web 服务器（34MB）
registry-sync.db         # SQLite 数据库（运行时生成）
web/build/              # 前端构建产物（npm run build）
```

## 📊 代码统计

| 模块 | 文件数 | 代码行数（约）| 说明 |
|------|--------|--------------|------|
| CLI 工具 | 1 | 200 | 命令行入口 |
| Web 服务器 | 1 | 200 | HTTP 服务器入口 |
| 核心引擎 | 10 | 2000 | Registry API、同步逻辑、过滤器 |
| Web API | 7 | 1500 | RESTful API、数据库、调度器 |
| 前端 | 15 | 2500 | React 组件、Hooks、API 客户端 |
| **总计** | **34** | **~6400** | - |

## 🌟 核心模块说明

### 1. pkg/registry - Registry API V2 客户端

负责与 Docker Registry V2 API 交互：
- **认证**：支持 Basic Auth 和 Bearer Token
- **Manifest 操作**：获取、上传、检查 Manifest
- **Blob 操作**：分块上传、下载、检查存在
- **多架构支持**：解析 Manifest List

### 2. pkg/sync - 同步引擎

核心同步逻辑：
- **Worker Pool**：并发控制，可配置并发数
- **智能重试**：指数退避，区分可重试错误
- **增量同步**：检查 Blob 是否存在，跳过已有层
- **进度回调**：实时报告同步进度

### 3. internal/scheduler - 任务调度器

后台任务管理：
- **Cron 调度**：支持 Cron 表达式定时执行
- **任务队列**：管理正在运行的任务
- **执行记录**：保存历史和日志
- **WebSocket 推送**：实时进度通知

### 4. web - React 前端

现代 Web 界面：
- **仪表盘**：统计卡片、最近执行
- **CRUD 操作**：Registry、Task 管理
- **实时更新**：WebSocket 连接
- **响应式设计**：适配各种屏幕

## 🔄 数据流

### CLI 模式
```
配置文件 (YAML)
    ↓
Config 解析
    ↓
Sync Engine
    ↓
Registry API Client
    ↓
源/目标 Registry
```

### Web 模式
```
前端 (React)
    ↓
RESTful API (Gin)
    ↓
数据库 (SQLite)
    ↓
Scheduler (Cron)
    ↓
Sync Engine
    ↓
Registry API Client
    ↓
WebSocket ← 实时进度
    ↓
前端更新
```

## 📦 依赖关系

### Go 依赖
- `gopkg.in/yaml.v3` - YAML 解析
- `golang.org/x/time/rate` - Rate Limiting
- `github.com/gin-gonic/gin` - Web 框架
- `gorm.io/gorm` - ORM
- `github.com/robfig/cron/v3` - Cron 调度
- `github.com/gorilla/websocket` - WebSocket

### 前端依赖
- `react` / `react-dom` - React 框架
- `react-router-dom` - 路由
- `antd` - UI 组件库
- `axios` - HTTP 客户端
- `dayjs` - 日期处理
- `vite` - 构建工具

---

**项目总览完成！👍**
