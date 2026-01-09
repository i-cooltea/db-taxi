<template>
  <div class="container">
    <header class="page-header">
      <div>
        <h1>🚕 DB-Taxi</h1>
        <p>MySQL 数据库同步管理系统</p>
      </div>
    </header>

    <div class="cards-grid">
      <router-link to="/connections" class="feature-card">
        <div class="feature-icon">🔌</div>
        <h2>连接管理</h2>
        <p>管理远程数据库连接配置</p>
      </router-link>

      <router-link to="/sync" class="feature-card">
        <div class="feature-icon">🔄</div>
        <h2>同步配置</h2>
        <p>配置数据库同步任务和表映射</p>
      </router-link>

      <router-link to="/monitoring" class="feature-card">
        <div class="feature-icon">📊</div>
        <h2>同步监控</h2>
        <p>监控同步任务状态和进度</p>
      </router-link>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 3rem 2rem;
  border-radius: 10px;
  margin-bottom: 2rem;
  text-align: center;
}

.page-header h1 {
  font-size: 3rem;
  margin-bottom: 0.5rem;
}

.page-header p {
  font-size: 1.2rem;
  opacity: 0.9;
}

.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
  margin-top: 2rem;
}

.feature-card {
  background: white;
  border-radius: 10px;
  padding: 2rem;
  text-align: center;
  text-decoration: none;
  color: inherit;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  transition: all 0.3s ease;
}

.feature-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 4px 20px rgba(102, 126, 234, 0.3);
}

.feature-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.feature-card h2 {
  color: #667eea;
  margin-bottom: 0.5rem;
}

.feature-card p {
  color: #666;
}
</style>
