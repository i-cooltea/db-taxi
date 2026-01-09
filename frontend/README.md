# DB-Taxi Frontend

基于 Vue 3 + Vite 的数据库同步管理前端应用。

## 技术栈

- **Vue 3**: 使用 Composition API
- **Vite**: 快速的开发服务器和构建工具
- **Vue Router**: 客户端路由
- **Pinia**: 状态管理
- **原生 CSS**: 无需额外的 UI 框架

## 项目结构

```
frontend/
├── src/
│   ├── components/          # 可复用组件
│   │   ├── ConfigCard.vue   # 配置卡片
│   │   ├── ConfigModal.vue  # 配置模态框
│   │   ├── TableItem.vue    # 表项组件
│   │   └── TableMappingModal.vue  # 表映射配置
│   ├── views/               # 页面组件
│   │   ├── Home.vue         # 首页
│   │   ├── Connections.vue  # 连接管理
│   │   ├── SyncConfig.vue   # 同步配置
│   │   └── Monitoring.vue   # 同步监控
│   ├── stores/              # Pinia stores
│   │   └── syncStore.js     # 同步配置状态管理
│   ├── router/              # 路由配置
│   │   └── index.js
│   ├── App.vue              # 根组件
│   ├── main.js              # 入口文件
│   └── style.css            # 全局样式
├── index.html               # HTML 模板
├── vite.config.js           # Vite 配置
└── package.json             # 依赖配置
```

## 开发

### 安装依赖

```bash
cd db-taxi/frontend
npm install
```

### 启动开发服务器

```bash
npm run dev
```

开发服务器将在 http://localhost:3000 启动。

API 请求会自动代理到 http://localhost:8080。

### 构建生产版本

```bash
npm run build
```

构建产物将输出到 `dist/` 目录。

### 预览生产构建

```bash
npm run preview
```

## 功能特性

### 同步配置管理

- ✅ 创建/编辑/删除同步配置
- ✅ 选择数据库连接
- ✅ 表选择和映射配置
- ✅ 自定义表名映射
- ✅ 配置同步模式（全量/增量）
- ✅ 高级选项配置（批处理、并发、冲突解决）
- ✅ 启动同步任务

### 表选择界面

- ✅ 浏览远程数据库表列表
- ✅ 多选表进行同步
- ✅ 为每个表配置同步模式
- ✅ 自定义目标表名称
- ✅ 配置 WHERE 过滤条件
- ✅ 全选/取消全选功能

### 用户体验

- ✅ 响应式设计
- ✅ 实时表单验证
- ✅ 加载状态指示
- ✅ 错误提示
- ✅ 模态框交互
- ✅ 标签页切换

## API 集成

前端通过 REST API 与后端通信：

- `GET /api/sync/connections` - 获取连接列表
- `GET /api/sync/configs` - 获取同步配置
- `POST /api/sync/configs` - 创建同步配置
- `PUT /api/sync/configs/:id` - 更新同步配置
- `DELETE /api/sync/configs/:id` - 删除同步配置
- `POST /api/sync/jobs` - 启动同步任务
- `GET /api/sync/connections/:id/tables` - 获取表列表

## 需求覆盖

此实现覆盖以下需求：

- **Requirement 3.1**: 浏览远程数据库并显示可用表列表
- **Requirement 3.2**: 选择表进行同步并保存配置
- **Requirement 3.3**: 配置表同步规则（全量/增量模式）
- **Requirement 3.4**: 自定义本地表名称
- **Requirement 3.5**: 启用/禁用表同步
- **Requirement 9.1**: 使用 Vite + Vue 3 框架提供响应式单页应用
- **Requirement 9.2**: 通过 REST API 与后端服务通信
- **Requirement 10.2**: 提供表选择、映射配置和同步模式设置界面

## 注意事项

1. 确保后端服务在 http://localhost:8080 运行
2. 开发时使用 Vite 的代理功能访问 API
3. 生产环境需要配置正确的 API 地址
4. 所有组件使用 Vue 3 Composition API
5. 状态管理使用 Pinia（Vue 3 推荐）
