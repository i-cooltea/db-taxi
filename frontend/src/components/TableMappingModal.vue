<template>
  <div class="modal" @click.self="$emit('close')">
    <div class="modal-content modal-small">
      <div class="modal-header">
        <h2>配置表映射</h2>
        <span class="close" @click="$emit('close')">&times;</span>
      </div>

      <div v-if="error" class="alert alert-error">{{ error }}</div>

      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="source-table">源表名称 *</label>
          <input 
            id="source-table" 
            v-model="formData.source_table" 
            type="text" 
            readonly
          >
        </div>

        <div class="form-group">
          <label for="target-table">目标表名称 *</label>
          <input 
            id="target-table" 
            v-model="formData.target_table" 
            type="text" 
            required
          >
          <small>本地数据库中的表名称（可自定义）</small>
        </div>

        <div class="form-group">
          <label for="sync-mode">同步模式 *</label>
          <select id="sync-mode" v-model="formData.sync_mode" required>
            <option value="full">全量同步</option>
            <option value="incremental">增量同步</option>
          </select>
        </div>

        <div class="form-group">
          <label for="where-clause">WHERE 条件</label>
          <textarea 
            id="where-clause" 
            v-model="formData.where_clause" 
            rows="3"
            placeholder="created_at > '2024-01-01'"
          ></textarea>
          <small>可选的过滤条件，只同步符合条件的数据</small>
        </div>

        <div class="form-group">
          <label class="checkbox-label">
            <input type="checkbox" v-model="formData.enabled">
            启用此表同步
          </label>
        </div>

        <div class="form-actions">
          <button type="button" class="btn btn-secondary" @click="$emit('close')">
            取消
          </button>
          <button type="submit" class="btn">
            保存
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'

const props = defineProps({
  tableName: {
    type: String,
    required: true
  },
  tableData: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'save'])

const error = ref(null)

const formData = reactive({
  source_table: '',
  target_table: '',
  sync_mode: 'full',
  where_clause: '',
  enabled: true
})

onMounted(() => {
  formData.source_table = props.tableName
  
  if (props.tableData) {
    formData.target_table = props.tableData.target_table
    formData.sync_mode = props.tableData.sync_mode
    formData.where_clause = props.tableData.where_clause || ''
    formData.enabled = props.tableData.enabled !== false
  } else {
    formData.target_table = props.tableName
  }
})

function handleSubmit() {
  emit('save', props.tableName, {
    target_table: formData.target_table,
    sync_mode: formData.sync_mode,
    where_clause: formData.where_clause,
    enabled: formData.enabled
  })
}
</script>

<style scoped>
.modal-small {
  max-width: 600px;
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
</style>
