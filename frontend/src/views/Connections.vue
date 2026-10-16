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
      <h1>连接管理</h1>
      <button class="btn btn-primary" @click="showAddModal = true">
        <Plus :size="16" />
        <span>添加连接</span>
      </button>
    </div>

    <!-- Content Card -->
    <div class="card">
      <!-- Loading State -->
      <div v-if="loading && connections.length === 0" class="loading-state">
        <div class="loading-spinner"></div>
        <p class="text-secondary">加载连接列表...</p>
      </div>

      <!-- Empty State -->
      <div v-else-if="connections.length === 0" class="empty-state">
        <Plug :size="48" class="empty-state-icon" />
        <h3>还没有数据库连接</h3>
        <p>点击上方"添加连接"按钮创建第一个数据库连接</p>
      </div>

      <template v-else>
        <!-- Filter Bar -->
        <div class="filter-bar">
          <input
            v-model="filters.name"
            type="text"
            class="form-control filter-name"
            placeholder="搜索连接名称..."
          >
          <button v-if="hasActiveFilters" class="btn btn-ghost filter-reset" @click="resetFilters">
            <X :size="14" />
            <span>重置</span>
          </button>
        </div>

        <!-- No Matching Connections -->
        <div v-if="filteredConnections.length === 0" class="empty-state">
          <Search :size="48" class="empty-state-icon" />
          <h3>没有匹配的连接</h3>
          <p>请调整过滤条件后重试</p>
        </div>

        <!-- Connection Table -->
        <div v-else class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>名称</th>
                <th>主机</th>
                <th>用户名</th>
                <th>状态</th>
                <th style="width: 160px;">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="connection in filteredConnections" :key="connection.config.id">
                <td class="font-medium">{{ connection.config.name }}</td>
                <td>{{ connection.config.host }}:{{ connection.config.port }}</td>
                <td>{{ connection.config.username }}</td>
                <td>
                  <span :class="['badge', statusClass(connection)]">
                    {{ statusText(connection) }}
                  </span>
                </td>
                <td>
                  <div class="action-buttons">
                    <button
                      class="btn btn-sm btn-ghost"
                      @click="testConnectionStatus(connection)"
                      :disabled="testingConnections.has(connection.config.id)"
                      title="测试连接"
                    >
                      <Loader2 v-if="testingConnections.has(connection.config.id)" :size="14" class="spin" />
                      <Activity v-else :size="14" />
                    </button>
                    <button class="btn btn-sm btn-ghost" @click="editConnection(connection)" title="编辑">
                      <Edit2 :size="14" />
                    </button>
                    <button class="btn btn-sm btn-ghost" @click="copyConnection(connection)" title="复制">
                      <Copy :size="14" />
                    </button>
                    <button class="btn btn-sm btn-ghost text-error" @click="confirmDelete(connection)" title="删除">
                      <Trash2 :size="14" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>

    <!-- Modals -->
    <ConnectionModal
      v-if="showAddModal || showEditModal"
      :connection="editingConnection"
      @close="closeModal"
      @save="saveConnection"
    />

    <!-- Delete Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click.self="showDeleteModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h2>确认删除</h2>
          <button class="modal-close" @click="showDeleteModal = false">&times;</button>
        </div>
        <div class="modal-body">
          <p>确定要删除连接 <strong>{{ deletingConnection?.config.name }}</strong> 吗？</p>
          <div class="alert alert-warning" style="margin-top: 16px;">
            <AlertTriangle :size="16" />
            <span>此操作将同时删除与此连接相关的所有同步配置和任务。</span>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="showDeleteModal = false">取消</button>
          <button class="btn btn-danger" @click="deleteConnection" :disabled="deleting">
            {{ deleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { Plus, Plug, AlertCircle, CheckCircle, AlertTriangle, Search, X, Loader2, Activity, Edit2, Copy, Trash2 } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import ConnectionModal from '../components/ConnectionModal.vue'

const syncStore = useSyncStore()
const { connections, loading } = storeToRefs(syncStore)

const error = ref(null)
const successMessage = ref(null)
const showAddModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const editingConnection = ref(null)
const deletingConnection = ref(null)
const deleting = ref(false)
const testingConnections = ref(new Set())
const filters = ref({ name: '' })

const hasActiveFilters = computed(() => filters.value.name.trim() !== '')

const filteredConnections = computed(() => {
  const keyword = filters.value.name.trim().toLowerCase()
  if (!keyword) return connections.value
  return connections.value.filter(c => c.config.name.toLowerCase().includes(keyword))
})

function resetFilters() {
  filters.value.name = ''
}

function statusClass(connection) {
  if (!connection.status) return 'badge-warning'
  return connection.status.connected ? 'badge-success' : 'badge-error'
}

function statusText(connection) {
  if (!connection.status) return '未知'
  return connection.status.connected ? '已连接' : '未连接'
}

onMounted(async () => {
  try {
    await syncStore.fetchConnections()
  } catch (err) {
    error.value = '加载连接列表失败: ' + err.message
  }
})

function editConnection(connection) {
  editingConnection.value = connection
  showEditModal.value = true
}

function copyConnection(connection) {
  const { id, created_at, updated_at, ...rest } = connection.config
  editingConnection.value = { config: { ...rest, name: `${connection.config.name} - 副本` } }
  showAddModal.value = true
}

function confirmDelete(connection) {
  deletingConnection.value = connection
  showDeleteModal.value = true
}

async function deleteConnection() {
  if (!deletingConnection.value) return
  deleting.value = true
  error.value = null
  try {
    await syncStore.deleteConnection(deletingConnection.value.config.id)
    successMessage.value = '连接已成功删除'
    showDeleteModal.value = false
    deletingConnection.value = null
  } catch (err) {
    error.value = '删除连接失败: ' + err.message
  } finally {
    deleting.value = false
  }
}

async function testConnectionStatus(connection) {
  const connectionId = connection.config.id
  testingConnections.value.add(connectionId)
  error.value = null
  try {
    const status = await syncStore.testConnection(connectionId)
    if (status.connected) {
      successMessage.value = `连接 ${connection.config.name} 测试成功 (延迟: ${status.latency_ms}ms)`
    } else {
      error.value = `连接 ${connection.config.name} 测试失败: ${status.error || '未知错误'}`
    }
  } catch (err) {
    error.value = `测试连接失败: ${err.message}`
  } finally {
    testingConnections.value.delete(connectionId)
  }
}

async function saveConnection(connectionData) {
  error.value = null
  try {
    if (showEditModal.value && editingConnection.value) {
      await syncStore.updateConnection(editingConnection.value.config.id, connectionData)
      successMessage.value = '连接已成功更新'
    } else {
      await syncStore.createConnection(connectionData)
      successMessage.value = '连接已成功创建'
    }
    closeModal()
  } catch (err) {
    error.value = (showEditModal.value ? '更新' : '创建') + '连接失败: ' + err.message
    throw err
  }
}

function closeModal() {
  showAddModal.value = false
  showEditModal.value = false
  editingConnection.value = null
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

.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.filter-name {
  flex: 1 1 240px;
}

.filter-reset {
  flex-shrink: 0;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
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
