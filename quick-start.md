# DB-Taxi 快速入门

一个便捷实用的 MySQL 数据克隆工具，基于 Go + Vue 3 构建。

## 功能特性

- **数据库连接管理** - MySQL 连接池管理
- **数据库同步** - 全量同步、表级同步、条件同步
- **批量操作** - 批量数据传输和处理
- **同步监控** - 实时任务状态和进度监控
- **Web 界面** - 现代化响应式 UI（Vue 3 + Vite）
- **高性能** - 基于 Go 和 Gin 框架，支持高并发
- **灵活配置** - 支持配置文件、命令行参数、环境变量

## 环境要求

- Go 1.21+
- MySQL 5.7+
- Node.js（前端开发）

## 安装

```bash
# 克隆仓库
git clone <repository-url>
cd db-taxi

# 安装 Go 依赖
go mod tidy

# 构建项目
go build -o db-taxi .
```

## 快速开始

### 1. 配置数据库连接

**方式一：配置文件**
```bash
cp config.yaml.example config.yaml
vim config.yaml
./db-taxi -config /path/to/config.yaml
```

**方式二：Docker Compose**
```bash
# 启动 DB-Taxi 及内置的 MySQL 实例
docker-compose up -d
```

### 2. 访问 Web 界面

启动服务后，打开浏览器：

```bash
./db-taxi -host localhost -user root -password secret -database mydb
```

访问：**http://localhost:8080**

## 同步功能使用

1. **添加远程连接** - 在"连接管理"页面添加远程数据库
2. **配置同步** - 选择要同步的表，设置同步模式（全量/增量）
3. **开始同步** - 点击"开始同步"按钮
4. **监控进度** - 在"同步监控"页面查看实时进度

## Docker 部署

```bash
# 使用 Docker Compose
docker-compose up -d

# 或者手动构建运行
docker build -t db-taxi .
docker run -p 8080:8080 \
  -e DBT_DATABASE_HOST=your-mysql-host \
  -e DBT_DATABASE_USERNAME=root \
  -e DBT_DATABASE_PASSWORD=secret \
  -e DBT_DATABASE_DATABASE=mydb \
  db-taxi
```

## 命令行选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-config` | 配置文件路径 | - |
| `-host` | 数据库主机 | localhost |
| `-port` | 数据库端口 | 3306 |
| `-user` | 数据库用户名 | - |
| `-password` | 数据库密码 | - |
| `-database` | 数据库名称 | - |
| `-ssl` | 启用 SSL 连接 | false |
| `-server-port` | Web 服务端口 | 8080 |
| `-help` | 显示帮助信息 | - |

## 技术栈

- **后端**：Go 1.21+，Gin Web 框架
- **数据库**：MySQL 5.7+
- **前端**：Vue 3，Vite，Vue Router，Pinia

## 项目结构

```
db-taxi/
├── main.go              # 程序入口
├── config.yaml.example  # 配置模板
├── static/              # 静态文件
├── frontend/            # Vue 3 前端
├── internal/            # Go 包
│   ├── config/          # 配置管理
│   ├── server/          # HTTP 服务
│   ├── database/        # 数据库操作
│   ├── sync/            # 同步系统
│   └── migration/       # 数据迁移
├── docs/                # 文档
└── scripts/             # 脚本
```

## 常用命令

```bash
# 运行所有测试
go test ./...

# 运行集成验证
./scripts/verify-integration.sh

# 构建前端
cd frontend && npm install && npm run build
```

更多详情请参阅 [README.md](README.md)。
