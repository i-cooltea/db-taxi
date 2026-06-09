## ADDED Requirements

### Requirement: 值不同的行支持单行复制 REPLACE

在"值不同的行"表格中，每行提供复制按钮，生成针对该行数据的 REPLACE SQL 语句。

#### Scenario: 点击单行复制按钮生成 REPLACE
- **WHEN** 用户在"值不同的行"表格中点击某行的"复制"按钮
- **THEN** 系统生成该行数据的 REPLACE SQL 语句并复制到剪贴板
- **THEN** 系统显示 Toast 提示"REPLACE 语句已复制"

#### Scenario: 生成的 REPLACE 语句格式正确
- **WHEN** 用户触发复制
- **THEN** 生成的 SQL 格式为 `REPLACE INTO \`目标库\`.\`目标表\` (col1, col2, ...) VALUES (val1, val2, ...);`
- **THEN** 所有非主键列使用源表值
- **THEN** 字符串类型值用单引号包裹并转义
- **THEN** NULL 值输出为 `NULL`（不加引号）
- **THEN** 数值类型值不加引号

### Requirement: 仅源表有的行支持单行复制 REPLACE

在"仅存在于源表"区域，每行提供复制按钮，生成针对该行数据的 REPLACE SQL 语句。

#### Scenario: 点击仅源表行的复制按钮
- **WHEN** 用户在"仅存在于源表"列表中点击某行的"复制"按钮
- **THEN** 系统生成该行数据的 REPLACE SQL 语句并复制到剪贴板

### Requirement: 支持批量生成全部差异行的 REPLACE

在数据差异区域提供"复制全部 REPLACE"按钮，生成所有差异行（值不同 + 仅源表有）的 REPLACE 语句。

#### Scenario: 点击批量复制按钮
- **WHEN** 用户点击"复制全部 REPLACE"按钮
- **THEN** 系统生成所有差异行的 REPLACE 语句（以分号分隔的多条 SQL）
- **THEN** 系统复制到剪贴板并显示 Toast 提示

### Requirement: 仅目标表有的行不生成 REPLACE

REPLACE 语句用于将源表数据覆盖到目标表，仅目标表有的行无法通过 REPLACE 删除，因此不提供复制功能。

#### Scenario: 仅目标表行无复制按钮
- **WHEN** 用户查看"仅存在于目标表"区域
- **THEN** 该区域不显示复制按钮

### Requirement: COPY 按钮使用剪贴板 API

复制功能使用 `navigator.clipboard.writeText()` API。

#### Scenario: 复制成功
- **WHEN** 剪贴板 API 调用成功
- **THEN** 系统显示成功 Toast

#### Scenario: 复制失败
- **WHEN** 剪贴板 API 调用失败（如权限问题）
- **THEN** 系统显示错误提示
