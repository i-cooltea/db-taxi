import { createRouter, createWebHistory } from 'vue-router'
import Home from '../views/Home.vue'
import Connections from '../views/Connections.vue'
import SyncConfig from '../views/SyncConfig.vue'
import Monitoring from '../views/Monitoring.vue'
import ConfigManagement from '../views/ConfigManagement.vue'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: Home
  },
  {
    path: '/connections',
    name: 'Connections',
    component: Connections
  },
  {
    path: '/sync',
    name: 'SyncConfig',
    component: SyncConfig
  },
  {
    path: '/monitoring',
    name: 'Monitoring',
    component: Monitoring
  },
  {
    path: '/config',
    name: 'ConfigManagement',
    component: ConfigManagement
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
