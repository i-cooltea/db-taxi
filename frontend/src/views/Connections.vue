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

      <!-- Connection Grid -->
      <div v-else class="connections-grid">
        <ConnectionCard
          v-for="connection in connections"
          :key="connection.config.id"
          :connection="connection"
          :testing="testingConnections.has(connection.config.id)"
          @edit="editConnection"
          @delete="confirmDelete"
          @test="testConnectionStatus"
        />
      </div>
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
import { ref, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { Plus, Plug, AlertCircle, CheckCircle, AlertTriangle } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import ConnectionCard from '../components/ConnectionCard.vue'
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
    if (editingConnection.value) {
      await syncStore.updateConnection(editingConnection.value.config.id, connectionData)
      successMessage.value = '连接已成功更新'
    } else {
      await syncStore.createConnection(connectionData)
      successMessage.value = '连接已成功创建'
    }
    closeModal()
  } catch (err) {
    error.value = (editingConnection.value ? '更新' : '创建') + '连接失败: ' + err.message
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

.connections-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: var(--space-4);
}

@media (max-width: 768px) {
  .connections-grid {
    grid-template-columns: 1fr;
  }
}
</style>
