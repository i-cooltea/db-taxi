<template>
  <div>

    <!-- Statistics Overview -->
    <div class="stats-grid">
      <div class="stat-card">
        <BarChart3 class="stat-icon" :size="32" />
        <div class="stat-content">
          <div class="stat-value">{{ stats.total_jobs || 0 }}</div>
          <div class="stat-label">总任务数</div>
        </div>
      </div>
      <div class="stat-card success">
        <CheckCircle class="stat-icon" :size="32" />
        <div class="stat-content">
          <div class="stat-value">{{ stats.completed_jobs || 0 }}</div>
          <div class="stat-label">已完成</div>
        </div>
      </div>
      <div class="stat-card running">
        <Loader class="stat-icon" :size="32" />
        <div class="stat-content">
          <div class="stat-value">{{ stats.running_jobs || 0 }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </div>
      <div class="stat-card error">
        <XCircle class="stat-icon" :size="32" />
        <div class="stat-content">
          <div class="stat-value">{{ stats.failed_jobs || 0 }}</div>
          <div class="stat-label">失败</div>
        </div>
      </div>
    </div>

    <!-- Active Jobs Section -->
    <div class="card" v-if="activeJobs.length > 0">
      <div class="card-header">
        <h2><Activity :size="20" class="inline-icon" /> 运行中的任务</h2>
        <button @click="refreshActiveJobs" class="btn btn-secondary" :disabled="loading">
          <RefreshCw :size="16" :class="{ 'spin': loading }" /> {{ loading ? '刷新中...' : '刷新' }}
        </button>
      </div>
      <div class="active-jobs">
        <div v-for="job in activeJobs" :key="job.job_id" class="job-card active">
          <div class="job-header">
            <div class="job-info">
              <h3>{{ getConfigName(job.config_id) }}</h3>
              <span class="job-id">任务 ID: {{ job.job_id }}</span>
            </div>
            <div class="job-status">
              <span class="status-badge running">运行中</span>
              <div class="job-actions">
                <button @click="stopJob(job.job_id)" class="btn btn-sm btn-warning" :disabled="loading">
                  <Pause :size="14" /> 停止
                </button>
                <button @click="cancelJob(job.job_id)" class="btn btn-sm btn-danger" :disabled="loading">
                  <X :size="14" /> 取消
                </button>
                <button @click="viewJobDetails(job.job_id)" class="btn btn-sm">查看详情</button>
              </div>
            </div>
          </div>
          
          <div class="job-progress">
            <div class="progress-info">
              <span>进度: {{ job.completed_tables }}/{{ job.total_tables }} 表</span>
              <span>{{ job.progress_percent?.toFixed(1) || 0 }}%</span>
            </div>
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: (job.progress_percent || 0) + '%' }"></div>
            </div>
          </div>

          <div class="job-stats">
            <div class="stat-item">
              <span class="stat-label">已处理行数:</span>
              <span class="stat-value time-value">{{ formatNumber(job.processed_rows) }} / {{ formatNumber(job.total_rows) }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">开始时间:</span>
              <span class="stat-value time-value">{{ formatTime(job.start_time) }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">错误数:</span>
              <span class="stat-value" :class="{ 'error-text': job.error_count > 0 }">{{ job.error_count || 0 }}</span>
            </div>
          </div>

          <!-- Table Progress Details -->
          <div v-if="job.table_progress && Object.keys(job.table_progress).length > 0" class="table-progress">
            <h4>表同步进度</h4>
            <div class="table-list">
              <div v-for="(table, tableName) in job.table_progress" :key="tableName" class="table-item">
                <div class="table-name">{{ table.table_name }}</div>
                <div class="table-status">
                  <span class="status-badge" :class="table.status">{{ getStatusText(table.status) }}</span>
                  <span v-if="table.status === 'running'" class="table-progress-text">
                    {{ formatNumber(table.processed_rows) }} / {{ formatNumber(table.total_rows) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Job History Section -->
    <div class="card">
      <div class="card-header">
        <h2><History :size="20" class="inline-icon" /> 同步历史</h2>
        <div class="header-actions">
          <button @click="showStartSyncModal = true" class="btn btn-primary" :disabled="loading">
            <Play :size="16" /> 启动同步
          </button>
          <button @click="refreshHistory" class="btn btn-secondary" :disabled="loading">
            <RefreshCw :size="16" :class="{ 'spin': loading }" /> {{ loading ? '刷新中...' : '刷新' }}
          </button>
        </div>
      </div>

      <div v-if="jobHistory.length === 0" class="empty-state">
        <p>暂无同步历史记录</p>
      </div>

      <div v-else class="history-list">
        <div v-for="job in jobHistory" :key="job.id" class="job-card">
          <div class="job-header">
            <div class="job-info">
              <h3>{{ job.config_name || '未知配置' }}</h3>
              <span class="job-id">任务 ID: {{ job.id }}</span>
              <span class="connection-name">连接: {{ job.connection_name || '未知' }}</span>
            </div>
            <div class="job-status">
              <span class="status-badge" :class="job.status">{{ getStatusText(job.status) }}</span>
              <button @click="viewJobLogs(job.id)" class="btn btn-sm">查看日志</button>
            </div>
          </div>

          <div class="job-stats">
            <div class="stat-item">
              <span class="stat-label">表数量:</span>
              <span class="stat-value time-value">{{ job.completed_tables }} / {{ job.total_tables }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">处理行数:</span>
              <span class="stat-value time-value">{{ formatNumber(job.processed_rows) }} / {{ formatNumber(job.total_rows) }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">开始时间:</span>
              <span class="stat-value time-value">{{ formatTime(job.start_time) }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">结束时间:</span>
              <span class="stat-value time-value">{{ job.end_time ? formatTime(job.end_time) : '-' }}</span>
            </div>
            <div class="stat-item" v-if="job.end_time">
              <span class="stat-label">耗时:</span>
              <span class="stat-value time-value">{{ calculateDuration(job.start_time, job.end_time) }}</span>
            </div>
          </div>

          <div v-if="job.error" class="job-error">
            <strong>错误信息:</strong> {{ job.error }}
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="jobHistory.length > 0" class="pagination">
        <button 
          @click="previousPage" 
          :disabled="currentPage === 1 || loading"
          class="btn btn-secondary"
        >
          上一页
        </button>
        <span class="page-info">第 {{ currentPage }} 页</span>
        <button 
          @click="nextPage" 
          :disabled="jobHistory.length < pageSize || loading"
          class="btn btn-secondary"
        >
          下一页
        </button>
      </div>
    </div>

    <!-- Job Logs Modal -->
    <div v-if="showLogsModal" class="modal-overlay" @click="closeLogsModal">
      <div class="modal-content logs-modal" @click.stop>
        <div class="modal-header">
          <h2><FileText :size="20" class="inline-icon" /> 任务日志</h2>
          <button @click="closeLogsModal" class="close-btn">×</button>
        </div>
        <div class="modal-body">
          <div v-if="loadingLogs" class="loading-state">
            <p>加载日志中...</p>
          </div>
          <div v-else-if="jobLogs.length === 0" class="empty-state">
            <p>暂无日志记录</p>
          </div>
          <div v-else class="logs-list">
            <div v-for="log in jobLogs" :key="log.id" class="log-entry" :class="log.level">
              <div class="log-header">
                <span class="log-level" :class="log.level">{{ log.level.toUpperCase() }}</span>
                <span class="log-time">{{ formatTime(log.created_at) }}</span>
              </div>
              <div class="log-table">表: {{ log.table_name }}</div>
              <div class="log-message">{{ log.message }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Start Sync Modal -->
    <div v-if="showStartSyncModal" class="modal-overlay" @click="closeStartSyncModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2><Play :size="20" class="inline-icon" /> 启动同步任务</h2>
          <button @click="closeStartSyncModal" class="close-btn">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>选择同步配置</label>
            <select v-model="selectedConfigId" class="form-control">
              <option value="">请选择配置...</option>
              <option v-for="config in availableConfigs" :key="config.id" :value="config.id">
                {{ config.name }} ({{ config.connection_name }})
              </option>
            </select>
          </div>
          <div v-if="selectedConfigId" class="config-info">
            <h4>配置详情</h4>
            <div class="info-item">
              <span class="info-label">连接:</span>
              <span class="info-value">{{ getSelectedConfigInfo().connection_name }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">同步模式:</span>
              <span class="info-value">{{ getSelectedConfigInfo().sync_mode === 'full' ? '全量同步' : '增量同步' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">表数量:</span>
              <span class="info-value">{{ getSelectedConfigInfo().table_count || 0 }}</span>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeStartSyncModal" class="btn btn-secondary">取消</button>
          <button @click="startSync" class="btn btn-primary" :disabled="!selectedConfigId || loading">
            {{ loading ? '启动中...' : '启动同步' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { 
  BarChart3, CheckCircle, XCircle, Loader, Activity, 
  RefreshCw, History, Play, Pause, X, FileText 
} from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'

const syncStore = useSyncStore()

// State
const stats = ref({})
const activeJobs = ref([])
const jobHistory = ref([])
const jobLogs = ref([])
const availableConfigs = ref([])
const selectedConfigId = ref('')
const loading = ref(false)
const loadingLogs = ref(false)
const showLogsModal = ref(false)
const showStartSyncModal = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const refreshInterval = ref(null)

// Load initial data
onMounted(async () => {
  console.log('Monitoring component mounted')
  await loadAllData()
  // Auto-refresh every 5 seconds
  refreshInterval.value = setInterval(async () => {
    await refreshActiveJobs()
  }, 5000)
})

onUnmounted(() => {
  console.log('Monitoring component unmounted')
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})

async function loadAllData() {
  console.log('Loading all monitoring data...')
  loading.value = true
  try {
    await Promise.all([
      loadStats(),
      loadActiveJobs(),
      loadHistory(),
      loadAvailableConfigs()
    ])
    console.log('All data loaded successfully')
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
    if (result.success) {
      stats.value = result.data || {}
    }
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

async function loadActiveJobs() {
  try {
    const response = await fetch('/api/sync/jobs/active')
    const result = await response.json()
    if (result.success) {
      activeJobs.value = result.data || []
    }
  } catch (error) {
    console.error('Failed to load active jobs:', error)
  }
}

async function loadHistory() {
  try {
    const offset = (currentPage.value - 1) * pageSize.value
    const url = `/api/sync/jobs/history?limit=${pageSize.value}&offset=${offset}`
    console.log('Loading history from:', url)
    const response = await fetch(url)
    const result = await response.json()
    console.log('History response:', result)
    if (result.success) {
      jobHistory.value = result.data || []
      console.log('Job history loaded:', jobHistory.value.length, 'items')
    } else {
      console.error('Failed to load history:', result.error)
    }
  } catch (error) {
    console.error('Failed to load job history:', error)
  }
}

async function loadAvailableConfigs() {
  try {
    const response = await fetch('/api/sync/configs')
    const result = await response.json()
    if (result.success) {
      availableConfigs.value = result.data || []
    }
  } catch (error) {
    console.error('Failed to load available configs:', error)
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
    if (result.success) {
      jobLogs.value = result.data || []
    }
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

function closeStartSyncModal() {
  showStartSyncModal.value = false
  selectedConfigId.value = ''
}

async function startSync() {
  if (!selectedConfigId.value) return
  
  loading.value = true
  try {
    const response = await fetch('/api/sync/jobs', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        config_id: selectedConfigId.value
      })
    })
    
    const result = await response.json()
    if (result.success) {
      alert('同步任务已启动！')
      closeStartSyncModal()
      await loadAllData()
    } else {
      alert('启动失败: ' + (result.error || '未知错误'))
    }
  } catch (error) {
    console.error('Failed to start sync:', error)
    alert('启动失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

async function stopJob(jobId) {
  if (!confirm('确定要停止这个同步任务吗？')) return
  
  loading.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/stop`, {
      method: 'POST'
    })
    
    const result = await response.json()
    if (result.success) {
      alert('任务已停止')
      await refreshActiveJobs()
    } else {
      alert('停止失败: ' + (result.error || '未知错误'))
    }
  } catch (error) {
    console.error('Failed to stop job:', error)
    alert('停止失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

async function cancelJob(jobId) {
  if (!confirm('确定要取消这个同步任务吗？取消后任务将无法恢复。')) return
  
  loading.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/cancel`, {
      method: 'POST'
    })
    
    const result = await response.json()
    if (result.success) {
      alert('任务已取消')
      await refreshActiveJobs()
    } else {
      alert('取消失败: ' + (result.error || '未知错误'))
    }
  } catch (error) {
    console.error('Failed to cancel job:', error)
    alert('取消失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

function getSelectedConfigInfo() {
  const config = availableConfigs.value.find(c => c.id === selectedConfigId.value)
  return config || {}
}

function viewJobDetails(jobId) {
  // Could navigate to a detailed job view or show a modal
  console.log('View job details:', jobId)
}

function getConfigName(configId) {
  const config = syncStore.configs.find(c => c.id === configId)
  return config?.name || configId
}

function getStatusText(status) {
  const statusMap = {
    'pending': '等待中',
    'running': '运行中',
    'completed': '已完成',
    'failed': '失败',
    'cancelled': '已取消'
  }
  return statusMap[status] || status
}

function formatNumber(num) {
  if (!num) return '0'
  return num.toLocaleString()
}

function formatTime(timestamp) {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function calculateDuration(startTime, endTime) {
  if (!startTime || !endTime) return '-'
  const start = new Date(startTime)
  const end = new Date(endTime)
  const durationMs = end - start
  const seconds = Math.floor(durationMs / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  
  if (hours > 0) {
    return `${hours}小时${minutes % 60}分钟`
  } else if (minutes > 0) {
    return `${minutes}分钟${seconds % 60}秒`
  } else {
    return `${seconds}秒`
  }
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
/* Statistics Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: white;
  border-radius: 10px;
  padding: 1.5rem;
  display: flex;
  align-items: center;
  gap: 1rem;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  border-left: 4px solid #667eea;
}

.stat-card.success {
  border-left-color: #10b981;
}

.stat-card.running {
  border-left-color: #3b82f6;
}

.stat-card.error {
  border-left-color: #ef4444;
}

.stat-icon {
  color: #667eea;
  flex-shrink: 0;
}

.stat-card.success .stat-icon {
  color: #10b981;
}

.stat-card.running .stat-icon {
  color: #3b82f6;
}

.stat-card.error .stat-icon {
  color: #ef4444;
}

.inline-icon {
  display: inline-block;
  vertical-align: middle;
  margin-right: 0.25rem;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 2rem;
  font-weight: bold;
  color: #1f2937;
}

.stat-label {
  font-size: 0.875rem;
  color: #6b7280;
}

/* Card Styles */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.card h2 {
  color: #667eea;
  font-size: 1.5rem;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

/* Job Cards */
.active-jobs,
.history-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.job-card {
  background: #f9fafb;
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid #e5e7eb;
}

.job-card.active {
  border-left: 4px solid #3b82f6;
  background: #eff6ff;
}

.job-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.job-info h3 {
  margin: 0 0 0.5rem 0;
  color: #1f2937;
  font-size: 1.125rem;
}

.job-id,
.connection-name {
  display: block;
  font-size: 0.875rem;
  color: #6b7280;
  margin-top: 0.25rem;
}

.job-status {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.5rem;
}

.job-actions {
  display: flex;
  gap: 0.5rem;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.875rem;
  font-weight: 500;
}

.status-badge.pending {
  background: #fef3c7;
  color: #92400e;
}

.status-badge.running {
  background: #dbeafe;
  color: #1e40af;
}

.status-badge.completed {
  background: #d1fae5;
  color: #065f46;
}

.status-badge.failed {
  background: #fee2e2;
  color: #991b1b;
}

.status-badge.cancelled {
  background: #f3f4f6;
  color: #4b5563;
}

/* Progress Bar */
.job-progress {
  margin: 1rem 0;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
  color: #6b7280;
}

.progress-bar {
  height: 8px;
  background: #e5e7eb;
  border-radius: 9999px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #2563eb);
  transition: width 0.3s ease;
}

/* Job Stats */
.job-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-top: 1rem;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stat-item .stat-label {
  font-size: 0.875rem;
  color: #6b7280;
}

.stat-item .stat-value {
  font-weight: 500;
  color: #1f2937;
}

.stat-item .stat-value.time-value {
  font-size: 18px;
}

.error-text {
  color: #ef4444 !important;
}

.job-error {
  margin-top: 1rem;
  padding: 1rem;
  background: #fee2e2;
  border-radius: 6px;
  color: #991b1b;
  font-size: 0.875rem;
}

/* Table Progress */
.table-progress {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid #e5e7eb;
}

.table-progress h4 {
  margin: 0 0 1rem 0;
  color: #1f2937;
  font-size: 1rem;
}

.table-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.table-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  background: white;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

.table-name {
  font-weight: 500;
  color: #1f2937;
}

.table-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.table-progress-text {
  font-size: 0.875rem;
  color: #6b7280;
}

/* Pagination */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  margin-top: 2rem;
  padding-top: 1rem;
  border-top: 1px solid #e5e7eb;
}

.page-info {
  color: #6b7280;
  font-size: 0.875rem;
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 3rem;
  color: #6b7280;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 10px;
  width: 90%;
  max-width: 800px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.logs-modal {
  max-height: 80vh;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e5e7eb;
}

.modal-header h2 {
  margin: 0;
  color: #1f2937;
  font-size: 1.5rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 2rem;
  color: #6b7280;
  cursor: pointer;
  padding: 0;
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.close-btn:hover {
  background: #f3f4f6;
}

.modal-body {
  padding: 1.5rem;
  overflow-y: auto;
}

/* Logs List */
.logs-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.log-entry {
  padding: 1rem;
  border-radius: 6px;
  border-left: 4px solid #e5e7eb;
}

.log-entry.info {
  background: #f0f9ff;
  border-left-color: #3b82f6;
}

.log-entry.warn {
  background: #fffbeb;
  border-left-color: #f59e0b;
}

.log-entry.error {
  background: #fef2f2;
  border-left-color: #ef4444;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.log-level {
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.log-level.info {
  background: #dbeafe;
  color: #1e40af;
}

.log-level.warn {
  background: #fef3c7;
  color: #92400e;
}

.log-level.error {
  background: #fee2e2;
  color: #991b1b;
}

.log-time {
  font-size: 0.875rem;
  color: #6b7280;
}

.log-table {
  font-size: 0.875rem;
  color: #6b7280;
  margin-bottom: 0.5rem;
}

.log-message {
  color: #1f2937;
  line-height: 1.5;
}

.loading-state {
  text-align: center;
  padding: 2rem;
  color: #6b7280;
}

/* Button Styles */
.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: #667eea;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #5568d3;
}

.btn-secondary {
  background: #f3f4f6;
  color: #1f2937;
}

.btn-secondary:hover:not(:disabled) {
  background: #e5e7eb;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.8125rem;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-warning {
  background: #f59e0b;
  color: white;
}

.btn-warning:hover:not(:disabled) {
  background: #d97706;
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #dc2626;
}

/* Modal Footer */
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 1.5rem;
  border-top: 1px solid #e5e7eb;
}

/* Form Styles */
.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #1f2937;
}

.form-control {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 0.875rem;
  transition: border-color 0.2s;
}

.form-control:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

/* Config Info */
.config-info {
  background: #f9fafb;
  border-radius: 6px;
  padding: 1rem;
  margin-top: 1rem;
}

.config-info h4 {
  margin: 0 0 1rem 0;
  color: #1f2937;
  font-size: 1rem;
}

.info-item {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem 0;
  border-bottom: 1px solid #e5e7eb;
}

.info-item:last-child {
  border-bottom: none;
}

.info-label {
  color: #6b7280;
  font-size: 0.875rem;
}

.info-value {
  color: #1f2937;
  font-weight: 500;
  font-size: 0.875rem;
}
</style>
