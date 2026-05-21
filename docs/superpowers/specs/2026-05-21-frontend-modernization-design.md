# DB-Taxi 前端现代化改造设计文档

## 概述

将 DB-Taxi 前端从现有的混合风格升级为极简现代风格（Minimalist），类似 Notion / Linear 的设计语言，提升视觉一致性、用户体验和代码可维护性。

## 设计系统

### 色彩方案

#### 亮色模式
| Token | 值 | 用途 |
|-------|-----|------|
| `--bg-primary` | `#FFFFFF` | 主背景 |
| `--bg-secondary` | `#F7F7F7` | 次级背景、页面背景 |
| `--bg-tertiary` | `#F3F4F6` | 表头、悬停状态 |
| `--text-primary` | `#111827` | 主要文字 |
| `--text-secondary` | `#6B7280` | 次要文字、说明 |
| `--text-tertiary` | `#9CA3AF` | 占位符、禁用 |
| `--border-primary` | `#E5E7EB` | 边框、分割线 |
| `--border-secondary` | `#F3F4F6` | 细微分隔 |
| `--accent-primary` | `#2563EB` | 主要操作、链接 |
| `--accent-hover` | `#1D4ED8` | 悬停状态 |
| `--success` | `#10B981` | 成功状态 |
| `--warning` | `#F59E0B` | 警告状态 |
| `--error` | `#EF4444` | 错误状态 |

#### 暗黑模式
| Token | 值 | 用途 |
|-------|-----|------|
| `--bg-primary` | `#0F0F0F` | 主背景 |
| `--bg-secondary` | `#171717` | 卡片背景 |
| `--bg-tertiary` | `#1F1F1F` | 表头、悬停 |
| `--text-primary` | `#FFFFFF` | 主要文字 |
| `--text-secondary` | `#A1A1A1` | 次要文字 |
| `--text-tertiary` | `#525252` | 占位符 |
| `--border-primary` | `#2E2E2E` | 边框 |
| `--border-secondary` | `#1F1F1F` | 细微分隔 |
| `--accent-primary` | `#3B82F6` | 主要操作 |
| `--accent-hover` | `#2563EB` | 悬停状态 |

### 字体系统

```css
--font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif;
--font-mono: "SF Mono", Monaco, "Cascadia Code", "Roboto Mono", Consolas, monospace;
```

| 元素 | 大小 | 字重 | 行高 |
|------|------|------|------|
| 页面标题 | 24px | 600 | 1.3 |
| 区块标题 | 18px | 600 | 1.4 |
| 正文 | 14px | 400 | 1.6 |
| 小字 | 12px | 400 | 1.5 |
| 按钮 | 14px | 500 | 1 |

### 间距系统

基于 8px 倍数：4, 8, 12, 16, 24, 32, 48, 64

### 圆角系统

| 元素 | 圆角 |
|------|------|
| 卡片/面板 | 8px |
| 按钮 | 6px |
| 输入框 | 6px |
| 标签 | 4px |
| 完全圆角 | 9999px |

## 组件规范

### 按钮

**主要按钮**
- 背景: `--accent-primary`
- 文字: 白色
- 圆角: 6px
- 内边距: 8px 16px
- Hover: `--accent-hover`
- 无阴影

**次要按钮**
- 背景: 透明
- 边框: 1px solid `--border-primary`
- 文字: `--text-primary`
- Hover: 背景 `--bg-secondary`

**幽灵按钮**
- 背景: 透明
- 无边框
- Hover: 背景 `--bg-tertiary`

### 卡片

- 背景: `--bg-primary`
- 边框: 1px solid `--border-primary`
- 圆角: 8px
- 内边距: 24px
- 无阴影（扁平设计）

### 表格

- 表头: `--bg-tertiary` 背景
- 行高: 48px
- 行分隔: 1px solid `--border-secondary`
- 悬停: `--bg-tertiary` 背景
- 无纵向边框

### 输入框

- 背景: `--bg-primary`
- 边框: 1px solid `--border-primary`
- 圆角: 6px
- Focus: 边框 `--accent-primary`
- 无阴影

### 标签页

- 底部指示器: 2px solid `--accent-primary`
- 无背景色切换
- 指示器动画过渡

### 开关

- 宽度: 40px
- 高度: 24px
- 圆角: 9999px
- 背景: `--border-primary` / `--accent-primary`

## 布局架构

### 页面结构

```
┌─────────────────────────────────────────────────┐
│  Header (固定, 高度 64px)                        │
│  [Logo]          [Nav: 连接 同步 对比 监控 配置]  [主题切换] │
├─────────────────────────────────────────────────┤
│                                                  │
│  Main Content (padding: 24px 32px)              │
│                                                  │
│  ┌───────────────────────────────────────────┐  │
│  │  Page Header                              │  │
│  │  [标题]                    [操作按钮]      │  │
│  └───────────────────────────────────────────┘  │
│                                                  │
│  ┌───────────────────────────────────────────┐  │
│  │  Content Card                             │  │
│  │                                           │  │
│  │  [表格/表单/内容]                          │  │
│  │                                           │  │
│  └───────────────────────────────────────────┘  │
│                                                  │
└─────────────────────────────────────────────────┘
```

### 响应式断点

| 断点 | 宽度 | 适配 |
|------|------|------|
| Mobile | < 768px | 单列布局，汉堡菜单 |
| Tablet | 768px - 1024px | 双列布局 |
| Desktop | > 1024px | 完整布局 |

### 移动端适配

- 导航变为汉堡菜单
- 表格横向滚动，冻结首列
- 卡片全宽堆叠
- 模态框全屏

## 暗黑模式实现

### 切换方式

用户通过右上角主题按钮手动切换，存储在 `localStorage`。

### 切换逻辑

```javascript
// 读取用户偏好
const theme = localStorage.getItem('theme') || 'light'
document.documentElement.setAttribute('data-theme', theme)
```

### CSS 实现

```css
:root {
  /* 亮色变量 */
}

[data-theme="dark"] {
  /* 暗色变量覆盖 */
}
```

## 动画规范

| 动画 | 时长 | 缓动 |
|------|------|------|
| 悬停状态 | 150ms | ease-out |
| 页面切换 | 200ms | ease-in-out |
| 模态框 | 200ms | cubic-bezier(0.4, 0, 0.2, 1) |
| 加载骨架 | 1.5s | ease-in-out (无限循环) |

## 待改造页面

1. **Home.vue** - 主框架和导航
2. **Connections.vue** - 连接管理
3. **SyncConfig.vue** - 同步配置
4. **TableDiff.vue** - 表对比
5. **Monitoring.vue** - 监控页面
6. **ConfigManagement.vue** - 配置管理

## 技术实现

### CSS 架构

使用 CSS 变量统一管理，位于 `style.css` 根文件：

```css
/* 设计变量 */
:root {
  /* 所有颜色、间距、圆角变量 */
}

/* 基础样式重置 */
/* 工具类 */
/* 动画定义 */
```

### 组件复用

提取公共组件：
- `BaseButton.vue` - 统一按钮
- `BaseCard.vue` - 卡片容器
- `BaseTable.vue` - 表格包装
- `BaseModal.vue` - 模态框
- `BaseInput.vue` - 输入框
- `ThemeToggle.vue` - 主题切换

### 中英文统一

所有界面文字使用中文，状态值映射：
- `pending` → 等待中
- `running` → 运行中
- `completed` → 已完成
- `failed` → 失败
- `cancelled` → 已取消

## 成功标准

1. 所有页面视觉风格统一
2. 支持手动切换暗黑/亮色模式
3. 移动端基本可用
4. 代码无 CSS 重复定义
5. 所有文字为中文
