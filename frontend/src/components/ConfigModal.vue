<template>
  <div class="modal" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ config && config.id ? '编辑同步配置' : '创建同步配置' }}</h2>
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
          <div class="sync-visual">
            <div class="sync-node">
              <Database class="db-icon db-icon-source" :size="26" />
              <div class="sync-node-text">
                <div class="sync-node-main">
                  {{ sourceConnection?.config.host || '源连接' }}
                </div>
                <div class="sync-node-sub">
                  {{ formData.source_database || '请选择源数据库' }}
                </div>
              </div>
            </div>

            <div class="sync-arrow">
              <ArrowRight :size="20" />
            </div>

            <div class="sync-node">
              <Database class="db-icon db-icon-target" :size="26" />
              <div class="sync-node-text">
                <div class="sync-node-main">
                  {{ targetConnection?.config.host || '目标' }}
                </div>
                <div class="sync-node-sub">
                  {{ formData.target_database || '目标数据库' }}
                </div>
              </div>
            </div>
          </div>

          <div class="connection-panels">
            <!-- 源连接方框 -->
            <div class="connection-panel connection-panel-source">
              <div class="connection-panel-header">
                <Database class="db-icon-source" :size="20" />
                <span>数据来源</span>
              </div>
              <div class="form-group">
                <label for="source-connection">源连接（数据来源）*</label>
                <select
                  id="source-connection"
                  class="form-control"
                  v-model="formData.source_connection_id"
                  required
                  @change="onSourceConnectionChange"
                >
                  <option value="">请选择源数据库连接</option>
                  <option
                    v-for="conn in connections"
                    :key="conn.config.id"
                    :value="conn.config.id"
                  >
                    {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
                  </option>
                </select>
                <small>选择数据来源的数据库连接</small>
              </div>
              <div class="form-group">
                <label for="source-database">源数据库 *</label>
                <select
                  id="source-database"
                  class="form-control"
                  v-model="formData.source_database"
                  required
                  :disabled="!formData.source_connection_id || loadingDatabases"
                  @change="onSourceDatabaseChange"
                >
                  <option value="">
                    {{ loadingDatabases ? '加载数据库列表...' : '请选择源数据库' }}
                  </option>
                  <option v-for="db in availableDatabases" :key="db" :value="db">
                    {{ db }}
                  </option>
                </select>
                <small>选择要作为数据来源的数据库（选择源连接后会自动加载）</small>
              </div>
            </div>

            <!-- 箭头 -->
            <div class="connection-arrow">
              <ArrowRight :size="28" />
            </div>

            <!-- 目标连接方框 -->
            <div class="connection-panel connection-panel-target">
              <div class="connection-panel-header">
                <Database class="db-icon-target" :size="20" />
                <span>数据目标</span>
              </div>
              <div class="form-group">
                <label for="target-connection">目标连接（数据目标）*</label>
                <select
                  id="target-connection"
                  class="form-control"
                  v-model="formData.target_connection_id"
                  required
                >
                  <option value="">请选择目标数据库连接</option>
                  <option
                    v-for="conn in connections"
                    :key="conn.config.id"
                    :value="conn.config.id"
                  >
                    {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
                  </option>
                </select>
                <small>选择数据同步到的目标数据库连接</small>
                <small style="color: var(--error);">⚠️ 警告：数据将会被覆盖</small>
              </div>
              <div class="form-group">
                <label for="target-database">目标数据库名称 *</label>
                <input
                  id="target-database"
                  class="form-control"
                  v-model="formData.target_database"
                  type="text"
                  required
                  placeholder="默认与源数据库同名，可修改"
                  @input="targetDatabaseTouched = true"
                >
                <small>默认与源数据库同名；可修改。若目标数据库不存在，将自动创建（不会报错）。</small>
              </div>
            </div>
          </div>

          <div class="form-group">
            <label for="name">配置名称 *</label>
            <input
              id="name"
              class="form-control"
              v-model="formData.name"
              type="text"
              required
            >
            <small>为此同步配置指定一个易于识别的名称</small>
          </div>

          <div class="form-group">
            <label for="sync-mode">默认同步模式 *</label>
            <select id="sync-mode" class="form-control" v-model="formData.sync_mode" required>
              <option value="full">全量同步</option>
              <option value="incremental">增量同步</option>
            </select>
            <small>全量同步会复制所有数据，增量同步只同步变更的数据</small>
          </div>

          <div class="form-group">
            <label for="schedule">同步计划</label>
            <input
              id="schedule"
              class="form-control"
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
          <div class="tables-layout tables-layout-three">
            <!-- Leftmost: 已选表顺序（始终展示） -->
            <div class="selected-order-panel">
              <div class="selected-order-header">
                <label>已选表顺序</label>
                <small>拖动可调整同步顺序</small>
              </div>
              <div v-if="selectedTablesList.length === 0" class="selected-order-empty">
                <p>在中间勾选表后，将在此显示同步顺序</p>
              </div>
              <div
                v-else
                class="selected-order-list"
                @dragover.prevent
              >
                <div
                  v-for="(item, index) in selectedTablesList"
                  :key="item.source_table"
                  class="selected-order-item"
                  :class="{ active: editingTable === item.source_table, dragging: dragFromIndex === index }"
                  :title="item.source_table"
                  draggable="true"
                  @click="selectAndEditTable(item.source_table)"
                  @dragstart="onDragStartSelected(index, $event)"
                  @dragover.prevent="dragOverIndex = index"
                  @dragleave="dragOverIndex = null"
                  @drop="onDropSelected(index, $event)"
                  @dragend="dragFromIndex = null; dragOverIndex = null"
                >
                  <span class="drag-handle" title="拖动排序">⋮⋮</span>
                  <span class="table-name">{{ item.source_table }}</span>
                  <span class="table-mode-badge">{{ item.sync_mode === 'full' ? '全量' : '增量' }}</span>
                </div>
              </div>
            </div>

            <!-- Middle: Table Selection List -->
            <div class="tables-selection">
              <div class="selection-header">
                <label>选择要同步的表</label>
                <div class="selection-actions">
                  <button 
                    type="button" 
                    class="btn-link" 
                    @click="selectAllTables"
                    :disabled="!formData.source_connection_id || !formData.source_database || availableTables.length === 0"
                  >
                    全选
                  </button>
                  <button 
                    type="button" 
                    class="btn-link" 
                    @click="deselectAllTables"
                    :disabled="selectedTablesList.length === 0"
                  >
                    清空
                  </button>
                </div>
              </div>

              <div v-if="loadingTables" class="loading-compact">
                <div class="spinner-small"></div>
                <span>加载中...</span>
              </div>

              <div v-else-if="!formData.source_connection_id || !formData.source_database" class="empty-state-compact">
                <p>请先选择源连接与源数据库</p>
              </div>

              <div v-else-if="availableTables.length === 0" class="empty-state-compact">
                <p>此数据库没有可用的表</p>
              </div>

              <div v-else class="table-list-compact">
                <div 
                  v-for="tableName in availableTables"
                  :key="tableName"
                  class="table-item-compact"
                  :class="{ selected: isTableSelected(tableName), active: editingTable === tableName }"
                  @click="selectAndEditTable(tableName)"
                >
                  <input 
                    type="checkbox" 
                    :checked="isTableSelected(tableName)"
                    @click.stop="toggleTable(tableName, !isTableSelected(tableName))"
                  >
                  <span class="table-name">{{ tableName }}</span>
                  <span v-if="isTableSelected(tableName)" class="table-mode-badge">
                    {{ getTableConfig(tableName).sync_mode === 'full' ? '全量' : '增量' }}
                  </span>
                </div>
              </div>

              <div class="selection-summary">
                已选择 <strong>{{ selectedTablesList.length }}</strong> / {{ availableTables.length }} 个表
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
                class="form-control"
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
                class="form-control"
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
            <select id="conflict-resolution" class="form-control" v-model="formData.options.conflict_resolution">
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
          <button type="submit" class="btn btn-primary" :disabled="saving">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted, computed } from 'vue'
import { Database, ArrowRight } from 'lucide-vue-next'

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
const loadingDatabases = ref(false)
const availableDatabases = ref([])
const availableTables = ref([])
// 已选表列表（数组保证顺序，用于同步执行顺序；可拖动排序）
const selectedTablesList = ref([])
const editingTable = ref(null)
const dragFromIndex = ref(null)
const dragOverIndex = ref(null)
const currentTableConfig = ref({
  source_table: '',
  target_table: '',
  sync_mode: 'full',
  enabled: true,
  where_clause: ''
})

const formData = reactive({
  source_connection_id: '',
  target_connection_id: '',
  source_database: '',
  target_database: '',
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

const targetDatabaseTouched = ref(false)

const sourceConnection = computed(() =>
  props.connections.find(c => c.config.id === formData.source_connection_id)
)

const targetConnection = computed(() =>
  props.connections.find(c => c.config.id === formData.target_connection_id)
)

onMounted(() => {
  if (props.config) {
    // Load existing config
    formData.source_connection_id = props.config.source_connection_id
    formData.target_connection_id = props.config.target_connection_id
    formData.source_database = props.config.source_database || ''
    formData.target_database = props.config.target_database || ''
    targetDatabaseTouched.value = !!formData.target_database && formData.target_database !== formData.source_database
    formData.name = props.config.name
    formData.sync_mode = props.config.sync_mode
    formData.schedule = props.config.schedule || ''
    formData.enabled = props.config.enabled
    
    if (props.config.options) {
      Object.assign(formData.options, props.config.options)
    }

    // Load selected tables (preserve API order = sync order)
    if (props.config.tables) {
      selectedTablesList.value = props.config.tables.map(t => ({
        source_table: t.source_table,
        target_table: t.target_table || t.source_table,
        sync_mode: t.sync_mode || 'full',
        enabled: t.enabled !== false,
        where_clause: t.where_clause || '',
        id: t.id
      }))
    }

    // Load databases for source connection, then load tables for selected source database
    loadDatabasesForConnection(props.config.source_connection_id).then(() => {
      if (formData.source_database) {
        loadTablesForConnection(props.config.source_connection_id, formData.source_database)
      }
    })
  }
})

function isTableSelected(tableName) {
  return selectedTablesList.value.some(t => t.source_table === tableName)
}

function getTableConfig(tableName) {
  const t = selectedTablesList.value.find(x => x.source_table === tableName)
  return t || { sync_mode: 'full' }
}

async function onSourceConnectionChange() {
  if (formData.source_connection_id) {
    // Reset selections when source connection changes
    availableDatabases.value = []
    availableTables.value = []
    selectedTablesList.value = []
    editingTable.value = null
    formData.source_database = ''
    if (!targetDatabaseTouched.value) {
      formData.target_database = ''
    }

    await loadDatabasesForConnection(formData.source_connection_id)
  } else {
    availableDatabases.value = []
    availableTables.value = []
  }
}

async function onSourceDatabaseChange() {
  if (formData.source_connection_id && formData.source_database) {
    // default target database name to source database if user hasn't edited it
    if (!targetDatabaseTouched.value) {
      formData.target_database = formData.source_database
    }
    availableTables.value = []
    selectedTablesList.value = []
    editingTable.value = null
    await loadTablesForConnection(formData.source_connection_id, formData.source_database)
  } else {
    availableTables.value = []
  }
}

async function loadDatabasesForConnection(connectionId) {
  loadingDatabases.value = true
  error.value = null
  try {
    const response = await fetch(`/api/sync/connections/${connectionId}/databases`)
    const result = await response.json()
    if (result.success) {
      availableDatabases.value = result.data || []
    } else {
      throw new Error(result.error || 'Failed to load databases')
    }
  } catch (err) {
    error.value = err.message
  } finally {
    loadingDatabases.value = false
  }
}

async function loadTablesForConnection(connectionId, database) {
  loadingTables.value = true
  error.value = null
  
  try {
    const response = await fetch(`/api/sync/connections/${connectionId}/tables?database=${encodeURIComponent(database)}`)
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
    if (!isTableSelected(tableName)) {
      selectedTablesList.value.push({
        source_table: tableName,
        target_table: tableName,
        sync_mode: formData.sync_mode,
        enabled: true,
        where_clause: ''
      })
    }
    selectAndEditTable(tableName)
  } else {
    selectedTablesList.value = selectedTablesList.value.filter(t => t.source_table !== tableName)
    if (editingTable.value === tableName) {
      editingTable.value = null
    }
  }
}

function onDragStartSelected(index, e) {
  dragFromIndex.value = index
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', String(index))
}

function onDropSelected(toIndex, e) {
  e.preventDefault()
  const from = dragFromIndex.value
  if (from == null) return
  if (from === toIndex) return
  const list = [...selectedTablesList.value]
  const [item] = list.splice(from, 1)
  list.splice(toIndex, 0, item)
  selectedTablesList.value = list
  dragFromIndex.value = null
}

function selectAndEditTable(tableName) {
  editingTable.value = tableName
  const item = selectedTablesList.value.find(t => t.source_table === tableName)
  if (item) {
    currentTableConfig.value = {
      source_table: item.source_table,
      target_table: item.target_table,
      sync_mode: item.sync_mode,
      enabled: item.enabled,
      where_clause: item.where_clause || ''
    }
  } else {
    currentTableConfig.value = {
      source_table: tableName,
      target_table: tableName,
      sync_mode: formData.sync_mode,
      enabled: true,
      where_clause: ''
    }
  }
}

function saveCurrentTableConfig() {
  if (editingTable.value) {
    const item = selectedTablesList.value.find(t => t.source_table === editingTable.value)
    if (item) {
      item.target_table = currentTableConfig.value.target_table || currentTableConfig.value.source_table
      item.sync_mode = currentTableConfig.value.sync_mode
      item.enabled = currentTableConfig.value.enabled
      item.where_clause = currentTableConfig.value.where_clause || ''
    }
    editingTable.value = null
  }
}

function cancelTableEdit() {
  editingTable.value = null
}

function updateTableMode(tableName, syncMode) {
  const item = selectedTablesList.value.find(t => t.source_table === tableName)
  if (item) item.sync_mode = syncMode
}

function selectAllTables() {
  availableTables.value.forEach(tableName => {
    if (!isTableSelected(tableName)) {
      selectedTablesList.value.push({
        source_table: tableName,
        target_table: tableName,
        sync_mode: formData.sync_mode,
        enabled: true,
        where_clause: ''
      })
    }
  })
}

function deselectAllTables() {
  selectedTablesList.value = []
  editingTable.value = null
}

async function handleSubmit() {
  error.value = null

  if (!formData.source_connection_id) {
    error.value = '请选择源数据库连接'
    return
  }

  if (!formData.source_database) {
    error.value = '请选择源数据库'
    return
  }

  if (!formData.target_connection_id) {
    error.value = '请选择目标数据库连接'
    return
  }

  if (!formData.target_database) {
    error.value = '请输入目标数据库名称'
    return
  }

  if (formData.source_connection_id === formData.target_connection_id) {
    error.value = '源连接和目标连接不能相同'
    return
  }

  if (selectedTablesList.value.length === 0) {
    error.value = '请至少选择一个表进行同步'
    return
  }

  const tables = selectedTablesList.value.map(t => ({
    source_table: t.source_table,
    target_table: t.target_table || t.source_table,
    sync_mode: t.sync_mode,
    enabled: t.enabled,
    where_clause: t.where_clause || undefined
  }))

  const configData = {
    source_connection_id: formData.source_connection_id,
    target_connection_id: formData.target_connection_id,
    source_database: formData.source_database,
    target_database: formData.target_database,
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
  background-color: var(--bg-primary);
  margin: 2% auto;
  padding: var(--space-5);
  border-radius: var(--radius-lg);
  width: 92%;
  max-width: 1200px;
  max-height: 92vh;
  overflow-y: auto;
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
  margin-bottom: var(--space-5);
}

.modal-header h2 {
  color: var(--text-primary);
  margin: 0;
}

.close {
  color: var(--text-tertiary);
  font-size: 28px;
  font-weight: bold;
  cursor: pointer;
  line-height: 1;
}

.close:hover {
  color: var(--text-primary);
}

.tabs {
  display: flex;
  gap: var(--space-1);
  margin-bottom: var(--space-5);
  border-bottom: 1px solid var(--border-primary);
}

.tab {
  padding: var(--space-3) var(--space-5);
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-secondary);
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.2s;
  font-weight: 500;
}

.tab:hover {
  color: var(--text-primary);
}

.tab.active {
  color: var(--accent-primary);
  border-bottom-color: var(--accent-primary);
}

.tab-content {
  min-height: 400px;
}

.tables-tab {
  min-height: 560px;
}

.tables-layout {
  display: grid;
  grid-template-columns: 350px 1fr;
  gap: var(--space-5);
  height: 560px;
}

.tables-layout.tables-layout-three {
  grid-template-columns: minmax(280px, 380px) 350px 1fr;
}

/* Leftmost: 手动排序面板（仅在有已选表时显示） */
.selected-order-panel {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--bg-secondary);
}

.selected-order-header {
  padding: var(--space-3) var(--space-4);
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-primary);
}

.selected-order-header label {
  display: block;
  margin: 0;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-primary);
}

.selected-order-header small {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.selected-order-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-4);
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
  line-height: 1.5;
}

.selected-order-empty p {
  margin: 0;
}

.selected-order-panel .selected-order-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-2) var(--space-3);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* Left Panel: Table Selection */
.tables-selection {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.selection-header {
  padding: var(--space-3) var(--space-4);
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-primary);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.selection-header label {
  margin: 0;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-primary);
}

.selection-actions {
  display: flex;
  gap: var(--space-2);
}

.btn-link {
  background: none;
  border: none;
  color: var(--accent-primary);
  cursor: pointer;
  font-size: 13px;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  transition: all 0.2s;
}

.btn-link:hover:not(:disabled) {
  background: var(--accent-light);
}

.btn-link:disabled {
  color: var(--text-tertiary);
  cursor: not-allowed;
}

.table-list-compact {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-2);
}

.table-item-compact {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s;
  font-size: 13px;
}

.table-item-compact:hover {
  background: var(--bg-hover);
}

.table-item-compact.selected {
  background: var(--accent-light);
}

.table-item-compact.active {
  background: var(--accent-primary);
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
  font-size: 11px;
  padding: 2px 8px;
  background: var(--accent-primary);
  color: white;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.loading-compact {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-5);
  color: var(--text-secondary);
  font-size: 13px;
}

.spinner-small {
  width: 16px;
  height: 16px;
  border: 2px solid var(--border-primary);
  border-top: 2px solid var(--accent-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty-state-compact {
  padding: var(--space-5) var(--space-4);
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}


.selected-order-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  cursor: grab;
  font-size: 13px;
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  transition: background 0.15s;
  min-width: 0;
}

.selected-order-item .table-name {
  min-width: 0;
  word-break: break-word;
  overflow-wrap: break-word;
  line-height: 1.35;
}

.selected-order-item:active {
  cursor: grabbing;
}

.selected-order-item:hover {
  background: var(--accent-light);
}

.selected-order-item.active {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
}

.selected-order-item.dragging {
  opacity: 0.5;
}

.drag-handle {
  color: var(--text-tertiary);
  cursor: grab;
  font-size: 13px;
  user-select: none;
}

.selected-order-item.active .drag-handle {
  color: rgba(255, 255, 255, 0.8);
}

.selection-summary {
  padding: var(--space-3) var(--space-4);
  background: var(--bg-tertiary);
  border-top: 1px solid var(--border-primary);
  font-size: 13px;
  color: var(--text-secondary);
}

.selection-summary strong {
  color: var(--accent-primary);
  font-weight: 600;
}

/* Right Panel: Table Configuration */
.table-config-panel {
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  padding: var(--space-5);
  overflow-y: auto;
}

.config-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-tertiary);
  font-size: 13px;
}

.config-form h3 {
  margin: 0 0 var(--space-5) 0;
  color: var(--accent-primary);
  font-size: 16px;
  padding-bottom: var(--space-3);
  border-bottom: 2px solid var(--border-primary);
}

.form-group-compact {
  margin-bottom: var(--space-4);
}

.form-group-compact label {
  display: block;
  margin-bottom: var(--space-2);
  font-weight: 500;
  font-size: 13px;
  color: var(--text-primary);
}

.form-group-compact input[type="text"],
.form-group-compact textarea {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-sm);
  font-size: 13px;
  transition: border-color 0.2s;
  background: var(--bg-primary);
  color: var(--text-primary);
}

.form-group-compact input[type="text"]:focus,
.form-group-compact textarea:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.form-group-compact textarea {
  resize: vertical;
  font-family: var(--font-mono);
}

.form-group-compact small {
  display: block;
  margin-top: 4px;
  color: var(--text-tertiary);
  font-size: 12px;
}

.radio-group {
  display: flex;
  gap: var(--space-4);
}

.radio-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
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
  gap: var(--space-3);
  justify-content: flex-end;
  margin-top: var(--space-5);
  padding-top: var(--space-5);
  border-top: 1px solid var(--border-primary);
}

.btn-sm {
  padding: var(--space-2) var(--space-4);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-sm.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.btn-sm.btn-secondary:hover {
  background: var(--border-primary);
}

.btn-sm.btn-primary {
  background: var(--accent-primary);
  color: white;
}

.btn-sm.btn-primary:hover {
  background: var(--accent-hover);
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
  gap: var(--space-4);
}

.form-actions {
  display: flex;
  gap: var(--space-4);
  justify-content: flex-end;
  margin-top: var(--space-5);
}

.connection-panels {
  display: flex;
  align-items: stretch;
  gap: var(--space-4);
  margin-bottom: var(--space-5);
}

.connection-panel {
  flex: 1;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  background: var(--bg-secondary);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.connection-panel-source {
  border-color: var(--success);
  border-width: 2px;
}

.connection-panel-target {
  border-color: var(--accent-primary);
  border-width: 2px;
}

.connection-panel-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-weight: 600;
  font-size: 14px;
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-primary);
  color: var(--text-primary);
}

.connection-panel .form-group {
  margin-bottom: 0;
}

.connection-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  flex-shrink: 0;
  align-self: center;
}

@media (max-width: 768px) {
  .connection-panels {
    flex-direction: column;
  }

  .connection-arrow {
    transform: rotate(90deg);
  }
}

.sync-visual {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-5);
  padding: var(--space-3) var(--space-4);
  margin: var(--space-2) 0 var(--space-4);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.sync-node {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.db-icon {
  flex-shrink: 0;
}

.db-icon-source {
  color: var(--success);
}

.db-icon-target {
  color: var(--accent-primary);
}

.sync-node-text {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  line-height: 1.2;
}

.sync-node-main {
  color: var(--text-primary);
  font-weight: 500;
  white-space: nowrap;
}

.sync-node-sub {
  color: var(--text-secondary);
  white-space: nowrap;
}

.sync-arrow {
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: auto;
}

.table-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  padding: var(--space-4);
}

.loading {
  text-align: center;
  padding: var(--space-5);
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: var(--space-5);
  color: var(--text-secondary);
}
</style>
