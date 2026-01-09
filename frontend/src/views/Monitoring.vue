<template>
  <div class="container">
    <header class="page-header">
      <div>
        <h1>📊 同步监控</h1>
        <p>监控同步任务状态和进度</p>
      </div>
      <nav class="nav-links">
        <router-link to="/" class="nav-link">🏠 首页</router-link>
        <router-link to="/sync" class="nav-link">🔄 同步配置</router-link>
      </nav>
    </header>

    <div class="card">
      <div class="card-header">
        <h2>📈 同步任务监控</h2>
      </div>
      <p style="text-align: center; color: #666; padding: 2rem;">
        监控功能将在任务 9.3 中实现
      </p>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 2rem;
  border-radius: 10px;
  margin-bottom: 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header h1 {
  font-size: 2rem;
  margin-bottom: 0.5rem;
}

.page-header p {
  font-size: 1rem;
  opacity: 0.9;
}

.nav-links {
  display: flex;
  gap: 1rem;
}

.nav-link {
  color: white;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 5px;
  transition: background 0.2s;
}

.nav-link:hover {
  background: rgba(255, 255, 255, 0.2);
}

.card-header {
  margin-bottom: 1.5rem;
}

.card h2 {
  color: #667eea;
  font-size: 1.5rem;
}
</style>
