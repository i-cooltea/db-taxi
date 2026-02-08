<template>
  <div class="progress-component">
    <div class="progress-header">
      <span class="progress-label">{{ label }}</span>
      <span v-if="!indeterminate" class="progress-percent">{{ percentFixed }}%</span>
      <span v-else class="progress-status">处理中...</span>
    </div>
    <div v-if="subtitle" class="progress-subtitle">{{ subtitle }}</div>
    <div class="progress-track">
      <div
        v-if="!indeterminate"
        class="progress-fill"
        :style="{ width: Math.min(100, Math.max(0, percent)) + '%' }"
      />
      <div v-else class="progress-fill indeterminate" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  /** 进度条标题 */
  label: {
    type: String,
    required: true
  },
  /** 进度百分比 0-100 */
  percent: {
    type: Number,
    default: 0
  },
  /** 副标题，如 "2/5 表"、"1,234 / 5,678 行" */
  subtitle: {
    type: String,
    default: ''
  },
  /** 未知总量时显示 indeterminate 动画 */
  indeterminate: {
    type: Boolean,
    default: false
  }
})

const percentFixed = computed(() => (props.percent ?? 0).toFixed(1))
</script>

<style scoped>
.progress-component {
  margin-bottom: 1rem;
}

.progress-component:last-child {
  margin-bottom: 0;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.35rem;
}

.progress-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #475569;
}

.progress-percent {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #1e40af;
  min-width: 3.5em;
  text-align: right;
}

.progress-status {
  font-size: 0.8125rem;
  color: #64748b;
}

.progress-subtitle {
  font-size: 0.75rem;
  color: #64748b;
  margin-bottom: 0.5rem;
}

.progress-track {
  height: 10px;
  background: #e2e8f0;
  border-radius: 9999px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #2563eb);
  border-radius: 9999px;
  transition: width 0.35s ease;
}

.progress-fill.indeterminate {
  width: 40%;
  animation: progress-indeterminate 1.2s ease-in-out infinite;
}

@keyframes progress-indeterminate {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(350%);
  }
}
</style>
