## Why

DB-Taxi 当前只支持表数据的同步（全量/增量），但缺少同步前的差异对比能力。用户在执行同步前无法直观了解源表和目标表之间在结构（schema）和数据（data）层面的差异，容易导致盲目同步或数据覆盖。新增表对比功能，让用户在同步前就能看清两张表的差异，做出更安全的同步决策。

## What Changes

- 新增表对比 API 端点，支持对任意两张表进行结构和数据差异对比
- 新增前端"表对比"页面，提供可视化差异展示（结构差异 + 数据差异）
- 对比结果包括：列差异（新增/删除/类型变更）、行差异（仅存在于源/目标的行、值不同的行）
- 支持指定连接和数据库选择对比的两张表
- 对比结果可导出

## Capabilities

### New Capabilities
- `table-diff`: 表对比核心能力，包括结构差异检测、数据差异检测、差异结果模型与 API

### Modified Capabilities
（无已有 spec 需要修改）

## Impact

- **后端**: 新增 table-diff 相关 handler 和 service 层代码于 `internal/server/server.go` 和 `internal/sync/` 下；数据库连接复用现有 connection manager
- **前端**: 新增 `TableDiff.vue` 视图组件、对应路由、store 方法；前端路由新增 `/diff` 路径
- **API**: 新增 `/api/sync/diff/*` 系列端点
- **依赖**: 无新增外部依赖，复用现有 sqlx + gin 技术栈
