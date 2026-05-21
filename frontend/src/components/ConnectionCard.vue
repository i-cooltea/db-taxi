<template>
  <div class="connection-card">
    <div class="card-header-section">
      <div class="connection-icon">
        <Database :size="20" />
      </div>
      <div class="connection-info">
        <h3>{{ connection.config.name }}</h3>
        <span :class="['status-badge', statusClass]">
          <span :class="['status-dot', statusClass]"></span>
          {{ statusText }}
        </span>
      </div>
    </div>

    <div class="connection-details">
      <div class="detail-item">
        <span class="detail-label">主机</span>
        <span class="detail-value">{{ connection.config.host }}:{{ connection.config.port }}</span>
      </div>
      <div class="detail-item">
        <span class="detail-label">数据库</span>
        <span class="detail-value">{{ connection.config.database }}</span>
      </div>
      <div class="detail-item">
        <span class="detail-label">用户名</span>
        <span class="detail-value">{{ connection.config.username }}</span>
      </div>
    </div>

    <div class="card-actions">
      <button class="btn btn-sm btn-ghost" @click="$emit('test', connection)" :disabled="testing">
        <Loader2 v-if="testing" :size="14" class="spin" />
        <Activity v-else :size="14" />
        <span>{{ testing ? '测试中...' : '测试' }}</span>
      </button>
      <button class="btn btn-sm btn-ghost" @click="$emit('edit', connection)">
        <Edit2 :size="14" />
        <span>编辑</span>
      </button>
      <button class="btn btn-sm btn-ghost text-error" @click="$emit('delete', connection)">
        <Trash2 :size="14" />
        <span>删除</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Database, Activity, Edit2, Trash2, Loader2 } from 'lucide-vue-next'

const props = defineProps({
  connection: { type: Object, required: true },
  testing: { type: Boolean, default: false }
})

defineEmits(['edit', 'delete', 'test'])

const statusClass = computed(() => {
  if (!props.connection.status) return 'unknown'
  return props.connection.status.connected ? 'connected' : 'disconnected'
})

const statusText = computed(() => {
  if (!props.connection.status) return '未知'
  return props.connection.status.connected ? '已连接' : '未连接'
})
</script>

<style scoped>
.connection-card {
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  transition: box-shadow var(--transition-fast);
}

.connection-card:hover {
  box-shadow: var(--shadow-sm);
}

.card-header-section {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.connection-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--accent-light);
  color: var(--accent-primary);
  border-radius: var(--radius-md);
}

.connection-info h3 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-1);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-size: 12px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
}

.status-badge.connected {
  background: var(--success-light);
  color: var(--success);
}

.status-badge.disconnected {
  background: var(--error-light);
  color: var(--error);
}

.status-badge.unknown {
  background: var(--warning-light);
  color: var(--warning);
}

.connection-details {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-bottom: var(--space-4);
  padding: var(--space-3);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.detail-item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.detail-label {
  color: var(--text-secondary);
}

.detail-value {
  color: var(--text-primary);
  font-weight: 500;
}

.card-actions {
  display: flex;
  gap: var(--space-2);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
