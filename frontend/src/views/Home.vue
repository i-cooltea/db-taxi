<template>
  <div class="app-layout">
    <!-- Header -->
    <header class="app-header">
      <div class="header-container">
        <div class="header-left">
          <div class="logo">
            <Database :size="24" />
            <span>DB-Taxi</span>
          </div>
          <nav class="main-nav">
            <button
              v-for="tab in tabs"
              :key="tab.name"
              :class="['nav-tab', { active: activeTab === tab.name }]"
              @click="activeTab = tab.name"
            >
              <component :is="tab.icon" :size="16" />
              <span>{{ tab.label }}</span>
            </button>
          </nav>
        </div>
        <div class="header-right">
          <ThemeToggle />
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="app-main">
      <div class="container">
        <div class="page-content">
          <Connections v-if="activeTab === 'connections'" />
          <SyncConfig v-else-if="activeTab === 'sync'" />
          <TableDiff v-else-if="activeTab === 'diff'" />
          <Monitoring v-else-if="activeTab === 'monitoring'" />
          <ConfigManagement v-else-if="activeTab === 'config'" />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Database, Plug, RefreshCw, BarChart3, Settings, GitCompareArrows } from 'lucide-vue-next'
import ThemeToggle from '../components/ThemeToggle.vue'
import Connections from './Connections.vue'
import SyncConfig from './SyncConfig.vue'
import Monitoring from './Monitoring.vue'
import ConfigManagement from './ConfigManagement.vue'
import TableDiff from './TableDiff.vue'

const activeTab = ref('connections')

const tabs = [
  { name: 'connections', icon: Plug, label: '连接管理' },
  { name: 'sync', icon: RefreshCw, label: '同步配置' },
  { name: 'diff', icon: GitCompareArrows, label: '表对比' },
  { name: 'monitoring', icon: BarChart3, label: '同步监控' },
  { name: 'config', icon: Settings, label: '配置管理' }
]
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.app-header {
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-primary);
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 var(--space-5);
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-7);
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.logo svg {
  color: var(--accent-primary);
}

.main-nav {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.nav-tab {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.nav-tab:hover {
  color: var(--text-primary);
  background: var(--bg-tertiary);
}

.nav-tab.active {
  color: var(--accent-primary);
  background: var(--accent-light);
}

.app-main {
  flex: 1;
  padding: var(--space-5) 0;
}

.page-content {
  animation: fadeIn var(--transition-normal);
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* Responsive */
@media (max-width: 768px) {
  .header-container {
    padding: 0 var(--space-4);
  }

  .logo span {
    display: none;
  }

  .main-nav {
    gap: 0;
  }

  .nav-tab {
    padding: var(--space-2);
  }

  .nav-tab span {
    display: none;
  }
}
</style>
