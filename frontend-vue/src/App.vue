<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from './stores/user'
import { useTenant } from './composables/useTenant'
import { serviceApi } from './api/platform'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const { currentTenantId, tenantOptions, fetchTenants, switchTenant } = useTenant()
const isSidebarCollapsed = ref(false)
const handleLogout = () => { userStore.logout(); router.push('/') }
const currentRoute = computed(() => route.name as string)

// 健康状态概览（从 /services API 实时统计）
const healthSummary = ref({ total: 0, healthy: 0, warning: 0, error: 0 })
let healthTimer: number | null = null

async function fetchHealthSummary() {
  if (!userStore.isLoggedIn) return
  try {
    const list = await serviceApi.list()
    const services = list || []
    healthSummary.value = {
      total: services.length,
      healthy: services.filter((s: any) => s.status === 'healthy').length,
      warning: services.filter((s: any) => s.status === 'warning').length,
      error: services.filter((s: any) => s.status === 'critical' || s.status === 'unreachable').length,
    }
  } catch {
    // 静默失败
  }
}

onMounted(() => {
  if (userStore.isLoggedIn) {
    fetchTenants()
    fetchHealthSummary()
    // 每 30 秒自动刷新
    healthTimer = window.setInterval(fetchHealthSummary, 30000)
  }
})

onUnmounted(() => {
  if (healthTimer) {
    clearInterval(healthTimer)
    healthTimer = null
  }
})

// 路由变化时也刷新
watch(() => route.fullPath, () => {
  if (userStore.isLoggedIn) fetchHealthSummary()
})

// 切换租户时也刷新
watch(currentTenantId, () => {
  if (userStore.isLoggedIn) fetchHealthSummary()
})

// ==================== 微服务治理平台菜单 ====================
const menuGroups = [
  {
    group: '服务治理',
    items: [
      { label: '服务拓扑', key: 'Topology', icon: 'Share', path: '/topology', desc: '服务依赖关系与健康状态' },
      { label: '服务管理', key: 'Services', icon: 'Monitor', path: '/services', desc: '服务注册发现与健康检查' },
      { label: 'API 网关', key: 'Gateway', icon: 'Connection', path: '/gateway', desc: '路由规则、限流、鉴权' },
      { label: '流量管理', key: 'Traffic', icon: 'DataLine', path: '/traffic', desc: '熔断、降级、重试策略' },
    ]
  },
  {
    group: '多租户',
    items: [
      { label: '租户管理', key: 'Tenants', icon: 'OfficeBuilding', path: '/tenants', desc: '租户配置、资源配额、隔离' },
      { label: '用户管理', key: 'UserList', icon: 'User', path: '/users', desc: '租户下的用户增删改查' },
    ]
  },
  {
    group: 'AI 能力',
    items: [
      { label: 'AI 接入', key: 'AIProviders', icon: 'Cpu', path: '/ai-providers', desc: '供应商注册、路由、配额' },
      { label: '智能校对', key: 'Proofread', icon: 'EditPen', path: '/proofread', desc: '校对API接入与分发示例' },
      { label: 'AI 对话', key: 'AIChat', icon: 'ChatDotRound', path: '/ai-chat', desc: '多模型对话与SSE流式' },
    ]
  },
  {
    group: '可观测性',
    items: [
      { label: '监控中心', key: 'Observability', icon: 'View', path: '/observability', desc: '指标、链路、日志聚合' },
      { label: '告警规则', key: 'Alerts', icon: 'AlarmClock', path: '/alerts', desc: '告警配置与历史' },
    ]
  },
]

const navigateTo = (item: { path: string }) => { router.push(item.path) }

</script>

<template>
  <div class="app-layout">
    <!-- 侧边栏 -->
    <aside class="app-sidebar" :class="{ collapsed: isSidebarCollapsed }">
      <!-- Logo -->
      <div class="sidebar-header">
        <div class="logo-icon">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
          </svg>
        </div>
        <div v-show="!isSidebarCollapsed" class="logo-area">
          <span class="logo-text">MicroHub</span>
          <span class="logo-sub">微服务治理平台</span>
        </div>
      </div>

      <!-- 健康概览条 -->
      <div v-show="!isSidebarCollapsed" class="health-bar">
        <div class="health-stats">
          <span class="health-dot green"></span>
          <span>{{ healthSummary.healthy }} 健康</span>
          <span v-if="healthSummary.warning > 0" class="health-stat">
            <span class="health-dot yellow"></span>
            <span>{{ healthSummary.warning }} 告警</span>
          </span>
          <span v-if="healthSummary.error > 0" class="health-stat">
            <span class="health-dot red"></span>
            <span>{{ healthSummary.error }} 异常</span>
          </span>
        </div>
      </div>

      <!-- 菜单分组 -->
      <nav class="sidebar-nav">
        <div v-for="group in menuGroups" :key="group.group" class="menu-group">
          <div v-show="!isSidebarCollapsed" class="group-label">{{ group.group }}</div>
          <div
            v-for="item in group.items"
            :key="item.key"
            class="nav-item"
            :class="{ active: currentRoute === item.key }"
            @click="navigateTo(item)"
          >
            <el-icon :size="18"><component :is="item.icon" /></el-icon>
            <div v-show="!isSidebarCollapsed" class="nav-content">
              <span class="nav-label">{{ item.label }}</span>
              <span class="nav-desc">{{ item.desc }}</span>
            </div>
          </div>
        </div>
      </nav>

      <div class="sidebar-toggle" @click="isSidebarCollapsed = !isSidebarCollapsed">
        <el-icon :size="16"><component :is="isSidebarCollapsed ? 'DArrowRight' : 'DArrowLeft'" /></el-icon>
      </div>
    </aside>

    <!-- 主区域 -->
    <div class="app-main">
      <!-- 顶栏 -->
      <header class="app-header">
        <div class="header-left">
          <el-icon :size="18" class="header-menu-btn" @click="isSidebarCollapsed = !isSidebarCollapsed"><Fold /></el-icon>

          <!-- 租户选择器 - 核心功能 -->
          <div class="tenant-selector">
            <el-icon :size="14" color="#22c55e"><OfficeBuilding /></el-icon>
            <span class="tenant-label">当前租户</span>
            <el-select
              v-model="currentTenantId"
              placeholder="选择租户"
              size="small"
              popper-class="tenant-popper"
              @change="switchTenant"
            >
              <el-option v-for="t in tenantOptions" :key="t.value" :label="t.label" :value="t.value" />
            </el-select>
          </div>

          <!-- 快捷状态指示 -->
          <div class="quick-status">
            <div class="status-chip" :class="{ healthy: true }">
              <span class="chip-dot"></span> Gateway
            </div>
            <div class="status-chip" :class="{ healthy: true }">
              <span class="chip-dot"></span> Consul
            </div>
            <div class="status-chip warning">
              <span class="chip-dot"></span> NATS
            </div>
          </div>
        </div>

        <div class="header-right">
          <el-dropdown trigger="click">
            <div class="user-info">
              <el-avatar :size="30" icon="UserFilled" />
              <span class="username">{{ userStore.username || '管理员' }}</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item><el-icon><User /></el-icon> 个人信息</el-dropdown-item>
                <el-dropdown-item><el-icon><Setting /></el-icon> 平台设置</el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout"><el-icon><SwitchButton /></el-icon> 退出</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <!-- 内容区 -->
      <main class="app-content">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<style scoped>
.app-layout { display: flex; height: 100vh; overflow: hidden; background: var(--bg-primary); }

/* ===== 侧边栏 ===== */
.app-sidebar {
  width: 240px; min-width: 64px;
  background: var(--bg-sidebar);
  display: flex; flex-direction: column;
  transition: width 0.25s cubic-bezier(.4,0,.2,1);
  border-right: 1px solid var(--border-color);
  overflow: hidden;
}
.app-sidebar.collapsed { width: 64px; }

.sidebar-header {
  display: flex; align-items: center; padding: 16px;
  gap: 10px; height: 60px;
  border-bottom: 1px solid var(--border-color);
}
.logo-icon {
  width: 32px; height: 32px; border-radius: 8px;
  background: linear-gradient(135deg, #22c55e, #3b82f6);
  display: flex; align-items: center; justify-content: center;
  color: #fff; flex-shrink: 0;
}
.logo-area { display: flex; flex-direction: column; }
.logo-text {
  font-size: 16px; font-weight: 800;
  background: linear-gradient(135deg, #22c55e, #3b82f6);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent;
  white-space: nowrap; line-height: 1.2;
}
.logo-sub { font-size: 10px; color: var(--text-muted); white-space: nowrap; }

/* 健康概览条 */
.health-bar {
  padding: 8px 16px;
  border-bottom: 1px solid var(--border-color);
  font-size: 12px; color: var(--text-secondary);
}
.health-stats { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.health-stat { display: flex; align-items: center; gap: 4px; }
.health-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.health-dot.green { background: var(--success); }
.health-dot.yellow { background: var(--warning); }
.health-dot.red { background: var(--danger); }

/* 菜单分组 */
.sidebar-nav { flex: 1; padding: 8px 0; overflow-y: auto; }
.menu-group { margin-bottom: 8px; }
.group-label {
  font-size: 11px; color: var(--text-muted); font-weight: 600;
  padding: 4px 20px; letter-spacing: 1px; text-transform: uppercase;
}
.nav-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 16px; margin: 2px 8px;
  border-radius: 6px; cursor: pointer;
  color: var(--text-secondary); transition: all 0.15s ease;
  font-size: 13px; white-space: nowrap;
}
.nav-item:hover { background: rgba(59,130,246,0.08); color: var(--text-primary); }
.nav-item.active {
  background: linear-gradient(135deg, rgba(59,130,246,0.15), rgba(34,197,94,0.08));
  color: #3b82f6; font-weight: 600;
}
.nav-content { display: flex; flex-direction: column; }
.nav-desc { font-size: 10px; color: var(--text-muted); line-height: 1.3; }

.sidebar-toggle {
  padding: 12px; display: flex; justify-content: center;
  cursor: pointer; color: var(--text-muted);
  border-top: 1px solid var(--border-color); transition: color 0.2s;
}
.sidebar-toggle:hover { color: #3b82f6; }

/* ===== 主区域 ===== */
.app-main { flex: 1; display: flex; flex-direction: column; min-width: 0; overflow: hidden; }

.app-header {
  height: 56px; display: flex; align-items: center; justify-content: space-between;
  padding: 0 20px; background: var(--bg-header);
  border-bottom: 1px solid var(--border-color); flex-shrink: 0;
}
.header-left { display: flex; align-items: center; gap: 16px; }
.header-menu-btn { cursor: pointer; color: var(--text-secondary); }

/* 租户选择器 */
.tenant-selector {
  display: flex; align-items: center; gap: 6px;
  background: rgba(34,197,94,0.08); padding: 4px 10px; border-radius: 6px;
  border: 1px solid rgba(34,197,94,0.2);
}
.tenant-label { font-size: 12px; color: var(--success); font-weight: 500; white-space: nowrap; }
.tenant-selector .el-select { width: 140px; }

/* 快捷状态 */
.quick-status { display: flex; gap: 6px; }
.status-chip {
  display: flex; align-items: center; gap: 4px;
  padding: 2px 8px; border-radius: 4px; font-size: 11px;
  background: rgba(255,255,255,0.04); color: var(--text-secondary);
}
.status-chip.healthy .chip-dot { background: var(--success); width: 6px; height: 6px; border-radius: 50%; }
.status-chip.warning .chip-dot { background: var(--warning); width: 6px; height: 6px; border-radius: 50%; }
.status-chip.warning { color: var(--warning); }

.header-right { display: flex; align-items: center; gap: 16px; }
.user-info {
  display: flex; align-items: center; gap: 6px; cursor: pointer;
  color: var(--text-primary); font-size: 13px; padding: 4px 6px; border-radius: 6px;
  transition: background 0.2s;
}
.user-info:hover { background: rgba(59,130,246,0.08); }
.username { font-weight: 500; }

.app-content { flex: 1; padding: 20px; overflow-y: auto; }
</style>
