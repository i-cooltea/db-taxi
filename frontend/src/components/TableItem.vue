<template>
  <div class="table-item">
    <div class="table-info">
      <div class="table-name">{{ tableName }}</div>
      <div v-if="selected && tableData" class="table-mapping">
        → {{ tableData.target_table }} ({{ tableData.sync_mode }})
      </div>
    </div>
    <div class="table-actions">
      <select 
        v-if="selected"
        class="sync-mode-select"
        :value="tableData?.sync_mode"
        @change="$emit('update-mode', tableName, $event.target.value)"
      >
        <option value="full">全量</option>
        <option value="incremental">增量</option>
      </select>
      <button 
        v-if="selected"
        type="button"
        class="btn btn-secondary btn-small"
        @click="$emit('configure', tableName)"
      >
        ⚙️ 配置
      </button>
      <label class="checkbox-label">
        <input 
          type="checkbox" 
          :checked="selected"
          @change="$emit('toggle', tableName, $event.target.checked)"
        >
        选择
      </label>
    </div>
  </div>
</template>

<script setup>
defineProps({
  tableName: {
    type: String,
    required: true
  },
  selected: {
    type: Boolean,
    default: false
  },
  tableData: {
    type: Object,
    default: null
  }
})

defineEmits(['toggle', 'configure', 'update-mode'])
</script>

<style scoped>
.table-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.2s;
}

.table-item:hover {
  background: #f8f9ff;
}

.table-item:last-child {
  border-bottom: none;
}

.table-info {
  flex: 1;
}

.table-name {
  font-weight: 500;
  color: #333;
}

.table-mapping {
  font-size: 0.85rem;
  color: #666;
  margin-top: 0.25rem;
}

.table-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.sync-mode-select {
  padding: 0.4rem 0.6rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.85rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: auto;
}
</style>
