<template>
  <div>
    <!-- Page Header -->
    <!-- Toast -->
    <Toast :show="toast.show" :message="toast.message" :type="toast.type" @close="toast.show = false" />

    <div class="page-header">
      <h1>表对比</h1>
    </div>

    <!-- Selection Panel -->
    <div class="card">
      <!-- Quick Config Selector -->
      <div v-if="syncConfigs.length > 0" class="config-selector">
        <label class="config-selector-label">
          <List :size="16" />
          <span>快速选择同步配置</span>
        </label>
        <select v-model="selectedConfigId" @change="onConfigChange" class="form-control config-select">
          <option value="">-- 请选择同步配置 --</option>
          <option v-for="cfg in syncConfigs" :key="cfg.id" :value="cfg.id">
            {{ cfg.name }}
          </option>
        </select>
      </div>

      <div class="selection-area">
        <!-- Source -->
        <div class="selection-panel">
          <h3 class="panel-title">源表</h3>
          <div class="form-group">
            <label>连接</label>
            <select v-model="source.connectionId" @change="onSourceConnectionChange" class="form-control">
              <option value="">请选择连接</option>
              <option v-for="conn in connections" :key="conn.config.id" :value="conn.config.id">
                {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
              </option>
            </select>
          </div>
          <div class="form-group">
            <label>数据库</label>
            <select v-model="source.database" @change="onSourceDatabaseChange" :disabled="!source.connectionId" class="form-control">
              <option value="">请选择数据库</option>
              <option v-for="db in sourceDatabases" :key="db" :value="db">{{ db }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>表</label>
            <select v-model="source.table" :disabled="!source.database" class="form-control">
              <option value="">请选择表</option>
              <option v-for="t in sourceTables" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>
        </div>

        <!-- Divider -->
        <div class="selection-divider">
          <ArrowRight :size="24" />
        </div>

        <!-- Target -->
        <div class="selection-panel">
          <h3 class="panel-title">目标表</h3>
          <div class="form-group">
            <label>连接</label>
            <select v-model="target.connectionId" @change="onTargetConnectionChange" class="form-control">
              <option value="">请选择连接</option>
              <option v-for="conn in connections" :key="conn.config.id" :value="conn.config.id">
                {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
              </option>
            </select>
          </div>
          <div class="form-group">
            <label>数据库</label>
            <select v-model="target.database" @change="onTargetDatabaseChange" :disabled="!target.connectionId" class="form-control">
              <option value="">请选择数据库</option>
              <option v-for="db in targetDatabases" :key="db" :value="db">{{ db }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>表</label>
            <select v-model="target.table" :disabled="!target.database" class="form-control">
              <option value="">请选择表</option>
              <option v-for="t in targetTables" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="action-bar">
        <button class="btn btn-primary btn-lg" @click="startCompare" :disabled="!canCompare || comparing">
          <Loader2 v-if="comparing" :size="18" class="spin" />
          <GitCompareArrows v-else :size="18" />
          <span>{{ comparing ? `对比中 ${progress}%...` : '开始对比' }}</span>
        </button>
        <button class="btn btn-secondary" @click="exportResult" v-if="diffResult" :disabled="comparing">
          <Download :size="16" />
          <span>导出结果</span>
        </button>
      </div>

      <!-- Warning -->
      <div v-if="diffResult && diffResult.warning" class="alert alert-warning" style="margin-top: 16px;">
        <AlertTriangle :size="18" />
        <span>{{ diffResult.warning }}</span>
      </div>
    </div>

    <!-- Results -->
    <div v-if="diffResult" class="card" style="margin-top: 24px;">
      <!-- Result Tabs -->
      <div class="result-tabs">
        <button :class="['result-tab', { active: activeResultTab === 'schema' }]" @click="activeResultTab = 'schema'">
          结构差异
          <span v-if="schemaDiffCount > 0" class="badge badge-info">{{ schemaDiffCount }}</span>
        </button>
        <button :class="['result-tab', { active: activeResultTab === 'data' }]" @click="activeResultTab = 'data'" :disabled="!diffResult.data_diff || !!diffResult.warning">
          数据差异
          <span v-if="dataDiffCount > 0" class="badge badge-info">{{ dataDiffCount }}</span>
        </button>
      </div>

      <!-- Schema Diff -->
      <div v-if="activeResultTab === 'schema'" class="result-content">
        <div v-if="diffResult.schema_diff && diffResult.schema_diff.is_same" class="empty-state">
          <CheckCircle :size="32" class="empty-state-icon" style="color: var(--success);" />
          <h3>结构完全一致</h3>
          <p>两个表的列结构完全相同</p>
        </div>
        <div v-else-if="diffResult.schema_diff" class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>列名</th>
                <th>状态</th>
                <th>源表类型</th>
                <th>目标表类型</th>
                <th>源表可空</th>
                <th>目标表可空</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="col in diffResult.schema_diff.added_columns" :key="'add-'+col.name" class="row-added">
                <td class="font-medium">{{ col.name }}</td>
                <td><span class="badge badge-success">新增</span></td>
                <td>{{ col.type }}</td>
                <td>-</td>
                <td>{{ col.nullable ? '是' : '否' }}</td>
                <td>-</td>
              </tr>
              <tr v-for="col in diffResult.schema_diff.removed_columns" :key="'rm-'+col.name" class="row-removed">
                <td class="font-medium">{{ col.name }}</td>
                <td><span class="badge badge-error">删除</span></td>
                <td>-</td>
                <td>{{ col.type }}</td>
                <td>-</td>
                <td>{{ col.nullable ? '是' : '否' }}</td>
              </tr>
              <tr v-for="col in diffResult.schema_diff.modified_columns" :key="'mod-'+col.name" class="row-modified">
                <td class="font-medium">{{ col.name }}</td>
                <td><span class="badge badge-warning">变更</span></td>
                <td>{{ col.source_type }}</td>
                <td>{{ col.target_type }}</td>
                <td>{{ col.source_nullable ? '是' : '否' }}</td>
                <td>{{ col.target_nullable ? '是' : '否' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Data Diff -->
      <div v-if="activeResultTab === 'data'" class="result-content">
        <div v-if="diffResult.data_diff" class="diff-summary">
          <div class="stat-item">
            <span class="stat-label">仅源表</span>
            <span class="stat-value">{{ diffResult.data_diff.only_in_source?.length || 0 }} 行</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">仅目标表</span>
            <span class="stat-value">{{ diffResult.data_diff.only_in_target?.length || 0 }} 行</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">值不同</span>
            <span class="stat-value">{{ diffResult.data_diff.value_diff?.length || 0 }} 行</span>
          </div>
          <div v-if="diffResult.data_diff.partial" class="stat-item warning">
            <span class="stat-label">部分对比</span>
            <span class="stat-value">{{ diffResult.data_diff.compared_rows }}/{{ diffResult.data_diff.total_rows }} 行</span>
          </div>
          <div class="stat-item batch-action">
            <button
              class="btn btn-primary btn-sm"
              :disabled="(diffResult.data_diff.value_diff?.length || 0) + (diffResult.data_diff.only_in_source?.length || 0) === 0"
              @click="copyToClipboard(generateAllReplaceSQL())"
              title="复制全部 REPLACE 语句"
            >
              <ClipboardList :size="16" />
              <span>复制全部 REPLACE</span>
            </button>
          </div>
        </div>

        <!-- Value Diff -->
        <div v-if="diffResult.data_diff?.value_diff?.length" class="diff-section">
          <h4>值不同的行</h4>
          <div v-if="valueDiffTotalPages > 1" class="pagination">
            <button class="btn btn-sm btn-secondary" @click="valueDiffPage--" :disabled="valueDiffPage <= 1">
              <ChevronLeft :size="14" />
            </button>
            <span class="page-info">{{ valueDiffPage }} / {{ valueDiffTotalPages }}</span>
            <button class="btn btn-sm btn-secondary" @click="valueDiffPage++" :disabled="valueDiffPage >= valueDiffTotalPages">
              <ChevronRight :size="14" />
            </button>
          </div>
          <div class="table-container">
            <table class="data-table">
              <thead>
                <tr>
                  <th>主键</th>
                  <th>差异列</th>
                  <th>源表值</th>
                  <th>目标表值</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in paginatedValueDiff" :key="JSON.stringify(entry.primary_key)">
                  <td class="font-medium">{{ formatPK(entry.primary_key) }}</td>
                  <td>
                    <span v-for="d in entry.differences" :key="d.column" class="tag">{{ d.column }}</span>
                  </td>
                  <td class="text-error">{{ formatVal(entry.differences[0]?.source_value) }}</td>
                  <td class="text-success">{{ formatVal(entry.differences[0]?.target_value) }}</td>
                  <td>
                    <button class="btn btn-sm btn-secondary" title="复制 REPLACE 语句" @click="copyToClipboard(generateReplaceForValueDiff(entry))">
                      <Copy :size="14" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Only in Source -->
        <div v-if="diffResult.data_diff?.only_in_source?.length" class="diff-section">
          <h4>仅存在于源表 ({{ diffResult.data_diff.only_in_source.length }} 行)</h4>
          <div class="row-cards">
            <div v-for="(row, idx) in diffResult.data_diff.only_in_source.slice(0, 10)" :key="idx" class="row-card source-only">
              <span v-for="(val, key) in row" :key="key" class="kv"><strong>{{ key }}:</strong> {{ formatVal(val) }}</span>
              <button class="btn btn-sm btn-secondary card-copy-btn" title="复制 REPLACE 语句" @click="copyToClipboard(generateReplaceForOnlySource(row))">
                <Copy :size="14" />
              </button>
            </div>
          </div>
        </div>

        <!-- Only in Target -->
        <div v-if="diffResult.data_diff?.only_in_target?.length" class="diff-section">
          <h4>仅存在于目标表 ({{ diffResult.data_diff.only_in_target.length }} 行)</h4>
          <div class="row-cards">
            <div v-for="(row, idx) in diffResult.data_diff.only_in_target.slice(0, 10)" :key="idx" class="row-card target-only">
              <span v-for="(val, key) in row" :key="key" class="kv"><strong>{{ key }}:</strong> {{ formatVal(val) }}</span>
            </div>
          </div>
        </div>

        <div v-if="dataDiffCount === 0 && diffResult.data_diff" class="empty-state">
          <CheckCircle :size="32" class="empty-state-icon" style="color: var(--success);" />
          <h3>数据完全一致</h3>
          <p>两个表的数据内容完全相同</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { GitCompareArrows, ArrowRight, Loader2, Download, CheckCircle, AlertTriangle, ChevronLeft, ChevronRight, Copy, ClipboardList, List } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import Toast from '../components/Toast.vue'

const store = useSyncStore()
const connections = computed(() => store.connections)

const source = ref({ connectionId: '', database: '', table: '' })
const target = ref({ connectionId: '', database: '', table: '' })
const sourceDatabases = ref([])
const sourceTables = ref([])
const targetDatabases = ref([])
const targetTables = ref([])

const comparing = ref(false)
const progress = ref(0)
const diffResult = ref(null)
const activeResultTab = ref('schema')

const pageSize = 10
const valueDiffPage = ref(1)

const toast = ref({ show: false, message: '', type: 'success' })

const selectedConfigId = ref('')
const syncConfigs = computed(() => store.configs.filter(c => c.enabled))

async function onConfigChange() {
  if (!selectedConfigId.value) return
  const config = store.configs.find(c => c.id === selectedConfigId.value)
  if (!config) return

  // Reset previous diff
  diffResult.value = null
  activeResultTab.value = 'schema'

  // Fill source
  source.value.connectionId = config.source_connection_id
  await onSourceConnectionChange()
  source.value.database = config.source_database
  await onSourceDatabaseChange()
  if (config.tables?.length > 0) {
    source.value.table = config.tables[0].source_table
  } else {
    source.value.table = ''
  }

  // Fill target
  target.value.connectionId = config.target_connection_id
  await onTargetConnectionChange()
  target.value.database = config.target_database
  await onTargetDatabaseChange()
  if (config.tables?.length > 0) {
    target.value.table = config.tables[0].target_table
  } else {
    target.value.table = ''
  }
}

const canCompare = computed(() =>
  source.value.connectionId && source.value.database && source.value.table &&
  target.value.connectionId && target.value.database && target.value.table
)

const schemaDiffCount = computed(() => {
  if (!diffResult.value?.schema_diff) return 0
  const s = diffResult.value.schema_diff
  return (s.added_columns?.length || 0) + (s.removed_columns?.length || 0) + (s.modified_columns?.length || 0)
})

const dataDiffCount = computed(() => {
  if (!diffResult.value?.data_diff) return 0
  const d = diffResult.value.data_diff
  return (d.only_in_source?.length || 0) + (d.only_in_target?.length || 0) + (d.value_diff?.length || 0)
})

const valueDiffTotalPages = computed(() => Math.ceil((diffResult.value?.data_diff?.value_diff?.length || 0) / pageSize))

const paginatedValueDiff = computed(() => {
  const arr = diffResult.value?.data_diff?.value_diff || []
  const start = (valueDiffPage.value - 1) * pageSize
  return arr.slice(start, start + pageSize)
})

// --- REPLACE SQL utility functions ---

function formatSQLValue(val) {
  if (val === null || val === undefined) return 'NULL'
  if (typeof val === 'number' || typeof val === 'boolean') return String(val)
  const escaped = String(val).replace(/'/g, "''")
  return `'${escaped}'`
}

function buildReplaceSQL(cols, vals) {
  const tableName = `\`${target.value.table}\``
  const colList = cols.map(c => `\`${c}\``).join(', ')
  const valList = vals.join(', ')
  return `REPLACE INTO ${tableName} (${colList}) VALUES (${valList});`
}

function generateReplaceForValueDiff(entry) {
  // Use the full source row to generate REPLACE with all columns
  const row = entry.source_row
  if (row) {
    const cols = Object.keys(row)
    const vals = cols.map(c => formatSQLValue(row[c]))
    return buildReplaceSQL(cols, vals)
  }
  // Fallback: only PK + diff columns
  const cols = []
  const vals = []
  const pkCols = new Set()

  // PK columns
  for (const [col, val] of Object.entries(entry.primary_key)) {
    cols.push(col)
    vals.push(formatSQLValue(val))
    pkCols.add(col)
  }

  // Diff columns (use source value) — skip if already added as PK
  for (const diff of entry.differences) {
    if (!pkCols.has(diff.column)) {
      cols.push(diff.column)
      vals.push(formatSQLValue(diff.source_value))
    }
  }

  return buildReplaceSQL(cols, vals)
}

function generateReplaceForOnlySource(row) {
  const cols = Object.keys(row)
  const vals = cols.map(c => formatSQLValue(row[c]))
  return buildReplaceSQL(cols, vals)
}

function generateAllReplaceSQL() {
  const lines = []
  const data = diffResult.value?.data_diff
  if (!data) return ''

  // Value diff rows
  for (const entry of data.value_diff || []) {
    lines.push(generateReplaceForValueDiff(entry))
  }

  // Only in source rows
  for (const row of data.only_in_source || []) {
    lines.push(generateReplaceForOnlySource(row))
  }

  return lines.join('\n')
}

async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    toast.value = { show: true, message: 'REPLACE 语句已复制', type: 'success' }
  } catch (e) {
    toast.value = { show: true, message: '复制失败: ' + e.message, type: 'error' }
  }
}

onMounted(() => {
  if (store.configs.length === 0) store.fetchConfigs()
})

async function onSourceConnectionChange() {
  source.value.database = ''
  source.value.table = ''
  sourceDatabases.value = []
  sourceTables.value = []
  if (!source.value.connectionId) return
  try {
    const res = await fetch(`/api/sync/connections/${source.value.connectionId}/databases`)
    const result = await res.json()
    if (result.success) sourceDatabases.value = result.data || []
  } catch (e) { console.error(e) }
}

async function onSourceDatabaseChange() {
  source.value.table = ''
  sourceTables.value = []
  if (!source.value.connectionId || !source.value.database) return
  try {
    const res = await fetch(`/api/sync/connections/${source.value.connectionId}/tables?database=${encodeURIComponent(source.value.database)}`)
    const result = await res.json()
    if (result.success) sourceTables.value = result.data || []
  } catch (e) { console.error(e) }
}

async function onTargetConnectionChange() {
  target.value.database = ''
  target.value.table = ''
  targetDatabases.value = []
  targetTables.value = []
  if (!target.value.connectionId) return
  try {
    const res = await fetch(`/api/sync/connections/${target.value.connectionId}/databases`)
    const result = await res.json()
    if (result.success) targetDatabases.value = result.data || []
  } catch (e) { console.error(e) }
}

async function onTargetDatabaseChange() {
  target.value.table = ''
  targetTables.value = []
  if (!target.value.connectionId || !target.value.database) return
  try {
    const res = await fetch(`/api/sync/connections/${target.value.connectionId}/tables?database=${encodeURIComponent(target.value.database)}`)
    const result = await res.json()
    if (result.success) targetTables.value = result.data || []
  } catch (e) { console.error(e) }
}

async function startCompare() {
  if (!canCompare.value) return
  comparing.value = true
  progress.value = 0
  diffResult.value = null
  activeResultTab.value = 'schema'
  valueDiffPage.value = 1

  try {
    const res = await fetch('/api/sync/diff/compare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        source: { connection_id: source.value.connectionId, database: source.value.database, table: source.value.table },
        target: { connection_id: target.value.connectionId, database: target.value.database, table: target.value.table }
      })
    })
    const result = await res.json()
    if (!result.success) throw new Error(result.error || '对比失败')

    const taskId = result.data.task_id
    while (true) {
      await new Promise(r => setTimeout(r, 500))
      const pollRes = await fetch(`/api/sync/diff/${taskId}`)
      const pollResult = await pollRes.json()
      if (!pollResult.success) throw new Error(pollResult.error || '查询任务失败')

      const task = pollResult.data
      progress.value = task.progress || 0

      if (task.status === 'done') {
        diffResult.value = task.result
        break
      }
      if (task.status === 'failed') throw new Error(task.error || '对比失败')
    }
  } catch (e) {
    alert(e.message)
  } finally {
    comparing.value = false
  }
}

function exportResult() {
  if (!diffResult.value) return
  const blob = new Blob([JSON.stringify(diffResult.value, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `diff-result-${source.value.database}.${source.value.table}-vs-${target.value.database}.${target.value.table}.json`
  a.click()
  URL.revokeObjectURL(url)
}

function formatPK(pk) {
  return Object.entries(pk).map(([k, v]) => `${k}=${v}`).join(', ')
}

function formatVal(v) {
  if (v === null || v === undefined) return 'NULL'
  return String(v).length > 50 ? String(v).slice(0, 50) + '...' : String(v)
}
</script>

<style scoped>
.page-header {
  margin-bottom: var(--space-5);
}

.page-header h1 {
  font-size: 24px;
  font-weight: 600;
}

.config-selector {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
  padding-bottom: var(--space-5);
  border-bottom: 1px solid var(--border-secondary);
}

.config-selector-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
}

.config-select {
  max-width: 400px;
}

.selection-area {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: var(--space-5);
  align-items: start;
}

.selection-panel {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-4);
  color: var(--text-primary);
}

.selection-divider {
  display: flex;
  align-items: center;
  justify-content: center;
  padding-top: 80px;
  color: var(--text-secondary);
}

.action-bar {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-5);
  padding-top: var(--space-5);
  border-top: 1px solid var(--border-secondary);
}

.result-tabs {
  display: flex;
  gap: var(--space-1);
  border-bottom: 1px solid var(--border-secondary);
  margin: calc(-1 * var(--space-5)) calc(-1 * var(--space-5)) var(--space-4);
  padding: 0 var(--space-5);
}

.result-tab {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all var(--transition-fast);
  margin-bottom: -1px;
}

.result-tab:hover:not(:disabled) {
  color: var(--text-primary);
}

.result-tab.active {
  color: var(--accent-primary);
  border-bottom-color: var(--accent-primary);
}

.result-tab:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.result-content {
  padding-top: var(--space-2);
}

.diff-summary {
  display: flex;
  gap: var(--space-6);
  margin-bottom: var(--space-5);
  padding: var(--space-4);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.stat-item.warning .stat-label,
.stat-item.warning .stat-value {
  color: var(--warning);
}

.stat-label {
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.diff-section {
  margin-bottom: var(--space-5);
}

.diff-section h4 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: var(--space-3);
  color: var(--text-primary);
}

.pagination {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.page-info {
  font-size: 14px;
  color: var(--text-secondary);
}

.row-cards {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.row-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font-size: 13px;
  border-left: 3px solid var(--border-primary);
}

.row-card.source-only {
  border-left-color: var(--success);
  background: var(--success-light);
}

.row-card.target-only {
  border-left-color: var(--error);
  background: var(--error-light);
}

.kv {
  display: inline-block;
  margin-right: var(--space-4);
  margin-bottom: var(--space-1);
}

.kv strong {
  color: var(--accent-primary);
}

.card-copy-btn {
  margin-top: var(--space-2);
}

.batch-action {
  margin-left: auto;
  align-self: center;
}

.tag {
  display: inline-block;
  padding: var(--space-1) var(--space-2);
  background: var(--warning-light);
  color: var(--warning);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  margin-right: var(--space-1);
}

.row-added {
  background: var(--success-light);
}

.row-removed {
  background: var(--error-light);
}

.row-modified {
  background: var(--warning-light);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 768px) {
  .selection-area {
    grid-template-columns: 1fr;
  }

  .selection-divider {
    padding: var(--space-2) 0;
    transform: rotate(90deg);
  }

  .diff-summary {
    flex-wrap: wrap;
    gap: var(--space-4);
  }

  .action-bar {
    flex-direction: column;
  }
}
</style>
