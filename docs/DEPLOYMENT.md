# DB-Taxi 部署指南

## 概述

本文档提供 DB-Taxi 的完整部署指南，包括开发环境、测试环境和生产环境的部署方法。

## 部署方式

DB-Taxi 支持多种部署方式：

1. **直接运行** - 适合开发和测试
2. **Docker 部署** - 适合快速部署和隔离环境
3. **Docker Compose** - 适合完整的应用栈部署
4. **生产环境部署** - 适合生产环境的高可用部署

## 前置要求

### 基础要求

- **操作系统**: Linux, macOS, 或 Windows
- **Go**: 1.21 或更高版本（直接运行时需要）
- **MySQL**: 5.7 或更高版本
- **Docker**: 20.10 或更高版本（Docker 部署时需要）
- **Docker Compose**: 1.29 或更高版本（Docker Compose 部署时需要）

### 网络要求

- 应用端口：8080（可配置）
- MySQL 端口：3306（可配置）
- 确保防火墙允许相应端口的访问

### 资源要求

#### 最小配置
- CPU: 1 核
- 内存: 512 MB
- 磁盘: 1 GB

#### 推荐配置
- CPU: 2 核或更多
- 内存: 2 GB 或更多
- 磁盘: 10 GB 或更多（取决于同步的数据量）

#### 生产环境配置
- CPU: 4 核或更多
- 内存: 4 GB 或更多
- 磁盘: 50 GB 或更多（SSD 推荐）

## 方式 1: 直接运行

### 1.1 构建应用

```bash
# 克隆代码库
git clone https://github.com/your-repo/db-taxi.git
cd db-taxi

# 安装依赖
go mod tidy

# 构建应用
go build -o db-taxi .
```

### 1.2 配置数据库

确保 MySQL 服务正在运行，并创建数据库：

```sql
CREATE DATABASE myapp CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'dbuser'@'%' IDENTIFIED BY 'secure_password';
GRANT ALL PRIVILEGES ON myapp.* TO 'dbuser'@'%';
FLUSH PRIVILEGES;
```

### 1.3 配置应用

创建配置文件：

```bash
cp configs/config.yaml.example config.yaml
vim config.yaml
```

或使用环境变量：

```bash
export DBT_DATABASE_HOST=localhost
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=dbuser
export DBT_DATABASE_PASSWORD=secure_password
export DBT_DATABASE_DATABASE=myapp
```

### 1.4 运行应用

```bash
# 使用配置文件
./db-taxi -config config.yaml

# 或使用命令行参数
./db-taxi -host localhost -user dbuser -password secure_password -database myapp

# 或使用环境变量
./db-taxi
```

### 1.5 验证部署

```bash
# 检查健康状态
curl http://localhost:8080/health

# 访问 Web 界面
open http://localhost:8080
```

## 方式 2: Docker 部署

### 2.1 准备环境

创建 `.env` 文件：

```bash
cp .env.example .env
vim .env
```

配置必要的环境变量：

```bash
DB_HOST=your-mysql-host
DB_PORT=3306
DB_USERNAME=dbuser
DB_PASSWORD=secure_password
DB_DATABASE=myapp
```

### 2.2 使用部署脚本

```bash
# 运行部署脚本
./scripts/deploy-docker.sh

# 或手动部署
docker build -t db-taxi:latest .
docker-compose up -d
```

### 2.3 查看状态

```bash
# 查看容器状态
docker-compose ps

# 查看日志
docker-compose logs -f db-taxi

# 检查健康状态
curl http://localhost:8080/health
```

### 2.4 停止和清理

```bash
# 停止容器
docker-compose down

# 停止并删除数据卷
docker-compose down -v
```

## 方式 3: 生产环境部署

### 3.1 准备生产配置

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑生产配置
vim .env
```

重要配置项：

```bash
# 数据库配置
DB_HOST=production-mysql.example.com
DB_PORT=3306
DB_USERNAME=prod_user
DB_PASSWORD=strong_secure_password
DB_DATABASE=production_db
DB_SSL=true

# 日志配置
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=/app/logs/db-taxi.log

# 安全配置
ENABLE_AUDIT=true
SESSION_TIMEOUT=30m

# 同步系统配置
SYNC_ENABLED=true
SYNC_MAX_CONCURRENCY=10
SYNC_BATCH_SIZE=2000
```

### 3.2 运行生产部署脚本

```bash
# 设置镜像标签（可选）
export IMAGE_TAG=v1.0.0

# 运行部署脚本
./scripts/deploy-production.sh
```

部署脚本会自动执行：
1. 检查系统要求
2. 验证环境配置
3. 备份当前配置
4. 构建 Docker 镜像
5. 运行数据库迁移
6. 部署应用
7. 健康检查
8. 清理旧镜像

### 3.3 配置反向代理（推荐）

#### 使用 Nginx

创建 Nginx 配置文件 `/etc/nginx/sites-available/db-taxi`:

```nginx
upstream db-taxi {
    server localhost:8080;
}

server {
    listen 80;
    server_name db-taxi.example.com;
    
    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name db-taxi.example.com;
    
    # SSL 证书配置
    ssl_certificate /etc/ssl/certs/db-taxi.crt;
    ssl_certificate_key /etc/ssl/private/db-taxi.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    # 日志配置
    access_log /var/log/nginx/db-taxi-access.log;
    error_log /var/log/nginx/db-taxi-error.log;
    
    # 代理配置
    location / {
        proxy_pass http://db-taxi;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # 超时配置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
        proxy_pass http://db-taxi;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

启用配置：

```bash
sudo ln -s /etc/nginx/sites-available/db-taxi /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

#### 使用 Traefik

创建 `docker-compose.traefik.yml`:

```yaml
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
    ports:
      - "80:80"
      - "443:443"
      - "8081:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - db-taxi-network

  db-taxi:
    image: db-taxi:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.db-taxi.rule=Host(`db-taxi.example.com`)"
      - "traefik.http.routers.db-taxi.entrypoints=websecure"
      - "traefik.http.routers.db-taxi.tls=true"
    networks:
      - db-taxi-network

networks:
  db-taxi-network:
    driver: bridge
```

### 3.4 配置系统服务（Systemd）

创建服务文件 `/etc/systemd/system/db-taxi.service`:

```ini
[Unit]
Description=DB-Taxi MySQL Web Explorer
After=network.target mysql.service
Wants=mysql.service

[Service]
Type=simple
User=dbtaxi
Group=dbtaxi
WorkingDirectory=/opt/db-taxi
EnvironmentFile=/opt/db-taxi/.env
ExecStart=/opt/db-taxi/db-taxi -config /opt/db-taxi/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=db-taxi

# 安全配置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/db-taxi/logs

[Install]
WantedBy=multi-user.target
```

启用和启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable db-taxi
sudo systemctl start db-taxi
sudo systemctl status db-taxi
```

## 监控和维护

### 健康检查

```bash
# HTTP 健康检查
curl http://localhost:8080/health

# 详细状态检查
curl http://localhost:8080/api/status

# 同步系统状态
curl http://localhost:8080/api/sync/status
```

### 日志管理

```bash
# Docker 日志
docker-compose logs -f db-taxi

# Systemd 日志
sudo journalctl -u db-taxi -f

# 应用日志文件
tail -f /opt/db-taxi/logs/db-taxi.log
```

### 备份

#### 备份配置

```bash
# 备份配置文件
tar -czf config-backup-$(date +%Y%m%d).tar.gz .env configs/

# 备份同步配置
curl http://localhost:8080/api/sync/config/export > sync-config-backup-$(date +%Y%m%d).json
```

#### 备份数据库

```bash
# 备份 MySQL 数据库
mysqldump -h localhost -u root -p myapp > myapp-backup-$(date +%Y%m%d).sql

# 或使用 Docker
docker exec mysql mysqldump -u root -p myapp > myapp-backup-$(date +%Y%m%d).sql
```

### 更新和升级

```bash
# 拉取最新代码
git pull origin main

# 重新构建
go build -o db-taxi .

# 或使用 Docker
docker build -t db-taxi:latest .

# 重启服务
sudo systemctl restart db-taxi

# 或使用 Docker Compose
docker-compose restart db-taxi
```

### 性能调优

#### 数据库连接池

```yaml
database:
  max_open_conns: 50      # 增加最大连接数
  max_idle_conns: 10      # 增加空闲连接数
  conn_max_lifetime: "10m" # 调整连接生命周期
```

#### 同步系统

```yaml
sync:
  max_concurrency: 10     # 增加并发数
  batch_size: 2000        # 增加批量大小
  enable_compression: true # 启用压缩
```

#### 系统资源

```yaml
# Docker Compose 资源限制
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 4G
    reservations:
      cpus: '2'
      memory: 2G
```

## 故障排除

### 应用无法启动

1. 检查配置文件是否正确
2. 验证数据库连接
3. 检查端口是否被占用
4. 查看应用日志

### 数据库连接失败

1. 验证数据库服务是否运行
2. 检查网络连接
3. 验证用户名和密码
4. 检查防火墙规则

### 同步任务失败

1. 查看任务日志
2. 检查远程数据库连接
3. 验证表结构兼容性
4. 检查磁盘空间

### 性能问题

1. 监控系统资源使用
2. 调整并发数和批量大小
3. 优化数据库索引
4. 启用数据压缩

## 安全建议

### 1. 网络安全

- 使用防火墙限制访问
- 启用 SSL/TLS 加密
- 使用 VPN 或专用网络
- 配置反向代理

### 2. 认证和授权

- 使用强密码
- 定期更换密码
- 限制数据库用户权限
- 启用审计日志

### 3. 数据安全

- 定期备份数据
- 加密敏感数据
- 使用只读账号进行同步
- 监控异常活动

### 4. 应用安全

- 保持应用更新
- 使用最新的依赖版本
- 配置适当的超时时间
- 限制资源使用

## 高可用部署

### 负载均衡

使用多个 DB-Taxi 实例和负载均衡器：

```yaml
# docker-compose.ha.yml
version: '3.8'

services:
  db-taxi-1:
    image: db-taxi:latest
    # ... 配置 ...
  
  db-taxi-2:
    image: db-taxi:latest
    # ... 配置 ...
  
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - db-taxi-1
      - db-taxi-2
```

### 数据库高可用

- 使用 MySQL 主从复制
- 配置自动故障转移
- 使用数据库集群（如 Galera）

## 监控和告警

### Prometheus 集成

添加 Prometheus 指标端点（如果实现）：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'db-taxi'
    static_configs:
      - targets: ['localhost:8080']
```

### 告警规则

配置告警规则监控关键指标：
- 应用健康状态
- 同步任务失败率
- 数据库连接状态
- 系统资源使用

## 参考资源

- [系统集成文档](SYSTEM_INTEGRATION.md)
- [API 文档](API.md)
- [同步功能使用指南](SYNC_USER_GUIDE.md)
- [数据库迁移文档](MIGRATIONS.md)

## 支持

如需帮助，请：
1. 查看相关文档
2. 检查应用日志
3. 在 GitHub 上提交 Issue
4. 联系技术支持团队
