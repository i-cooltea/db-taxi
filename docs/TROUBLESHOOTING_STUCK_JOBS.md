# 同步任务卡住问题排查指南

## 问题描述

同步任务创建后一直卡在"处理中"(running)状态，无法完成或失败。

## 可能的原因

1. **JobEngine未启动** - 应用程序启动时JobEngine没有正确初始化
2. **远程数据库连接失败** - 无法连接到远程数据库
3. **网络问题** - 网络中断或超时
4. **数据库锁** - 源表或目标表被锁定
5. **应用程序崩溃** - 应用程序在任务执行过程中崩溃
6. **资源不足** - 内存或CPU资源不足导致任务挂起

## 诊断步骤

### 1. 使用诊断脚本

```bash
cd db-taxi
chmod +x scripts/diagnose-stuck-jobs.sh
./scripts/diagnose-stuck-jobs.sh
```

这个脚本会检查：
- 数据库连接状态
- 当前运行中的任务
- 长时间运行的任务
- 同步配置
- 表映射配置
- 最近的同步日志

### 2. 使用Go诊断工具

```bash
cd db-taxi
go run cmd/fix-jobs/main.go -config configs/config.yaml -dry-run
```

参数说明：
- `-config`: 配置文件路径（默认: configs/config.yaml）
- `-dry-run`: 只显示问题，不进行修复（默认: false）
- `-timeout`: 任务超时时间（分钟，默认: 30）

### 3. 直接查询数据库

```sql
-- 查看所有运行中的任务
SELECT 
    id,
    config_id,
    status,
    start_time,
    TIMESTAMPDIFF(MINUTE, start_time, NOW()) as running_minutes,
    total_tables,
    completed_tables,
    error_message
FROM sync_jobs 
WHERE status = 'running'
ORDER BY start_time DESC;

-- 查看任务日志
SELECT 
    job_id,
    table_name,
    level,
    message,
    created_at
FROM sync_logs
WHERE job_id = 'YOUR_JOB_ID'
ORDER BY created_at DESC;
```

## 修复方法

### 方法1: 使用自动修复工具

```bash
# 修复超过30分钟的卡住任务
cd db-taxi
go run cmd/fix-jobs/main.go -config configs/config.yaml

# 修复超过60分钟的卡住任务
go run cmd/fix-jobs/main.go -config configs/config.yaml -timeout 60
```

### 方法2: 使用SQL脚本

```bash
cd db-taxi
mysql -h localhost -u root -p db_taxi < scripts/fix-stuck-jobs.sql
```

### 方法3: 手动修复

```sql
-- 将卡住的任务标记为失败
UPDATE sync_jobs 
SET 
    status = 'failed',
    error_message = 'Manually marked as failed due to timeout',
    end_time = NOW()
WHERE status = 'running' 
AND TIMESTAMPDIFF(MINUTE, start_time, NOW()) > 30;
```

## 预防措施

### 1. 确保JobEngine正常启动

检查应用程序日志，确认看到以下消息：
```
INFO Job engine started successfully worker_count=5
INFO Sync system initialized successfully
```

### 2. 配置合理的超时时间

在同步配置中设置合理的超时时间：

```yaml
sync:
  enabled: true
  job_timeout: 3600  # 任务超时时间（秒）
  worker_count: 5    # 并发工作线程数
```

### 3. 监控任务状态

定期检查任务状态，及时发现问题：

```bash
# 每5分钟检查一次
watch -n 300 'mysql -u root -p -e "SELECT status, COUNT(*) FROM db_taxi.sync_jobs GROUP BY status"'
```

### 4. 设置任务超时自动清理

创建一个cron任务定期清理超时任务：

```bash
# 添加到crontab
*/30 * * * * cd /path/to/db-taxi && go run cmd/fix-jobs/main.go -config configs/config.yaml >> /var/log/db-taxi-fix-jobs.log 2>&1
```

### 5. 优化同步配置

- 减小批处理大小（batch_size）
- 增加工作线程数（max_concurrency）
- 使用增量同步而不是全量同步
- 为大表添加WHERE子句限制数据范围

## 常见问题

### Q1: 为什么任务会卡住？

**A:** 最常见的原因是：
1. 远程数据库连接超时或断开
2. 网络不稳定
3. 源表数据量太大，同步时间过长
4. 应用程序在同步过程中重启或崩溃

### Q2: 如何避免任务卡住？

**A:** 
1. 确保网络连接稳定
2. 使用增量同步而不是全量同步
3. 为大表设置合理的批处理大小
4. 定期监控任务状态
5. 设置任务超时自动清理机制

### Q3: 修复后任务还是卡住怎么办？

**A:**
1. 检查应用程序日志，查找错误信息
2. 验证远程数据库连接是否正常
3. 检查是否有防火墙或网络策略阻止连接
4. 尝试手动连接远程数据库测试
5. 重启应用程序重新初始化JobEngine

### Q4: 如何查看任务执行的详细日志？

**A:**
```sql
-- 查看特定任务的所有日志
SELECT 
    table_name,
    level,
    message,
    created_at
FROM sync_logs
WHERE job_id = 'YOUR_JOB_ID'
ORDER BY created_at;

-- 查看错误日志
SELECT 
    job_id,
    table_name,
    message,
    created_at
FROM sync_logs
WHERE level = 'error'
ORDER BY created_at DESC
LIMIT 20;
```

### Q5: 如何重新运行失败的任务？

**A:**
失败的任务不能直接重新运行，需要创建新的同步任务：

1. 通过Web界面：进入同步配置页面，点击"开始同步"按钮
2. 通过API：调用 `POST /api/sync/configs/{config_id}/start` 端点
3. 通过命令行：使用curl命令调用API

```bash
curl -X POST http://localhost:8080/api/sync/configs/{config_id}/start
```

## 联系支持

如果问题仍然无法解决，请：
1. 收集应用程序日志
2. 导出数据库中的任务记录
3. 记录问题发生的时间和环境
4. 联系技术支持团队

## 相关文档

- [同步系统用户指南](SYNC_USER_GUIDE.md)
- [API文档](API.md)
- [部署指南](DEPLOYMENT.md)
- [系统集成文档](SYSTEM_INTEGRATION.md)
