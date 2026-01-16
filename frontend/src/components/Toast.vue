<template>
  <Transition name="toast">
    <div v-if="visible" class="toast" :class="type">
      <div class="toast-content">
        <component :is="getIcon()" :size="20" class="toast-icon" />
        <span class="toast-message">{{ message }}</span>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, watch } from 'vue'
import { CheckCircle, XCircle, AlertCircle, Info } from 'lucide-vue-next'

const props = defineProps({
  message: {
    type: String,
    default: ''
  },
  type: {
    type: String,
    default: 'info', // success, error, warning, info
    validator: (value) => ['success', 'error', 'warning', 'info'].includes(value)
  },
  duration: {
    type: Number,
    default: 3000
  },
  show: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close'])

const visible = ref(false)
let timer = null

watch(() => props.show, (newVal) => {
  if (newVal) {
    visible.value = true
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      visible.value = false
      emit('close')
    }, props.duration)
  }
})

function getIcon() {
  const icons = {
    success: CheckCircle,
    error: XCircle,
    warning: AlertCircle,
    info: Info
  }
  return icons[props.type] || Info
}
</script>

<style scoped>
.toast {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  min-width: 300px;
  max-width: 500px;
  padding: 1rem 1.25rem;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  background: white;
}

.toast.success {
  border-left: 4px solid #10b981;
  background: #f0fdf4;
}

.toast.error {
  border-left: 4px solid #ef4444;
  background: #fef2f2;
}

.toast.warning {
  border-left: 4px solid #f59e0b;
  background: #fffbeb;
}

.toast.info {
  border-left: 4px solid #3b82f6;
  background: #eff6ff;
}

.toast-content {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.toast-icon {
  flex-shrink: 0;
}

.toast.success .toast-icon {
  color: #10b981;
}

.toast.error .toast-icon {
  color: #ef4444;
}

.toast.warning .toast-icon {
  color: #f59e0b;
}

.toast.info .toast-icon {
  color: #3b82f6;
}

.toast-message {
  color: #1f2937;
  font-size: 0.9375rem;
  line-height: 1.5;
}

.toast-enter-active {
  animation: toast-in 0.3s ease-out;
}

.toast-leave-active {
  animation: toast-out 0.3s ease-in;
}

@keyframes toast-in {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@keyframes toast-out {
  from {
    transform: translateX(0);
    opacity: 1;
  }
  to {
    transform: translateX(100%);
    opacity: 0;
  }
}
</style>
