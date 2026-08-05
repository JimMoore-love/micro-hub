import { ref, watch, type Ref } from 'vue'
import { useUserStore } from '../stores/user'
import client from '../api/client'

export interface TenantOption {
  label: string
  value: string
}

export function useTenant() {
  const userStore = useUserStore()

  const currentTenantId = ref<string>(userStore.currentTenantId)
  const tenantOptions: Ref<TenantOption[]> = ref<TenantOption[]>([
    { label: '默认租户', value: 'default' },
  ])
  const loading = ref(false)
  const switching = ref(false)

  async function fetchTenants() {
    loading.value = true
    try {
      const data = await client.get<any, any[]>('/tenants')
      tenantOptions.value = data.map((t: any) => ({
        label: t.name,
        value: t.id,
      }))
      if (!tenantOptions.value.find(t => t.value === currentTenantId.value)) {
        const defaultTenant = tenantOptions.value.find(t => t.value === 'default')
        currentTenantId.value = defaultTenant?.value || tenantOptions.value[0]?.value || 'default'
        userStore.setTenantId(currentTenantId.value)
      }
    } catch {
      // 降级：保持默认列表
    } finally {
      loading.value = false
    }
  }

  function switchTenant(tenantId: string) {
    if (tenantId === currentTenantId.value) return
    switching.value = true
    currentTenantId.value = tenantId
    userStore.setTenantId(tenantId)
    // 不再刷新页面：依赖 currentTenantId 响应式数据的视图会自动重载
    setTimeout(() => { switching.value = false }, 500)
  }

  // 同步外部 store 变化
  watch(() => userStore.currentTenantId, (val) => {
    if (val !== currentTenantId.value) {
      currentTenantId.value = val
    }
  })

  return {
    currentTenantId,
    tenantOptions,
    loading,
    switching,
    fetchTenants,
    switchTenant,
  }
}
