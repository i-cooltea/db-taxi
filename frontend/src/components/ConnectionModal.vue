<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ connection && connection.config.id ? '编辑数据库连接' : '添加数据库连接' }}</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <form @submit.prevent="handleSubmit" class="modal-form">
        <div class="modal-body">
          <div v-if="error" class="alert alert-error">
            <AlertCircle :size="18" />
            <span>{{ error }}</span>
          </div>

          <div class="form-group">
            <label for="name">连接名称 *</label>
            <input
              id="name"
              v-model="formData.name"
              type="text"
              class="form-control"
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
                class="form-control"
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
                class="form-control"
                required
                min="1"
                max="65535"
                placeholder="3306"
              >
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="username">用户名 *</label>
              <input
                id="username"
                v-model="formData.username"
                type="text"
                class="form-control"
                required
                placeholder="数据库用户名"
              >
            </div>

            <div class="form-group">
              <label for="password">密码 *</label>
              <div class="password-wrapper">
                <input
                  id="password"
                  v-model="formData.password"
                  :type="showPassword ? 'text' : 'password'"
                  class="form-control"
                  required
                  placeholder="数据库密码"
                >
                <button
                  type="button"
                  class="password-toggle"
                  :title="showPassword ? '隐藏密码' : '显示密码'"
                  @click="showPassword = !showPassword"
                >
                  <EyeOff v-if="showPassword" :size="16" />
                  <Eye v-else :size="16" />
                </button>
              </div>
            </div>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="formData.ssl">
              <span>启用 SSL 连接</span>
            </label>
            <small>建议在生产环境中启用 SSL 以确保数据传输安全</small>
          </div>

          <div v-if="testResult" :class="['alert', testResult.success ? 'alert-success' : 'alert-error']">
            <CheckCircle v-if="testResult.success" :size="18" />
            <AlertCircle v-else :size="18" />
            <span>
              {{ testResult.success ? '连接成功' : '连接失败' }}
              <template v-if="testResult.message"> - {{ testResult.message }}</template>
              <template v-if="testResult.latency !== undefined && testResult.latency !== null"> (延迟: {{ testResult.latency }}ms)</template>
            </span>
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="$emit('close')">
            取消
          </button>
          <button type="button" class="btn btn-secondary" :disabled="testing" @click="testConnection">
            <Loader2 v-if="testing" :size="14" class="spin" />
            <Activity v-else :size="14" />
            <span>{{ testing ? '测试中...' : '测试连接' }}</span>
          </button>
          <button type="submit" class="btn btn-primary" :disabled="saving">
            <Loader2 v-if="saving" :size="14" class="spin" />
            <span>{{ saving ? '保存中...' : '保存' }}</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Eye, EyeOff, Activity, Loader2, CheckCircle, AlertCircle } from 'lucide-vue-next'

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
  // database 字段保留给后端兼容使用，但不再在连接配置中展示或必填
  database: '',
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

  // Validate required fields（数据库名称不再在连接层面必填）
  if (!formData.name || !formData.host ||
      !formData.username || !formData.password) {
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
.modal-form {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.form-group small {
  display: block;
  margin-top: var(--space-1);
  color: var(--text-tertiary);
  font-size: 12px;
}

.password-wrapper {
  position: relative;
}

.password-wrapper .form-control {
  padding-right: 40px;
}

.password-toggle {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  padding: var(--space-1);
  cursor: pointer;
  color: var(--text-tertiary);
  transition: color var(--transition-fast);
}

.password-toggle:hover {
  color: var(--text-primary);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  margin-bottom: 0;
}

.checkbox-label input[type="checkbox"] {
  width: auto;
  accent-color: var(--accent-primary);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 560px) {
  .form-row {
    grid-template-columns: 1fr;
  }
}
</style>
