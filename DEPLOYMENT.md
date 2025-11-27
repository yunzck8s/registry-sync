# Registry Sync 部署指南

本文档提供了 Registry Sync 的详细部署说明，支持 Docker Compose 和 Kubernetes 两种部署方式。

## 📦 镜像信息

- **Docker Hub**: `zunshen/registry-sync:latest`
- **支持架构**: `linux/amd64`, `linux/arm64`

## 🚀 快速开始

### 方式一：Docker Compose（推荐用于单机部署）

#### 1. 下载 docker-compose.yml

```bash
curl -O https://raw.githubusercontent.com/YOUR_USERNAME/registry-sync/main/docker-compose.yml
```

#### 2. 创建数据目录

```bash
mkdir -p data
```

#### 3. 启动服务

```bash
docker-compose up -d
```

#### 4. 访问应用

打开浏览器访问：http://localhost:8080

#### 5. 查看日志

```bash
docker-compose logs -f
```

#### 6. 停止服务

```bash
docker-compose down
```

### 方式二：Kubernetes（推荐用于生产环境）

#### 1. 下载 Kubernetes 配置文件

```bash
git clone https://github.com/YOUR_USERNAME/registry-sync.git
cd registry-sync/k8s
```

#### 2. 部署到 Kubernetes

使用 kubectl 直接部署：

```bash
kubectl apply -f namespace.yaml
kubectl apply -f pvc.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

或使用 kustomize 一键部署：

```bash
kubectl apply -k .
```

#### 3. 检查部署状态

```bash
kubectl get pods -n registry-sync
kubectl get svc -n registry-sync
```

#### 4. 访问应用

##### 方式 A：端口转发（用于测试）

```bash
kubectl port-forward -n registry-sync svc/registry-sync 8080:8080
```

然后访问：http://localhost:8080

##### 方式 B：使用 Ingress（推荐用于生产）

1. 编辑 `k8s/ingress.yaml`，修改域名：

```yaml
spec:
  rules:
  - host: registry-sync.yourdomain.com  # 修改为你的域名
```

2. 部署 Ingress：

```bash
kubectl apply -f ingress.yaml
```

3. 配置 DNS 解析到 Ingress 控制器的 IP

4. 访问：http://registry-sync.yourdomain.com

#### 5. 查看日志

```bash
kubectl logs -n registry-sync -l app=registry-sync -f
```

#### 6. 卸载

```bash
kubectl delete -k k8s/
# 或
kubectl delete namespace registry-sync
```

## 🔧 配置说明

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `TZ` | 时区 | `Asia/Shanghai` |
| `GIN_MODE` | Gin 模式 | `release` |

### 数据持久化

- **Docker Compose**: 数据存储在 `./data` 目录
- **Kubernetes**: 使用 PVC，默认申请 10Gi 存储

### 资源配置（Kubernetes）

默认资源配置：

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

根据实际负载调整 `k8s/deployment.yaml` 中的资源配置。

## 🛠 GitHub Actions CI/CD 设置

### 1. 配置 Secrets

在 GitHub 仓库设置中添加以下 Secrets：

- `DOCKERHUB_USERNAME`: Docker Hub 用户名
- `DOCKERHUB_TOKEN`: Docker Hub 访问令牌

获取 Docker Hub Token：
1. 登录 Docker Hub
2. Account Settings → Security → New Access Token
3. 复制生成的 token

### 2. 触发构建

CI/CD 会在以下情况自动触发：

- **推送到 main/master 分支**: 构建并推送 `latest` 标签
- **创建 tag（如 v1.0.0）**: 构建并推送版本标签
- **PR 请求**: 仅构建，不推送

### 3. 手动触发构建

```bash
# 创建并推送 tag
git tag v1.0.0
git push origin v1.0.0
```

## 📊 健康检查

- **端点**: `/api/v1/health`
- **Docker Compose**: 内置健康检查，30秒间隔
- **Kubernetes**: 配置了 liveness 和 readiness 探针

## 🔐 安全建议

1. **生产环境建议**：
   - 使用 Ingress + TLS 证书
   - 配置网络策略限制访问
   - 定期备份数据库文件

2. **密码管理**：
   - Registry 密码加密存储在数据库中
   - 定期更换 Registry 凭证

3. **资源限制**：
   - 设置合理的 CPU 和内存限制
   - 配置存储配额

## 🐛 故障排查

### Docker Compose

```bash
# 查看容器状态
docker-compose ps

# 查看详细日志
docker-compose logs -f registry-sync

# 重启服务
docker-compose restart

# 完全重建
docker-compose down && docker-compose up -d --build
```

### Kubernetes

```bash
# 查看 Pod 状态
kubectl get pods -n registry-sync

# 查看 Pod 详情
kubectl describe pod -n registry-sync <pod-name>

# 查看日志
kubectl logs -n registry-sync <pod-name> -f

# 进入容器
kubectl exec -it -n registry-sync <pod-name> -- sh

# 检查 PVC
kubectl get pvc -n registry-sync
```

### 常见问题

#### 1. 数据库文件权限错误

**Docker Compose**:
```bash
sudo chown -R 1000:1000 ./data
```

**Kubernetes**:
检查 PVC 的访问模式和存储类配置

#### 2. 前端无法访问后端 API

检查：
- 服务是否正常启动：`/api/v1/health`
- 网络配置是否正确
- 防火墙规则

#### 3. 镜像拉取失败

```bash
# Docker Compose
docker pull zunshen/registry-sync:latest

# Kubernetes
kubectl describe pod -n registry-sync <pod-name>
# 检查 imagePullPolicy 和镜像名称
```

## 📈 监控和日志

### 日志位置

- **容器内**: 标准输出（stdout）
- **采集建议**:
  - Docker Compose: 使用 `docker logs` 或日志采集工具
  - Kubernetes: 使用 EFK/ELK 或 Loki 采集日志

### 监控指标

建议监控：
- Pod/容器状态
- CPU 和内存使用率
- 存储空间使用
- API 响应时间
- 同步任务成功率

## 🔄 升级

### Docker Compose

```bash
# 拉取最新镜像
docker-compose pull

# 重启服务
docker-compose up -d
```

### Kubernetes

```bash
# 方式一：更新镜像
kubectl set image deployment/registry-sync \
  registry-sync=zunshen/registry-sync:v1.1.0 \
  -n registry-sync

# 方式二：重新应用配置
kubectl apply -k k8s/

# 查看滚动更新状态
kubectl rollout status deployment/registry-sync -n registry-sync
```

## 📞 支持

如有问题，请提交 Issue：https://github.com/YOUR_USERNAME/registry-sync/issues
