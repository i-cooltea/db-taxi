<template>
  <div>

    <div v-if="error" class="alert alert-error">
      {{ error }}
      <button class="alert-close" @click="error = null">&times;</button>
    </div>

    <div v-if="successMessage" class="alert alert-success">
      {{ successMessage }}
      <button class="alert-close" @click="successMessage = null">&times;</button>
    </div>

    <div class="card">
      <div class="card-header">
        <h2><List :size="20" class="inline-icon" /> 同步配置列表</h2>
        <button class="btn" @click="showCreateModal">
          <Plus :size="18" /> 创建同步配置
        </button>
      </div>

      <div v-if="store.loading" class="loading">
        正在加载同步配置...
      </div>

      <div v-else-if="store.error" class="alert alert-error">
        {{ store.error }}
      </div>

      <div v-else-if="store.configs.length === 0" class="empty-state">
        <RefreshCw class="empty-icon" :size="64" />
        <h3>还没有配置任何同步任务</h3>
        <p>点击上方"创建同步配置"按钮开始配置数据库同步</p>
      </div>

      <div v-else class="configs-list">
        <table class="config-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>源连接</th>
              <th>目标连接</th>
              <th>同步模式</th>
              <th>表数量</th>
              <th>计划</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="config in store.configs" :key="config.id">
              <td class="config-name-cell">{{ config.name }}</td>
              <td>{{ getConnection(config.source_connection_id)?.config.name || 'Unknown' }}</td>
              <td>{{ getConnection(config.target_connection_id)?.config.name || 'Unknown' }}</td>
              <td>
                <span :class="['badge', config.sync_mode === 'full' ? 'badge-full' : 'badge-incremental']">
                  {{ config.sync_mode === 'full' ? '全量同步' : '增量同步' }}
                </span>
              </td>
              <td>{{ config.tables ? config.tables.length : 0 }} 个表</td>
              <td>{{ config.schedule || '-' }}</td>
              <td>
                <span :class="['status-badge', config.enabled ? 'status-enabled' : 'status-disabled']">
                  {{ config.enabled ? '已启用' : '已禁用' }}
                </span>
              </td>
              <td class="actions-cell">
                <button class="btn btn-success btn-small" @click="startSync(config)" title="启动同步">
                  <Play :size="14" />
                </button>
                <button class="btn btn-info btn-small" @click="viewTables(config)" title="查看表">
                  <List :size="14" />
                </button>
                <button class="btn btn-secondary btn-small" @click="editConfig(config)" title="编辑">
                  <Edit2 :size="14" />
                </button>
                <button class="btn btn-secondary btn-small" @click="copyConfig(config)" title="复制">
                  <Copy :size="14" />
                </button>
                <button class="btn btn-danger btn-small" @click="deleteConfig(config)" title="删除">
                  <Trash2 :size="14" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <ConfigModal
      v-if="showModal"
      :config="activeConfig"
      :connections="store.connections"
      @close="closeModal"
      @save="saveConfig"
    />

    <TableMappingsModal
      v-if="showTablesModal"
      :config="viewingConfig"
      @close="closeTablesModal"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { RefreshCw, List, Plus, Play, Edit2, Copy, Trash2 } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import ConfigModal from '../components/ConfigModal.vue'
import TableMappingsModal from '../components/TableMappingsModal.vue'

const store = useSyncStore()

const showModal = ref(false)
const editingConfig = ref(null) // null 表示创建模式，非空表示编辑已有配置
const activeConfig = ref(null)  // 传给弹窗用于预填表单的数据（编辑或复制）
const error = ref(null)
const successMessage = ref(null)
const showTablesModal = ref(false)
const viewingConfig = ref(null)

onMounted(async () => {
  await store.fetchConfigs()
})

function getConnection(connectionId) {
  return store.connections.find(c => c.config.id === connectionId)
}

function showCreateModal() {
  editingConfig.value = null
  activeConfig.value = null
  showModal.value = true
}

function editConfig(config) {
  editingConfig.value = config
  activeConfig.value = config
  showModal.value = true
}

function copyConfig(config) {
  // 基于当前配置生成一份用于创建的新配置：
  // - 去掉 id 等后端字段
  // - 名称增加「副本」后缀，避免与原配置混淆
  const { id, created_at, updated_at, ...rest } = config
  activeConfig.value = {
    ...rest,
    name: `${config.name} - 副本`
  }
  // 保持 editingConfig 为 null，这样保存时走创建逻辑
  editingConfig.value = null
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingConfig.value = null
}

async function saveConfig(configData) {
  try {
    if (editingConfig.value) {
      await store.updateConfig(editingConfig.value.id, configData)
    } else {
      await store.createConfig(configData)
    }
    closeModal()
  } catch (error) {
    console.error('Failed to save config:', error)
  }
}

async function deleteConfig(config) {
  if (confirm(`确定要删除同步配置 "${config.name}" 吗？\n\n此操作将删除所有相关的表映射配置。`)) {
    try {
      await store.deleteConfig(config.id)
    } catch (error) {
      console.error('Failed to delete config:', error)
    }
  }
}

async function startSync(config) {
  if (confirm(`确定要启动同步任务 "${config.name}" 吗？`)) {
    try {
      error.value = null
      const job = await store.startSync(config.id)
      successMessage.value = `同步任务已成功启动！任务ID: ${job.id}`
      
      // 自动清除成功消息
      setTimeout(() => {
        successMessage.value = null
      }, 5000)
    } catch (err) {
      error.value = `启动同步任务失败: ${err.message}`
      console.error('Failed to start sync:', err)
    }
  }
}

function viewTables(config) {
  viewingConfig.value = config
  showTablesModal.value = true
}

function closeTablesModal() {
  showTablesModal.value = false
  viewingConfig.value = null
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.card h2 {
  color: #667eea;
  font-size: 1.5rem;
}

.configs-list {
  overflow-x: auto;
}

.config-table {
  width: 100%;
  border-collapse: collapse;
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.config-table th,
.config-table td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid #e1e5f2;
}

.config-table th {
  background: #f8f9ff;
  font-weight: 600;
  color: #667eea;
  font-size: 0.9rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.config-table tr:last-child td {
  border-bottom: none;
}

.config-table tr:hover td {
  background: #f8f9ff;
}

.config-name-cell {
  font-weight: 600;
  color: #333;
}

.actions-cell {
  white-space: nowrap;
}

.actions-cell .btn {
  margin-right: 0.25rem;
}

.actions-cell .btn:last-child {
  margin-right: 0;
}

.loading {
  text-align: center;
  padding: 2rem;
  color: #666;
}

.empty-state {
  text-align: center;
  padding: 3rem;
  color: #666;
}

.empty-icon {
  color: #667eea;
  opacity: 0.5;
  margin-bottom: 1rem;
}

.inline-icon {
  display: inline-block;
  vertical-align: middle;
  margin-right: 0.25rem;
}

.alert {
  padding: 1rem;
  border-radius: 5px;
  margin-bottom: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  animation: slideIn 0.3s;
}

.alert-success {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.alert-error {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.alert-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: inherit;
  opacity: 0.7;
  padding: 0;
  line-height: 1;
}

.alert-close:hover {
  opacity: 1;
}

@keyframes slideIn {
  from { transform: translateY(-20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
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

.btn-small {
  padding: 0.35rem 0.5rem;
}
</style>
