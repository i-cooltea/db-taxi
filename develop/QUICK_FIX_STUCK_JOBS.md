# 快速修复卡住的同步任务

## 问题：任务一直显示"处理中"状态

### 快速诊断

```bash
# 方法1: 使用Makefile（推荐）
cd db-taxi
make diagnose-jobs

# 方法2: 直接运行Go程序
cd db-taxi
go run cmd/fix-jobs/main.go -config configs/config.yaml -dry-run

# 方法3: 使用Shell脚本
cd db-taxi
./scripts/diagnose-stuck-jobs.sh
```

### 快速修复

```bash
# 方法1: 使用Makefile（推荐）
cd db-taxi
make fix-jobs

# 自定义超时时间（默认30分钟）
make fix-jobs TIMEOUT=60

# 方法2: 直接运行Go程序
cd db-taxi
go run cmd/fix-jobs/main.go -config configs/config.yaml

# 方法3: 使用SQL脚本
cd db-taxi
mysql -h localhost -u root -p db_taxi < scripts/fix-stuck-jobs.sql
```

### 手动修复（最快）

```sql
-- 连接到MySQL
mysql -h localhost -u root -p

-- 切换到数据库
USE db_taxi;

-- 查看卡住的任务
SELECT 
    id,
    config_id,
    status,
    start_time,
    TIMESTAMPDIFF(MINUTE, start_time, NOW()) as running_minutes
FROM sync_jobs 
WHERE status = 'running'
ORDER BY start_time;

-- 修复超过30分钟的任务
UPDATE sync_jobs 
SET 
    status = 'failed',
    error_message = 'Task timeout - manually fixed',
    end_time = NOW()
WHERE status = 'running' 
AND TIMESTAMPDIFF(MINUTE, start_time, NOW()) > 30;

-- 验证修复结果
SELECT status, COUNT(*) as count 
FROM sync_jobs 
GROUP BY status;
```

## 根本原因分析

### 1. 检查JobEngine是否启动

```bash
# 查看应用程序日志
tail -f /path/to/db-taxi.log | grep "Job engine"

# 应该看到：
# INFO Job engine started successfully worker_count=5
# INFO Sync system initialized successfully
```

如果没有看到这些日志，说明JobEngine没有启动。

**解决方法：**
```bash
# 重启应用程序
systemctl restart db-taxi
# 或
docker-compose restart db-taxi
```

### 2. 检查远程数据库连接

```bash
# 测试远程数据库连接
mysql -h REMOTE_HOST -P REMOTE_PORT -u REMOTE_USER -p REMOTE_DB -e "SELECT 1"
```

如果连接失败，检查：
- 网络连接
- 防火墙规则
- 数据库用户权限
- SSL配置

### 3. 检查数据库锁

```sql
-- 查看当前锁定的表
SHOW OPEN TABLES WHERE In_use > 0;

-- 查看正在运行的进程
SHOW PROCESSLIST;

-- 杀死长时间运行的查询
KILL QUERY process_id;
```

## 预防措施

### 1. 启用自动清理

添加到crontab：
```bash
# 每30分钟自动清理超时任务
*/30 * * * * cd /path/to/db-taxi && make fix-jobs >> /var/log/db-taxi-fix-jobs.log 2>&1
```

### 2. 监控任务状态

```bash
# 实时监控任务状态
watch -n 60 'mysql -u root -p -e "SELECT status, COUNT(*) FROM db_taxi.sync_jobs GROUP BY status"'
```

### 3. 优化同步配置

编辑同步配置，减小批处理大小：
```json
{
  "options": {
    "batch_size": 500,
    "max_concurrency": 3
  }
}
```

## 常见错误及解决方法

### 错误1: "connection refused"
**原因：** 无法连接到远程数据库  
**解决：** 检查网络连接和防火墙规则

### 错误2: "table not found"
**原因：** 源表不存在或已被删除  
**解决：** 更新表映射配置，移除不存在的表

### 错误3: "timeout"
**原因：** 查询执行时间过长  
**解决：** 增加超时时间或使用增量同步

### 错误4: "out of memory"
**原因：** 内存不足  
**解决：** 减小批处理大小或增加系统内存

## 需要帮助？

查看详细文档：
- [完整故障排查指南](docs/TROUBLESHOOTING_STUCK_JOBS.md)
- [同步系统用户指南](docs/SYNC_USER_GUIDE.md)
- [API文档](docs/API.md)

或联系技术支持团队。
