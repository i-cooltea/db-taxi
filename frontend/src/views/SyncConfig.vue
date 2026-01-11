<template>
  <div class="container">
    <header class="page-header">
      <div>
        <h1>🔄 同步配置</h1>
        <p>管理数据库同步配置和表映射</p>
      </div>
      <nav class="nav-links">
        <router-link to="/" class="nav-link">🏠 首页</router-link>
        <router-link to="/connections" class="nav-link">🔌 连接管理</router-link>
        <router-link to="/config" class="nav-link">⚙️ 配置管理</router-link>
      </nav>
    </header>

    <div class="card">
      <div class="card-header">
        <h2>📋 同步配置列表</h2>
        <button class="btn" @click="showCreateModal">
          ➕ 创建同步配置
        </button>
      </div>

      <div v-if="store.loading" class="loading">
        正在加载同步配置...
      </div>

      <div v-else-if="store.error" class="alert alert-error">
        {{ store.error }}
      </div>

      <div v-else-if="store.configs.length === 0" class="empty-state">
        <div class="empty-state-icon">📋</div>
        <h3>还没有配置任何同步任务</h3>
        <p>点击上方"创建同步配置"按钮开始配置数据库同步</p>
      </div>

      <div v-else class="configs-grid">
        <ConfigCard
          v-for="config in store.configs"
          :key="config.id"
          :config="config"
          :connection="getConnection(config.connection_id)"
          @edit="editConfig"
          @delete="deleteConfig"
          @start="startSync"
          @view-tables="viewTables"
        />
      </div>
    </div>

    <ConfigModal
      v-if="showModal"
      :config="editingConfig"
      :connections="store.connections"
      @close="closeModal"
      @save="saveConfig"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSyncStore } from '../stores/syncStore'
import ConfigCard from '../components/ConfigCard.vue'
import ConfigModal from '../components/ConfigModal.vue'

const router = useRouter()
const store = useSyncStore()

const showModal = ref(false)
const editingConfig = ref(null)

onMounted(async () => {
  await store.fetchConfigs()
})

function getConnection(connectionId) {
  return store.connections.find(c => c.config.id === connectionId)
}

function showCreateModal() {
  editingConfig.value = null
  showModal.value = true
}

function editConfig(config) {
  editingConfig.value = config
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
  if (confirm('确定要启动同步任务吗？')) {
    try {
      const job = await store.startSync(config.id)
      router.push(`/monitoring?job_id=${job.id}`)
    } catch (error) {
      console.error('Failed to start sync:', error)
    }
  }
}

function viewTables(config) {
  const tableList = config.tables.map(t => 
    `${t.source_table} → ${t.target_table} (${t.sync_mode})`
  ).join('\n')
  alert(`配置: ${config.name}\n\n同步表列表:\n${tableList || '(无)'}`)
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

.configs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 1.5rem;
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

.empty-state-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}
</style>
