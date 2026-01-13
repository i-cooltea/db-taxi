<template>
  <div class="container">
    <header class="page-header">
      <div class="header-content">
        <div class="logo">
          <Truck class="logo-icon" />
          <h1>DB-Taxi</h1>
        </div>
        <p>MySQL 数据库同步管理系统</p>
      </div>
    </header>

    <div class="tabs-container">
      <div class="tabs">
        <button 
          v-for="tab in tabs" 
          :key="tab.name"
          :class="['tab', { active: activeTab === tab.name }]"
          @click="activeTab = tab.name"
        >
          <component :is="tab.icon" class="tab-icon" :size="20" />
          <span class="tab-label">{{ tab.label }}</span>
        </button>
      </div>

      <div class="tab-content">
        <Connections v-if="activeTab === 'connections'" />
        <SyncConfig v-else-if="activeTab === 'sync'" />
        <Monitoring v-else-if="activeTab === 'monitoring'" />
        <ConfigManagement v-else-if="activeTab === 'config'" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Truck, Plug, RefreshCw, BarChart3, Settings } from 'lucide-vue-next'
import Connections from './Connections.vue'
import SyncConfig from './SyncConfig.vue'
import Monitoring from './Monitoring.vue'
import ConfigManagement from './ConfigManagement.vue'

const activeTab = ref('connections')

const tabs = [
  { name: 'connections', icon: Plug, label: '连接管理' },
  { name: 'sync', icon: RefreshCw, label: '同步配置' },
  { name: 'monitoring', icon: BarChart3, label: '同步监控' },
  { name: 'config', icon: Settings, label: '配置管理' }
]
</script>

<style scoped>
.page-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 2rem;
  border-radius: 10px;
  margin-bottom: 2rem;
  text-align: center;
}

.header-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.logo-icon {
  width: 2.5rem;
  height: 2.5rem;
}

.page-header h1 {
  font-size: 2.5rem;
  margin: 0;
}

.page-header p {
  font-size: 1.1rem;
  opacity: 0.9;
  margin: 0;
}

.tabs-container {
  background: white;
  border-radius: 10px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  overflow: hidden;
}

.tabs {
  display: flex;
  background: #f8f9fa;
  border-bottom: 2px solid #e9ecef;
}

.tab {
  flex: 1;
  padding: 1.2rem 1.5rem;
  background: none;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-size: 1rem;
  color: #666;
  transition: all 0.3s ease;
  position: relative;
}

.tab:hover {
  background: rgba(102, 126, 234, 0.05);
  color: #667eea;
}

.tab.active {
  color: #667eea;
  font-weight: 600;
  background: white;
}

.tab.active::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  right: 0;
  height: 2px;
  background: #667eea;
}

.tab-icon {
  flex-shrink: 0;
}

.tab-label {
  font-size: 1rem;
}

.tab-content {
  padding: 2rem;
  min-height: 500px;
}

@media (max-width: 768px) {
  .tabs {
    flex-wrap: wrap;
  }
  
  .tab {
    flex: 1 1 50%;
  }
  
  .tab-label {
    font-size: 0.9rem;
  }
}
</style>
