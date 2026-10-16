# DB-Taxi

[English](README.md) | [中文](README.zh.md)

![image](https://raw.githubusercontent.com/i-cooltea/resource/refs/heads/master/image/db-taxi-logo.png)

一款非常方便且实用的MySQL数据克隆工具。

## 功能特性

- 🔌 **数据库连接管理** - 支持 MySQL 数据库连接，包括连接池管理
- 🔄 **数据库同步** - 支持数据库全量同步，指定表同步，按筛选条件同步
- 📦 **批量操作** - 支持批量数据传输和分批处理
- 🔍 **同步监控** - 实时监控同步任务状态和进度
- 🌐 **Web 界面** - 现代化的响应式 Web 界面（Vue 3 + Vite）
- ⚡ **高性能** - 基于 Go 语言和 Gin 框架，支持高并发
- 🔧 **配置灵活** - 支持配置文件和环境变量配置

## 快速开始

### 1. 配置数据库连接

有多种方式来配置数据库连接：

#### 方式1: 使用配置文件
```bash
# 复制配置文件模板
cp config.yaml.example config.yaml

# 编辑配置文件
vim config.yaml

./db-taxi -config /path/to/your/config.yaml
```

#### 方式2: 使用 Docker Compose
```bash
# 启动 DB-Taxi 及内置的 MySQL 实例
docker-compose up -d

# 查看日志
docker-compose logs -f db-taxi

# 停止服务
docker-compose down
```

Web 界面访问地址：http://localhost:8080，内置 MySQL 暴露在 13306 端口。

### 2. 构建和运行

```bash
# 安装依赖
go mod tidy

# 构建项目
go build -o db-taxi .

# 运行（使用默认配置）
./db-taxi

# 或使用命令行参数
./db-taxi -host localhost -user root -password secret -database mydb -server-port 9090
```

### 3. 访问 Web 界面

打开浏览器访问：http://localhost:8080

## 数据库迁移

DB-Taxi 包含自动数据库迁移系统，在应用启动时自动创建和更新所需的数据库表。

### 自动迁移（推荐）

应用启动时会自动运行迁移：

```bash
./db-taxi -host localhost -user root -password secret -database mydb
```

### 手动迁移

如需手动控制迁移，可使用以下命令：

```bash
# 运行所有待执行的迁移
make migrate HOST=localhost USER=root PASSWORD=secret DB=mydb

# 检查迁移状态
make migrate-status HOST=localhost USER=root DB=mydb

# 查看当前版本
make migrate-version HOST=localhost USER=root DB=mydb
```

或使用便捷脚本：

```bash
./scripts/migrate.sh -h localhost -u root -P secret -d mydb
```

详细文档请参考：
- [完整迁移文档](docs/MIGRATIONS.md)
- [快速入门指南](docs/MIGRATION_QUICK_START.md)

## 同步功能使用

DB-Taxi 提供强大的数据库同步功能，支持多数据库连接管理和选择性表同步。

### 快速开始同步

1. **添加远程连接**：在 Web 界面的"连接管理"页面添加远程数据库连接
2. **配置同步**：选择要同步的表，设置同步模式（全量/增量）
3. **启动同步**：点击"立即同步"开始数据同步
4. **监控进度**：在"同步监控"页面查看实时进度和日志

### 同步功能特性

- ✅ 多数据库实例连接管理
- ✅ 选择性表同步
- ✅ 全量和增量同步模式
- ✅ 实时进度监控
- ✅ 同步失败 支持查看错误信息
- ✅ 配置导入导出
- ✅ 批量操作和性能优化
- ✅ 定时同步计划

详细使用指南请参考：
- [同步功能使用指南](docs/SYNC_USER_GUIDE.md)
- [API 文档](docs/API.md)

## 命令行选项

```bash
db-taxi [options]

配置选项:
  -config string      指定配置文件路径
  -host string        数据库主机地址
  -port int           数据库端口
  -user string        数据库用户名
  -password string    数据库密码
  -database string    数据库名称
  -ssl                启用SSL连接
  -server-port int    Web服务器端口
  -help               显示帮助信息
```

## 使用示例

### 基本使用
```bash
# 使用默认配置文件
./db-taxi

# 显示帮助
./db-taxi -help
```

### 指定配置文件
```bash
# 使用自定义配置文件
./db-taxi -config /etc/db-taxi/production.yaml

# 使用预设的配置文件
./db-taxi -config configs/local.yaml      # 本地开发
./db-taxi -config configs/production.yaml # 生产环境
```

### 命令行参数覆盖
```bash
# 完全通过命令行指定
./db-taxi -host 192.168.1.100 -port 3306 -user admin -password secret123 -database myapp

# 使用配置文件，但覆盖部分参数
./db-taxi -config configs/local.yaml -password newsecret -server-port 9090

# 混合使用环境变量和命令行参数
export DBT_DATABASE_HOST=remote-mysql
./db-taxi -user admin -password secret -database production_db
```


### 生产环境部署
```bash
# 使用环境变量（推荐用于生产环境）
export DBT_DATABASE_HOST=mysql-server.internal
export DBT_DATABASE_USERNAME=app_user
export DBT_DATABASE_PASSWORD=secure_password
export DBT_DATABASE_DATABASE=production_db
export DBT_SERVER_PORT=8080
./db-taxi -config configs/production.yaml
```

## 环境变量配置

你也可以使用环境变量来配置应用：

```bash
export DBT_DATABASE_HOST=localhost
export DBT_DATABASE_PORT=3306
export DBT_DATABASE_USERNAME=root
export DBT_DATABASE_PASSWORD=your_password
export DBT_DATABASE_DATABASE=your_database
export DBT_SERVER_PORT=8080
```

## API 端点

### 健康检查
- `GET /health` - 服务器健康检查

### 数据库操作
- `GET /api/status` - 获取服务器和数据库状态
- `GET /api/connection/test` - 测试数据库连接
- `GET /api/databases` - 获取数据库列表
- `GET /api/databases/{database}/tables` - 获取指定数据库的表列表
- `GET /api/databases/{database}/tables/{table}` - 获取表的详细信息
- `GET /api/databases/{database}/tables/{table}/data` - 获取表数据（支持分页）

### 同步系统 API
- `GET /api/sync/status` - 获取同步系统状态
- `GET /api/sync/stats` - 获取同步系统统计信息

#### 连接管理
- `GET /api/sync/connections` - 获取所有同步连接
- `POST /api/sync/connections` - 创建新的同步连接
- `GET /api/sync/connections/{id}` - 获取指定连接详情
- `PUT /api/sync/connections/{id}` - 更新连接配置
- `DELETE /api/sync/connections/{id}` - 删除连接
- `POST /api/sync/connections/{id}/test` - 测试连接

#### 同步配置
- `GET /api/sync/configs` - 获取同步配置列表
- `POST /api/sync/configs` - 创建同步配置
- `GET /api/sync/configs/{id}` - 获取配置详情
- `PUT /api/sync/configs/{id}` - 更新配置
- `DELETE /api/sync/configs/{id}` - 删除配置

#### 任务管理
- `GET /api/sync/jobs` - 获取同步任务列表
- `POST /api/sync/jobs` - 启动新的同步任务
- `GET /api/sync/jobs/{id}` - 获取任务详情
- `POST /api/sync/jobs/{id}/stop` - 停止任务
- `GET /api/sync/jobs/{id}/logs` - 获取任务日志

#### 配置管理
- `GET /api/sync/config/export` - 导出同步配置
- `POST /api/sync/config/import` - 导入同步配置
- `POST /api/sync/config/validate` - 验证配置文件

### 查询参数
- `limit` - 限制返回的记录数（默认：10，最大：1000）
- `offset` - 偏移量（默认：0）

## 项目结构

```
db-taxi/
├── main.go                    # 应用程序入口
├── config.yaml.example       # 配置文件模板
├── static/                    # 静态文件
│   └── index.html            # Web 界面
├── frontend/                  # Vue 3 前端应用
│   ├── src/
│   │   ├── components/       # Vue 组件
│   │   ├── views/            # 页面视图
│   │   ├── stores/           # 状态管理
│   │   └── router/           # 路由配置
│   └── package.json
├── internal/
│   ├── config/               # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   ├── server/               # HTTP 服务器
│   │   ├── server.go
│   │   ├── middleware.go
│   │   └── server_test.go
│   ├── database/             # 数据库操作
│   │   ├── connection.go     # 连接池管理
│   │   ├── schema.go         # 数据库结构探索
│   │   └── connection_test.go
│   ├── sync/                 # 同步系统
│   │   ├── sync.go           # 同步管理器
│   │   ├── interfaces.go     # 接口定义
│   │   ├── repository.go     # 数据访问层
│   │   ├── service.go        # 业务逻辑层
│   │   ├── job_engine.go     # 任务引擎
│   │   ├── sync_engine.go    # 同步引擎
│   │   └── mapping_manager.go # 映射管理器
│   ├── migration/            # 数据库迁移
│   │   ├── migration.go
│   │   └── sql/              # SQL 迁移文件
│   └── integration_test.go   # 集成测试
├── docs/                     # 文档
│   ├── SYSTEM_INTEGRATION.md # 系统集成文档
│   ├── MIGRATIONS.md         # 迁移文档
│   └── MIGRATION_QUICK_START.md
├── scripts/                  # 脚本
│   ├── migrate.sh            # 迁移脚本
│   └── verify-integration.sh # 集成验证脚本
└── go.mod                    # Go 模块定义
```

## 配置选项

### 服务器配置
- `server.port` - 服务器端口（默认：8080）
- `server.host` - 服务器主机（默认：0.0.0.0）
- `server.read_timeout` - 读取超时时间
- `server.write_timeout` - 写入超时时间

### 数据库配置
- `database.host` - MySQL 主机地址
- `database.port` - MySQL 端口
- `database.username` - 用户名
- `database.password` - 密码
- `database.database` - 数据库名
- `database.ssl` - 是否启用 SSL
- `database.max_open_conns` - 最大连接数
- `database.max_idle_conns` - 最大空闲连接数
- `database.conn_max_lifetime` - 连接最大生存时间

### 安全配置
- `security.session_timeout` - 会话超时时间
- `security.read_only_mode` - 只读模式
- `security.enable_audit` - 启用审计日志

### 日志配置
- `logging.level` - 日志级别（debug, info, warn, error）
- `logging.format` - 日志格式（json, text）
- `logging.output` - 日志输出（stdout, stderr, 文件路径）

### 同步系统配置
- `sync.enabled` - 是否启用同步系统（默认：true）
- `sync.max_concurrency` - 最大并发同步任务数（默认：5）
- `sync.batch_size` - 批量操作大小（默认：1000）
- `sync.retry_attempts` - 重试次数（默认：3）
- `sync.retry_delay` - 重试延迟时间（默认：30s）
- `sync.job_timeout` - 任务超时时间（默认：1h）
- `sync.cleanup_age` - 历史记录清理时间（默认：720h ）

## 开发

### 运行测试
```bash
# 运行所有测试
go test ./...

# 运行单元测试（跳过集成测试）
go test ./... -short

# 运行集成测试
go test ./internal/integration_test.go -v

# 运行特定包的测试
go test ./internal/sync/... -v
```

### 验证系统集成
```bash
# 运行集成验证脚本
./scripts/verify-integration.sh
```

### 前端开发
```bash
# 进入前端目录
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 构建前端代码到static文件夹（在项目根目录执行）
make build-frontend
```

### Docker 部署
```bash
# 使用 Docker Compose（包含 MySQL）
docker-compose up -d

# 或者单独构建和运行
docker build -t db-taxi .
docker run -p 8080:8080 \
  -e DBT_DATABASE_HOST=your-mysql-host \
  -e DBT_DATABASE_USERNAME=root \
  -e DBT_DATABASE_PASSWORD=secret \
  -e DBT_DATABASE_DATABASE=mydb \
  db-taxi
```

### 快速启动脚本
```bash
# 本地开发
chmod +x scripts/start-local.sh
./scripts/start-local.sh

# 生产环境
export DB_PASSWORD=your_production_password
chmod +x scripts/start-production.sh
./scripts/start-production.sh
```

## 技术栈

- **后端**: Go 1.21+, Gin Web Framework
- **数据库**: MySQL 5.7+
- **前端**: Vue 3, Vite, Vue Router, Pinia
- **依赖管理**: Go Modules, npm

## 依赖项

- `github.com/gin-gonic/gin` - Web 框架
- `github.com/jmoiron/sqlx` - SQL 扩展库
- `github.com/go-sql-driver/mysql` - MySQL 驱动
- `github.com/sirupsen/logrus` - 日志库
- `github.com/spf13/viper` - 配置管理

## 实现状态

基于规范文档中的实施计划，当前实现包括：

### 核心功能
- ✅ 项目初始化和基础架构设置
- ✅ 数据库连接池管理器
- ✅ 数据库元数据探索器（Schema Explorer）
- ✅ Web 界面和用户体验（Vue 3 + Vite）
- ✅ REST API 接口实现

### 同步系统
- ✅ 连接管理器（Connection Manager）
- ✅ 同步管理器（Sync Manager）
- ✅ 映射管理器（Mapping Manager）
- ✅ 任务引擎（Job Engine）
- ✅ 同步引擎（Sync Engine）
- ✅ 批量处理和性能优化
- ✅ 错误处理和恢复机制
- ✅ 配置导入导出
- ✅ 实时监控和统计

### 系统集成
- ✅ 所有组件依赖注入
- ✅ 系统启动和关闭逻辑
- ✅ 数据库迁移系统
- ✅ 健康检查和监控
- ✅ 集成测试

### 待实现功能
- ⏳ 会话管理系统
- ⏳ SQL 查询引擎
- ⏳ 数据导出功能

详细的系统集成文档请参考：[SYSTEM_INTEGRATION.md](docs/SYSTEM_INTEGRATION.md)

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 支持

如果你遇到任何问题，请查看：
1. 确保 MySQL 服务正在运行
2. 检查配置文件中的数据库连接信息
3. 查看应用程序日志获取详细错误信息
4. 运行集成验证脚本：`./scripts/verify-integration.sh`
5. 查看系统集成文档：[SYSTEM_INTEGRATION.md](docs/SYSTEM_INTEGRATION.md)
