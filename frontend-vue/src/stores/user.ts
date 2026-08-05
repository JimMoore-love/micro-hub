import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem('token') || '')
  const username = ref<string>(localStorage.getItem('username') || '')
  const currentTenantId = ref<string>(localStorage.getItem('tenantId') || 'default')

  const isLoggedIn = computed(() => !!token.value)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem('token', t)
  }

  function setUsername(name: string) {
    username.value = name
    localStorage.setItem('username', name)
  }

  function setTenantId(id: string) {
    currentTenantId.value = id
    localStorage.setItem('tenantId', id)
  }

  function logout() {
    token.value = ''
    username.value = ''
    currentTenantId.value = 'default'
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.setItem('tenantId', 'default')
  }

  return {
    token,
    username,
    currentTenantId,
    isLoggedIn,
    setToken,
    setUsername,
    setTenantId,
    logout,
  }
})
