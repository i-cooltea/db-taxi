## Why

表对比功能目前只展示源表和目标表的数据差异（值不同、仅源表有、仅目标表有），用户无法便捷地将源表数据覆盖到目标表。需要在值不同的行上提供"复制"按钮，一键生成 REPLACE 语句，简化数据修复流程。

## What Changes

- **值不同的行增加复制按钮**：在"值不同的行"表格中，每行增加一个操作列，点击后生成针对该行数据的 REPLACE SQL 语句
- **仅源表有的行增加复制按钮**：在"仅存在于源表"区域，每行增加一个复制按钮，生成 REPLACE 语句
- **整体同步按钮**：在数据差异区域增加"生成全部 REPLACE"按钮，一键复制所有差异行的 REPLACE 语句
- REPLACE 语句格式：`REPLACE INTO \`目标库\`.\`目标表\` (col1, col2, ...) VALUES (val1, val2, ...);`
- 复制成功后显示 Toast 提示

## Capabilities

### New Capabilities
- `table-diff-copy`: 表对比结果复制功能，支持单行和批量生成 REPLACE SQL 语句

### Modified Capabilities

（无，纯新增功能）

## Impact

- **Frontend**: `frontend/src/views/TableDiff.vue` — 修改表格模板，增加操作列和复制按钮
- **Frontend**: `frontend/src/components/Toast.vue` — 复制成功提示（已有组件，无需修改）
- 不涉及后端 API 改动，纯前端功能
