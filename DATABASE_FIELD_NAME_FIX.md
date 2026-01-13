# 数据库字段名错误修复

## 问题描述

在更新 sync_jobs 表时出现错误：

```
failed to update sync job: could not find name error in &sync.SyncJob{...}
```

## 根本原因

在 `internal/sync/repository.go` 的 `UpdateSyncJob` 方法中，SQL 语句使用的命名参数与 `SyncJob` 结构体的 db tag 不匹配：

**错误的代码：**
```go
query := `
    UPDATE sync_jobs 
    SET ... error_message = :error  -- ❌ 错误：使用 :error
    WHERE id = :id
`
```

**SyncJob 结构体定义：**
```go
type SyncJob struct {
    ...
    Error string `json:"error,omitempty" db:"error_message"`  // ✅ db tag 是 error_message
}
```

**问题：**
- SQL 中使用 `:error`
- 但结构体的 db tag 是 `error_message`
- sqlx 的 NamedExec 无法找到名为 `error` 的字段

## 解决方案

修改 SQL 语句，使用正确的字段名：

```go
query := `
    UPDATE sync_jobs 
    SET ... error_message = :error_message  -- ✅ 正确：使用 :error_message
    WHERE id = :id
`
```

## 修复文件

- `internal/sync/repository.go` - UpdateSyncJob 方法

## 验证结果

修复后，任务可以正常更新状态：

```bash
# 测试前 - 错误
❌ failed to update sync job: could not find name error

# 测试后 - 成功
✅ 所有任务成功完成
✅ 状态正确更新为 "completed"
✅ 没有数据库更新错误
```

## 相关知识

### sqlx NamedExec 的工作原理

sqlx 的 `NamedExec` 使用结构体的 `db` tag 来映射命名参数：

```go
type Example struct {
    Name  string `db:"user_name"`   // SQL 中使用 :user_name
    Email string `db:"email"`       // SQL 中使用 :email
}

// 正确的 SQL
query := "UPDATE users SET user_name = :user_name, email = :email"

// 错误的 SQL
query := "UPDATE users SET user_name = :Name, email = :Email"  // ❌ 会失败
```

### 最佳实践

1. **保持一致性**
   - SQL 命名参数应该与 db tag 完全匹配
   - 使用 snake_case 命名数据库字段

2. **检查清单**
   - [ ] 结构体字段有正确的 db tag
   - [ ] SQL 命名参数与 db tag 匹配
   - [ ] 字段名大小写正确

3. **测试方法**
   ```go
   // 使用 NamedExec 前，可以先打印 SQL 和结构体
   log.Printf("Query: %s", query)
   log.Printf("Struct: %+v", job)
   ```

## 其他可能的字段名问题

检查其他 SQL 语句是否有类似问题：

```bash
# 搜索所有使用 NamedExec 的地方
grep -r "NamedExec" internal/sync/*.go

# 检查 SQL 命名参数
grep -r ":\w\+" internal/sync/*.go | grep -v "error_message"
```

## 总结

✅ **问题已修复**

- 修改了 `UpdateSyncJob` 方法中的 SQL 语句
- 将 `:error` 改为 `:error_message`
- 与 SyncJob 结构体的 db tag 保持一致
- 所有任务现在可以正常更新状态

这是一个简单但重要的修复，确保了数据库操作的正确性。
