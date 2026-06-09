## 1. REPLACE SQL 生成工具函数

- [x] 1.1 创建 `formatSQLValue(val)` 工具函数：处理字符串转义、数值判断、NULL 处理
- [x] 1.2 创建 `buildReplaceSQL(tableName, columns, values)` 工具函数：生成单条 REPLACE 语句
- [x] 1.3 创建 `generateReplaceForValueDiff(entry, sourceTable, targetTable)`：为值不同的行生成 REPLACE
- [x] 1.4 创建 `generateReplaceForOnlySource(row, sourceTable, targetTable)`：为仅源表行生成 REPLACE

## 2. 值不同行表格增加操作列

- [x] 2.1 在值不同行表格的 `<thead>` 增加"操作"列
- [x] 2.2 在值不同行表格的 `<tbody>` 每行增加复制按钮（使用 Clipboard API）
- [x] 2.3 复制成功后调用 Toast 显示"REPLACE 语句已复制"

## 3. 仅源表行增加复制按钮

- [x] 3.1 在"仅存在于源表"区域的每行卡片增加复制按钮
- [x] 3.2 点击后生成完整 REPLACE 语句并复制到剪贴板

## 4. 批量复制全部 REPLACE

- [x] 4.1 在数据差异区域（diff-summary 附近）增加"复制全部 REPLACE"按钮
- [x] 4.2 点击后收集所有值不同行 + 仅源表行的 REPLACE 语句，合并复制

## 5. 验证

- [ ] 5.1 验证单行复制按钮生成正确的 REPLACE 语句格式
- [ ] 5.2 验证批量复制功能正常工作
- [ ] 5.3 验证 Toast 提示正确显示
- [ ] 5.4 验证仅目标表行没有复制按钮
