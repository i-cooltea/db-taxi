<template>
  <div>

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
import { RefreshCw, List, Plus } from 'lucide-vue-next'
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
</style>
