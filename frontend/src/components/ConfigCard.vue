<template>
  <div class="config-card">
    <div class="config-header">
      <div>
        <div class="config-name">{{ config.name }}</div>
        <div class="config-connection">
          连接: {{ connection?.config.name || 'Unknown' }}
        </div>
      </div>
      <span :class="['status-badge', statusClass]">
        {{ statusText }}
      </span>
    </div>

    <div class="config-details">
      <div class="detail-row">
        <span class="detail-label">同步模式:</span>
        <span :class="['badge', syncModeClass]">{{ syncModeText }}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">同步表数量:</span>
        <span class="detail-value">{{ tableCount }} 个表</span>
      </div>
      <div v-if="config.schedule" class="detail-row">
        <span class="detail-label">同步计划:</span>
        <span class="detail-value">{{ config.schedule }}</span>
      </div>
      <div v-if="config.options" class="detail-row">
        <span class="detail-label">批处理大小:</span>
        <span class="detail-value">{{ config.options.batch_size || 1000 }}</span>
      </div>
      <div v-if="config.options" class="detail-row">
        <span class="detail-label">最大并发:</span>
        <span class="detail-value">{{ config.options.max_concurrency || 5 }}</span>
      </div>
    </div>

    <div class="config-actions">
      <button class="btn btn-success btn-small" @click="$emit('start', config)">
        <Play :size="14" /> 启动同步
      </button>
      <button class="btn btn-secondary btn-small" @click="$emit('view-tables', config)">
        <List :size="14" /> 查看表
      </button>
      <button class="btn btn-secondary btn-small" @click="$emit('edit', config)">
        <Edit2 :size="14" /> 编辑
      </button>
      <button class="btn btn-danger btn-small" @click="$emit('delete', config)">
        <Trash2 :size="14" /> 删除
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Play, List, Edit2, Trash2 } from 'lucide-vue-next'

const props = defineProps({
  config: {
    type: Object,
    required: true
  },
  connection: {
    type: Object,
    default: null
  }
})

defineEmits(['edit', 'delete', 'start', 'view-tables'])

const statusClass = computed(() => 
  props.config.enabled ? 'status-enabled' : 'status-disabled'
)

const statusText = computed(() => 
  props.config.enabled ? '已启用' : '已禁用'
)

const syncModeClass = computed(() => 
  props.config.sync_mode === 'full' ? 'badge-full' : 'badge-incremental'
)

const syncModeText = computed(() => 
  props.config.sync_mode === 'full' ? '全量同步' : '增量同步'
)

const tableCount = computed(() => 
  props.config.tables ? props.config.tables.length : 0
)
</script>

<style scoped>
.config-card {
  background: #f8f9ff;
  border: 2px solid #e1e5f2;
  border-radius: 8px;
  padding: 1.5rem;
  transition: all 0.3s ease;
}

.config-card:hover {
  border-color: #667eea;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.15);
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: start;
  margin-bottom: 1rem;
}

.config-name {
  font-size: 1.2rem;
  font-weight: bold;
  color: #333;
  margin-bottom: 0.25rem;
}

.config-connection {
  font-size: 0.9rem;
  color: #666;
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

.status-enabled {
  background: #d4edda;
  color: #155724;
}

.status-disabled {
  background: #f8d7da;
  color: #721c24;
}

.config-details {
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

.badge-full {
  background: #e3f2fd;
  color: #1976d2;
}

.badge-incremental {
  background: #f3e5f5;
  color: #7b1fa2;
}

.config-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
  flex-wrap: wrap;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}
</style>
