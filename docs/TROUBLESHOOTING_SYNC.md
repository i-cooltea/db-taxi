# 同步常见问题排查

## 源表与目标表字符集一致

同步创建目标表时，会从源表读取字符集与排序规则并应用到目标表，使**源表与目标表字符编码一致**：

- **表级别**：从 `INFORMATION_SCHEMA.TABLES` 读取源表的 `TABLE_COLLATION`，在目标表上使用 `DEFAULT CHARSET=... COLLATE=...`。
- **列级别**：从 `INFORMATION_SCHEMA.COLUMNS` 读取各列的 `CHARACTER_SET_NAME`、`COLLATION_NAME`（仅字符串类型有值），在目标表列定义中加上 `CHARACTER SET ... COLLATE ...`。

因此若源表为 utf8mb4，目标表也会按 utf8mb4 创建，无需再单独改目标库/表。

---

## Error 1366: Incorrect string value (emoji / 4-byte UTF-8)

### 现象

同步时报错类似：

```text
Table sync failed: failed to sync data: failed to insert batch: failed to execute batch insert: Error 1366 (HY000): Incorrect string value: '\xF0\x9F\x98\x84\xE5\x93...' for column 'xxx' at row N
```

其中 `\xF0\x9F\x98\x84` 为 emoji（如 😄）的 UTF-8 编码，属于 4 字节 UTF-8 字符。

### 原因

- MySQL 的 **utf8** 实际是 utf8mb3，**最多只支持 3 字节**，无法存储 emoji 等 4 字节字符。
- 目标库/表/列若不是 **utf8mb4**，插入 4 字节字符就会报 1366。

### 解决方式

**1. 连接已使用 utf8mb4（代码已改）**

本项目的 MySQL 连接已统一加上 `charset=utf8mb4`，保证客户端按 utf8mb4 传数据。若仍报 1366，说明问题在**目标端字符集**。

**2. 把目标库/表改为 utf8mb4（必做）**

在**目标 MySQL** 上执行（把 `your_db`、`your_table`、`your_column` 换成实际库名、表名、列名）：

```sql
-- 库级别（新建库会继承，已有库可改）
ALTER DATABASE your_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 整张表转 utf8mb4（推荐，一次改完所有字符串列）
ALTER TABLE your_table CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 或只改某一列（若不想动整表）
ALTER TABLE your_table MODIFY your_column VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

改完后重新执行同步即可。

**3. 不想改目标库时的替代做法**

若目标必须保持 utf8（3 字节），只能在同步前在应用层过滤/替换 4 字节字符（例如替换成空或占位符），一般不建议，推荐直接使用 utf8mb4。
