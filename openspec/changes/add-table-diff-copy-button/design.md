## Context

表对比功能（`TableDiff.vue`）目前支持：
- 选择源表和目标表进行结构和数据对比
- 展示"值不同的行"（分页）、"仅源表有"、"仅目标表有"三个数据差异类别
- 导出 JSON 格式的差异结果

现有数据流：
- 后端 `diff_engine.go` 返回 `ValueDiffEntry`（含 PrimaryKey 和 Differences 列表）以及 `OnlyInSource`（完整行数据）
- 前端通过轮询获取对比结果并渲染

用户需要将源表数据覆盖到目标表，当前只能手动逐条编写 SQL，效率低且容易出错。

本设计为纯前端改动，不涉及后端 API 变更。

## Goals / Non-Goals

**Goals:**
- 在"值不同的行"表格每行增加复制按钮，生成单行 REPLACE 语句
- 在"仅存在于源表"区域的每行增加复制按钮，生成单行 REPLACE 语句
- 提供"复制全部 REPLACE"按钮，生成所有差异行的批量 REPLACE 语句
- 复制成功后通过 Toast 组件反馈

**Non-Goals:**
- 不修改后端 API 或数据结构
- 不实现 REPLACE 语句的自动执行（仅复制到剪贴板）
- 不处理仅目标表有的行（这些行需要 DELETE 而非 REPLACE）
- 不处理结构差异导致的列不匹配

## Decisions

### Decision 1: 纯前端生成 REPLACE 语句

**选择**：在前端根据 diffResult 数据直接拼接 SQL 字符串。

**理由**：
- 后端 `ValueDiffEntry` 已包含完整的源表值（`source_value`）和主键信息
- `OnlyInSource` 包含完整的行数据
- 将数据库名和表名提交给后端拼接 SQL 会增加不必要的网络开销
- 前端生成逻辑简单直接，无安全风险（SQL 不自动执行）

### Decision 2: 值不同的行使用源表值构造 REPLACE

**选择**：对于 `ValueDiffEntry`，遍历该行的所有列（从更完整的行数据推断），将所有列的值用源表值填充。

**说明**：`ValueDiffEntry.Differences` 只包含有差异的列，不包含完整行数据。需要从 `diffResult.data_diff` 中获取当前行的完整源表数据来生成 REPLACE。但由于后端接口返回的结构限制，实际上我们可以通过 Differences 列表 + PrimaryKey 来重建完整行——主键值从 `primary_key` 取，其他列值从 `differences` 中取 source_value（对于无差异列，值相同，直接用源表值或目标表值均可）。

**更简单的方案**：由于 REPLACE 要求所有列的值，对于值不同的行：
- 主键列从 `entry.primary_key` 获取
- 差异列使用 `diff.source_value`
- 非差异列（值相同）可任意取源表或目标表值

但实际后端没有返回完整的列列表。我们无法直接知道哪些列"值相同但没有差异"。所以我们需要换个思路：

**最终方案**：在值不同的行中，生成 REPLACE 时只包含主键列和有差异的列（即 Differences 中涉及的列），并使用源表值。因为 REPLACE INTO 只需要提供要覆盖的列即可，未指定的列会保持原值。

对于"仅源表有"的行，`OnlyInSource` 包含完整行数据，可以生成完整的 REPLACE 语句。

### Decision 3: 复制按钮放在操作列

**选择**：在"值不同的行"表格增加"操作"列，放置复制按钮。

**理由**：
- 每行一个操作按钮，最直观
- 不与现有数据列混淆
- 可利用现有表格结构

### Decision 4: SQL 值转义策略

**选择**：字符串值用单引号包裹并转义内部的单引号（`'` → `''`），数值不加引号，NULL 输出为 `NULL`。

**理由**：
- MySQL REPLACE 语句的标准格式
- 单引号转义是 MySQL 的标准做法（两个单引号表示一个文字单引号）
- 通过类型判断来区分数值和字符串

## Risks / Trade-offs

- [Risk] 值不同的行缺少完整列列表 → Mitigation: REPLACE 只包含主键列和有差异的列，不指定无差异列（它们值相同，无需覆盖）
- [Risk] 数值类型判断不准确（JavaScript 中所有值都是字符串形式） → Mitigation: 使用正则或简单启发式判断（纯数字为数值，否则为字符串），用户可手动修正
- [Risk] 大数据量下"复制全部"生成大量 SQL → Mitigation: 剪贴板 API 可以处理较大文本，无需额外分片
