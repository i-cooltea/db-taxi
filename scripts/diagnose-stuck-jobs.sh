#!/bin/bash

# 诊断卡在"处理中"状态的同步任务

echo "=== 数据库同步任务诊断工具 ==="
echo ""

# 检查MySQL连接
echo "1. 检查数据库连接..."
mysql -h localhost -u root -p -e "SELECT 1" 2>/dev/null
if [ $? -ne 0 ]; then
    echo "❌ 无法连接到MySQL数据库"
    echo "请检查数据库配置和连接信息"
    exit 1
fi
echo "✅ 数据库连接正常"
echo ""

# 检查同步任务状态
echo "2. 检查同步任务状态..."
mysql -h localhost -u root -p -e "
USE db_taxi;
SELECT 
    id,
    config_id,
    status,
    start_time,
    end_time,
    total_tables,
    completed_tables,
    TIMESTAMPDIFF(MINUTE, start_time, NOW()) as running_minutes,
    error_message
FROM sync_jobs 
WHERE status = 'running'
ORDER BY start_time DESC;
" 2>/dev/null

echo ""

# 检查是否有长时间运行的任务
echo "3. 检查长时间运行的任务（超过30分钟）..."
mysql -h localhost -u root -p -e "
USE db_taxi;
SELECT 
    id,
    config_id,
    status,
    start_time,
    TIMESTAMPDIFF(MINUTE, start_time, NOW()) as running_minutes
FROM sync_jobs 
WHERE status = 'running' 
AND TIMESTAMPDIFF(MINUTE, start_time, NOW()) > 30;
" 2>/dev/null

echo ""

# 检查同步配置
echo "4. 检查同步配置..."
mysql -h localhost -u root -p -e "
USE db_taxi;
SELECT 
    sc.id,
    sc.name,
    sc.connection_id,
    sc.enabled,
    COUNT(tm.id) as table_count
FROM sync_configs sc
LEFT JOIN table_mappings tm ON sc.id = tm.sync_config_id
GROUP BY sc.id;
" 2>/dev/null

echo ""

# 检查表映射
echo "5. 检查表映射配置..."
mysql -h localhost -u root -p -e "
USE db_taxi;
SELECT 
    tm.id,
    tm.sync_config_id,
    tm.source_table,
    tm.target_table,
    tm.sync_mode,
    tm.enabled
FROM table_mappings tm
ORDER BY tm.sync_config_id, tm.source_table;
" 2>/dev/null

echo ""

# 检查同步日志
echo "6. 检查最近的同步日志..."
mysql -h localhost -u root -p -e "
USE db_taxi;
SELECT 
    sl.job_id,
    sl.table_name,
    sl.level,
    sl.message,
    sl.created_at
FROM sync_logs sl
ORDER BY sl.created_at DESC
LIMIT 20;
" 2>/dev/null

echo ""

# 提供修复建议
echo "=== 修复建议 ==="
echo ""
echo "如果发现任务卡在'running'状态："
echo ""
echo "1. 手动将卡住的任务标记为失败："
echo "   UPDATE sync_jobs SET status='failed', error_message='Manually marked as failed due to timeout', end_time=NOW() WHERE status='running' AND TIMESTAMPDIFF(MINUTE, start_time, NOW()) > 30;"
echo ""
echo "2. 检查应用程序日志："
echo "   tail -f /path/to/db-taxi.log"
echo ""
echo "3. 重启应用程序以重新初始化JobEngine"
echo ""
echo "4. 检查远程数据库连接是否正常"
echo ""
echo "5. 检查是否有死锁或长时间运行的查询："
echo "   SHOW PROCESSLIST;"
echo ""

