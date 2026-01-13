#!/bin/bash

# Job Engine 状态检查脚本

echo "========================================="
echo "Job Engine 状态检查"
echo "========================================="
echo ""

# 检查服务器是否运行
echo "1. 检查服务器状态..."
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✓ 服务器正在运行"
else
    echo "✗ 服务器未运行"
    exit 1
fi
echo ""

# 检查 Sync 系统状态
echo "2. 检查 Sync 系统状态..."
SYNC_STATUS=$(curl -s http://localhost:8080/api/sync/status)
if echo "$SYNC_STATUS" | grep -q '"success":true'; then
    echo "✓ Sync 系统健康"
    echo "$SYNC_STATUS" | jq -r '.data.status' 2>/dev/null || echo "$SYNC_STATUS"
else
    echo "✗ Sync 系统异常"
    echo "$SYNC_STATUS"
    exit 1
fi
echo ""

# 获取统计信息
echo "3. 获取系统统计..."
STATS=$(curl -s http://localhost:8080/api/sync/stats)
if echo "$STATS" | grep -q '"success":true'; then
    echo "✓ 统计信息:"
    echo "$STATS" | jq '.data' 2>/dev/null || echo "$STATS"
else
    echo "✗ 获取统计信息失败"
    echo "$STATS"
fi
echo ""

# 检查活跃任务
echo "4. 检查活跃任务..."
ACTIVE_JOBS=$(curl -s http://localhost:8080/api/sync/jobs/active)
if echo "$ACTIVE_JOBS" | grep -q '"success":true'; then
    JOB_COUNT=$(echo "$ACTIVE_JOBS" | jq '.meta.total' 2>/dev/null || echo "0")
    echo "✓ 活跃任务数: $JOB_COUNT"
    if [ "$JOB_COUNT" != "0" ]; then
        echo ""
        echo "活跃任务列表:"
        echo "$ACTIVE_JOBS" | jq '.data' 2>/dev/null || echo "$ACTIVE_JOBS"
    fi
else
    echo "⚠ 无法获取活跃任务信息"
fi
echo ""

# 检查任务历史
echo "5. 检查最近的任务..."
RECENT_JOBS=$(curl -s "http://localhost:8080/api/sync/jobs?limit=5")
if echo "$RECENT_JOBS" | grep -q '"success":true'; then
    JOB_COUNT=$(echo "$RECENT_JOBS" | jq '.meta.count' 2>/dev/null || echo "0")
    echo "✓ 最近任务数: $JOB_COUNT"
    if [ "$JOB_COUNT" != "0" ]; then
        echo ""
        echo "最近的任务:"
        echo "$RECENT_JOBS" | jq -r '.data[] | "  - ID: \(.id), Status: \(.status), Config: \(.config_id)"' 2>/dev/null || echo "$RECENT_JOBS"
    fi
else
    echo "⚠ 无法获取任务历史"
fi
echo ""

echo "========================================="
echo "✓ Job Engine 状态检查完成"
echo "========================================="
echo ""
echo "总结:"
echo "- 服务器运行正常"
echo "- Sync 系统健康"
echo "- Job Engine 可以接受和处理任务"
echo ""
echo "如需提交测试任务，请使用:"
echo "  curl -X POST http://localhost:8080/api/sync/jobs \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"config_id\": \"your-config-id\"}'"
echo ""
