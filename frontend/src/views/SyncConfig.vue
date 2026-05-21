<template>
  <div>
    <!-- Alerts -->
    <div v-if="error" class="alert alert-error">
      <AlertCircle :size="18" />
      <span>{{ error }}</span>
      <button class="alert-close" @click="error = null">&times;</button>
    </div>

    <div v-if="successMessage" class="alert alert-success">
      <CheckCircle :size="18" />
      <span>{{ successMessage }}</span>
      <button class="alert-close" @click="successMessage = null">&times;</button>
    </div>

    <!-- Page Header -->
    <div class="page-header">
      <h1>同步配置</h1>
      <button class="btn btn-primary" @click="showCreateModal">
        <Plus :size="16" />
        <span>创建配置</span>
      </button>
    </div>

    <!-- Content Card -->
    <div class="card">
      <!-- Loading -->
      <div v-if="store.loading" class="loading-state">
        <div class="loading-spinner"></div>
        <p class="text-secondary">正在加载同步配置...</p>
      </div>

      <!-- Error -->
      <div v-else-if="store.error" class="alert alert-error">
        <AlertCircle :size="18" />
        <span>{{ store.error }}</span>
      </div>

      <!-- Empty State -->
      <div v-else-if="store.configs.length === 0" class="empty-state">
        <RefreshCw :size="48" class="empty-state-icon" />
        <h3>还没有配置任何同步任务</h3>
        <p>点击上方"创建配置"按钮开始配置数据库同步</p>
      </div>

      <!-- Config Table -->
      <div v-else class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>源连接</th>
              <th>目标连接</th>
              <th>同步模式</th>
              <th>表数量</th>
              <th>计划</th>
              <th>状态</th>
              <th style="width: 200px;">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="config in store.configs" :key="config.id">
              <td class="font-medium">{{ config.name }}</td>
              <td>{{ getConnection(config.source_connection_id)?.config.name || 'Unknown' }}</td>
              <td>{{ getConnection(config.target_connection_id)?.config.name || 'Unknown' }}</td>
              <td>
                <span :class="['badge', config.sync_mode === 'full' ? 'badge-info' : 'badge-success']">
                  {{ config.sync_mode === 'full' ? '全量同步' : '增量同步' }}
                </span>
              </td>
              <td>{{ config.tables ? config.tables.length : 0 }} 个表</td>
              <td class="text-secondary">{{ config.schedule || '-' }}</td>
              <td>
                <span :class="['badge', config.enabled ? 'badge-success' : 'badge-error']">
                  {{ config.enabled ? '已启用' : '已禁用' }}
                </span>
              </td>
              <td>
                <div class="action-buttons">
                  <button class="btn btn-sm btn-ghost" @click="startSync(config)" title="启动同步">
                    <Play :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost" @click="viewTables(config)" title="查看表">
                    <List :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost" @click="editConfig(config)" title="编辑">
                    <Edit2 :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost" @click="copyConfig(config)" title="复制">
                    <Copy :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost text-error" @click="deleteConfig(config)" title="删除">
                    <Trash2 :size="14" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modals -->
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
import { RefreshCw, Plus, Play, Edit2, Copy, Trash2, List, AlertCircle, CheckCircle } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import ConfigModal from '../components/ConfigModal.vue'
import TableMappingsModal from '../components/TableMappingsModal.vue'

const store = useSyncStore()
const showModal = ref(false)
const editingConfig = ref(null)
const activeConfig = ref(null)
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
  const { id, created_at, updated_at, ...rest } = config
  activeConfig.value = { ...rest, name: `${config.name} - 副本` }
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
      setTimeout(() => { successMessage.value = null }, 5000)
    } catch (err) {
      error.value = `启动同步任务失败: ${err.message}`
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
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-5);
}

.page-header h1 {
  font-size: 24px;
  font-weight: 600;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-7);
  gap: var(--space-4);
}

.action-buttons {
  display: flex;
  gap: var(--space-1);
}

@media (max-width: 768px) {
  .table-container {
    margin: 0 calc(-1 * var(--space-4));
    border-radius: 0;
    border-left: none;
    border-right: none;
  }
}
</style>
