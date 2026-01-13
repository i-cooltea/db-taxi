-- 修复卡在"running"状态的同步任务
-- 使用方法: mysql -h localhost -u root -p db_taxi < fix-stuck-jobs.sql

-- 1. 查看当前卡住的任务
SELECT 
    id,
    config_id,
    status,
    start_time,
    TIMESTAMPDIFF(MINUTE, start_time, NOW()) as running_minutes,
    total_tables,
    completed_tables
FROM sync_jobs 
WHERE status = 'running'
ORDER BY start_time;

-- 2. 将超过30分钟仍在运行的任务标记为失败
UPDATE sync_jobs 
SET 
    status = 'failed',
    error_message = 'Task timeout - automatically marked as failed after 30 minutes',
    end_time = NOW()
WHERE status = 'running' 
AND TIMESTAMPDIFF(MINUTE, start_time, NOW()) > 30;

-- 3. 显示修复结果
SELECT 
    CONCAT('Fixed ', ROW_COUNT(), ' stuck jobs') as result;

-- 4. 查看所有任务的状态统计
SELECT 
    status,
    COUNT(*) as count,
    MIN(start_time) as earliest,
    MAX(start_time) as latest
FROM sync_jobs
GROUP BY status
ORDER BY status;

-- 5. 查看最近失败的任务及错误信息
SELECT 
    id,
    config_id,
    start_time,
    end_time,
    error_message
FROM sync_jobs
WHERE status = 'failed'
ORDER BY start_time DESC
LIMIT 10;
