<template>
  <div class="table-diff">
    <!-- Selection Area -->
    <div class="selection-area">
      <div class="selection-panel">
        <h3>源表</h3>
        <div class="form-group">
          <label>连接</label>
          <select v-model="source.connectionId" @change="onSourceConnectionChange">
            <option value="">请选择连接</option>
            <option v-for="conn in connections" :key="conn.config.id" :value="conn.config.id">
              {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
            </option>
          </select>
        </div>
        <div class="form-group">
          <label>数据库</label>
          <select v-model="source.database" @change="onSourceDatabaseChange" :disabled="!source.connectionId">
            <option value="">请选择数据库</option>
            <option v-for="db in sourceDatabases" :key="db" :value="db">{{ db }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>表</label>
          <select v-model="source.table" :disabled="!source.database">
            <option value="">请选择表</option>
            <option v-for="t in sourceTables" :key="t" :value="t">{{ t }}</option>
          </select>
        </div>
      </div>

      <div class="selection-divider">
        <GitCompareArrows :size="32" />
      </div>

      <div class="selection-panel">
        <h3>目标表</h3>
        <div class="form-group">
          <label>连接</label>
          <select v-model="target.connectionId" @change="onTargetConnectionChange">
            <option value="">请选择连接</option>
            <option v-for="conn in connections" :key="conn.config.id" :value="conn.config.id">
              {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
            </option>
          </select>
        </div>
        <div class="form-group">
          <label>数据库</label>
          <select v-model="target.database" @change="onTargetDatabaseChange" :disabled="!target.connectionId">
            <option value="">请选择数据库</option>
            <option v-for="db in targetDatabases" :key="db" :value="db">{{ db }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>表</label>
          <select v-model="target.table" :disabled="!target.database">
            <option value="">请选择表</option>
            <option v-for="t in targetTables" :key="t" :value="t">{{ t }}</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Action Bar -->
    <div class="action-bar">
      <button class="btn-primary" @click="startCompare" :disabled="!canCompare || comparing">
        <span v-if="comparing">对比中 {{ progress }}%...</span>
        <span v-else>开始对比</span>
      </button>
      <button class="btn-secondary" @click="exportResult" v-if="diffResult" :disabled="comparing">
        导出结果
      </button>
    </div>

    <!-- Warning -->
    <div class="warning-banner" v-if="diffResult && diffResult.warning">
      {{ diffResult.warning }}
    </div>

    <!-- Results Area -->
    <div class="results-area" v-if="diffResult">
      <div class="result-tabs">
        <button :class="['result-tab', { active: activeResultTab === 'schema' }]" @click="activeResultTab = 'schema'">
          结构差异
          <span class="badge" v-if="schemaDiffCount > 0">{{ schemaDiffCount }}</span>
        </button>
        <button :class="['result-tab', { active: activeResultTab === 'data' }]" @click="activeResultTab = 'data'" :disabled="!diffResult.data_diff || !!diffResult.warning">
          数据差异
          <span class="badge" v-if="dataDiffCount > 0">{{ dataDiffCount }}</span>
        </button>
      </div>

      <!-- Schema Diff -->
      <div v-if="activeResultTab === 'schema'" class="schema-diff">
        <div v-if="diffResult.schema_diff && diffResult.schema_diff.is_same" class="no-diff">
          结构完全一致，无差异
        </div>
        <template v-else>
          <table class="diff-table" v-if="diffResult.schema_diff">
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
                <td>{{ col.name }}</td>
                <td><span class="status-tag added">新增</span></td>
                <td>{{ col.type }}</td>
                <td>-</td>
                <td>{{ col.nullable ? 'YES' : 'NO' }}</td>
                <td>-</td>
              </tr>
              <tr v-for="col in diffResult.schema_diff.removed_columns" :key="'rm-'+col.name" class="row-removed">
                <td>{{ col.name }}</td>
                <td><span class="status-tag removed">删除</span></td>
                <td>-</td>
                <td>{{ col.type }}</td>
                <td>-</td>
                <td>{{ col.nullable ? 'YES' : 'NO' }}</td>
              </tr>
              <tr v-for="col in diffResult.schema_diff.modified_columns" :key="'mod-'+col.name" class="row-modified">
                <td>{{ col.name }}</td>
                <td><span class="status-tag modified">变更</span></td>
                <td>{{ col.source_type }}</td>
                <td>{{ col.target_type }}</td>
                <td>{{ col.source_nullable ? 'YES' : 'NO' }}</td>
                <td>{{ col.target_nullable ? 'YES' : 'NO' }}</td>
              </tr>
            </tbody>
          </table>
        </template>
      </div>

      <!-- Data Diff -->
      <div v-if="activeResultTab === 'data'" class="data-diff">
        <div v-if="diffResult.data_diff" class="data-diff-summary">
          <span class="diff-stat">仅源表: {{ diffResult.data_diff.only_in_source?.length || 0 }} 行</span>
          <span class="diff-stat">仅目标表: {{ diffResult.data_diff.only_in_target?.length || 0 }} 行</span>
          <span class="diff-stat">值不同: {{ diffResult.data_diff.value_diff?.length || 0 }} 行</span>
          <span class="diff-stat partial" v-if="diffResult.data_diff.partial">
            (部分对比: {{ diffResult.data_diff.compared_rows }}/{{ diffResult.data_diff.total_rows }} 行)
          </span>
        </div>

        <!-- Value Diff -->
        <div v-if="diffResult.data_diff && diffResult.data_diff.value_diff?.length" class="diff-section">
          <h4>值不同的行</h4>
          <div class="pagination" v-if="valueDiffTotalPages > 1">
            <button @click="valueDiffPage > 1 && valueDiffPage--" :disabled="valueDiffPage <= 1">&lt;</button>
            <span>{{ valueDiffPage }} / {{ valueDiffTotalPages }}</span>
            <button @click="valueDiffPage < valueDiffTotalPages && valueDiffPage++" :disabled="valueDiffPage >= valueDiffTotalPages">&gt;</button>
          </div>
          <table class="diff-table data-table">
            <thead>
              <tr>
                <th>主键</th>
                <th>差异列</th>
                <th>源表值</th>
                <th>目标表值</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="entry in paginatedValueDiff" :key="JSON.stringify(entry.primary_key)">
                <td>{{ formatPK(entry.primary_key) }}</td>
                <td>
                  <span v-for="d in entry.differences" :key="d.column" class="col-name">{{ d.column }} </span>
                </td>
                <td>
                  <span v-for="d in entry.differences" :key="'s-'+d.column" class="val-diff">{{ formatVal(d.source_value) }} </span>
                </td>
                <td>
                  <span v-for="d in entry.differences" :key="'t-'+d.column" class="val-diff">{{ formatVal(d.target_value) }} </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Only in Source -->
        <div v-if="diffResult.data_diff && diffResult.data_diff.only_in_source?.length" class="diff-section">
          <h4>仅存在于源表 ({{ diffResult.data_diff.only_in_source.length }} 行)</h4>
          <div class="pagination" v-if="onlySourceTotalPages > 1">
            <button @click="onlySourcePage > 1 && onlySourcePage--" :disabled="onlySourcePage <= 1">&lt;</button>
            <span>{{ onlySourcePage }} / {{ onlySourceTotalPages }}</span>
            <button @click="onlySourcePage < onlySourceTotalPages && onlySourcePage++" :disabled="onlySourcePage >= onlySourceTotalPages">&gt;</button>
          </div>
          <div class="row-cards">
            <div v-for="(row, idx) in paginatedOnlySource" :key="idx" class="row-card source-only">
              <span v-for="(val, key) in row" :key="key" class="kv"><strong>{{ key }}</strong>: {{ formatVal(val) }}</span>
            </div>
          </div>
        </div>

        <!-- Only in Target -->
        <div v-if="diffResult.data_diff && diffResult.data_diff.only_in_target?.length" class="diff-section">
          <h4>仅存在于目标表 ({{ diffResult.data_diff.only_in_target.length }} 行)</h4>
          <div class="pagination" v-if="onlyTargetTotalPages > 1">
            <button @click="onlyTargetPage > 1 && onlyTargetPage--" :disabled="onlyTargetPage <= 1">&lt;</button>
            <span>{{ onlyTargetPage }} / {{ onlyTargetTotalPages }}</span>
            <button @click="onlyTargetPage < onlyTargetTotalPages && onlyTargetPage++" :disabled="onlyTargetPage >= onlyTargetTotalPages">&gt;</button>
          </div>
          <div class="row-cards">
            <div v-for="(row, idx) in paginatedOnlyTarget" :key="idx" class="row-card target-only">
              <span v-for="(val, key) in row" :key="key" class="kv"><strong>{{ key }}</strong>: {{ formatVal(val) }}</span>
            </div>
          </div>
        </div>

        <div v-if="dataDiffCount === 0 && diffResult.data_diff" class="no-diff">
          数据完全一致，无差异
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { GitCompareArrows } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'

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

const pageSize = 20
const valueDiffPage = ref(1)
const onlySourcePage = ref(1)
const onlyTargetPage = ref(1)

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
const onlySourceTotalPages = computed(() => Math.ceil((diffResult.value?.data_diff?.only_in_source?.length || 0) / pageSize))
const onlyTargetTotalPages = computed(() => Math.ceil((diffResult.value?.data_diff?.only_in_target?.length || 0) / pageSize))

const paginatedValueDiff = computed(() => {
  const arr = diffResult.value?.data_diff?.value_diff || []
  const start = (valueDiffPage.value - 1) * pageSize
  return arr.slice(start, start + pageSize)
})

const paginatedOnlySource = computed(() => {
  const arr = diffResult.value?.data_diff?.only_in_source || []
  const start = (onlySourcePage.value - 1) * pageSize
  return arr.slice(start, start + pageSize)
})

const paginatedOnlyTarget = computed(() => {
  const arr = diffResult.value?.data_diff?.only_in_target || []
  const start = (onlyTargetPage.value - 1) * pageSize
  return arr.slice(start, start + pageSize)
})

onMounted(() => {
  if (store.connections.length === 0) {
    store.fetchConnections()
  }
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
  onlySourcePage.value = 1
  onlyTargetPage.value = 1

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

    // Poll for result
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
      if (task.status === 'failed') {
        throw new Error(task.error || '对比失败')
      }
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
  return String(v)
}
</script>

<style scoped>
.table-diff {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.selection-area {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.selection-panel {
  flex: 1;
  background: #f8f9fa;
  border-radius: 8px;
  padding: 1.25rem;
}

.selection-panel h3 {
  margin: 0 0 1rem 0;
  font-size: 1.1rem;
  color: #333;
}

.selection-divider {
  display: flex;
  align-items: center;
  padding-top: 2rem;
  color: #667eea;
}

.form-group {
  margin-bottom: 0.75rem;
}

.form-group label {
  display: block;
  font-size: 0.85rem;
  font-weight: 600;
  color: #555;
  margin-bottom: 0.25rem;
}

.form-group select {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
  background: white;
}

.form-group select:disabled {
  background: #eee;
  cursor: not-allowed;
}

.action-bar {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.btn-primary {
  padding: 0.65rem 2rem;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 1rem;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:disabled {
  background: #aaa;
  cursor: not-allowed;
}

.btn-primary:hover:not(:disabled) {
  background: #5a6fd6;
}

.btn-secondary {
  padding: 0.65rem 1.5rem;
  background: white;
  color: #667eea;
  border: 1px solid #667eea;
  border-radius: 6px;
  font-size: 1rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:hover:not(:disabled) {
  background: #f0f2ff;
}

.warning-banner {
  background: #fff3cd;
  border: 1px solid #ffc107;
  color: #856404;
  padding: 0.75rem 1rem;
  border-radius: 6px;
  font-size: 0.9rem;
}

.results-area {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
}

.result-tabs {
  display: flex;
  background: #f8f9fa;
  border-bottom: 2px solid #e9ecef;
}

.result-tab {
  flex: 1;
  padding: 0.75rem 1rem;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 0.95rem;
  color: #666;
  transition: all 0.2s;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.result-tab:hover:not(:disabled) {
  color: #667eea;
}

.result-tab.active {
  color: #667eea;
  font-weight: 600;
  background: white;
}

.result-tab.active::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  right: 0;
  height: 2px;
  background: #667eea;
}

.result-tab:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.badge {
  background: #667eea;
  color: white;
  border-radius: 10px;
  padding: 0.1rem 0.5rem;
  font-size: 0.75rem;
  min-width: 1.2rem;
  text-align: center;
}

.schema-diff, .data-diff {
  padding: 1.25rem;
}

.no-diff {
  text-align: center;
  color: #28a745;
  padding: 2rem;
  font-size: 1.1rem;
}

.diff-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.diff-table th {
  background: #f0f2ff;
  padding: 0.6rem 0.75rem;
  text-align: left;
  font-weight: 600;
  color: #444;
  border-bottom: 2px solid #ddd;
}

.diff-table td {
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid #eee;
}

.row-added td { background: #e8f5e9; }
.row-removed td { background: #ffebee; }
.row-modified td { background: #fff8e1; }

.status-tag {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 600;
}

.status-tag.added { background: #c8e6c9; color: #2e7d32; }
.status-tag.removed { background: #ffcdd2; color: #c62828; }
.status-tag.modified { background: #fff9c4; color: #f57f17; }

.data-diff-summary {
  display: flex;
  gap: 1.5rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.diff-stat {
  font-size: 0.9rem;
  color: #555;
}

.diff-stat.partial {
  color: #e65100;
  font-weight: 600;
}

.diff-section {
  margin-bottom: 1.5rem;
}

.diff-section h4 {
  font-size: 0.95rem;
  color: #333;
  margin: 0 0 0.75rem 0;
}

.pagination {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.pagination button {
  padding: 0.3rem 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: white;
  cursor: pointer;
}

.pagination button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.val-diff {
  color: #e65100;
  font-weight: 600;
}

.col-name {
  display: inline-block;
  background: #fff9c4;
  padding: 0.1rem 0.35rem;
  border-radius: 3px;
  margin-right: 0.25rem;
  font-size: 0.85rem;
}

.row-cards {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.row-card {
  background: #f8f9fa;
  border-radius: 6px;
  padding: 0.5rem 0.75rem;
  font-size: 0.85rem;
  border-left: 3px solid #ccc;
}

.row-card.source-only { border-left-color: #4caf50; }
.row-card.target-only { border-left-color: #f44336; }

.kv {
  display: inline-block;
  margin-right: 1rem;
}

.kv strong {
  color: #667eea;
}

@media (max-width: 768px) {
  .selection-area {
    flex-direction: column;
  }
  .selection-divider {
    padding-top: 0;
    transform: rotate(90deg);
  }
}
</style>
