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
        <div v-show="activeTab === 'tables'" class="tab-content tables-tab">
          <div class="tables-layout">
            <!-- Left: Table Selection List -->
            <div class="tables-selection">
              <div class="selection-header">
                <label>选择要同步的表</label>
                <div class="selection-actions">
                  <button 
                    type="button" 
                    class="btn-link" 
                    @click="selectAllTables"
                    :disabled="!formData.connection_id || availableTables.length === 0"
                  >
                    全选
                  </button>
                  <button 
                    type="button" 
                    class="btn-link" 
                    @click="deselectAllTables"
                    :disabled="selectedTables.size === 0"
                  >
                    清空
                  </button>
                </div>
              </div>

              <div v-if="loadingTables" class="loading-compact">
                <div class="spinner-small"></div>
                <span>加载中...</span>
              </div>

              <div v-else-if="!formData.connection_id" class="empty-state-compact">
                <p>请先选择数据库连接</p>
              </div>

              <div v-else-if="availableTables.length === 0" class="empty-state-compact">
                <p>此数据库没有可用的表</p>
              </div>

              <div v-else class="table-list-compact">
                <div 
                  v-for="tableName in availableTables"
                  :key="tableName"
                  class="table-item-compact"
                  :class="{ selected: selectedTables.has(tableName), active: editingTable === tableName }"
                  @click="selectAndEditTable(tableName)"
                >
                  <input 
                    type="checkbox" 
                    :checked="selectedTables.has(tableName)"
                    @click.stop="toggleTable(tableName, !selectedTables.has(tableName))"
                  >
                  <span class="table-name">{{ tableName }}</span>
                  <span v-if="selectedTables.has(tableName)" class="table-mode-badge">
                    {{ selectedTables.get(tableName).sync_mode === 'full' ? '全量' : '增量' }}
                  </span>
                </div>
              </div>

              <div class="selection-summary">
                已选择 <strong>{{ selectedTables.size }}</strong> / {{ availableTables.length }} 个表
              </div>
            </div>

            <!-- Right: Table Configuration Panel -->
            <div class="table-config-panel">
              <div v-if="!editingTable" class="config-placeholder">
                <p>← 点击左侧表名进行配置</p>
              </div>

              <div v-else class="config-form">
                <h3>{{ editingTable }}</h3>

                <div class="form-group-compact">
                  <label>目标表名</label>
                  <input 
                    v-model="currentTableConfig.target_table" 
                    type="text"
                    placeholder="留空则使用源表名"
                  >
                </div>

                <div class="form-group-compact">
                  <label>同步模式</label>
                  <div class="radio-group">
                    <label class="radio-label">
                      <input 
                        type="radio" 
                        v-model="currentTableConfig.sync_mode" 
                        value="full"
                      >
                      <span>全量同步</span>
                    </label>
                    <label class="radio-label">
                      <input 
                        type="radio" 
                        v-model="currentTableConfig.sync_mode" 
                        value="incremental"
                      >
                      <span>增量同步</span>
                    </label>
                  </div>
                </div>

                <div class="form-group-compact">
                  <label>WHERE 条件</label>
                  <textarea 
                    v-model="currentTableConfig.where_clause" 
                    rows="3"
                    placeholder="例如: status = 'active' AND created_at > '2024-01-01'"
                  ></textarea>
                  <small>可选：用于过滤要同步的数据</small>
                </div>

                <div class="form-group-compact">
                  <label class="checkbox-label">
                    <input type="checkbox" v-model="currentTableConfig.enabled">
                    启用此表同步
                  </label>
                </div>

                <div class="config-actions">
                  <button type="button" class="btn-sm btn-secondary" @click="cancelTableEdit">
                    取消
                  </button>
                  <button type="button" class="btn-sm btn-primary" @click="saveCurrentTableConfig">
                    保存配置
                  </button>
                </div>
              </div>
            </div>
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
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from 'vue'

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
const editingTable = ref(null)
const currentTableConfig = ref({
  target_table: '',
  sync_mode: 'full',
  enabled: true,
  where_clause: ''
})

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
    // Auto-select for editing
    selectAndEditTable(tableName)
  } else {
    selectedTables.value.delete(tableName)
    if (editingTable.value === tableName) {
      editingTable.value = null
    }
  }
}

function selectAndEditTable(tableName) {
  editingTable.value = tableName
  
  if (selectedTables.value.has(tableName)) {
    // Load existing config
    const tableData = selectedTables.value.get(tableName)
    currentTableConfig.value = { ...tableData }
  } else {
    // Create new config
    currentTableConfig.value = {
      target_table: tableName,
      sync_mode: formData.sync_mode,
      enabled: true,
      where_clause: ''
    }
    selectedTables.value.set(tableName, { ...currentTableConfig.value })
  }
}

function saveCurrentTableConfig() {
  if (editingTable.value) {
    selectedTables.value.set(editingTable.value, { ...currentTableConfig.value })
    editingTable.value = null
  }
}

function cancelTableEdit() {
  editingTable.value = null
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
  min-height: 400px;
}

.tables-tab {
  min-height: 500px;
}

.tables-layout {
  display: grid;
  grid-template-columns: 350px 1fr;
  gap: 1.5rem;
  height: 500px;
}

/* Left Panel: Table Selection */
.tables-selection {
  display: flex;
  flex-direction: column;
  border: 1px solid #e1e5f2;
  border-radius: 8px;
  overflow: hidden;
}

.selection-header {
  padding: 0.75rem 1rem;
  background: #f8f9fa;
  border-bottom: 1px solid #e1e5f2;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.selection-header label {
  margin: 0;
  font-weight: 600;
  font-size: 0.875rem;
  color: #333;
}

.selection-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-link {
  background: none;
  border: none;
  color: #667eea;
  cursor: pointer;
  font-size: 0.8125rem;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  transition: all 0.2s;
}

.btn-link:hover:not(:disabled) {
  background: #667eea;
  color: white;
}

.btn-link:disabled {
  color: #ccc;
  cursor: not-allowed;
}

.table-list-compact {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
}

.table-item-compact {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 0.875rem;
}

.table-item-compact:hover {
  background: #f0f9ff;
}

.table-item-compact.selected {
  background: #e0f2fe;
}

.table-item-compact.active {
  background: #667eea;
  color: white;
}

.table-item-compact.active .table-mode-badge {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.table-item-compact input[type="checkbox"] {
  width: auto;
  margin: 0;
  cursor: pointer;
}

.table-name {
  flex: 1;
  font-weight: 500;
}

.table-mode-badge {
  font-size: 0.75rem;
  padding: 0.125rem 0.5rem;
  background: #667eea;
  color: white;
  border-radius: 10px;
}

.loading-compact {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
  color: #666;
  font-size: 0.875rem;
}

.spinner-small {
  width: 16px;
  height: 16px;
  border: 2px solid #f3f3f3;
  border-top: 2px solid #667eea;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty-state-compact {
  padding: 2rem 1rem;
  text-align: center;
  color: #999;
  font-size: 0.875rem;
}

.selection-summary {
  padding: 0.75rem 1rem;
  background: #f8f9fa;
  border-top: 1px solid #e1e5f2;
  font-size: 0.8125rem;
  color: #666;
}

.selection-summary strong {
  color: #667eea;
  font-weight: 600;
}

/* Right Panel: Table Configuration */
.table-config-panel {
  border: 1px solid #e1e5f2;
  border-radius: 8px;
  padding: 1.5rem;
  overflow-y: auto;
}

.config-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #999;
  font-size: 0.875rem;
}

.config-form h3 {
  margin: 0 0 1.5rem 0;
  color: #667eea;
  font-size: 1.125rem;
  padding-bottom: 0.75rem;
  border-bottom: 2px solid #e1e5f2;
}

.form-group-compact {
  margin-bottom: 1.25rem;
}

.form-group-compact label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  font-size: 0.875rem;
  color: #333;
}

.form-group-compact input[type="text"],
.form-group-compact textarea {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.875rem;
  transition: border-color 0.2s;
}

.form-group-compact input[type="text"]:focus,
.form-group-compact textarea:focus {
  outline: none;
  border-color: #667eea;
}

.form-group-compact textarea {
  resize: vertical;
  font-family: 'Courier New', monospace;
}

.form-group-compact small {
  display: block;
  margin-top: 0.25rem;
  color: #999;
  font-size: 0.75rem;
}

.radio-group {
  display: flex;
  gap: 1rem;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-weight: normal;
}

.radio-label input[type="radio"] {
  width: auto;
  margin: 0;
  cursor: pointer;
}

.config-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid #e1e5f2;
}

.btn-sm {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-sm.btn-secondary {
  background: #f3f4f6;
  color: #1f2937;
}

.btn-sm.btn-secondary:hover {
  background: #e5e7eb;
}

.btn-sm.btn-primary {
  background: #667eea;
  color: white;
}

.btn-sm.btn-primary:hover {
  background: #5568d3;
}

@media (max-width: 768px) {
  .tables-layout {
    grid-template-columns: 1fr;
    height: auto;
  }

  .tables-selection {
    max-height: 300px;
  }
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
