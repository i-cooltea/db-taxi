<template>
  <div class="container">
    <header class="page-header">
      <div>
        <h1>🔌 连接管理</h1>
        <p>管理远程数据库连接配置</p>
      </div>
      <nav class="nav-links">
        <router-link to="/" class="nav-link">🏠 首页</router-link>
        <router-link to="/sync" class="nav-link">🔄 同步配置</router-link>
        <router-link to="/monitoring" class="nav-link">📊 监控</router-link>
      </nav>
    </header>

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
        <h2>📋 连接列表</h2>
        <button class="btn" @click="showAddModal = true">
          ➕ 添加连接
        </button>
      </div>

      <div v-if="loading && connections.length === 0" class="loading">
        <div class="spinner"></div>
        <p>加载连接列表...</p>
      </div>

      <div v-else-if="connections.length === 0" class="empty-state">
        <div class="empty-icon">🔌</div>
        <h3>还没有数据库连接</h3>
        <p>点击上方"添加连接"按钮创建第一个数据库连接</p>
      </div>

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

    <!-- Add/Edit Connection Modal -->
    <ConnectionModal
      v-if="showAddModal || showEditModal"
      :connection="editingConnection"
      @close="closeModal"
      @save="saveConnection"
    />

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal" @click.self="showDeleteModal = false">
      <div class="modal-content modal-small">
        <div class="modal-header">
          <h2>确认删除</h2>
          <span class="close" @click="showDeleteModal = false">&times;</span>
        </div>
        <p>确定要删除连接 <strong>{{ deletingConnection?.config.name }}</strong> 吗？</p>
        <p class="warning">⚠️ 此操作将同时删除与此连接相关的所有同步配置和任务。</p>
        <div class="form-actions">
          <button class="btn btn-secondary" @click="showDeleteModal = false">
            取消
          </button>
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
import { useSyncStore } from '../stores/syncStore'
import ConnectionCard from '../components/ConnectionCard.vue'
import ConnectionModal from '../components/ConnectionModal.vue'

const syncStore = useSyncStore()

const error = ref(null)
const successMessage = ref(null)
const showAddModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const editingConnection = ref(null)
const deletingConnection = ref(null)
const deleting = ref(false)
const testingConnections = ref(new Set())

const { connections, loading } = syncStore

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
      // Update existing connection
      await syncStore.updateConnection(editingConnection.value.config.id, connectionData)
      successMessage.value = '连接已成功更新'
    } else {
      // Create new connection
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 2rem;
  border-radius: 10px;
  margin-bottom: 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header h1 {
  font-size: 2rem;
  margin-bottom: 0.5rem;
}

.page-header p {
  font-size: 1rem;
  opacity: 0.9;
}

.nav-links {
  display: flex;
  gap: 1rem;
}

.nav-link {
  color: white;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 5px;
  transition: background 0.2s;
}

.nav-link:hover {
  background: rgba(255, 255, 255, 0.2);
}

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

.loading {
  text-align: center;
  padding: 3rem;
}

.spinner {
  border: 4px solid #f3f3f3;
  border-top: 4px solid #667eea;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 0 auto 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty-state {
  text-align: center;
  padding: 3rem;
  color: #666;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.empty-state h3 {
  color: #333;
  margin-bottom: 0.5rem;
}

.connections-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 1.5rem;
}

.modal-small {
  max-width: 500px;
}

.warning {
  color: #856404;
  background: #fff3cd;
  padding: 0.75rem;
  border-radius: 5px;
  border: 1px solid #ffeaa7;
  margin: 1rem 0;
}

.form-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
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
</style>
