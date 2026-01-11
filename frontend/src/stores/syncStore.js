import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useSyncStore = defineStore('sync', () => {
  // State
  const configs = ref([])
  const connections = ref([])
  const loading = ref(false)
  const error = ref(null)

  // Getters
  const configCount = computed(() => configs.value.length)
  const enabledConfigs = computed(() => 
    configs.value.filter(c => c.enabled)
  )

  // Actions
  async function fetchConnections() {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/connections')
      const result = await response.json()
      if (result.success) {
        connections.value = result.data || []
      } else {
        throw new Error(result.error || 'Failed to load connections')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createConnection(connectionData) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/connections', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(connectionData)
      })
      const result = await response.json()
      if (result.success) {
        await fetchConnections()
        return result.data
      } else {
        throw new Error(result.error || 'Failed to create connection')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateConnection(connectionId, connectionData) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`/api/sync/connections/${connectionId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(connectionData)
      })
      const result = await response.json()
      if (result.success) {
        await fetchConnections()
      } else {
        throw new Error(result.error || 'Failed to update connection')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteConnection(connectionId) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`/api/sync/connections/${connectionId}`, {
        method: 'DELETE'
      })
      const result = await response.json()
      if (result.success) {
        await fetchConnections()
      } else {
        throw new Error(result.error || 'Failed to delete connection')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function testConnection(connectionId) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`/api/sync/connections/${connectionId}/test`, {
        method: 'POST'
      })
      const result = await response.json()
      if (result.success) {
        // Update the connection status in the local state
        const conn = connections.value.find(c => c.config.id === connectionId)
        if (conn) {
          conn.status = result.data
        }
        return result.data
      } else {
        throw new Error(result.error || 'Failed to test connection')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchConfigs() {
    loading.value = true
    error.value = null
    try {
      // Load all connections first
      await fetchConnections()
      
      configs.value = []
      // Load configs for each connection
      for (const conn of connections.value) {
        const response = await fetch(`/api/sync/configs?connection_id=${conn.config.id}`)
        const result = await response.json()
        if (result.success && result.data) {
          configs.value.push(...result.data)
        }
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createConfig(configData) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/configs', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(configData)
      })
      const result = await response.json()
      if (result.success) {
        await fetchConfigs()
        return result.data
      } else {
        throw new Error(result.error || 'Failed to create config')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(configId, configData) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`/api/sync/configs/${configId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(configData)
      })
      const result = await response.json()
      if (result.success) {
        await fetchConfigs()
      } else {
        throw new Error(result.error || 'Failed to update config')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteConfig(configId) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`/api/sync/configs/${configId}`, {
        method: 'DELETE'
      })
      const result = await response.json()
      if (result.success) {
        await fetchConfigs()
      } else {
        throw new Error(result.error || 'Failed to delete config')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function startSync(configId) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/jobs', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ config_id: configId })
      })
      const result = await response.json()
      if (result.success) {
        return result.data
      } else {
        throw new Error(result.error || 'Failed to start sync')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function exportConfig() {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/config/export')
      const result = await response.json()
      if (result.success) {
        return result.data
      } else {
        throw new Error(result.error || 'Failed to export config')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function importConfig(configData, resolveConflicts = false) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/config/import', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          config: configData,
          resolve_conflicts: resolveConflicts
        })
      })
      const result = await response.json()
      if (result.success) {
        // Refresh configs after import
        await fetchConfigs()
        return result
      } else {
        throw new Error(result.error || 'Failed to import config')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function validateConfig(configData) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/config/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(configData)
      })
      const result = await response.json()
      return result
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function backupConfig() {
    loading.value = true
    error.value = null
    try {
      const response = await fetch('/api/sync/config/backup')
      const result = await response.json()
      if (result.success) {
        return result.data
      } else {
        throw new Error(result.error || 'Failed to backup config')
      }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    configs,
    connections,
    loading,
    error,
    configCount,
    enabledConfigs,
    fetchConnections,
    createConnection,
    updateConnection,
    deleteConnection,
    testConnection,
    fetchConfigs,
    createConfig,
    updateConfig,
    deleteConfig,
    startSync,
    exportConfig,
    importConfig,
    validateConfig,
    backupConfig
  }
})
