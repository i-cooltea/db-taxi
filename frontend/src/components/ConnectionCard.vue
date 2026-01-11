<template>
  <div class="connection-card">
    <div class="connection-header">
      <div>
        <div class="connection-name">{{ connection.config.name }}</div>
        <div class="connection-info">
          {{ connection.config.host }}:{{ connection.config.port }} / {{ connection.config.database }}
        </div>
        <div class="connection-local">
          本地数据库: {{ connection.config.local_db_name }}
        </div>
      </div>
      <span :class="['status-badge', statusClass]">
        {{ statusText }}
      </span>
    </div>

    <div class="connection-details">
      <div class="detail-row">
        <span class="detail-label">用户名:</span>
        <span class="detail-value">{{ connection.config.username }}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">SSL:</span>
        <span :class="['badge', connection.config.ssl ? 'badge-enabled' : 'badge-disabled']">
          {{ connection.config.ssl ? '已启用' : '未启用' }}
        </span>
      </div>
      <div v-if="connection.status" class="detail-row">
        <span class="detail-label">延迟:</span>
        <span class="detail-value">{{ connection.status.latency_ms }}ms</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">创建时间:</span>
        <span class="detail-value">{{ formatDate(connection.created_at) }}</span>
      </div>
    </div>

    <div class="connection-actions">
      <button class="btn btn-success btn-small" @click="$emit('test', connection)" :disabled="testing">
        {{ testing ? '测试中...' : '🔍 测试连接' }}
      </button>
      <button class="btn btn-secondary btn-small" @click="$emit('edit', connection)">
        ✏️ 编辑
      </button>
      <button class="btn btn-danger btn-small" @click="$emit('delete', connection)">
        🗑️ 删除
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  connection: {
    type: Object,
    required: true
  },
  testing: {
    type: Boolean,
    default: false
  }
})

defineEmits(['edit', 'delete', 'test'])

const statusClass = computed(() => {
  if (!props.connection.status) return 'status-unknown'
  return props.connection.status.connected ? 'status-connected' : 'status-disconnected'
})

const statusText = computed(() => {
  if (!props.connection.status) return '未知'
  return props.connection.status.connected ? '已连接' : '未连接'
})

function formatDate(dateString) {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN')
}
</script>

<style scoped>
.connection-card {
  background: #f8f9ff;
  border: 2px solid #e1e5f2;
  border-radius: 8px;
  padding: 1.5rem;
  transition: all 0.3s ease;
}

.connection-card:hover {
  border-color: #667eea;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.15);
}

.connection-header {
  display: flex;
  justify-content: space-between;
  align-items: start;
  margin-bottom: 1rem;
}

.connection-name {
  font-size: 1.2rem;
  font-weight: bold;
  color: #333;
  margin-bottom: 0.25rem;
}

.connection-info {
  font-size: 0.9rem;
  color: #666;
  margin-bottom: 0.25rem;
}

.connection-local {
  font-size: 0.85rem;
  color: #667eea;
  font-weight: 500;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.85rem;
  font-weight: 500;
}

.status-connected {
  background: #d4edda;
  color: #155724;
}

.status-disconnected {
  background: #f8d7da;
  color: #721c24;
}

.status-unknown {
  background: #fff3cd;
  color: #856404;
}

.connection-details {
  margin: 1rem 0;
  padding: 1rem;
  background: white;
  border-radius: 5px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem 0;
  border-bottom: 1px solid #f0f0f0;
}

.detail-row:last-child {
  border-bottom: none;
}

.detail-label {
  font-weight: 500;
  color: #666;
}

.detail-value {
  color: #333;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.85rem;
  font-weight: 500;
}

.badge-enabled {
  background: #d4edda;
  color: #155724;
}

.badge-disabled {
  background: #f8d7da;
  color: #721c24;
}

.connection-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
  flex-wrap: wrap;
}
</style>
