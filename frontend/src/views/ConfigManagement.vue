<template>
  <div>

    <div class="card">
      <div class="card-header">
        <h2><Upload :size="20" class="inline-icon" /> 导出配置</h2>
      </div>
      <p class="card-description">
        导出当前所有的同步配置，包括连接信息、数据库映射和表映射配置。
      </p>
      <div class="action-buttons">
        <button class="btn btn-primary" @click="handleExport" :disabled="store.loading">
          <Download :size="18" v-if="!store.loading" /> 
          <Loader2 :size="18" class="spin" v-else />
          <span>{{ store.loading ? '导出中...' : '导出配置' }}</span>
        </button>
        <button class="btn btn-secondary" @click="handleBackup" :disabled="store.loading">
          <Save :size="18" v-if="!store.loading" />
          <Loader2 :size="18" class="spin" v-else />
          <span>{{ store.loading ? '备份中...' : '创建备份' }}</span>
        </button>
      </div>
    </div>

    <div class="card">
      <div class="card-header">
        <h2><Download :size="20" class="inline-icon" /> 导入配置</h2>
      </div>
      <p class="card-description">
        从配置文件导入同步配置。系统会自动验证配置文件的有效性。
      </p>

      <div class="import-section">
        <div class="file-upload">
          <label for="config-file" class="file-label">
            <Folder :size="24" class="file-icon" />
            <span v-if="!selectedFile">选择配置文件 (JSON)</span>
            <span v-else class="file-name">{{ selectedFile.name }}</span>
          </label>
          <input
            id="config-file"
            type="file"
            accept=".json"
            @change="handleFileSelect"
            class="file-input"
          />
        </div>

        <div v-if="selectedFile" class="file-actions">
          <button class="btn btn-secondary" @click="handleValidate" :disabled="store.loading || validating">
            <CheckCircle :size="16" v-if="!validating" />
            <Loader2 :size="16" class="spin" v-else />
            <span>{{ validating ? '验证中...' : '验证配置' }}</span>
          </button>
          <button class="btn" @click="clearFile">
            <X :size="16" /> 清除
          </button>
        </div>

        <div v-if="validationResult" class="validation-result" :class="validationResult.valid ? 'success' : 'error'">
          <CheckCircle v-if="validationResult.valid" class="result-icon" :size="32" />
          <XCircle v-else class="result-icon" :size="32" />
          <div class="result-message">
            <strong>{{ validationResult.valid ? '配置有效' : '配置无效' }}</strong>
            <p v-if="validationResult.error">{{ validationResult.error }}</p>
            <p v-else>配置文件格式正确，可以导入</p>
          </div>
        </div>

        <div v-if="validationResult && validationResult.valid" class="import-options">
          <label class="checkbox-label">
            <input type="checkbox" v-model="resolveConflicts" />
            <span>自动解决冲突（覆盖现有配置）</span>
          </label>
          <p class="option-description">
            如果启用，导入时遇到冲突的配置将自动覆盖。否则，导入将失败并显示冲突详情。
          </p>
        </div>

        <div v-if="validationResult && validationResult.valid" class="action-buttons">
          <button class="btn btn-primary" @click="handleImport" :disabled="store.loading">
            <Download :size="18" v-if="!store.loading" />
            <Loader2 :size="18" class="spin" v-else />
            <span>{{ store.loading ? '导入中...' : '导入配置' }}</span>
          </button>
        </div>
      </div>
    </div>

    <div v-if="store.error" class="alert alert-error">
      <strong>错误：</strong>{{ store.error }}
    </div>

    <div v-if="successMessage" class="alert alert-success">
      <strong>成功：</strong>{{ successMessage }}
    </div>

    <!-- Config Preview Modal -->
    <div v-if="showPreview" class="modal-overlay" @click="closePreview">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>配置预览</h3>
          <button class="close-btn" @click="closePreview">✕</button>
        </div>
        <div class="modal-body">
          <pre class="config-preview">{{ JSON.stringify(previewConfig, null, 2) }}</pre>
        </div>
        <div class="modal-footer">
          <button class="btn btn-primary" @click="downloadConfig">
            💾 下载配置文件
          </button>
          <button class="btn" @click="closePreview">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { 
  Upload, Download, Save, Folder, CheckCircle, 
  XCircle, X, Loader2 
} from 'lucide-vue-next'
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
    
    // Read file content
    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        configData.value = JSON.parse(e.target.result)
      } catch (error) {
        validationResult.value = {
          valid: false,
          error: '无效的 JSON 格式'
        }
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
    validationResult.value = {
      valid: false,
      error: '请先选择配置文件'
    }
    return
  }

  validating.value = true
  try {
    const result = await store.validateConfig(configData.value)
    validationResult.value = result
  } catch (error) {
    validationResult.value = {
      valid: false,
      error: error.message
    }
  } finally {
    validating.value = false
  }
}

async function handleImport() {
  if (!configData.value) {
    return
  }

  try {
    await store.importConfig(configData.value, resolveConflicts.value)
    successMessage.value = '配置导入成功！'
    clearFile()
    
    // Clear success message after 5 seconds
    setTimeout(() => {
      successMessage.value = ''
    }, 5000)
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
  setTimeout(() => {
    successMessage.value = ''
  }, 5000)
}
</script>

<style scoped>
.card {
  background: white;
  border-radius: 10px;
  padding: 2rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
}

.card-header {
  margin-bottom: 1rem;
}

.card-header h2 {
  color: #667eea;
  font-size: 1.5rem;
  margin: 0;
}

.card-description {
  color: #666;
  margin-bottom: 1.5rem;
  line-height: 1.6;
}

.action-buttons {
  display: flex;
  gap: 1rem;
  margin-top: 1.5rem;
}

.btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 5px;
  font-size: 1rem;
  cursor: pointer;
  transition: all 0.2s;
  background: #e0e0e0;
  color: #333;
}

.btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-secondary {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: white;
}

.import-section {
  margin-top: 1.5rem;
}

.file-upload {
  margin-bottom: 1rem;
}

.file-label {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 2rem;
  background: #f5f5f5;
  border: 2px dashed #ccc;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.file-label:hover {
  background: #e8e8e8;
  border-color: #667eea;
}

.file-icon {
  color: #667eea;
  flex-shrink: 0;
}

.file-name {
  color: #667eea;
  font-weight: 500;
}

.file-input {
  display: none;
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

.file-actions {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.validation-result {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1rem;
  border-radius: 8px;
  margin-bottom: 1.5rem;
}

.validation-result.success {
  background: #e8f5e9;
  border: 1px solid #4caf50;
}

.validation-result.error {
  background: #ffebee;
  border: 1px solid #f44336;
}

.result-icon {
  flex-shrink: 0;
}

.validation-result.success .result-icon {
  color: #4caf50;
}

.validation-result.error .result-icon {
  color: #f44336;
}

.result-message {
  flex: 1;
}

.result-message strong {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 1.1rem;
}

.result-message p {
  margin: 0;
  color: #666;
}

.import-options {
  background: #f9f9f9;
  padding: 1rem;
  border-radius: 8px;
  margin-bottom: 1.5rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-weight: 500;
}

.checkbox-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.option-description {
  margin: 0.5rem 0 0 1.75rem;
  color: #666;
  font-size: 0.9rem;
  line-height: 1.5;
}

.alert {
  padding: 1rem;
  border-radius: 8px;
  margin-bottom: 1rem;
}

.alert-error {
  background: #ffebee;
  border: 1px solid #f44336;
  color: #c62828;
}

.alert-success {
  background: #e8f5e9;
  border: 1px solid #4caf50;
  color: #2e7d32;
}

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
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e0e0e0;
}

.modal-header h3 {
  margin: 0;
  color: #667eea;
  font-size: 1.5rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #999;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s;
}

.close-btn:hover {
  background: #f5f5f5;
  color: #333;
}

.modal-body {
  flex: 1;
  overflow: auto;
  padding: 1.5rem;
}

.config-preview {
  background: #f5f5f5;
  padding: 1rem;
  border-radius: 8px;
  overflow: auto;
  font-family: 'Courier New', monospace;
  font-size: 0.9rem;
  line-height: 1.5;
  margin: 0;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid #e0e0e0;
}
</style>
