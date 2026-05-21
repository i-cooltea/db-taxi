## ADDED Requirements

### Requirement: 表对比 API
系统 SHALL 提供 `/api/sync/diff/compare` POST 端点，接收源表和目标表的连接ID、数据库名、表名，发起异步对比任务并返回任务 ID。

#### Scenario: 成功发起对比任务
- **WHEN** 用户提交源表（connection_id=1, database="db1", table="users"）和目标表（connection_id=2, database="db2", table="users"）
- **THEN** 系统创建对比任务并返回 `{task_id: "xxx", status: "running"}`

#### Scenario: 表不存在
- **WHEN** 用户提交的源表或目标表在对应数据库中不存在
- **THEN** 系统返回 400 错误并提示表不存在

### Requirement: 对比任务状态查询
系统 SHALL 提供 `/api/sync/diff/:task_id` GET 端点，返回对比任务的状态和结果。

#### Scenario: 任务进行中
- **WHEN** 对比任务仍在执行
- **THEN** 返回 `{task_id: "xxx", status: "running", progress: 50}`

#### Scenario: 任务完成
- **WHEN** 对比任务执行完毕
- **THEN** 返回 `{task_id: "xxx", status: "done", result: {schema_diff: {...}, data_diff: {...}}}`

#### Scenario: 任务失败
- **WHEN** 对比任务执行出错（如连接断开）
- **THEN** 返回 `{task_id: "xxx", status: "failed", error: "connection lost"}`

### Requirement: 结构差异检测
系统 SHALL 检测两张表之间的列结构差异，包括列新增、列删除、列属性变更（类型、是否允许 NULL、默认值）。

#### Scenario: 列新增
- **WHEN** 源表包含列 "email" 而目标表不包含该列
- **THEN** 结果中 schema_diff.added_columns 包含 `{name: "email", type: "varchar(255)", nullable: false}`

#### Scenario: 列删除
- **WHEN** 目标表包含列 "legacy_field" 而源表不包含该列
- **THEN** 结果中 schema_diff.removed_columns 包含 `{name: "legacy_field", type: "int", nullable: true}`

#### Scenario: 列类型变更
- **WHEN** 源表的 "age" 列类型为 "int" 而目标表为 "bigint"
- **THEN** 结果中 schema_diff.modified_columns 包含 `{name: "age", source_type: "int", target_type: "bigint"}`

### Requirement: 数据差异检测
系统 SHALL 检测两张表之间的行数据差异，基于主键列进行行匹配，分类为仅存在于源表、仅存在于目标表、值不同的行。

#### Scenario: 行仅存在于源表
- **WHEN** 源表存在主键 id=100 的行但目标表不存在
- **THEN** 结果中 data_diff.only_in_source 包含该行的主键值和完整数据

#### Scenario: 行仅存在于目标表
- **WHEN** 目标表存在主键 id=200 的行但源表不存在
- **THEN** 结果中 data_diff.only_in_target 包含该行的主键值和完整数据

#### Scenario: 行值不同
- **WHEN** 源表和目标表都存在主键 id=50 的行，但 "status" 列值不同
- **THEN** 结果中 data_diff.value_diff 包含 `{primary_key: {id: 50}, differences: {column: "status", source_value: "active", target_value: "inactive"}}`

### Requirement: 无主键表处理
系统 SHALL 检测表是否拥有主键，若源表或目标表无主键，则仅执行结构对比，跳过数据对比，并在结果中明确提示。

#### Scenario: 无主键表对比
- **WHEN** 源表没有主键
- **THEN** 结果包含 schema_diff，data_diff 为空，提示 "源表无主键，无法进行数据差异对比"

### Requirement: 大表行数限制
系统 SHALL 在数据对比时设置最大对比行数（默认 100,000），超过时仅对比前 N 行并在结果中标记为部分对比。

#### Scenario: 表行数超限
- **WHEN** 源表包含 500,000 行数据
- **THEN** 系统仅对比前 100,000 行，结果中标记 `partial: true, total_rows: 500000, compared_rows: 100000`

### Requirement: 对比结果导出
系统 SHALL 提供对比结果的 JSON 导出功能，用户可在前端下载完整对比结果。

#### Scenario: 导出对比结果
- **WHEN** 用户点击"导出"按钮
- **THEN** 系统下载包含完整 schema_diff 和 data_diff 的 JSON 文件

### Requirement: 前端表对比页面
系统 SHALL 在前端提供独立的表对比页面（路由 `/diff`），包含源表和目标表的三级选择器（连接→数据库→表）、对比触发按钮、以及结构和数据差异的可视化展示。

#### Scenario: 选择表并发起对比
- **WHEN** 用户分别选择源表和目标表后点击"开始对比"
- **THEN** 系统发起对比请求并展示进度，完成后显示差异结果

#### Scenario: 查看结构差异
- **WHEN** 对比完成后用户切换到"结构差异"标签
- **THEN** 页面以表格展示列差异，新增列标绿、删除列标红、变更列标黄

#### Scenario: 查看数据差异
- **WHEN** 对比完成后用户切换到"数据差异"标签
- **THEN** 页面以表格展示行差异，支持分页浏览，差异值高亮标记
