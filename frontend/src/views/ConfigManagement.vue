<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <h1>配置管理</h1>
    </div>

    <!-- Export Card -->
    <div class="card" style="margin-bottom: 24px;">
      <div class="card-header">
        <h2>
          <Upload :size="18" />
          导出配置
        </h2>
      </div>
      <p class="text-secondary" style="margin-bottom: 16px;">
        导出当前所有的同步配置，包括连接信息、数据库映射和表映射配置。
      </p>
      <div class="action-buttons">
        <button class="btn btn-primary" @click="handleExport" :disabled="store.loading">
          <Loader2 v-if="store.loading" :size="16" class="spin" />
          <Download v-else :size="16" />
          <span>{{ store.loading ? '导出中...' : '导出配置' }}</span>
        </button>
        <button class="btn btn-secondary" @click="handleBackup" :disabled="store.loading">
          <Save :size="16" />
          <span>创建备份</span>
        </button>
      </div>
    </div>

    <!-- Import Card -->
    <div class="card">
      <div class="card-header">
        <h2>
          <Download :size="18" />
          导入配置
        </h2>
      </div>
      <p class="text-secondary" style="margin-bottom: 16px;">
        从配置文件导入同步配置。系统会自动验证配置文件的有效性。
      </p>

      <div class="import-section">
        <!-- File Upload -->
        <div class="file-upload" :class="{ 'has-file': selectedFile }">
          <input
            id="config-file"
            type="file"
            accept=".json"
            @change="handleFileSelect"
            class="file-input"
          />
          <label for="config-file" class="file-label">
            <FolderOpen :size="24" />
            <span v-if="!selectedFile">选择配置文件 (JSON)</span>
            <span v-else class="file-name">{{ selectedFile.name }}</span>
          </label>
        </div>

        <!-- File Actions -->
        <div v-if="selectedFile" class="file-actions">
          <button class="btn btn-secondary" @click="handleValidate" :disabled="store.loading || validating">
            <Loader2 v-if="validating" :size="14" class="spin" />
            <CheckCircle v-else :size="14" />
            <span>{{ validating ? '验证中...' : '验证配置' }}</span>
          </button>
          <button class="btn btn-ghost" @click="clearFile">
            <X :size="14" />
            <span>清除</span>
          </button>
        </div>

        <!-- Validation Result -->
        <div v-if="validationResult" :class="['validation-result', validationResult.valid ? 'success' : 'error']">
          <CheckCircle v-if="validationResult.valid" :size="24" />
          <XCircle v-else :size="24" />
          <div>
            <strong>{{ validationResult.valid ? '配置有效' : '配置无效' }}</strong>
            <p v-if="validationResult.error" class="text-secondary">{{ validationResult.error }}</p>
            <p v-else class="text-secondary">配置文件格式正确，可以导入</p>
          </div>
        </div>

        <!-- Import Options -->
        <div v-if="validationResult && validationResult.valid" class="import-options">
          <label class="checkbox-label">
            <input type="checkbox" v-model="resolveConflicts" />
            <span>自动解决冲突（覆盖现有配置）</span>
          </label>
          <p class="option-hint">如果启用，导入时遇到冲突的配置将自动覆盖。否则，导入将失败并显示冲突详情。</p>
        </div>

        <!-- Import Button -->
        <div v-if="validationResult && validationResult.valid" class="file-actions">
          <button class="btn btn-primary" @click="handleImport" :disabled="store.loading">
            <Loader2 v-if="store.loading" :size="16" class="spin" />
            <Download v-else :size="16" />
            <span>{{ store.loading ? '导入中...' : '导入配置' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Alert Messages -->
    <div v-if="store.error" class="alert alert-error" style="margin-top: 16px;">
      <AlertCircle :size="18" />
      <span>{{ store.error }}</span>
    </div>

    <div v-if="successMessage" class="alert alert-success" style="margin-top: 16px;">
      <CheckCircle :size="18" />
      <span>{{ successMessage }}</span>
    </div>

    <!-- Preview Modal -->
    <div v-if="showPreview" class="modal-overlay" @click="closePreview">
      <div class="modal-content modal-lg" @click.stop>
        <div class="modal-header">
          <h2>配置预览</h2>
          <button class="modal-close" @click="closePreview">&times;</button>
        </div>
        <div class="modal-body">
          <pre class="config-preview">{{ JSON.stringify(previewConfig, null, 2) }}</pre>
        </div>
        <div class="modal-footer">
          <button class="btn btn-primary" @click="downloadConfig">
            <Download :size="16" />
            <span>下载配置文件</span>
          </button>
          <button class="btn btn-secondary" @click="closePreview">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Upload, Download, Save, FolderOpen, CheckCircle, XCircle, X, Loader2, AlertCircle } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'

const store = useSyncStore()

const selectedFile = ref(null)
const configData = ref(null)
const validating = ref(false)
const validationResult = ref(null)
const resolveConflicts = ref(false)
const successMessage = ref('')
const showPreview = ref(false)
const previewConfig = ref(null)

function handleFileSelect(event) {
  const file = event.target.files[0]
  if (file) {
    selectedFile.value = file
    validationResult.value = null
    successMessage.value = ''

    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        configData.value = JSON.parse(e.target.result)
      } catch (error) {
        validationResult.value = { valid: false, error: '无效的 JSON 格式' }
      }
    }
    reader.readAsText(file)
  }
}

function clearFile() {
  selectedFile.value = null
  configData.value = null
  validationResult.value = null
  successMessage.value = ''
  document.getElementById('config-file').value = ''
}

async function handleValidate() {
  if (!configData.value) {
    validationResult.value = { valid: false, error: '请先选择配置文件' }
    return
  }
  validating.value = true
  try {
    validationResult.value = await store.validateConfig(configData.value)
  } catch (error) {
    validationResult.value = { valid: false, error: error.message }
  } finally {
    validating.value = false
  }
}

async function handleImport() {
  if (!configData.value) return
  try {
    await store.importConfig(configData.value, resolveConflicts.value)
    successMessage.value = '配置导入成功！'
    clearFile()
    setTimeout(() => { successMessage.value = '' }, 5000)
  } catch (error) {
    console.error('Import failed:', error)
  }
}

async function handleExport() {
  try {
    const config = await store.exportConfig()
    previewConfig.value = config
    showPreview.value = true
  } catch (error) {
    console.error('Export failed:', error)
  }
}

async function handleBackup() {
  try {
    const backup = await store.backupConfig()
    previewConfig.value = backup
    showPreview.value = true
  } catch (error) {
    console.error('Backup failed:', error)
  }
}

function closePreview() {
  showPreview.value = false
  previewConfig.value = null
}

function downloadConfig() {
  if (!previewConfig.value) return
  const dataStr = JSON.stringify(previewConfig.value, null, 2)
  const dataBlob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(dataBlob)
  const link = document.createElement('a')
  link.href = url
  link.download = `sync-config-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  closePreview()
  successMessage.value = '配置文件已下载！'
  setTimeout(() => { successMessage.value = '' }, 5000)
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

.action-buttons {
  display: flex;
  gap: var(--space-3);
}

.import-section {
  margin-top: var(--space-4);
}

.file-upload {
  margin-bottom: var(--space-4);
}

.file-input {
  display: none;
}

.file-label {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  padding: var(--space-6);
  border: 2px dashed var(--border-primary);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--text-secondary);
}

.file-label:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
  background: var(--accent-light);
}

.file-upload.has-file .file-label {
  border-color: var(--success);
  color: var(--success);
  background: var(--success-light);
}

.file-name {
  font-weight: 500;
}

.file-actions {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.validation-result {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-4);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-4);
}

.validation-result.success {
  background: var(--success-light);
  color: var(--success);
}

.validation-result.error {
  background: var(--error-light);
  color: var(--error);
}

.validation-result strong {
  display: block;
  margin-bottom: var(--space-1);
}

.import-options {
  background: var(--bg-secondary);
  padding: var(--space-4);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-4);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  cursor: pointer;
  font-weight: 500;
}

.checkbox-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.option-hint {
  margin: var(--space-2) 0 0 30px;
  font-size: 13px;
  color: var(--text-secondary);
}

.config-preview {
  background: var(--bg-secondary);
  padding: var(--space-4);
  border-radius: var(--radius-md);
  overflow: auto;
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.6;
  margin: 0;
  max-height: 60vh;
}

.modal-lg {
  max-width: 800px;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
