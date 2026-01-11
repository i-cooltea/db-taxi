<template>
  <div class="modal" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ connection ? '编辑数据库连接' : '添加数据库连接' }}</h2>
        <span class="close" @click="$emit('close')">&times;</span>
      </div>

      <div v-if="error" class="alert alert-error">{{ error }}</div>

      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="name">连接名称 *</label>
          <input 
            id="name" 
            v-model="formData.name" 
            type="text" 
            required
            placeholder="例如: 生产数据库"
          >
          <small>为此连接指定一个易于识别的名称</small>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="host">主机地址 *</label>
            <input 
              id="host" 
              v-model="formData.host" 
              type="text" 
              required
              placeholder="例如: localhost 或 192.168.1.100"
            >
          </div>

          <div class="form-group">
            <label for="port">端口 *</label>
            <input 
              id="port" 
              v-model.number="formData.port" 
              type="number" 
              required
              min="1"
              max="65535"
              placeholder="3306"
            >
          </div>
        </div>

        <div class="form-group">
          <label for="database">数据库名称 *</label>
          <input 
            id="database" 
            v-model="formData.database" 
            type="text" 
            required
            placeholder="例如: myapp_production"
          >
          <small>远程数据库的名称</small>
        </div>

        <div class="form-group">
          <label for="local-db-name">本地数据库名称 *</label>
          <input 
            id="local-db-name" 
            v-model="formData.local_db_name" 
            type="text" 
            required
            placeholder="例如: myapp_local"
          >
          <small>数据将同步到此本地数据库（如不存在将自动创建）</small>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="username">用户名 *</label>
            <input 
              id="username" 
              v-model="formData.username" 
              type="text" 
              required
              placeholder="数据库用户名"
            >
          </div>

          <div class="form-group">
            <label for="password">密码 *</label>
            <input 
              id="password" 
              v-model="formData.password" 
              :type="showPassword ? 'text' : 'password'"
              required
              placeholder="数据库密码"
            >
            <button 
              type="button" 
              class="password-toggle"
              @click="showPassword = !showPassword"
            >
              {{ showPassword ? '👁️' : '👁️‍🗨️' }}
            </button>
          </div>
        </div>

        <div class="form-group">
          <label class="checkbox-label">
            <input type="checkbox" v-model="formData.ssl">
            启用 SSL 连接
          </label>
          <small>建议在生产环境中启用 SSL 以确保数据传输安全</small>
        </div>

        <div class="form-actions">
          <button type="button" class="btn btn-secondary" @click="$emit('close')">
            取消
          </button>
          <button 
            type="button" 
            class="btn btn-secondary" 
            @click="testConnection"
            :disabled="testing"
          >
            {{ testing ? '测试中...' : '🔍 测试连接' }}
          </button>
          <button type="submit" class="btn" :disabled="saving">
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </form>

      <div v-if="testResult" class="test-result" :class="testResult.success ? 'success' : 'error'">
        <strong>{{ testResult.success ? '✅ 连接成功' : '❌ 连接失败' }}</strong>
        <p v-if="testResult.message">{{ testResult.message }}</p>
        <p v-if="testResult.latency">延迟: {{ testResult.latency }}ms</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'

const props = defineProps({
  connection: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'save'])

const error = ref(null)
const saving = ref(false)
const testing = ref(false)
const showPassword = ref(false)
const testResult = ref(null)

const formData = reactive({
  name: '',
  host: '',
  port: 3306,
  database: '',
  local_db_name: '',
  username: '',
  password: '',
  ssl: false
})

onMounted(() => {
  if (props.connection) {
    // Load existing connection data
    Object.assign(formData, props.connection.config)
  }
})

async function testConnection() {
  error.value = null
  testResult.value = null
  testing.value = true

  try {
    // Create a temporary connection config for testing
    const testConfig = { ...formData }
    
    const response = await fetch('/api/sync/connections/test', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(testConfig)
    })

    const result = await response.json()

    if (result.success) {
      testResult.value = {
        success: true,
        message: '数据库连接正常',
        latency: result.data?.latency_ms
      }
    } else {
      testResult.value = {
        success: false,
        message: result.error || '连接失败'
      }
    }
  } catch (err) {
    testResult.value = {
      success: false,
      message: err.message
    }
  } finally {
    testing.value = false
  }
}

async function handleSubmit() {
  error.value = null
  testResult.value = null

  // Validate required fields
  if (!formData.name || !formData.host || !formData.database || 
      !formData.local_db_name || !formData.username || !formData.password) {
    error.value = '请填写所有必填字段'
    return
  }

  saving.value = true
  try {
    await emit('save', { ...formData })
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
  max-width: 700px;
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

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-group {
  position: relative;
}

.password-toggle {
  position: absolute;
  right: 10px;
  top: 38px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0;
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

.test-result {
  margin-top: 1.5rem;
  padding: 1rem;
  border-radius: 5px;
  animation: slideIn 0.3s;
}

.test-result.success {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.test-result.error {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.test-result strong {
  display: block;
  margin-bottom: 0.5rem;
}

.test-result p {
  margin: 0.25rem 0;
}
</style>
