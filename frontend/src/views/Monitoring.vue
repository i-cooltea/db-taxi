<template>
  <div>
    <!-- Toast -->
    <Toast :show="toast.show" :message="toast.message" :type="toast.type" @close="toast.show = false" />

    <!-- Page Header -->
    <div class="page-header">
      <h1>同步监控</h1>
    </div>

    <!-- Stats Grid -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--accent-light); color: var(--accent-primary);">
          <BarChart3 :size="20" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.total_jobs || 0 }}</div>
          <div class="stat-label">总任务数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--success-light); color: var(--success);">
          <CheckCircle :size="20" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.completed_jobs || 0 }}</div>
          <div class="stat-label">已完成</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--accent-light); color: var(--accent-primary);">
          <Loader2 :size="20" class="spin" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.running_jobs || 0 }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--error-light); color: var(--error);">
          <XCircle :size="20" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.failed_jobs || 0 }}</div>
          <div class="stat-label">失败</div>
        </div>
      </div>
    </div>

    <!-- Active Jobs -->
    <div v-if="activeJobs.length > 0" class="card" style="margin-bottom: 24px;">
      <div class="card-header">
        <h2>
          <Activity :size="18" />
          运行中的任务
        </h2>
        <button class="btn btn-sm btn-secondary" @click="refreshActiveJobs" :disabled="loading">
          <RefreshCw :size="14" :class="{ spin: loading }" />
          刷新
        </button>
      </div>

      <div class="active-jobs">
        <div v-for="job in activeJobs" :key="job.job_id" class="job-card">
          <div class="job-header">
            <div class="job-info">
              <h3>{{ getConfigName(job.config_id) }}</h3>
              <span class="job-id">任务 ID: {{ job.job_id }}</span>
            </div>
            <div class="job-actions">
              <span class="badge badge-info">
                <span class="status-dot running"></span>
                运行中
              </span>
              <button class="btn btn-sm btn-ghost" @click="stopJob(job.job_id)" title="停止">
                <Pause :size="14" />
              </button>
              <button class="btn btn-sm btn-ghost text-error" @click="cancelJob(job.job_id)" title="取消">
                <X :size="14" />
              </button>
            </div>
          </div>

          <!-- Progress -->
          <div class="progress-section">
            <div class="progress-item">
              <div class="progress-header">
                <span class="progress-label">表进度</span>
                <span class="progress-value">{{ job.completed_tables }} / {{ job.total_tables }} 张表</span>
              </div>
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: job.total_tables ? (job.completed_tables / job.total_tables * 100) + '%' : '0%' }"></div>
              </div>
            </div>

            <div class="progress-item">
              <div class="progress-header">
                <span class="progress-label">当前表</span>
                <span class="progress-value">{{ formatNumber(job.processed_rows) }} / {{ formatNumber(job.total_rows) }} 行</span>
              </div>
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: currentTablePercent(job) + '%' }"></div>
              </div>
            </div>
          </div>

          <!-- Table Progress -->
          <div v-if="job.table_progress && Object.keys(job.table_progress).length > 0" class="table-progress">
            <h4>表同步进度</h4>
            <div class="table-list">
              <div v-for="(table, tableName) in job.table_progress" :key="tableName" class="table-item">
                <span class="table-name">{{ table.table_name }}</span>
                <div class="table-status">
                  <span :class="['badge', table.status === 'running' ? 'badge-info' : table.status === 'completed' ? 'badge-success' : 'badge-error']">
                    {{ getStatusText(table.status) }}
                  </span>
                  <span v-if="table.status === 'running'" class="table-rows">
                    {{ formatNumber(table.processed_rows) }} / {{ formatNumber(table.total_rows) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- History -->
    <div class="card">
      <div class="card-header">
        <h2>
          <History :size="18" />
          同步历史
        </h2>
        <button class="btn btn-sm btn-secondary" @click="refreshHistory" :disabled="loading">
          <RefreshCw :size="14" :class="{ spin: loading }" />
          刷新
        </button>
      </div>

      <div v-if="jobHistory.length === 0" class="empty-state">
        <History :size="48" class="empty-state-icon" />
        <h3>暂无同步历史</h3>
        <p>同步任务完成后会在这里显示记录</p>
      </div>

      <div v-else class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>状态</th>
              <th>配置名称</th>
              <th>任务 ID</th>
              <th>表进度</th>
              <th>行数</th>
              <th>开始时间</th>
              <th>耗时</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="job in jobHistory" :key="job.id">
              <td>
                <span :class="['badge', getStatusClass(job.status)]">
                  {{ getStatusText(job.status) }}
                </span>
              </td>
              <td class="font-medium">{{ job.config_name || '未知配置' }}</td>
              <td class="text-secondary">{{ job.id.substring(0, 8) }}...</td>
              <td>{{ job.completed_tables }}/{{ job.total_tables }}</td>
              <td>{{ formatNumber(job.processed_rows) }}</td>
              <td class="text-secondary">{{ formatTime(job.start_time) }}</td>
              <td class="text-secondary">{{ calculateDuration(job.start_time, job.end_time) }}</td>
              <td>
                <button class="btn btn-sm btn-ghost" @click="viewJobLogs(job.id)">日志</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="jobHistory.length > 0" class="pagination">
        <button class="btn btn-sm btn-secondary" @click="previousPage" :disabled="currentPage === 1 || loading">
          <ChevronLeft :size="14" />
          上一页
        </button>
        <span class="page-info">第 {{ currentPage }} 页</span>
        <button class="btn btn-sm btn-secondary" @click="nextPage" :disabled="jobHistory.length < pageSize || loading">
          下一页
          <ChevronRight :size="14" />
        </button>
      </div>
    </div>

    <!-- Logs Modal -->
    <div v-if="showLogsModal" class="modal-overlay" @click="closeLogsModal">
      <div class="modal-content modal-lg" @click.stop>
        <div class="modal-header">
          <h2>
            <FileText :size="18" />
            任务日志
          </h2>
          <button class="modal-close" @click="closeLogsModal">&times;</button>
        </div>
        <div class="modal-body">
          <div v-if="loadingLogs" class="loading-state">
            <div class="loading-spinner"></div>
            <p>加载日志中...</p>
          </div>
          <div v-else-if="jobLogs.length === 0" class="empty-state">
            <p>暂无日志记录</p>
          </div>
          <div v-else class="logs-list">
            <div v-for="log in jobLogs" :key="log.id" class="log-entry" :class="log.level">
              <div class="log-header">
                <span :class="['log-level', log.level]">{{ log.level.toUpperCase() }}</span>
                <span class="log-table">{{ log.table_name }}</span>
                <span class="log-time">{{ formatTime(log.created_at) }}</span>
              </div>
              <div class="log-message">{{ log.message }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { BarChart3, CheckCircle, XCircle, Loader2, Activity, RefreshCw, History, Pause, X, FileText, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import Toast from '../components/Toast.vue'

const syncStore = useSyncStore()

const stats = ref({})
const activeJobs = ref([])
const jobHistory = ref([])
const jobLogs = ref([])
const loading = ref(false)
const loadingLogs = ref(false)
const showLogsModal = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const progressEventSource = ref(null)
const toast = ref({ show: false, message: '', type: 'info' })

onMounted(async () => {
  await loadAllData()
  startProgressStream()
})

onUnmounted(() => {
  stopProgressStream()
})

function startProgressStream() {
  if (progressEventSource.value) return
  const es = new EventSource('/api/sync/jobs/progress/stream')
  progressEventSource.value = es

  es.addEventListener('progress', (e) => {
    try {
      const payload = JSON.parse(e.data)
      const jobs = payload?.data ?? []
      const hadJobs = activeJobs.value.length > 0
      activeJobs.value = jobs
      if (hadJobs && jobs.length === 0) {
        loadStats()
        loadHistory()
      }
    } catch (err) {
      console.error('Failed to parse progress SSE data:', err)
    }
  })

  es.onerror = () => {
    es.close()
    progressEventSource.value = null
    setTimeout(startProgressStream, 3000)
  }
}

function stopProgressStream() {
  if (progressEventSource.value) {
    progressEventSource.value.close()
    progressEventSource.value = null
  }
}

async function loadAllData() {
  loading.value = true
  try {
    await Promise.all([loadStats(), loadActiveJobs(), loadHistory()])
  } catch (error) {
    console.error('Failed to load monitoring data:', error)
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const response = await fetch('/api/sync/stats')
    const result = await response.json()
    if (result.success) stats.value = result.data || {}
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

async function loadActiveJobs() {
  try {
    const response = await fetch('/api/sync/jobs/active')
    const result = await response.json()
    if (result.success) activeJobs.value = result.data || []
  } catch (error) {
    console.error('Failed to load active jobs:', error)
  }
}

async function loadHistory() {
  try {
    const offset = (currentPage.value - 1) * pageSize.value
    const url = `/api/sync/jobs/history?limit=${pageSize.value}&offset=${offset}`
    const response = await fetch(url)
    const result = await response.json()
    if (result.success) jobHistory.value = result.data || []
  } catch (error) {
    console.error('Failed to load job history:', error)
  }
}

async function refreshActiveJobs() {
  await loadActiveJobs()
  await loadStats()
}

async function refreshHistory() {
  loading.value = true
  try {
    await loadHistory()
    await loadStats()
  } finally {
    loading.value = false
  }
}

async function viewJobLogs(jobId) {
  showLogsModal.value = true
  loadingLogs.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/logs`)
    const result = await response.json()
    if (result.success) jobLogs.value = result.data || []
  } catch (error) {
    console.error('Failed to load job logs:', error)
  } finally {
    loadingLogs.value = false
  }
}

function closeLogsModal() {
  showLogsModal.value = false
  jobLogs.value = []
}

async function stopJob(jobId) {
  if (!confirm('确定要停止这个同步任务吗？')) return
  loading.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/stop`, { method: 'POST' })
    const result = await response.json()
    if (result.success) {
      toast.value = { show: true, message: '任务已成功停止', type: 'success' }
      await refreshActiveJobs()
    } else {
      toast.value = { show: true, message: '停止失败: ' + (result.error || '未知错误'), type: 'error' }
    }
  } catch (error) {
    toast.value = { show: true, message: '停止失败: ' + error.message, type: 'error' }
  } finally {
    loading.value = false
  }
}

async function cancelJob(jobId) {
  if (!confirm('确定要取消这个同步任务吗？取消后任务将无法恢复。')) return
  loading.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/cancel`, { method: 'POST' })
    const result = await response.json()
    if (result.success) {
      toast.value = { show: true, message: '任务已成功取消', type: 'success' }
      await refreshActiveJobs()
    } else {
      toast.value = { show: true, message: '取消失败: ' + (result.error || '未知错误'), type: 'error' }
    }
  } catch (error) {
    toast.value = { show: true, message: '取消失败: ' + error.message, type: 'error' }
  } finally {
    loading.value = false
  }
}

function getConfigName(configId) {
  const config = syncStore.configs.find(c => c.id === configId)
  return config?.name || configId.substring(0, 8)
}

function getStatusText(status) {
  const map = { pending: '等待中', running: '运行中', completed: '已完成', failed: '失败', cancelled: '已取消' }
  return map[status] || status
}

function getStatusClass(status) {
  const map = { pending: 'badge-warning', running: 'badge-info', completed: 'badge-success', failed: 'badge-error', cancelled: 'badge-error' }
  return map[status] || 'badge-info'
}

function formatNumber(num) {
  if (!num) return '0'
  return num.toLocaleString()
}

function currentTablePercent(job) {
  const table = getCurrentTableProgress(job)
  if (!table || table.total_rows <= 0) return 0
  return Math.min(100, (table.processed_rows / table.total_rows) * 100)
}

function getCurrentTableProgress(job) {
  if (!job?.table_progress) return null
  const name = job.current_table
  if (name && job.table_progress[name]) return job.table_progress[name]
  return Object.values(job.table_progress).find(t => t.status === 'running') || null
}

function formatTime(timestamp) {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function calculateDuration(startTime, endTime) {
  if (!startTime || !endTime) return '-'
  const seconds = Math.floor((new Date(endTime) - new Date(startTime)) / 1000)
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  return `${Math.floor(seconds / 3600)}时${Math.floor((seconds % 3600) / 60)}分`
}

function previousPage() {
  if (currentPage.value > 1) {
    currentPage.value--
    loadHistory()
  }
}

function nextPage() {
  currentPage.value++
  loadHistory()
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

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-5);
}

.stat-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
}

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 4px;
}

.job-card {
  padding: var(--space-4);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-4);
}

.job-card:last-child {
  margin-bottom: 0;
}

.job-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--space-4);
}

.job-info h3 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-1);
}

.job-id {
  font-size: 12px;
  color: var(--text-secondary);
}

.job-actions {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}

.progress-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.progress-item {
  background: var(--bg-primary);
  padding: var(--space-3);
  border-radius: var(--radius-md);
}

.progress-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--space-2);
}

.progress-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.progress-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.progress-bar {
  height: 6px;
  background: var(--border-secondary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent-primary);
  border-radius: var(--radius-full);
  transition: width 0.3s ease;
}

.table-progress {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-primary);
}

.table-progress h4 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: var(--space-3);
}

.table-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.table-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3);
  background: var(--bg-primary);
  border-radius: var(--radius-md);
}

.table-name {
  font-weight: 500;
}

.table-status {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.table-rows {
  font-size: 13px;
  color: var(--text-secondary);
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: var(--space-4);
  margin-top: var(--space-5);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-secondary);
}

.page-info {
  font-size: 14px;
  color: var(--text-secondary);
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.log-entry {
  padding: var(--space-3);
  border-radius: var(--radius-md);
  border-left: 3px solid var(--border-primary);
}

.log-entry.info {
  background: var(--accent-light);
  border-left-color: var(--accent-primary);
}

.log-entry.warn {
  background: var(--warning-light);
  border-left-color: var(--warning);
}

.log-entry.error {
  background: var(--error-light);
  border-left-color: var(--error);
}

.log-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
}

.log-level {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  text-transform: uppercase;
}

.log-entry.info .log-level {
  background: var(--accent-primary);
  color: white;
}

.log-entry.warn .log-level {
  background: var(--warning);
  color: white;
}

.log-entry.error .log-level {
  background: var(--error);
  color: white;
}

.log-table {
  font-size: 13px;
  color: var(--text-secondary);
}

.log-time {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-left: auto;
}

.log-message {
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.6;
}

.modal-lg {
  max-width: 900px;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .job-header {
    flex-direction: column;
    gap: var(--space-3);
  }

  .job-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
