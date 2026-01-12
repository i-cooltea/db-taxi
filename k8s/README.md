# Kubernetes 部署指南

## 概述

本目录包含在 Kubernetes 集群上部署 DB-Taxi 的配置文件。

## 前置要求

- Kubernetes 集群（1.19+）
- kubectl 命令行工具
- 访问 Docker 镜像仓库的权限
- （可选）Helm 3.0+

## 快速开始

### 1. 构建和推送 Docker 镜像

```bash
# 构建镜像
docker build -t your-registry/db-taxi:latest .

# 推送到镜像仓库
docker push your-registry/db-taxi:latest
```

### 2. 更新配置

编辑 `deployment.yaml` 文件，更新以下内容：

- 镜像地址：`image: your-registry/db-taxi:latest`
- 数据库密码：在 Secret 中设置
- Ingress 域名：`host: db-taxi.example.com`
- 其他配置参数

### 3. 部署到 Kubernetes

```bash
# 应用所有配置
kubectl apply -f deployment.yaml

# 或分步部署
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
kubectl apply -f hpa.yaml
```

### 4. 验证部署

```bash
# 检查 Pod 状态
kubectl get pods -n db-taxi

# 检查服务状态
kubectl get svc -n db-taxi

# 检查 Ingress
kubectl get ingress -n db-taxi

# 查看日志
kubectl logs -f deployment/db-taxi -n db-taxi
```

## 配置说明

### ConfigMap

ConfigMap 包含应用的配置文件。主要配置项：

- 服务器配置（端口、超时等）
- 数据库连接配置
- 日志配置
- 同步系统配置

### Secret

Secret 存储敏感信息：

- 数据库密码
- SSL 证书（如需要）
- API 密钥（如需要）

创建 Secret：

```bash
kubectl create secret generic db-taxi-secrets \
  --from-literal=db-password='your-secure-password' \
  -n db-taxi
```

### Deployment

Deployment 定义应用的部署规格：

- 副本数：默认 2 个
- 资源限制：CPU 和内存
- 健康检查：liveness 和 readiness 探针
- 环境变量和配置挂载

### Service

Service 暴露应用：

- 类型：ClusterIP（集群内部访问）
- 端口：80 映射到容器的 8080

### Ingress

Ingress 提供外部访问：

- 域名路由
- SSL/TLS 终止
- 负载均衡

### HorizontalPodAutoscaler

HPA 实现自动扩缩容：

- 最小副本数：2
- 最大副本数：10
- 扩缩容指标：CPU 和内存使用率

## 高级配置

### 使用外部数据库

如果使用外部 MySQL 数据库，更新 ConfigMap 中的数据库配置：

```yaml
database:
  host: "external-mysql.example.com"
  port: 3306
  username: "db_user"
  database: "production_db"
  ssl: true
```

### 配置持久化存储

如需持久化日志或数据：

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: db-taxi-logs
  namespace: db-taxi
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

在 Deployment 中挂载：

```yaml
volumeMounts:
- name: logs
  mountPath: /app/logs

volumes:
- name: logs
  persistentVolumeClaim:
    claimName: db-taxi-logs
```

### 配置资源限制

根据实际负载调整资源：

```yaml
resources:
  requests:
    memory: "1Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi"
    cpu: "4000m"
```

### 配置多环境

创建不同环境的配置文件：

- `deployment-dev.yaml` - 开发环境
- `deployment-staging.yaml` - 测试环境
- `deployment-prod.yaml` - 生产环境

## 监控和日志

### 查看日志

```bash
# 查看所有 Pod 日志
kubectl logs -l app=db-taxi -n db-taxi

# 查看特定 Pod 日志
kubectl logs <pod-name> -n db-taxi

# 实时跟踪日志
kubectl logs -f deployment/db-taxi -n db-taxi

# 查看之前的日志
kubectl logs <pod-name> -n db-taxi --previous
```

### 监控指标

如果集成了 Prometheus：

```bash
# 查看 HPA 指标
kubectl get hpa -n db-taxi

# 查看 Pod 资源使用
kubectl top pods -n db-taxi

# 查看节点资源使用
kubectl top nodes
```

### 事件查看

```bash
# 查看命名空间事件
kubectl get events -n db-taxi

# 查看特定资源事件
kubectl describe pod <pod-name> -n db-taxi
```

## 更新和回滚

### 更新应用

```bash
# 更新镜像
kubectl set image deployment/db-taxi \
  db-taxi=your-registry/db-taxi:v2.0.0 \
  -n db-taxi

# 或重新应用配置
kubectl apply -f deployment.yaml

# 查看更新状态
kubectl rollout status deployment/db-taxi -n db-taxi
```

### 回滚

```bash
# 查看历史版本
kubectl rollout history deployment/db-taxi -n db-taxi

# 回滚到上一个版本
kubectl rollout undo deployment/db-taxi -n db-taxi

# 回滚到特定版本
kubectl rollout undo deployment/db-taxi \
  --to-revision=2 \
  -n db-taxi
```

## 扩缩容

### 手动扩缩容

```bash
# 扩容到 5 个副本
kubectl scale deployment/db-taxi --replicas=5 -n db-taxi

# 查看副本状态
kubectl get deployment db-taxi -n db-taxi
```

### 自动扩缩容

HPA 会根据 CPU 和内存使用率自动调整副本数。

查看 HPA 状态：

```bash
kubectl get hpa db-taxi-hpa -n db-taxi
kubectl describe hpa db-taxi-hpa -n db-taxi
```

## 故障排除

### Pod 无法启动

```bash
# 查看 Pod 状态
kubectl get pods -n db-taxi

# 查看 Pod 详情
kubectl describe pod <pod-name> -n db-taxi

# 查看日志
kubectl logs <pod-name> -n db-taxi
```

常见问题：
- 镜像拉取失败：检查镜像地址和权限
- 配置错误：检查 ConfigMap 和 Secret
- 资源不足：检查节点资源

### 服务无法访问

```bash
# 检查 Service
kubectl get svc -n db-taxi
kubectl describe svc db-taxi-service -n db-taxi

# 检查 Endpoints
kubectl get endpoints -n db-taxi

# 检查 Ingress
kubectl get ingress -n db-taxi
kubectl describe ingress db-taxi-ingress -n db-taxi
```

### 数据库连接失败

1. 检查数据库配置
2. 验证网络连接
3. 检查 Secret 中的密码
4. 查看应用日志

## 清理

删除所有资源：

```bash
# 删除整个命名空间
kubectl delete namespace db-taxi

# 或单独删除资源
kubectl delete -f deployment.yaml
```

## 最佳实践

1. **使用命名空间隔离环境**
2. **配置资源限制和请求**
3. **设置健康检查探针**
4. **使用 Secret 管理敏感信息**
5. **配置 HPA 实现自动扩缩容**
6. **使用 Ingress 管理外部访问**
7. **配置日志收集和监控**
8. **定期备份配置和数据**
9. **使用滚动更新策略**
10. **配置资源配额和限制**

## 参考资源

- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [DB-Taxi 部署指南](../docs/DEPLOYMENT.md)
- [DB-Taxi API 文档](../docs/API.md)
