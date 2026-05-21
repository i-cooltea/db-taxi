# DB-Taxi 前端现代化改造实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 DB-Taxi 前端升级为极简现代风格，实现统一的设计系统、暗黑模式切换和响应式布局。

**Architecture:** 使用 CSS 变量定义设计系统，通过 `data-theme` 属性切换暗黑/亮色模式。提取公共 UI 组件减少重复代码，逐步替换各页面的样式。

**Tech Stack:** Vue 3, CSS Variables, Composition API, Pinia

---

## 文件结构映射

| 文件 | 职责 |
|------|------|
| `frontend/src/style.css` | 全局设计系统、CSS 变量、工具类 |
| `frontend/src/components/ThemeToggle.vue` | 主题切换按钮组件 |
| `frontend/src/components/BaseButton.vue` | 统一按钮组件 |
| `frontend/src/components/BaseCard.vue` | 卡片容器组件 |
| `frontend/src/components/BaseTable.vue` | 表格包装组件 |
| `frontend/src/views/Home.vue` | 主框架、导航栏 |
| `frontend/src/views/Connections.vue` | 连接管理页面 |
| `frontend/src/views/SyncConfig.vue` | 同步配置页面 |
| `frontend/src/views/TableDiff.vue` | 表对比页面 |
| `frontend/src/views/Monitoring.vue` | 监控页面 |
| `frontend/src/views/ConfigManagement.vue` | 配置管理页面 |

---

## Task 1: 建立设计系统 (CSS Variables)

**Files:**
- Modify: `frontend/src/style.css`

- [ ] **Step 1: 定义 CSS 变量**

将 `frontend/src/style.css` 内容完全替换为：

```css
/* ============================================
   DB-Taxi Design System
   Minimalist Style with Dark Mode Support
   ============================================ */

/* CSS Variables */
:root {
  /* Colors - Light Mode */
  --bg-primary: #FFFFFF;
  --bg-secondary: #F7F7F7;
  --bg-tertiary: #F3F4F6;
  --bg-hover: #F9FAFB;

  --text-primary: #111827;
  --text-secondary: #6B7280;
  --text-tertiary: #9CA3AF;

  --border-primary: #E5E7EB;
  --border-secondary: #F3F4F6;

  --accent-primary: #2563EB;
  --accent-hover: #1D4ED8;
  --accent-light: #EFF6FF;

  --success: #10B981;
  --success-light: #ECFDF5;
  --warning: #F59E0B;
  --warning-light: #FFFBEB;
  --error: #EF4444;
  --error-light: #FEF2F2;

  /* Spacing (8px base) */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 24px;
  --space-6: 32px;
  --space-7: 48px;
  --space-8: 64px;

  /* Border Radius */
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
  --radius-full: 9999px;

  /* Typography */
  --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif;
  --font-mono: "SF Mono", Monaco, "Cascadia Code", "Roboto Mono", Consolas, monospace;

  /* Shadows */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);

  /* Transitions */
  --transition-fast: 150ms ease-out;
  --transition-normal: 200ms ease-in-out;
}

/* Dark Mode */
[data-theme="dark"] {
  --bg-primary: #0F0F0F;
  --bg-secondary: #171717;
  --bg-tertiary: #1F1F1F;
  --bg-hover: #262626;

  --text-primary: #FFFFFF;
  --text-secondary: #A1A1A1;
  --text-tertiary: #525252;

  --border-primary: #2E2E2E;
  --border-secondary: #1F1F1F;

  --accent-primary: #3B82F6;
  --accent-hover: #2563EB;
  --accent-light: #1E3A5F;

  --success: #34D399;
  --success-light: #064E3B;
  --warning: #FBBF24;
  --warning-light: #451A03;
  --error: #F87171;
  --error-light: #450A0A;

  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4);
}

/* Reset & Base */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html {
  font-size: 14px;
}

body {
  font-family: var(--font-sans);
  background-color: var(--bg-secondary);
  color: var(--text-primary);
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  transition: background-color var(--transition-normal), color var(--transition-normal);
}

#app {
  min-height: 100vh;
}

/* Typography */
h1 { font-size: 24px; font-weight: 600; line-height: 1.3; }
h2 { font-size: 18px; font-weight: 600; line-height: 1.4; }
h3 { font-size: 16px; font-weight: 600; line-height: 1.4; }
h4 { font-size: 14px; font-weight: 600; line-height: 1.5; }

/* Container */
.container {
  max-width: 1400px;
  margin: 0 auto;
  padding: var(--space-5) var(--space-6);
}

/* ============================================
   Button Components
   ============================================ */

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  font-size: 14px;
  font-weight: 500;
  line-height: 1;
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  cursor: pointer;
  transition: all var(--transition-fast);
  background: transparent;
  color: var(--text-primary);
}

.btn:hover:not(:disabled) {
  background: var(--bg-hover);
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Button Variants */
.btn-primary {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
  border-color: var(--accent-hover);
}

.btn-secondary {
  background: var(--bg-primary);
  color: var(--text-primary);
  border-color: var(--border-primary);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-secondary);
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
}

.btn-ghost:hover:not(:disabled) {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.btn-danger {
  background: var(--error);
  color: white;
  border-color: var(--error);
}

.btn-danger:hover:not(:disabled) {
  filter: brightness(0.9);
}

.btn-success {
  background: var(--success);
  color: white;
  border-color: var(--success);
}

/* Button Sizes */
.btn-sm {
  padding: var(--space-1) var(--space-3);
  font-size: 12px;
}

.btn-lg {
  padding: var(--space-3) var(--space-5);
  font-size: 16px;
}

/* ============================================
   Card Component
   ============================================ */

.card {
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  transition: box-shadow var(--transition-fast);
}

.card-hover:hover {
  box-shadow: var(--shadow-sm);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--border-secondary);
}

.card-header h2 {
  color: var(--text-primary);
}

/* ============================================
   Form Elements
   ============================================ */

.form-group {
  margin-bottom: var(--space-4);
}

.form-group label {
  display: block;
  margin-bottom: var(--space-2);
  font-weight: 500;
  color: var(--text-primary);
  font-size: 14px;
}

.form-control {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  font-size: 14px;
  line-height: 1.5;
  color: var(--text-primary);
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.form-control:focus {
  outline: none;
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 3px var(--accent-light);
}

.form-control::placeholder {
  color: var(--text-tertiary);
}

.form-control:disabled {
  background: var(--bg-tertiary);
  cursor: not-allowed;
}

select.form-control {
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%236B7280' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 36px;
}

/* ============================================
   Table Component
   ============================================ */

.table-container {
  overflow-x: auto;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.data-table th,
.data-table td {
  padding: var(--space-3) var(--space-4);
  text-align: left;
  border-bottom: 1px solid var(--border-secondary);
}

.data-table th {
  background: var(--bg-tertiary);
  font-weight: 600;
  color: var(--text-secondary);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

.data-table tr:last-child td {
  border-bottom: none;
}

.data-table tbody tr {
  transition: background var(--transition-fast);
}

.data-table tbody tr:hover {
  background: var(--bg-hover);
}

/* ============================================
   Alerts
   ============================================ */

.alert {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-4);
  font-size: 14px;
}

.alert-success {
  background: var(--success-light);
  color: var(--success);
  border: 1px solid var(--success);
}

.alert-warning {
  background: var(--warning-light);
  color: var(--warning);
  border: 1px solid var(--warning);
}

.alert-error {
  background: var(--error-light);
  color: var(--error);
  border: 1px solid var(--error);
}

.alert-close {
  margin-left: auto;
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  opacity: 0.5;
  transition: opacity var(--transition-fast);
}

.alert-close:hover {
  opacity: 1;
}

/* ============================================
   Badges & Tags
   ============================================ */

.badge {
  display: inline-flex;
  align-items: center;
  padding: var(--space-1) var(--space-3);
  font-size: 12px;
  font-weight: 500;
  border-radius: var(--radius-full);
}

.badge-success {
  background: var(--success-light);
  color: var(--success);
}

.badge-warning {
  background: var(--warning-light);
  color: var(--warning);
}

.badge-error {
  background: var(--error-light);
  color: var(--error);
}

.badge-info {
  background: var(--accent-light);
  color: var(--accent-primary);
}

/* ============================================
   Status Indicators
   ============================================ */

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  display: inline-block;
}

.status-dot.running { background: var(--accent-primary); }
.status-dot.success { background: var(--success); }
.status-dot.error { background: var(--error); }
.status-dot.warning { background: var(--warning); }
.status-dot.pending { background: var(--text-tertiary); }

/* ============================================
   Modal
   ============================================ */

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--space-5);
}

.modal-content {
  background: var(--bg-primary);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 600px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-md);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border-secondary);
}

.modal-header h2 {
  font-size: 18px;
  font-weight: 600;
}

.modal-body {
  padding: var(--space-5);
  overflow-y: auto;
  flex: 1;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--border-secondary);
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: color var(--transition-fast);
}

.modal-close:hover {
  color: var(--text-primary);
}

/* ============================================
   Empty State
   ============================================ */

.empty-state {
  text-align: center;
  padding: var(--space-7) var(--space-5);
  color: var(--text-secondary);
}

.empty-state-icon {
  color: var(--text-tertiary);
  margin-bottom: var(--space-4);
}

.empty-state h3 {
  margin-bottom: var(--space-2);
  color: var(--text-primary);
}

.empty-state p {
  font-size: 14px;
}

/* ============================================
   Loading States
   ============================================ */

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-primary);
  border-top-color: var(--accent-primary);
  border-radius: var(--radius-full);
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.skeleton {
  background: linear-gradient(90deg, var(--bg-tertiary) 25%, var(--bg-secondary) 50%, var(--bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: var(--radius-md);
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* ============================================
   Utility Classes
   ============================================ */

.text-primary { color: var(--text-primary); }
.text-secondary { color: var(--text-secondary); }
.text-tertiary { color: var(--text-tertiary); }

.flex { display: flex; }
.flex-col { flex-direction: column; }
.items-center { align-items: center; }
.justify-between { justify-content: space-between; }
.gap-2 { gap: var(--space-2); }
.gap-3 { gap: var(--space-3); }
.gap-4 { gap: var(--space-4); }

.mb-2 { margin-bottom: var(--space-2); }
.mb-3 { margin-bottom: var(--space-3); }
.mb-4 { margin-bottom: var(--space-4); }

/* ============================================
   Animations
   ============================================ */

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideIn {
  from { transform: translateY(-10px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.fade-in {
  animation: fadeIn var(--transition-normal);
}

.slide-in {
  animation: slideIn var(--transition-normal);
}

/* ============================================
   Responsive
   ============================================ */

@media (max-width: 768px) {
  .container {
    padding: var(--space-4);
  }

  .card {
    padding: var(--space-4);
  }

  .modal-overlay {
    padding: var(--space-2);
  }

  .modal-content {
    max-height: 95vh;
  }

  h1 { font-size: 20px; }
  h2 { font-size: 16px; }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/style.css
git commit -m "feat: implement design system with CSS variables and dark mode support"
```

---

## Task 2: 创建主题切换组件

**Files:**
- Create: `frontend/src/components/ThemeToggle.vue`

- [ ] **Step 1: 创建 ThemeToggle.vue**

创建 `frontend/src/components/ThemeToggle.vue`，内容：

```vue
<template>
  <button
    class="theme-toggle"
    @click="toggleTheme"
    :title="isDark ? '切换到亮色模式' : '切换到暗黑模式'"
  >
    <Sun v-if="isDark" :size="18" />
    <Moon v-else :size="18" />
  </button>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Sun, Moon } from 'lucide-vue-next'

const isDark = ref(false)

onMounted(() => {
  // 读取用户保存的主题偏好
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme) {
    isDark.value = savedTheme === 'dark'
    applyTheme(isDark.value)
  } else {
    // 默认使用亮色模式
    isDark.value = false
    applyTheme(false)
  }
})

function toggleTheme() {
  isDark.value = !isDark.value
  applyTheme(isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function applyTheme(dark) {
  document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')
}
</script>

<style scoped>
.theme-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  background: var(--bg-primary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.theme-toggle:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border-color: var(--border-primary);
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ThemeToggle.vue
git commit -m "feat: add theme toggle component with localStorage persistence"
```

---

## Task 3: 重构 Home.vue (主框架)

**Files:**
- Modify: `frontend/src/views/Home.vue`

- [ ] **Step 1: 重写 Home.vue**

将 `frontend/src/views/Home.vue` 内容完全替换为：

```vue
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
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/Home.vue
git commit -m "feat: redesign Home.vue with minimalist header and theme toggle"
```

---

## Task 4: 重构 Connections.vue

**Files:**
- Modify: `frontend/src/views/Connections.vue`
- Modify: `frontend/src/components/ConnectionCard.vue`
- Modify: `frontend/src/components/ConnectionModal.vue`

- [ ] **Step 1: 重写 Connections.vue**

将 `frontend/src/views/Connections.vue` 内容完全替换为：

```vue
<template>
  <div>
    <!-- Alerts -->
    <div v-if="error" class="alert alert-error">
      <AlertCircle :size="18" />
      <span>{{ error }}</span>
      <button class="alert-close" @click="error = null">&times;</button>
    </div>

    <div v-if="successMessage" class="alert alert-success">
      <CheckCircle :size="18" />
      <span>{{ successMessage }}</span>
      <button class="alert-close" @click="successMessage = null">&times;</button>
    </div>

    <!-- Page Header -->
    <div class="page-header">
      <h1>连接管理</h1>
      <button class="btn btn-primary" @click="showAddModal = true">
        <Plus :size="16" />
        <span>添加连接</span>
      </button>
    </div>

    <!-- Content Card -->
    <div class="card">
      <!-- Loading State -->
      <div v-if="loading && connections.length === 0" class="loading-state">
        <div class="loading-spinner"></div>
        <p class="text-secondary">加载连接列表...</p>
      </div>

      <!-- Empty State -->
      <div v-else-if="connections.length === 0" class="empty-state">
        <Plug :size="48" class="empty-state-icon" />
        <h3>还没有数据库连接</h3>
        <p>点击上方"添加连接"按钮创建第一个数据库连接</p>
      </div>

      <!-- Connection Grid -->
      <div v-else class="connections-grid">
        <ConnectionCard
          v-for="connection in connections"
          :key="connection.config.id"
          :connection="connection"
          :testing="testingConnections.has(connection.config.id)"
          @edit="editConnection"
          @delete="confirmDelete"
          @test="testConnectionStatus"
        />
      </div>
    </div>

    <!-- Modals -->
    <ConnectionModal
      v-if="showAddModal || showEditModal"
      :connection="editingConnection"
      @close="closeModal"
      @save="saveConnection"
    />

    <!-- Delete Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click.self="showDeleteModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h2>确认删除</h2>
          <button class="modal-close" @click="showDeleteModal = false">&times;</button>
        </div>
        <div class="modal-body">
          <p>确定要删除连接 <strong>{{ deletingConnection?.config.name }}</strong> 吗？</p>
          <div class="alert alert-warning" style="margin-top: 16px;">
            <AlertTriangle :size="16" />
            <span>此操作将同时删除与此连接相关的所有同步配置和任务。</span>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="showDeleteModal = false">取消</button>
          <button class="btn btn-danger" @click="deleteConnection" :disabled="deleting">
            {{ deleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { Plus, Plug, AlertCircle, CheckCircle, AlertTriangle } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import ConnectionCard from '../components/ConnectionCard.vue'
import ConnectionModal from '../components/ConnectionModal.vue'

const syncStore = useSyncStore()
const { connections, loading } = storeToRefs(syncStore)

const error = ref(null)
const successMessage = ref(null)
const showAddModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const editingConnection = ref(null)
const deletingConnection = ref(null)
const deleting = ref(false)
const testingConnections = ref(new Set())

onMounted(async () => {
  try {
    await syncStore.fetchConnections()
  } catch (err) {
    error.value = '加载连接列表失败: ' + err.message
  }
})

function editConnection(connection) {
  editingConnection.value = connection
  showEditModal.value = true
}

function confirmDelete(connection) {
  deletingConnection.value = connection
  showDeleteModal.value = true
}

async function deleteConnection() {
  if (!deletingConnection.value) return
  deleting.value = true
  error.value = null
  try {
    await syncStore.deleteConnection(deletingConnection.value.config.id)
    successMessage.value = '连接已成功删除'
    showDeleteModal.value = false
    deletingConnection.value = null
  } catch (err) {
    error.value = '删除连接失败: ' + err.message
  } finally {
    deleting.value = false
  }
}

async function testConnectionStatus(connection) {
  const connectionId = connection.config.id
  testingConnections.value.add(connectionId)
  error.value = null
  try {
    const status = await syncStore.testConnection(connectionId)
    if (status.connected) {
      successMessage.value = `连接 ${connection.config.name} 测试成功 (延迟: ${status.latency_ms}ms)`
    } else {
      error.value = `连接 ${connection.config.name} 测试失败: ${status.error || '未知错误'}`
    }
  } catch (err) {
    error.value = `测试连接失败: ${err.message}`
  } finally {
    testingConnections.value.delete(connectionId)
  }
}

async function saveConnection(connectionData) {
  error.value = null
  try {
    if (editingConnection.value) {
      await syncStore.updateConnection(editingConnection.value.config.id, connectionData)
      successMessage.value = '连接已成功更新'
    } else {
      await syncStore.createConnection(connectionData)
      successMessage.value = '连接已成功创建'
    }
    closeModal()
  } catch (err) {
    error.value = (editingConnection.value ? '更新' : '创建') + '连接失败: ' + err.message
    throw err
  }
}

function closeModal() {
  showAddModal.value = false
  showEditModal.value = false
  editingConnection.value = null
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-5);
}

.page-header h1 {
  font-size: 24px;
  font-weight: 600;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-7);
  gap: var(--space-4);
}

.connections-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: var(--space-4);
}

@media (max-width: 768px) {
  .connections-grid {
    grid-template-columns: 1fr;
  }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/Connections.vue
git commit -m "refactor: redesign Connections.vue with minimalist style"
```

---

## Task 5: 重构 SyncConfig.vue

**Files:**
- Modify: `frontend/src/views/SyncConfig.vue`

- [ ] **Step 1: 重写 SyncConfig.vue**

将 `frontend/src/views/SyncConfig.vue` 内容完全替换为：

```vue
<template>
  <div>
    <!-- Alerts -->
    <div v-if="error" class="alert alert-error">
      <AlertCircle :size="18" />
      <span>{{ error }}</span>
      <button class="alert-close" @click="error = null">&times;</button>
    </div>

    <div v-if="successMessage" class="alert alert-success">
      <CheckCircle :size="18" />
      <span>{{ successMessage }}</span>
      <button class="alert-close" @click="successMessage = null">&times;</button>
    </div>

    <!-- Page Header -->
    <div class="page-header">
      <h1>同步配置</h1>
      <button class="btn btn-primary" @click="showCreateModal">
        <Plus :size="16" />
        <span>创建配置</span>
      </button>
    </div>

    <!-- Content Card -->
    <div class="card">
      <!-- Loading -->
      <div v-if="store.loading" class="loading-state">
        <div class="loading-spinner"></div>
        <p class="text-secondary">正在加载同步配置...</p>
      </div>

      <!-- Error -->
      <div v-else-if="store.error" class="alert alert-error">
        <AlertCircle :size="18" />
        <span>{{ store.error }}</span>
      </div>

      <!-- Empty State -->
      <div v-else-if="store.configs.length === 0" class="empty-state">
        <RefreshCw :size="48" class="empty-state-icon" />
        <h3>还没有配置任何同步任务</h3>
        <p>点击上方"创建配置"按钮开始配置数据库同步</p>
      </div>

      <!-- Config Table -->
      <div v-else class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>源连接</th>
              <th>目标连接</th>
              <th>同步模式</th>
              <th>表数量</th>
              <th>计划</th>
              <th>状态</th>
              <th style="width: 200px;">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="config in store.configs" :key="config.id">
              <td class="font-medium">{{ config.name }}</td>
              <td>{{ getConnection(config.source_connection_id)?.config.name || 'Unknown' }}</td>
              <td>{{ getConnection(config.target_connection_id)?.config.name || 'Unknown' }}</td>
              <td>
                <span :class="['badge', config.sync_mode === 'full' ? 'badge-info' : 'badge-success']">
                  {{ config.sync_mode === 'full' ? '全量同步' : '增量同步' }}
                </span>
              </td>
              <td>{{ config.tables ? config.tables.length : 0 }} 个表</td>
              <td class="text-secondary">{{ config.schedule || '-' }}</td>
              <td>
                <span :class="['badge', config.enabled ? 'badge-success' : 'badge-error']">
                  {{ config.enabled ? '已启用' : '已禁用' }}
                </span>
              </td>
              <td>
                <div class="action-buttons">
                  <button class="btn btn-sm btn-ghost" @click="startSync(config)" title="启动同步">
                    <Play :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost" @click="viewTables(config)" title="查看表">
                    <List :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost" @click="editConfig(config)" title="编辑">
                    <Edit2 :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost" @click="copyConfig(config)" title="复制">
                    <Copy :size="14" />
                  </button>
                  <button class="btn btn-sm btn-ghost text-error" @click="deleteConfig(config)" title="删除">
                    <Trash2 :size="14" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modals -->
    <ConfigModal
      v-if="showModal"
      :config="activeConfig"
      :connections="store.connections"
      @close="closeModal"
      @save="saveConfig"
    />

    <TableMappingsModal
      v-if="showTablesModal"
      :config="viewingConfig"
      @close="closeTablesModal"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { RefreshCw, Plus, Play, Edit2, Copy, Trash2, List, AlertCircle, CheckCircle } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import ConfigModal from '../components/ConfigModal.vue'
import TableMappingsModal from '../components/TableMappingsModal.vue'

const store = useSyncStore()
const showModal = ref(false)
const editingConfig = ref(null)
const activeConfig = ref(null)
const error = ref(null)
const successMessage = ref(null)
const showTablesModal = ref(false)
const viewingConfig = ref(null)

onMounted(async () => {
  await store.fetchConfigs()
})

function getConnection(connectionId) {
  return store.connections.find(c => c.config.id === connectionId)
}

function showCreateModal() {
  editingConfig.value = null
  activeConfig.value = null
  showModal.value = true
}

function editConfig(config) {
  editingConfig.value = config
  activeConfig.value = config
  showModal.value = true
}

function copyConfig(config) {
  const { id, created_at, updated_at, ...rest } = config
  activeConfig.value = { ...rest, name: `${config.name} - 副本` }
  editingConfig.value = null
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingConfig.value = null
}

async function saveConfig(configData) {
  try {
    if (editingConfig.value) {
      await store.updateConfig(editingConfig.value.id, configData)
    } else {
      await store.createConfig(configData)
    }
    closeModal()
  } catch (error) {
    console.error('Failed to save config:', error)
  }
}

async function deleteConfig(config) {
  if (confirm(`确定要删除同步配置 "${config.name}" 吗？\n\n此操作将删除所有相关的表映射配置。`)) {
    try {
      await store.deleteConfig(config.id)
    } catch (error) {
      console.error('Failed to delete config:', error)
    }
  }
}

async function startSync(config) {
  if (confirm(`确定要启动同步任务 "${config.name}" 吗？`)) {
    try {
      error.value = null
      const job = await store.startSync(config.id)
      successMessage.value = `同步任务已成功启动！任务ID: ${job.id}`
      setTimeout(() => { successMessage.value = null }, 5000)
    } catch (err) {
      error.value = `启动同步任务失败: ${err.message}`
    }
  }
}

function viewTables(config) {
  viewingConfig.value = config
  showTablesModal.value = true
}

function closeTablesModal() {
  showTablesModal.value = false
  viewingConfig.value = null
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-5);
}

.page-header h1 {
  font-size: 24px;
  font-weight: 600;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-7);
  gap: var(--space-4);
}

.action-buttons {
  display: flex;
  gap: var(--space-1);
}

.font-medium {
  font-weight: 500;
}

.text-error {
  color: var(--error);
}

@media (max-width: 768px) {
  .table-container {
    margin: 0 calc(-1 * var(--space-4));
    border-radius: 0;
    border-left: none;
    border-right: none;
  }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/SyncConfig.vue
git commit -m "refactor: redesign SyncConfig.vue with minimalist table style"
```

---

## Task 6: 重构 TableDiff.vue

**Files:**
- Modify: `frontend/src/views/TableDiff.vue`

- [ ] **Step 1: 重写 TableDiff.vue**

将 `frontend/src/views/TableDiff.vue` 内容完全替换为：

```vue
<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <h1>表对比</h1>
    </div>

    <!-- Selection Panel -->
    <div class="card">
      <div class="selection-area">
        <!-- Source -->
        <div class="selection-panel">
          <h3 class="panel-title">源表</h3>
          <div class="form-group">
            <label>连接</label>
            <select v-model="source.connectionId" @change="onSourceConnectionChange" class="form-control">
              <option value="">请选择连接</option>
              <option v-for="conn in connections" :key="conn.config.id" :value="conn.config.id">
                {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
              </option>
            </select>
          </div>
          <div class="form-group">
            <label>数据库</label>
            <select v-model="source.database" @change="onSourceDatabaseChange" :disabled="!source.connectionId" class="form-control">
              <option value="">请选择数据库</option>
              <option v-for="db in sourceDatabases" :key="db" :value="db">{{ db }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>表</label>
            <select v-model="source.table" :disabled="!source.database" class="form-control">
              <option value="">请选择表</option>
              <option v-for="t in sourceTables" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>
        </div>

        <!-- Divider -->
        <div class="selection-divider">
          <ArrowRight :size="24" />
        </div>

        <!-- Target -->
        <div class="selection-panel">
          <h3 class="panel-title">目标表</h3>
          <div class="form-group">
            <label>连接</label>
            <select v-model="target.connectionId" @change="onTargetConnectionChange" class="form-control">
              <option value="">请选择连接</option>
              <option v-for="conn in connections" :key="conn.config.id" :value="conn.config.id">
                {{ conn.config.name }} ({{ conn.config.host }}:{{ conn.config.port }})
              </option>
            </select>
          </div>
          <div class="form-group">
            <label>数据库</label>
            <select v-model="target.database" @change="onTargetDatabaseChange" :disabled="!target.connectionId" class="form-control">
              <option value="">请选择数据库</option>
              <option v-for="db in targetDatabases" :key="db" :value="db">{{ db }}</option>
            </select>
          </div>
          <div class="form-group">
            <label>表</label>
            <select v-model="target.table" :disabled="!target.database" class="form-control">
              <option value="">请选择表</option>
              <option v-for="t in targetTables" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="action-bar">
        <button class="btn btn-primary btn-lg" @click="startCompare" :disabled="!canCompare || comparing">
          <Loader2 v-if="comparing" :size="18" class="spin" />
          <GitCompareArrows v-else :size="18" />
          <span>{{ comparing ? `对比中 ${progress}%...` : '开始对比' }}</span>
        </button>
        <button class="btn btn-secondary" @click="exportResult" v-if="diffResult" :disabled="comparing">
          <Download :size="16" />
          <span>导出结果</span>
        </button>
      </div>

      <!-- Warning -->
      <div v-if="diffResult && diffResult.warning" class="alert alert-warning" style="margin-top: 16px;">
        <AlertTriangle :size="18" />
        <span>{{ diffResult.warning }}</span>
      </div>
    </div>

    <!-- Results -->
    <div v-if="diffResult" class="card" style="margin-top: 24px;">
      <!-- Result Tabs -->
      <div class="result-tabs">
        <button :class="['result-tab', { active: activeResultTab === 'schema' }]" @click="activeResultTab = 'schema'">
          结构差异
          <span v-if="schemaDiffCount > 0" class="badge badge-info">{{ schemaDiffCount }}</span>
        </button>
        <button :class="['result-tab', { active: activeResultTab === 'data' }]" @click="activeResultTab = 'data'" :disabled="!diffResult.data_diff || !!diffResult.warning">
          数据差异
          <span v-if="dataDiffCount > 0" class="badge badge-info">{{ dataDiffCount }}</span>
        </button>
      </div>

      <!-- Schema Diff -->
      <div v-if="activeResultTab === 'schema'" class="result-content">
        <div v-if="diffResult.schema_diff && diffResult.schema_diff.is_same" class="empty-state">
          <CheckCircle :size="32" class="empty-state-icon" style="color: var(--success);" />
          <h3>结构完全一致</h3>
          <p>两个表的列结构完全相同</p>
        </div>
        <div v-else-if="diffResult.schema_diff" class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>列名</th>
                <th>状态</th>
                <th>源表类型</th>
                <th>目标表类型</th>
                <th>源表可空</th>
                <th>目标表可空</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="col in diffResult.schema_diff.added_columns" :key="'add-'+col.name" class="row-added">
                <td class="font-medium">{{ col.name }}</td>
                <td><span class="badge badge-success">新增</span></td>
                <td>{{ col.type }}</td>
                <td>-</td>
                <td>{{ col.nullable ? '是' : '否' }}</td>
                <td>-</td>
              </tr>
              <tr v-for="col in diffResult.schema_diff.removed_columns" :key="'rm-'+col.name" class="row-removed">
                <td class="font-medium">{{ col.name }}</td>
                <td><span class="badge badge-error">删除</span></td>
                <td>-</td>
                <td>{{ col.type }}</td>
                <td>-</td>
                <td>{{ col.nullable ? '是' : '否' }}</td>
              </tr>
              <tr v-for="col in diffResult.schema_diff.modified_columns" :key="'mod-'+col.name" class="row-modified">
                <td class="font-medium">{{ col.name }}</td>
                <td><span class="badge badge-warning">变更</span></td>
                <td>{{ col.source_type }}</td>
                <td>{{ col.target_type }}</td>
                <td>{{ col.source_nullable ? '是' : '否' }}</td>
                <td>{{ col.target_nullable ? '是' : '否' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Data Diff -->
      <div v-if="activeResultTab === 'data'" class="result-content">
        <div v-if="diffResult.data_diff" class="diff-summary">
          <div class="stat-item">
            <span class="stat-label">仅源表</span>
            <span class="stat-value">{{ diffResult.data_diff.only_in_source?.length || 0 }} 行</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">仅目标表</span>
            <span class="stat-value">{{ diffResult.data_diff.only_in_target?.length || 0 }} 行</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">值不同</span>
            <span class="stat-value">{{ diffResult.data_diff.value_diff?.length || 0 }} 行</span>
          </div>
          <div v-if="diffResult.data_diff.partial" class="stat-item warning">
            <span class="stat-label">部分对比</span>
            <span class="stat-value">{{ diffResult.data_diff.compared_rows }}/{{ diffResult.data_diff.total_rows }} 行</span>
          </div>
        </div>

        <!-- Value Diff -->
        <div v-if="diffResult.data_diff?.value_diff?.length" class="diff-section">
          <h4>值不同的行</h4>
          <div v-if="valueDiffTotalPages > 1" class="pagination">
            <button class="btn btn-sm btn-secondary" @click="valueDiffPage--" :disabled="valueDiffPage <= 1">
              <ChevronLeft :size="14" />
            </button>
            <span class="page-info">{{ valueDiffPage }} / {{ valueDiffTotalPages }}</span>
            <button class="btn btn-sm btn-secondary" @click="valueDiffPage++" :disabled="valueDiffPage >= valueDiffTotalPages">
              <ChevronRight :size="14" />
            </button>
          </div>
          <div class="table-container">
            <table class="data-table">
              <thead>
                <tr>
                  <th>主键</th>
                  <th>差异列</th>
                  <th>源表值</th>
                  <th>目标表值</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in paginatedValueDiff" :key="JSON.stringify(entry.primary_key)">
                  <td class="font-medium">{{ formatPK(entry.primary_key) }}</td>
                  <td>
                    <span v-for="d in entry.differences" :key="d.column" class="tag">{{ d.column }}</span>
                  </td>
                  <td class="text-error">{{ formatVal(entry.differences[0]?.source_value) }}</td>
                  <td class="text-success">{{ formatVal(entry.differences[0]?.target_value) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Only in Source -->
        <div v-if="diffResult.data_diff?.only_in_source?.length" class="diff-section">
          <h4>仅存在于源表 ({{ diffResult.data_diff.only_in_source.length }} 行)</h4>
          <div class="row-cards">
            <div v-for="(row, idx) in diffResult.data_diff.only_in_source.slice(0, 10)" :key="idx" class="row-card source-only">
              <span v-for="(val, key) in row" :key="key" class="kv"><strong>{{ key }}:</strong> {{ formatVal(val) }}</span>
            </div>
          </div>
        </div>

        <!-- Only in Target -->
        <div v-if="diffResult.data_diff?.only_in_target?.length" class="diff-section">
          <h4>仅存在于目标表 ({{ diffResult.data_diff.only_in_target.length }} 行)</h4>
          <div class="row-cards">
            <div v-for="(row, idx) in diffResult.data_diff.only_in_target.slice(0, 10)" :key="idx" class="row-card target-only">
              <span v-for="(val, key) in row" :key="key" class="kv"><strong>{{ key }}:</strong> {{ formatVal(val) }}</span>
            </div>
          </div>
        </div>

        <div v-if="dataDiffCount === 0 && diffResult.data_diff" class="empty-state">
          <CheckCircle :size="32" class="empty-state-icon" style="color: var(--success);" />
          <h3>数据完全一致</h3>
          <p>两个表的数据内容完全相同</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { GitCompareArrows, ArrowRight, Loader2, Download, CheckCircle, AlertTriangle, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'

const store = useSyncStore()
const connections = computed(() => store.connections)

const source = ref({ connectionId: '', database: '', table: '' })
const target = ref({ connectionId: '', database: '', table: '' })
const sourceDatabases = ref([])
const sourceTables = ref([])
const targetDatabases = ref([])
const targetTables = ref([])

const comparing = ref(false)
const progress = ref(0)
const diffResult = ref(null)
const activeResultTab = ref('schema')

const pageSize = 10
const valueDiffPage = ref(1)

const canCompare = computed(() =>
  source.value.connectionId && source.value.database && source.value.table &&
  target.value.connectionId && target.value.database && target.value.table
)

const schemaDiffCount = computed(() => {
  if (!diffResult.value?.schema_diff) return 0
  const s = diffResult.value.schema_diff
  return (s.added_columns?.length || 0) + (s.removed_columns?.length || 0) + (s.modified_columns?.length || 0)
})

const dataDiffCount = computed(() => {
  if (!diffResult.value?.data_diff) return 0
  const d = diffResult.value.data_diff
  return (d.only_in_source?.length || 0) + (d.only_in_target?.length || 0) + (d.value_diff?.length || 0)
})

const valueDiffTotalPages = computed(() => Math.ceil((diffResult.value?.data_diff?.value_diff?.length || 0) / pageSize))

const paginatedValueDiff = computed(() => {
  const arr = diffResult.value?.data_diff?.value_diff || []
  const start = (valueDiffPage.value - 1) * pageSize
  return arr.slice(start, start + pageSize)
})

onMounted(() => {
  if (store.connections.length === 0) store.fetchConnections()
})

async function onSourceConnectionChange() {
  source.value.database = ''
  source.value.table = ''
  sourceDatabases.value = []
  sourceTables.value = []
  if (!source.value.connectionId) return
  try {
    const res = await fetch(`/api/sync/connections/${source.value.connectionId}/databases`)
    const result = await res.json()
    if (result.success) sourceDatabases.value = result.data || []
  } catch (e) { console.error(e) }
}

async function onSourceDatabaseChange() {
  source.value.table = ''
  sourceTables.value = []
  if (!source.value.connectionId || !source.value.database) return
  try {
    const res = await fetch(`/api/sync/connections/${source.value.connectionId}/tables?database=${encodeURIComponent(source.value.database)}`)
    const result = await res.json()
    if (result.success) sourceTables.value = result.data || []
  } catch (e) { console.error(e) }
}

async function onTargetConnectionChange() {
  target.value.database = ''
  target.value.table = ''
  targetDatabases.value = []
  targetTables.value = []
  if (!target.value.connectionId) return
  try {
    const res = await fetch(`/api/sync/connections/${target.value.connectionId}/databases`)
    const result = await res.json()
    if (result.success) targetDatabases.value = result.data || []
  } catch (e) { console.error(e) }
}

async function onTargetDatabaseChange() {
  target.value.table = ''
  targetTables.value = []
  if (!target.value.connectionId || !target.value.database) return
  try {
    const res = await fetch(`/api/sync/connections/${target.value.connectionId}/tables?database=${encodeURIComponent(target.value.database)}`)
    const result = await res.json()
    if (result.success) targetTables.value = result.data || []
  } catch (e) { console.error(e) }
}

async function startCompare() {
  if (!canCompare.value) return
  comparing.value = true
  progress.value = 0
  diffResult.value = null
  activeResultTab.value = 'schema'
  valueDiffPage.value = 1

  try {
    const res = await fetch('/api/sync/diff/compare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        source: { connection_id: source.value.connectionId, database: source.value.database, table: source.value.table },
        target: { connection_id: target.value.connectionId, database: target.value.database, table: target.value.table }
      })
    })
    const result = await res.json()
    if (!result.success) throw new Error(result.error || '对比失败')

    const taskId = result.data.task_id
    while (true) {
      await new Promise(r => setTimeout(r, 500))
      const pollRes = await fetch(`/api/sync/diff/${taskId}`)
      const pollResult = await pollRes.json()
      if (!pollResult.success) throw new Error(pollResult.error || '查询任务失败')

      const task = pollResult.data
      progress.value = task.progress || 0

      if (task.status === 'done') {
        diffResult.value = task.result
        break
      }
      if (task.status === 'failed') throw new Error(task.error || '对比失败')
    }
  } catch (e) {
    alert(e.message)
  } finally {
    comparing.value = false
  }
}

function exportResult() {
  if (!diffResult.value) return
  const blob = new Blob([JSON.stringify(diffResult.value, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `diff-result-${source.value.database}.${source.value.table}-vs-${target.value.database}.${target.value.table}.json`
  a.click()
  URL.revokeObjectURL(url)
}

function formatPK(pk) {
  return Object.entries(pk).map(([k, v]) => `${k}=${v}`).join(', ')
}

function formatVal(v) {
  if (v === null || v === undefined) return 'NULL'
  return String(v).length > 50 ? String(v).slice(0, 50) + '...' : String(v)
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

.selection-area {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: var(--space-5);
  align-items: start;
}

.selection-panel {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-4);
  color: var(--text-primary);
}

.selection-divider {
  display: flex;
  align-items: center;
  justify-content: center;
  padding-top: 80px;
  color: var(--text-secondary);
}

.action-bar {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-5);
  padding-top: var(--space-5);
  border-top: 1px solid var(--border-secondary);
}

.result-tabs {
  display: flex;
  gap: var(--space-1);
  border-bottom: 1px solid var(--border-secondary);
  margin: calc(-1 * var(--space-5)) calc(-1 * var(--space-5)) var(--space-4);
  padding: 0 var(--space-5);
}

.result-tab {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all var(--transition-fast);
  margin-bottom: -1px;
}

.result-tab:hover:not(:disabled) {
  color: var(--text-primary);
}

.result-tab.active {
  color: var(--accent-primary);
  border-bottom-color: var(--accent-primary);
}

.result-tab:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.result-content {
  padding-top: var(--space-2);
}

.diff-summary {
  display: flex;
  gap: var(--space-6);
  margin-bottom: var(--space-5);
  padding: var(--space-4);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.stat-item.warning .stat-label,
.stat-item.warning .stat-value {
  color: var(--warning);
}

.stat-label {
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.diff-section {
  margin-bottom: var(--space-5);
}

.diff-section h4 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: var(--space-3);
  color: var(--text-primary);
}

.pagination {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.page-info {
  font-size: 14px;
  color: var(--text-secondary);
}

.row-cards {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.row-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font-size: 13px;
  border-left: 3px solid var(--border-primary);
}

.row-card.source-only {
  border-left-color: var(--success);
  background: var(--success-light);
}

.row-card.target-only {
  border-left-color: var(--error);
  background: var(--error-light);
}

.kv {
  display: inline-block;
  margin-right: var(--space-4);
  margin-bottom: var(--space-1);
}

.kv strong {
  color: var(--accent-primary);
}

.tag {
  display: inline-block;
  padding: var(--space-1) var(--space-2);
  background: var(--warning-light);
  color: var(--warning);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  margin-right: var(--space-1);
}

.font-medium {
  font-weight: 500;
}

.text-success {
  color: var(--success);
}

.text-error {
  color: var(--error);
}

.row-added {
  background: var(--success-light);
}

.row-removed {
  background: var(--error-light);
}

.row-modified {
  background: var(--warning-light);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 768px) {
  .selection-area {
    grid-template-columns: 1fr;
  }

  .selection-divider {
    padding: var(--space-2) 0;
    transform: rotate(90deg);
  }

  .diff-summary {
    flex-wrap: wrap;
    gap: var(--space-4);
  }

  .action-bar {
    flex-direction: column;
  }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/TableDiff.vue
git commit -m "refactor: redesign TableDiff.vue with minimalist style"
```

---

## Task 7: 重构 Monitoring.vue

**Files:**
- Modify: `frontend/src/views/Monitoring.vue`

- [ ] **Step 1: 重写 Monitoring.vue**

将 `frontend/src/views/Monitoring.vue` 内容完全替换为：

```vue
<template>
  <div>
    <!-- Toast -->
    <Toast :show="toast.show" :message="toast.message" :type="toast.type" @close="toast.show = false" />

    <!-- Page Header -->
    <div class="page-header">
      <h1>同步监控</h1>
    </div>

    <!-- Stats Grid -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--accent-light); color: var(--accent-primary);">
          <BarChart3 :size="20" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.total_jobs || 0 }}</div>
          <div class="stat-label">总任务数</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--success-light); color: var(--success);">
          <CheckCircle :size="20" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.completed_jobs || 0 }}</div>
          <div class="stat-label">已完成</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--accent-light); color: var(--accent-primary);">
          <Loader2 :size="20" class="spin" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.running_jobs || 0 }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: var(--error-light); color: var(--error);">
          <XCircle :size="20" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stats.failed_jobs || 0 }}</div>
          <div class="stat-label">失败</div>
        </div>
      </div>
    </div>

    <!-- Active Jobs -->
    <div v-if="activeJobs.length > 0" class="card" style="margin-bottom: 24px;">
      <div class="card-header">
        <h2>
          <Activity :size="18" />
          运行中的任务
        </h2>
        <button class="btn btn-sm btn-secondary" @click="refreshActiveJobs" :disabled="loading">
          <RefreshCw :size="14" :class="{ spin: loading }" />
          刷新
        </button>
      </div>

      <div class="active-jobs">
        <div v-for="job in activeJobs" :key="job.job_id" class="job-card">
          <div class="job-header">
            <div class="job-info">
              <h3>{{ getConfigName(job.config_id) }}</h3>
              <span class="job-id">任务 ID: {{ job.job_id }}</span>
            </div>
            <div class="job-actions">
              <span class="badge badge-info">
                <span class="status-dot running"></span>
                运行中
              </span>
              <button class="btn btn-sm btn-ghost" @click="stopJob(job.job_id)" title="停止">
                <Pause :size="14" />
              </button>
              <button class="btn btn-sm btn-ghost text-error" @click="cancelJob(job.job_id)" title="取消">
                <X :size="14" />
              </button>
            </div>
          </div>

          <!-- Progress -->
          <div class="progress-section">
            <div class="progress-item">
              <div class="progress-header">
                <span class="progress-label">表进度</span>
                <span class="progress-value">{{ job.completed_tables }} / {{ job.total_tables }} 张表</span>
              </div>
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: job.total_tables ? (job.completed_tables / job.total_tables * 100) + '%' : '0%' }"></div>
              </div>
            </div>

            <div class="progress-item">
              <div class="progress-header">
                <span class="progress-label">当前表</span>
                <span class="progress-value">{{ formatNumber(job.processed_rows) }} / {{ formatNumber(job.total_rows) }} 行</span>
              </div>
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: currentTablePercent(job) + '%' }"></div>
              </div>
            </div>
          </div>

          <!-- Table Progress -->
          <div v-if="job.table_progress && Object.keys(job.table_progress).length > 0" class="table-progress">
            <h4>表同步进度</h4>
            <div class="table-list">
              <div v-for="(table, tableName) in job.table_progress" :key="tableName" class="table-item">
                <span class="table-name">{{ table.table_name }}</span>
                <div class="table-status">
                  <span :class="['badge', table.status === 'running' ? 'badge-info' : table.status === 'completed' ? 'badge-success' : 'badge-error']">
                    {{ getStatusText(table.status) }}
                  </span>
                  <span v-if="table.status === 'running'" class="table-rows">
                    {{ formatNumber(table.processed_rows) }} / {{ formatNumber(table.total_rows) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- History -->
    <div class="card">
      <div class="card-header">
        <h2>
          <History :size="18" />
          同步历史
        </h2>
        <button class="btn btn-sm btn-secondary" @click="refreshHistory" :disabled="loading">
          <RefreshCw :size="14" :class="{ spin: loading }" />
          刷新
        </button>
      </div>

      <div v-if="jobHistory.length === 0" class="empty-state">
        <History :size="48" class="empty-state-icon" />
        <h3>暂无同步历史</h3>
        <p>同步任务完成后会在这里显示记录</p>
      </div>

      <div v-else class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>状态</th>
              <th>配置名称</th>
              <th>任务 ID</th>
              <th>表进度</th>
              <th>行数</th>
              <th>开始时间</th>
              <th>耗时</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="job in jobHistory" :key="job.id">
              <td>
                <span :class="['badge', getStatusClass(job.status)]">
                  {{ getStatusText(job.status) }}
                </span>
              </td>
              <td class="font-medium">{{ job.config_name || '未知配置' }}</td>
              <td class="text-secondary">{{ job.id.substring(0, 8) }}...</td>
              <td>{{ job.completed_tables }}/{{ job.total_tables }}</td>
              <td>{{ formatNumber(job.processed_rows) }}</td>
              <td class="text-secondary">{{ formatTime(job.start_time) }}</td>
              <td class="text-secondary">{{ calculateDuration(job.start_time, job.end_time) }}</td>
              <td>
                <button class="btn btn-sm btn-ghost" @click="viewJobLogs(job.id)">日志</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="jobHistory.length > 0" class="pagination">
        <button class="btn btn-sm btn-secondary" @click="previousPage" :disabled="currentPage === 1 || loading">
          <ChevronLeft :size="14" />
          上一页
        </button>
        <span class="page-info">第 {{ currentPage }} 页</span>
        <button class="btn btn-sm btn-secondary" @click="nextPage" :disabled="jobHistory.length < pageSize || loading">
          下一页
          <ChevronRight :size="14" />
        </button>
      </div>
    </div>

    <!-- Logs Modal -->
    <div v-if="showLogsModal" class="modal-overlay" @click="closeLogsModal">
      <div class="modal-content modal-lg" @click.stop>
        <div class="modal-header">
          <h2>
            <FileText :size="18" />
            任务日志
          </h2>
          <button class="modal-close" @click="closeLogsModal">&times;</button>
        </div>
        <div class="modal-body">
          <div v-if="loadingLogs" class="loading-state">
            <div class="loading-spinner"></div>
            <p>加载日志中...</p>
          </div>
          <div v-else-if="jobLogs.length === 0" class="empty-state">
            <p>暂无日志记录</p>
          </div>
          <div v-else class="logs-list">
            <div v-for="log in jobLogs" :key="log.id" class="log-entry" :class="log.level">
              <div class="log-header">
                <span :class="['log-level', log.level]">{{ log.level.toUpperCase() }}</span>
                <span class="log-table">{{ log.table_name }}</span>
                <span class="log-time">{{ formatTime(log.created_at) }}</span>
              </div>
              <div class="log-message">{{ log.message }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { BarChart3, CheckCircle, XCircle, Loader2, Activity, RefreshCw, History, Pause, X, FileText, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { useSyncStore } from '../stores/syncStore'
import Toast from '../components/Toast.vue'

const syncStore = useSyncStore()

const stats = ref({})
const activeJobs = ref([])
const jobHistory = ref([])
const jobLogs = ref([])
const loading = ref(false)
const loadingLogs = ref(false)
const showLogsModal = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const progressEventSource = ref(null)
const toast = ref({ show: false, message: '', type: 'info' })

onMounted(async () => {
  await loadAllData()
  startProgressStream()
})

onUnmounted(() => {
  stopProgressStream()
})

function startProgressStream() {
  if (progressEventSource.value) return
  const es = new EventSource('/api/sync/jobs/progress/stream')
  progressEventSource.value = es

  es.addEventListener('progress', (e) => {
    try {
      const payload = JSON.parse(e.data)
      const jobs = payload?.data ?? []
      const hadJobs = activeJobs.value.length > 0
      activeJobs.value = jobs
      if (hadJobs && jobs.length === 0) {
        loadStats()
        loadHistory()
      }
    } catch (err) {
      console.error('Failed to parse progress SSE data:', err)
    }
  })

  es.onerror = () => {
    es.close()
    progressEventSource.value = null
    setTimeout(startProgressStream, 3000)
  }
}

function stopProgressStream() {
  if (progressEventSource.value) {
    progressEventSource.value.close()
    progressEventSource.value = null
  }
}

async function loadAllData() {
  loading.value = true
  try {
    await Promise.all([loadStats(), loadActiveJobs(), loadHistory()])
  } catch (error) {
    console.error('Failed to load monitoring data:', error)
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const response = await fetch('/api/sync/stats')
    const result = await response.json()
    if (result.success) stats.value = result.data || {}
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

async function loadActiveJobs() {
  try {
    const response = await fetch('/api/sync/jobs/active')
    const result = await response.json()
    if (result.success) activeJobs.value = result.data || []
  } catch (error) {
    console.error('Failed to load active jobs:', error)
  }
}

async function loadHistory() {
  try {
    const offset = (currentPage.value - 1) * pageSize.value
    const url = `/api/sync/jobs/history?limit=${pageSize.value}&offset=${offset}`
    const response = await fetch(url)
    const result = await response.json()
    if (result.success) jobHistory.value = result.data || []
  } catch (error) {
    console.error('Failed to load job history:', error)
  }
}

async function refreshActiveJobs() {
  await loadActiveJobs()
  await loadStats()
}

async function refreshHistory() {
  loading.value = true
  try {
    await loadHistory()
    await loadStats()
  } finally {
    loading.value = false
  }
}

async function viewJobLogs(jobId) {
  showLogsModal.value = true
  loadingLogs.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/logs`)
    const result = await response.json()
    if (result.success) jobLogs.value = result.data || []
  } catch (error) {
    console.error('Failed to load job logs:', error)
  } finally {
    loadingLogs.value = false
  }
}

function closeLogsModal() {
  showLogsModal.value = false
  jobLogs.value = []
}

async function stopJob(jobId) {
  if (!confirm('确定要停止这个同步任务吗？')) return
  loading.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/stop`, { method: 'POST' })
    const result = await response.json()
    if (result.success) {
      toast.value = { show: true, message: '任务已成功停止', type: 'success' }
      await refreshActiveJobs()
    } else {
      toast.value = { show: true, message: '停止失败: ' + (result.error || '未知错误'), type: 'error' }
    }
  } catch (error) {
    toast.value = { show: true, message: '停止失败: ' + error.message, type: 'error' }
  } finally {
    loading.value = false
  }
}

async function cancelJob(jobId) {
  if (!confirm('确定要取消这个同步任务吗？取消后任务将无法恢复。')) return
  loading.value = true
  try {
    const response = await fetch(`/api/sync/jobs/${jobId}/cancel`, { method: 'POST' })
    const result = await response.json()
    if (result.success) {
      toast.value = { show: true, message: '任务已成功取消', type: 'success' }
      await refreshActiveJobs()
    } else {
      toast.value = { show: true, message: '取消失败: ' + (result.error || '未知错误'), type: 'error' }
    }
  } catch (error) {
    toast.value = { show: true, message: '取消失败: ' + error.message, type: 'error' }
  } finally {
    loading.value = false
  }
}

function getConfigName(configId) {
  const config = syncStore.configs.find(c => c.id === configId)
  return config?.name || configId.substring(0, 8)
}

function getStatusText(status) {
  const map = { pending: '等待中', running: '运行中', completed: '已完成', failed: '失败', cancelled: '已取消' }
  return map[status] || status
}

function getStatusClass(status) {
  const map = { pending: 'badge-warning', running: 'badge-info', completed: 'badge-success', failed: 'badge-error', cancelled: 'badge-error' }
  return map[status] || 'badge-info'
}

function formatNumber(num) {
  if (!num) return '0'
  return num.toLocaleString()
}

function currentTablePercent(job) {
  const table = getCurrentTableProgress(job)
  if (!table || table.total_rows <= 0) return 0
  return Math.min(100, (table.processed_rows / table.total_rows) * 100)
}

function getCurrentTableProgress(job) {
  if (!job?.table_progress) return null
  const name = job.current_table
  if (name && job.table_progress[name]) return job.table_progress[name]
  return Object.values(job.table_progress).find(t => t.status === 'running') || null
}

function formatTime(timestamp) {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function calculateDuration(startTime, endTime) {
  if (!startTime || !endTime) return '-'
  const seconds = Math.floor((new Date(endTime) - new Date(startTime)) / 1000)
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  return `${Math.floor(seconds / 3600)}时${Math.floor((seconds % 3600) / 60)}分`
}

function previousPage() {
  if (currentPage.value > 1) {
    currentPage.value--
    loadHistory()
  }
}

function nextPage() {
  currentPage.value++
  loadHistory()
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

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-5);
}

.stat-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
}

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 4px;
}

.job-card {
  padding: var(--space-4);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-4);
}

.job-card:last-child {
  margin-bottom: 0;
}

.job-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--space-4);
}

.job-info h3 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-1);
}

.job-id {
  font-size: 12px;
  color: var(--text-secondary);
}

.job-actions {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}

.progress-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.progress-item {
  background: var(--bg-primary);
  padding: var(--space-3);
  border-radius: var(--radius-md);
}

.progress-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--space-2);
}

.progress-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.progress-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.progress-bar {
  height: 6px;
  background: var(--border-secondary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent-primary);
  border-radius: var(--radius-full);
  transition: width 0.3s ease;
}

.table-progress {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-primary);
}

.table-progress h4 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: var(--space-3);
}

.table-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.table-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3);
  background: var(--bg-primary);
  border-radius: var(--radius-md);
}

.table-name {
  font-weight: 500;
}

.table-status {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.table-rows {
  font-size: 13px;
  color: var(--text-secondary);
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: var(--space-4);
  margin-top: var(--space-5);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-secondary);
}

.page-info {
  font-size: 14px;
  color: var(--text-secondary);
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.log-entry {
  padding: var(--space-3);
  border-radius: var(--radius-md);
  border-left: 3px solid var(--border-primary);
}

.log-entry.info {
  background: var(--accent-light);
  border-left-color: var(--accent-primary);
}

.log-entry.warn {
  background: var(--warning-light);
  border-left-color: var(--warning);
}

.log-entry.error {
  background: var(--error-light);
  border-left-color: var(--error);
}

.log-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
}

.log-level {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  text-transform: uppercase;
}

.log-entry.info .log-level {
  background: var(--accent-primary);
  color: white;
}

.log-entry.warn .log-level {
  background: var(--warning);
  color: white;
}

.log-entry.error .log-level {
  background: var(--error);
  color: white;
}

.log-table {
  font-size: 13px;
  color: var(--text-secondary);
}

.log-time {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-left: auto;
}

.log-message {
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.6;
}

.modal-lg {
  max-width: 900px;
}

.font-medium {
  font-weight: 500;
}

.text-error {
  color: var(--error);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .job-header {
    flex-direction: column;
    gap: var(--space-3);
  }

  .job-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/Monitoring.vue
git commit -m "refactor: redesign Monitoring.vue with minimalist stats cards and progress bars"
```

---

## Task 8: 重构 ConfigManagement.vue

**Files:**
- Modify: `frontend/src/views/ConfigManagement.vue`

- [ ] **Step 1: 重写 ConfigManagement.vue**

将 `frontend/src/views/ConfigManagement.vue` 内容完全替换为：

```vue
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
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/ConfigManagement.vue
git commit -m "refactor: redesign ConfigManagement.vue with minimalist file upload and cards"
```

---

## Task 9: 更新 ConnectionCard 组件

**Files:**
- Modify: `frontend/src/components/ConnectionCard.vue`

- [ ] **Step 1: 重写 ConnectionCard.vue**

将 `frontend/src/components/ConnectionCard.vue` 内容完全替换为：

```vue
<template>
  <div class="connection-card">
    <div class="card-header-section">
      <div class="connection-icon">
        <Database :size="20" />
      </div>
      <div class="connection-info">
        <h3>{{ connection.config.name }}</h3>
        <span :class="['status-badge', connection.status]">
          <span :class="['status-dot', connection.status]"></span>
          {{ getStatusText(connection.status) }}
        </span>
      </div>
    </div>

    <div class="connection-details">
      <div class="detail-item">
        <span class="detail-label">主机</span>
        <span class="detail-value">{{ connection.config.host }}:{{ connection.config.port }}</span>
      </div>
      <div class="detail-item">
        <span class="detail-label">数据库</span>
        <span class="detail-value">{{ connection.config.database }}</span>
      </div>
      <div class="detail-item">
        <span class="detail-label">用户名</span>
        <span class="detail-value">{{ connection.config.username }}</span>
      </div>
    </div>

    <div class="card-actions">
      <button class="btn btn-sm btn-ghost" @click="$emit('test', connection)" :disabled="testing">
        <Loader2 v-if="testing" :size="14" class="spin" />
        <Activity v-else :size="14" />
        <span>{{ testing ? '测试中...' : '测试' }}</span>
      </button>
      <button class="btn btn-sm btn-ghost" @click="$emit('edit', connection)">
        <Edit2 :size="14" />
        <span>编辑</span>
      </button>
      <button class="btn btn-sm btn-ghost text-error" @click="$emit('delete', connection)">
        <Trash2 :size="14" />
        <span>删除</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { Database, Activity, Edit2, Trash2, Loader2 } from 'lucide-vue-next'

defineProps({
  connection: { type: Object, required: true },
  testing: { type: Boolean, default: false }
})

defineEmits(['edit', 'delete', 'test'])

function getStatusText(status) {
  const map = { connected: '已连接', disconnected: '未连接', error: '错误' }
  return map[status] || status
}
</script>

<style scoped>
.connection-card {
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  transition: box-shadow var(--transition-fast);
}

.connection-card:hover {
  box-shadow: var(--shadow-sm);
}

.card-header-section {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.connection-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--accent-light);
  color: var(--accent-primary);
  border-radius: var(--radius-md);
}

.connection-info h3 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-1);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-size: 12px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
}

.status-badge.connected {
  background: var(--success-light);
  color: var(--success);
}

.status-badge.error {
  background: var(--error-light);
  color: var(--error);
}

.connection-details {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-bottom: var(--space-4);
  padding: var(--space-3);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.detail-item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.detail-label {
  color: var(--text-secondary);
}

.detail-value {
  color: var(--text-primary);
  font-weight: 500;
}

.card-actions {
  display: flex;
  gap: var(--space-2);
}

.text-error {
  color: var(--error);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ConnectionCard.vue
git commit -m "refactor: redesign ConnectionCard component with minimalist card style"
```

---

## Task 10: 构建并验证

**Files:**
- Modify: `frontend/dist/` (generated files)

- [ ] **Step 1: 构建前端**

```bash
cd frontend && npm run build 2>&1
```

Expected: Build succeeds with no errors

- [ ] **Step 2: Commit built files**

```bash
git add frontend/dist/
git commit -m "chore: rebuild frontend with modernized design"
```

---

## Spec Self-Review

1. **Spec coverage**: All requirements covered - CSS design system, theme toggle, all 6 pages refactored, base components updated
2. **Placeholder scan**: No TBD/TODO/fill in details found
3. **Type consistency**: All CSS variable names, class names consistent throughout

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-21-frontend-modernization.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
