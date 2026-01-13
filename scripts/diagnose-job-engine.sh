#!/bin/bash

# Job Engine 诊断脚本
# 用于检查 jobEngine 为什么没有启动

echo "========================================="
echo "Job Engine 诊断工具"
echo "========================================="
echo ""

# 检查配置文件
echo "1. 检查配置文件..."
if [ -f ".env" ]; then
    echo "✓ 找到 .env 文件"
    
    # 检查 SYNC_ENABLED
    if grep -q "^SYNC_ENABLED=true" .env; then
        echo "✓ SYNC_ENABLED=true"
    elif grep -q "^SYNC_ENABLED=false" .env; then
        echo "✗ SYNC_ENABLED=false - Sync 系统被禁用！"
        echo "  解决方案: 将 .env 中的 SYNC_ENABLED 设置为 true"
    else
        echo "⚠ SYNC_ENABLED 未设置，使用默认值 (true)"
    fi
    
    # 检查数据库配置
    echo ""
    echo "数据库配置:"
    grep "^DB_" .env | while read line; do
        key=$(echo $line | cut -d'=' -f1)
        if [[ $key == *"PASSWORD"* ]]; then
            echo "  $key=***"
        else
            echo "  $line"
        fi
    done
else
    echo "⚠ 未找到 .env 文件，使用默认配置"
fi

echo ""
echo "2. 检查数据库连接..."
if command -v mysql &> /dev/null; then
    # 从 .env 读取配置
    if [ -f ".env" ]; then
        export $(grep -v '^#' .env | xargs)
    fi
    
    DB_HOST=${DB_HOST:-localhost}
    DB_PORT=${DB_PORT:-3306}
    DB_USERNAME=${DB_USERNAME:-root}
    DB_DATABASE=${DB_DATABASE:-myapp}
    
    if mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USERNAME" -p"$DB_PASSWORD" -e "SELECT 1" &> /dev/null; then
        echo "✓ 数据库连接成功"
        
        # 检查 sync 表是否存在
        echo ""
        echo "3. 检查 Sync 系统表..."
        TABLES=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USERNAME" -p"$DB_PASSWORD" "$DB_DATABASE" -e "SHOW TABLES LIKE 'sync_%'" -s)
        
        if [ -z "$TABLES" ]; then
            echo "✗ Sync 系统表不存在"
            echo "  解决方案: 运行 'make migrate' 或 './scripts/migrate.sh up'"
        else
            echo "✓ 找到以下 Sync 系统表:"
            echo "$TABLES" | while read table; do
                echo "  - $table"
            done
            
            # 检查表结构
            echo ""
            echo "4. 检查关键表..."
            
            # 检查 connections 表
            if echo "$TABLES" | grep -q "connections"; then
                COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USERNAME" -p"$DB_PASSWORD" "$DB_DATABASE" -e "SELECT COUNT(*) FROM connections" -s)
                echo "  connections: $COUNT 条记录"
            fi
            
            # 检查 sync_configs 表
            if echo "$TABLES" | grep -q "sync_configs"; then
                COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USERNAME" -p"$DB_PASSWORD" "$DB_DATABASE" -e "SELECT COUNT(*) FROM sync_configs" -s)
                echo "  sync_configs: $COUNT 条记录"
            fi
            
            # 检查 sync_jobs 表
            if echo "$TABLES" | grep -q "sync_jobs"; then
                COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USERNAME" -p"$DB_PASSWORD" "$DB_DATABASE" -e "SELECT COUNT(*) FROM sync_jobs" -s)
                echo "  sync_jobs: $COUNT 条记录"
                
                # 检查是否有卡住的任务
                STUCK=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USERNAME" -p"$DB_PASSWORD" "$DB_DATABASE" -e "SELECT COUNT(*) FROM sync_jobs WHERE status='running'" -s)
                if [ "$STUCK" -gt 0 ]; then
                    echo "  ⚠ 发现 $STUCK 个运行中的任务"
                fi
            fi
        fi
    else
        echo "✗ 数据库连接失败"
        echo "  请检查数据库配置和连接信息"
    fi
else
    echo "⚠ 未安装 mysql 客户端，跳过数据库检查"
fi

echo ""
echo "5. 检查应用日志..."
if [ -f "db-taxi.log" ]; then
    echo "检查最近的日志..."
    
    # 检查 jobEngine 启动日志
    if grep -q "Job engine started" db-taxi.log; then
        echo "✓ 找到 Job engine 启动日志"
        grep "Job engine started" db-taxi.log | tail -1
    else
        echo "✗ 未找到 Job engine 启动日志"
    fi
    
    # 检查错误日志
    echo ""
    echo "最近的错误:"
    grep -i "error\|failed" db-taxi.log | tail -5
else
    echo "⚠ 未找到日志文件 db-taxi.log"
    echo "  提示: 运行应用时使用 'make run 2>&1 | tee db-taxi.log' 来记录日志"
fi

echo ""
echo "6. 检查进程状态..."
if pgrep -f "db-taxi" > /dev/null; then
    echo "✓ db-taxi 进程正在运行"
    echo "  PID: $(pgrep -f 'db-taxi')"
else
    echo "✗ db-taxi 进程未运行"
fi

echo ""
echo "========================================="
echo "诊断建议"
echo "========================================="
echo ""
echo "如果 Job Engine 没有启动，请按以下步骤排查:"
echo ""
echo "1. 确保 SYNC_ENABLED=true (在 .env 文件中)"
echo "2. 确保数据库连接正常"
echo "3. 运行数据库迁移: make migrate"
echo "4. 检查应用启动日志中的错误信息"
echo "5. 使用以下命令启动应用并查看详细日志:"
echo "   LOG_LEVEL=debug make run 2>&1 | tee db-taxi.log"
echo ""
echo "如果问题仍然存在，请查看日志文件中的详细错误信息"
echo ""
