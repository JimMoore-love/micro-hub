import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'Login', component: () => import('../views/Login.vue'), meta: { public: true } },
    { path: '/', redirect: '/topology' },
    { path: '/topology', name: 'Topology', component: () => import('../views/Topology.vue') },
    { path: '/services', name: 'Services', component: () => import('../views/Services.vue') },
    { path: '/nodes', name: 'Nodes', component: () => import('../views/Nodes.vue') },
    { path: '/gateway', name: 'Gateway', component: () => import('../views/Gateway.vue') },
    { path: '/traffic', name: 'Traffic', component: () => import('../views/Traffic.vue') },
    { path: '/tenants', name: 'Tenants', component: () => import('../views/Tenants.vue') },
    { path: '/users', name: 'UserList', component: () => import('../views/UserList.vue') },
    { path: '/ai-providers', name: 'AIProviders', component: () => import('../views/AIProviders.vue') },
    { path: '/proofread', name: 'Proofread', component: () => import('../views/Proofread.vue') },
    { path: '/ai-chat', name: 'AIChat', component: () => import('../views/AIChat.vue') },
    { path: '/observability', name: 'Observability', component: () => import('../views/Observability.vue') },
    { path: '/alerts', name: 'Alerts', component: () => import('../views/Alerts.vue') },
    { path: '/dashboard', redirect: '/topology' },
    { path: '/ai', redirect: '/ai-chat' },
    { path: '/orders', redirect: '/topology' },
    { path: '/settings', redirect: '/gateway' },
  ],
})

// 路由守卫：未登录跳转登录页
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (!token && !to.meta.public) {
    next('/login')
  } else if (token && to.name === 'Login') {
    next('/topology')
  } else {
    next()
  }
})

export default router
