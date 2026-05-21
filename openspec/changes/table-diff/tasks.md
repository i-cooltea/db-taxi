## 1. 后端数据模型与结构差异检测

- [x] 1.1 在 `internal/sync/` 下新增 `diff_engine.go`，定义 DiffResult、SchemaDiff、ColumnDiff 等结构体
- [x] 1.2 实现 `CompareSchema` 函数：通过 `SHOW COLUMNS` 获取源表和目标表列信息，按列名做差集和交集，输出新增列、删除列、变更列（类型、NULL、默认值）

## 2. 后端数据差异检测

- [x] 2.1 实现 `GetPrimaryKeys` 函数：通过 `SHOW INDEX` 获取表的主键列，无主键时返回错误
- [x] 2.2 实现 `CompareData` 函数：基于主键 LEFT/RIGHT JOIN 匹配行，对匹配行计算 MD5 哈希比对值差异，分类输出 only_in_source、only_in_target、value_diff
- [x] 2.3 实现大表行数限制：CompareData 支持 maxRows 参数（默认 500），超过时截断并标记 partial=true、total_rows、compared_rows
- [x] 2.4 实现对比入口 `RunDiff`：编排主键检测→结构对比→数据对比流程，无主键时跳过数据对比并在结果中提示

## 3. 后端异步任务与 API

- [x] 3.1 新增 `diff_task.go`，实现对比任务管理器（创建任务、goroutine 执行、查询状态）
- [x] 3.2 在 `internal/server/server.go` 注册 `/api/sync/diff/` 路由组
- [x] 3.3 实现 `POST /api/sync/diff/compare` handler：解析请求（source/target 各含 connection_id、database、table），创建异步任务，返回 task_id
- [x] 3.4 实现 `GET /api/sync/diff/:task_id` handler：返回任务状态（running/done/failed）及对比结果

## 4. 前端路由与页面框架

- [x] 4.1 在 `frontend/src/router/index.js` 添加 `/diff` 路由指向 `TableDiff` 视图
- [x] 4.2 创建 `frontend/src/views/TableDiff.vue` 基础框架：左侧源表选择区、右侧目标表选择区、底部结果展示区
- [x] 4.3 在 `frontend/src/views/Home.vue` 导航栏新增"表对比"标签页

## 5. 前端三级选择器与对比触发

- [x] 5.1 实现源表三级选择器：连接下拉→加载数据库列表→加载表列表
- [x] 5.2 实现目标表三级选择器（同上结构）
- [x] 5.3 实现"开始对比"按钮，调用 `POST /api/sync/diff/compare`，轮询 `GET /api/sync/diff/:task_id` 获取结果，展示加载状态

## 6. 前端差异结果展示

- [x] 6.1 实现结构差异 Tab：表格展示列差异，新增列绿色、删除列红色、变更列黄色
- [x] 6.2 实现数据差异 Tab：表格展示 only_in_source / only_in_target / value_diff 行，支持分页
- [x] 6.3 实现差异值高亮：value_diff 行中不同列值高亮标记
- [x] 6.4 实现无主键提示：数据对比被跳过时显示提示信息

## 7. 导出与验证

- [x] 7.1 实现对比结果 JSON 导出按钮：前端下载完整结果的 JSON 文件
- [ ] 7.2 端到端验证：选表对比→结构差异和数据差异结果正确
- [ ] 7.3 大表场景验证：确认 500 行限制和 partial 标记正常
- [ ] 7.4 无主键表场景验证：结构对比正常、数据对比跳过并提示
