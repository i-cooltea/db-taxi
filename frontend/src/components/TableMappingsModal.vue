<template>
  <div class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <div>
          <h2><List :size="24" class="inline-icon" /> 表映射配置</h2>
          <p class="config-name">{{ config.name }}</p>
        </div>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>

      <div class="modal-body">
        <div v-if="!config.tables || config.tables.length === 0" class="empty-state">
          <Database :size="64" class="empty-icon" />
          <h3>暂无表映射</h3>
          <p>此配置还没有添加任何表映射</p>
        </div>

        <div v-else class="tables-list">
          <div class="tables-header">
            <div class="header-item">源表</div>
            <div class="header-item">目标表</div>
            <div class="header-item">同步模式</div>
            <div class="header-item">状态</div>
          </div>

          <div 
            v-for="(table, index) in config.tables" 
            :key="index"
            class="table-row"
            :class="{ disabled: !table.enabled }"
          >
            <div class="table-cell">
              <Database :size="16" class="cell-icon" />
              <span class="table-name">{{ table.source_table }}</span>
            </div>
            <div class="table-cell">
              <ArrowRight :size="16" class="arrow-icon" />
              <span class="table-name">{{ table.target_table }}</span>
            </div>
            <div class="table-cell">
              <span class="sync-mode-badge" :class="getSyncModeClass(table.sync_mode)">
                <component :is="getSyncModeIcon(table.sync_mode)" :size="14" />
                {{ getSyncModeText(table.sync_mode) }}
              </span>
            </div>
            <div class="table-cell">
              <span class="status-badge" :class="table.enabled ? 'enabled' : 'disabled'">
                <component :is="table.enabled ? CheckCircle : XCircle" :size="14" />
                {{ table.enabled ? '已启用' : '已禁用' }}
              </span>
            </div>
          </div>
        </div>

        <div v-if="config.tables && config.tables.length > 0" class="summary">
          <div class="summary-item">
            <span class="summary-label">总表数:</span>
            <span class="summary-value">{{ config.tables.length }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-label">已启用:</span>
            <span class="summary-value enabled">{{ enabledCount }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-label">已禁用:</span>
            <span class="summary-value disabled">{{ disabledCount }}</span>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" @click="$emit('close')">
          关闭
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { 
  List, Database, ArrowRight, CheckCircle, XCircle, 
  RefreshCw, Download 
} from 'lucide-vue-next'

const props = defineProps({
  config: {
    type: Object,
    required: true
  }
})

defineEmits(['close'])

const enabledCount = computed(() => {
  if (!props.config.tables) return 0
  return props.config.tables.filter(t => t.enabled).length
})

const disabledCount = computed(() => {
  if (!props.config.tables) return 0
  return props.config.tables.filter(t => !t.enabled).length
})

function getSyncModeClass(mode) {
  return mode === 'full' ? 'mode-full' : 'mode-incremental'
}

function getSyncModeIcon(mode) {
  return mode === 'full' ? Download : RefreshCw
}

function getSyncModeText(mode) {
  return mode === 'full' ? '全量同步' : '增量同步'
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.2s;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.modal-content {
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 900px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s;
}

@keyframes slideUp {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.5rem 2rem;
  border-bottom: 1px solid #e5e7eb;
}

.modal-header h2 {
  margin: 0;
  color: #667eea;
  font-size: 1.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.config-name {
  margin: 0.5rem 0 0 0;
  color: #6b7280;
  font-size: 0.9rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 2rem;
  color: #6b7280;
  cursor: pointer;
  padding: 0;
  width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s;
}

.close-btn:hover {
  background: #f3f4f6;
  color: #1f2937;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem 2rem;
}

.empty-state {
  text-align: center;
  padding: 3rem 2rem;
  color: #6b7280;
}

.empty-icon {
  color: #667eea;
  opacity: 0.3;
  margin-bottom: 1rem;
}

.empty-state h3 {
  color: #1f2937;
  margin-bottom: 0.5rem;
}

.tables-list {
  background: #f9fafb;
  border-radius: 8px;
  overflow: hidden;
}

.tables-header {
  display: grid;
  grid-template-columns: 2fr 2fr 1.5fr 1fr;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: #667eea;
  color: white;
  font-weight: 600;
  font-size: 0.875rem;
}

.header-item {
  display: flex;
  align-items: center;
}

.table-row {
  display: grid;
  grid-template-columns: 2fr 2fr 1.5fr 1fr;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: white;
  border-bottom: 1px solid #e5e7eb;
  transition: all 0.2s;
}

.table-row:last-child {
  border-bottom: none;
}

.table-row:hover {
  background: #f0f9ff;
}

.table-row.disabled {
  opacity: 0.5;
}

.table-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.cell-icon {
  color: #667eea;
  flex-shrink: 0;
}

.arrow-icon {
  color: #9ca3af;
  flex-shrink: 0;
}

.table-name {
  color: #1f2937;
  font-weight: 500;
  word-break: break-all;
}

.sync-mode-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 500;
}

.mode-full {
  background: #dbeafe;
  color: #1e40af;
}

.mode-incremental {
  background: #f3e8ff;
  color: #6b21a8;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 500;
}

.status-badge.enabled {
  background: #d1fae5;
  color: #065f46;
}

.status-badge.disabled {
  background: #fee2e2;
  color: #991b1b;
}

.summary {
  display: flex;
  gap: 2rem;
  margin-top: 1.5rem;
  padding: 1rem 1.5rem;
  background: #f9fafb;
  border-radius: 8px;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.summary-label {
  color: #6b7280;
  font-size: 0.875rem;
}

.summary-value {
  color: #1f2937;
  font-weight: 600;
  font-size: 1.125rem;
}

.summary-value.enabled {
  color: #059669;
}

.summary-value.disabled {
  color: #dc2626;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1.5rem 2rem;
  border-top: 1px solid #e5e7eb;
}

.btn {
  padding: 0.625rem 1.5rem;
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-secondary {
  background: #f3f4f6;
  color: #1f2937;
}

.btn-secondary:hover {
  background: #e5e7eb;
}

.inline-icon {
  display: inline-block;
  vertical-align: middle;
}

@media (max-width: 768px) {
  .modal-content {
    width: 95%;
    max-height: 90vh;
  }

  .tables-header,
  .table-row {
    grid-template-columns: 1fr;
    gap: 0.5rem;
  }

  .table-cell {
    padding: 0.25rem 0;
  }

  .summary {
    flex-direction: column;
    gap: 0.75rem;
  }
}
</style>
