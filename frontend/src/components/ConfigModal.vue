<template>
  <div class="modal" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ config ? '编辑同步配置' : '创建同步配置' }}</h2>
        <span class="close" @click="$emit('close')">&times;</span>
      </div>

      <div v-if="error" class="alert alert-error">{{ error }}</div>

      <div class="tabs">
        <button 
          :class="['tab', { active: activeTab === 'basic' }]"
          @click="activeTab = 'basic'"
        >
          基本配置
        </button>
        <button 
          :class="['tab', { active: activeTab === 'tables' }]"
          @click="activeTab = 'tables'"
        >
          表选择
        </button>
        <button 
          :class="['tab', { active: activeTab === 'options' }]"
          @click="activeTab = 'options'"
        >
          高级选项
        </button>
      </div>

      <form @submit.prevent="handleSubmit">
        <!-- Basic Configuration Tab -->
        <div v-show="activeTab === 'basic'" class="tab-content">
          <div class="form-group">
            <label for="connection">选择连接 *</label>
            <select 
              id="connection" 
              v-model="formData.connection_id" 
              required
              @change="onConnectionChange"
            >
              <option value="">请选择数据库连接</option>
              <option 
                v-for="conn in connections" 
                :key="conn.config.id"
                :value="conn.config.id"
              >
                {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
              </option>
            </select>
            <small>选择要同步的远程数据库连接</small>
          </div>

          <div class="form-group">
            <label for="name">配置名称 *</label>
            <input 
              id="name" 
              v-model="formData.name" 
              type="text" 
              required
            >
            <small>为此同步配置指定一个易于识别的名称</small>
          </div>

          <div class="form-group">
            <label for="sync-mode">默认同步模式 *</label>
            <select id="sync-mode" v-model="formData.sync_mode" required>
              <option value="full">全量同步</option>
              <option value="incremental">增量同步</option>
            </select>
            <small>全量同步会复制所有数据，增量同步只同步变更的数据</small>
          </div>

          <div class="form-group">
            <label for="schedule">同步计划</label>
            <input 
              id="schedule" 
              v-model="formData.schedule" 
              type="text" 
              placeholder="0 */6 * * *"
            >
            <small>使用 Cron 表达式设置定时同步（留空表示手动触发）</small>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="formData.enabled">
              启用此配置
            </label>
          </div>
        </div>

        <!-- Tables Selection Tab -->
        <div v-show="activeTab === 'tables'" class="tab-content">
          <div class="form-group">
            <label>选择要同步的表</label>
            
            <div v-if="loadingTables" class="loading">
              正在加载表列表...
            </div>

            <div v-else-if="!formData.connection_id" class="empty-state">
              <p>请先在"基本配置"中选择数据库连接</p>
            </div>

            <div v-else-if="availableTables.length === 0" class="empty-state">
              <p>此数据库没有可用的表</p>
            </div>

            <div v-else class="table-list">
              <TableItem
                v-for="tableName in availableTables"
                :key="tableName"
                :table-name="tableName"
                :selected="selectedTables.has(tableName)"
                :table-data="selectedTables.get(tableName)"
                @toggle="toggleTable"
                @configure="configureTable"
                @update-mode="updateTableMode"
              />
            </div>
          </div>

          <div v-if="availableTables.length > 0" class="form-group">
            <button type="button" class="btn btn-secondary" @click="selectAllTables">
              全选
            </button>
            <button type="button" class="btn btn-secondary" @click="deselectAllTables">
              取消全选
            </button>
          </div>
        </div>

        <!-- Advanced Options Tab -->
        <div v-show="activeTab === 'options'" class="tab-content">
          <div class="form-row">
            <div class="form-group">
              <label for="batch-size">批处理大小</label>
              <input 
                id="batch-size" 
                v-model.number="formData.options.batch_size" 
                type="number" 
                min="100" 
                max="10000"
              >
              <small>每批处理的记录数</small>
            </div>

            <div class="form-group">
              <label for="max-concurrency">最大并发数</label>
              <input 
                id="max-concurrency" 
                v-model.number="formData.options.max_concurrency" 
                type="number" 
                min="1" 
                max="20"
              >
              <small>同时同步的表数量</small>
            </div>
          </div>

          <div class="form-group">
            <label for="conflict-resolution">冲突解决策略</label>
            <select id="conflict-resolution" v-model="formData.options.conflict_resolution">
              <option value="skip">跳过</option>
              <option value="overwrite">覆盖</option>
              <option value="error">报错</option>
            </select>
            <small>当数据冲突时的处理方式</small>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="formData.options.enable_compression">
              启用数据压缩传输
            </label>
          </div>
        </div>

        <div class="form-actions">
          <button type="button" class="btn btn-secondary" @click="$emit('close')">
            取消
          </button>
          <button type="submit" class="btn" :disabled="saving">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </form>
    </div>

    <TableMappingModal
      v-if="showTableModal"
      :table-name="editingTable"
      :table-data="selectedTables.get(editingTable)"
      @close="showTableModal = false"
      @save="saveTableMapping"
    />
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import TableItem from './TableItem.vue'
import TableMappingModal from './TableMappingModal.vue'

const props = defineProps({
  config: {
    type: Object,
    default: null
  },
  connections: {
    type: Array,
    required: true
  }
})

const emit = defineEmits(['close', 'save'])

const activeTab = ref('basic')
const error = ref(null)
const saving = ref(false)
const loadingTables = ref(false)
const availableTables = ref([])
const selectedTables = ref(new Map())
const showTableModal = ref(false)
const editingTable = ref(null)

const formData = reactive({
  connection_id: '',
  name: '',
  sync_mode: 'full',
  schedule: '',
  enabled: true,
  options: {
    batch_size: 1000,
    max_concurrency: 5,
    conflict_resolution: 'skip',
    enable_compression: true
  }
})

onMounted(() => {
  if (props.config) {
    // Load existing config
    formData.connection_id = props.config.connection_id
    formData.name = props.config.name
    formData.sync_mode = props.config.sync_mode
    formData.schedule = props.config.schedule || ''
    formData.enabled = props.config.enabled
    
    if (props.config.options) {
      Object.assign(formData.options, props.config.options)
    }

    // Load selected tables
    if (props.config.tables) {
      props.config.tables.forEach(table => {
        selectedTables.value.set(table.source_table, {
          target_table: table.target_table,
          sync_mode: table.sync_mode,
          enabled: table.enabled,
          where_clause: table.where_clause || ''
        })
      })
    }

    // Load tables for the connection
    loadTablesForConnection(props.config.connection_id)
  }
})

async function onConnectionChange() {
  if (formData.connection_id) {
    await loadTablesForConnection(formData.connection_id)
  } else {
    availableTables.value = []
  }
}

async function loadTablesForConnection(connectionId) {
  loadingTables.value = true
  error.value = null
  
  try {
    const response = await fetch(`/api/sync/connections/${connectionId}/tables`)
    const result = await response.json()
    
    if (result.success) {
      availableTables.value = result.data || []
    } else {
      throw new Error(result.error || 'Failed to load tables')
    }
  } catch (err) {
    error.value = err.message
  } finally {
    loadingTables.value = false
  }
}

function toggleTable(tableName, selected) {
  if (selected) {
    selectedTables.value.set(tableName, {
      target_table: tableName,
      sync_mode: formData.sync_mode,
      enabled: true,
      where_clause: ''
    })
  } else {
    selectedTables.value.delete(tableName)
  }
}

function updateTableMode(tableName, syncMode) {
  const tableData = selectedTables.value.get(tableName)
  if (tableData) {
    tableData.sync_mode = syncMode
  }
}

function configureTable(tableName) {
  editingTable.value = tableName
  showTableModal.value = true
}

function saveTableMapping(tableName, mappingData) {
  selectedTables.value.set(tableName, mappingData)
  showTableModal.value = false
}

function selectAllTables() {
  availableTables.value.forEach(tableName => {
    if (!selectedTables.value.has(tableName)) {
      selectedTables.value.set(tableName, {
        target_table: tableName,
        sync_mode: formData.sync_mode,
        enabled: true,
        where_clause: ''
      })
    }
  })
}

function deselectAllTables() {
  selectedTables.value.clear()
}

async function handleSubmit() {
  error.value = null

  if (!formData.connection_id) {
    error.value = '请选择数据库连接'
    return
  }

  if (selectedTables.value.size === 0) {
    error.value = '请至少选择一个表进行同步'
    return
  }

  const tables = Array.from(selectedTables.value.entries()).map(([source, data]) => ({
    source_table: source,
    target_table: data.target_table,
    sync_mode: data.sync_mode,
    enabled: data.enabled,
    where_clause: data.where_clause || undefined
  }))

  const configData = {
    connection_id: formData.connection_id,
    name: formData.name,
    tables: tables,
    sync_mode: formData.sync_mode,
    schedule: formData.schedule || undefined,
    enabled: formData.enabled,
    options: formData.options
  }

  saving.value = true
  try {
    await emit('save', configData)
  } catch (err) {
    error.value = err.message
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.modal {
  position: fixed;
  z-index: 1000;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0,0,0,0.5);
  animation: fadeIn 0.3s;
  overflow-y: auto;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.modal-content {
  background-color: white;
  margin: 2% auto;
  padding: 2rem;
  border-radius: 10px;
  width: 90%;
  max-width: 900px;
  animation: slideIn 0.3s;
}

@keyframes slideIn {
  from { transform: translateY(-50px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.modal-header h2 {
  color: #667eea;
  margin: 0;
}

.close {
  color: #aaa;
  font-size: 28px;
  font-weight: bold;
  cursor: pointer;
  line-height: 1;
}

.close:hover {
  color: #000;
}

.tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  border-bottom: 2px solid #e1e5f2;
}

.tab {
  padding: 0.75rem 1.5rem;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1rem;
  color: #666;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: all 0.2s;
}

.tab:hover {
  color: #667eea;
}

.tab.active {
  color: #667eea;
  border-bottom-color: #667eea;
  font-weight: 500;
}

.tab-content {
  min-height: 300px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
  margin-top: 2rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: auto;
}

.table-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid #ddd;
  border-radius: 5px;
  padding: 1rem;
}

.loading {
  text-align: center;
  padding: 2rem;
  color: #666;
}

.empty-state {
  text-align: center;
  padding: 2rem;
  color: #666;
}
</style>
