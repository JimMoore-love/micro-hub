<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { gatewayApi } from '@/api/platform'

// ==================== 统计卡片 ====================
const statCards = [
  { title: '总路由数', value: '12', icon: 'Connection', color: '#3b82f6', bg: 'rgba(59,130,246,0.1)' },
  { title: '活跃限流规则', value: '5', icon: 'Odometer', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)' },
  { title: 'JWT 验证/小时', value: '1,250', icon: 'Key', color: '#22c55e', bg: 'rgba(34,197,94,0.1)' },
  { title: '租户路由命中率', value: '100%', icon: 'Connection', color: '#8b5cf6', bg: 'rgba(139,92,246,0.1)' },
]

// ==================== 路由规则 ====================
interface RouteRule {
  id: string
  path: string
  upstream: string
  methods: string[]
  rateLimit: number | null
  tenantRouting: boolean
  enabled: boolean
}

const routes = ref<RouteRule[]>([
  { id: 'r1', path: '/api/v1/users/*', upstream: 'user-service:8081', methods: ['GET', 'POST'], rateLimit: 100, tenantRouting: true, enabled: true },
  { id: 'r2', path: '/api/v1/orders/*', upstream: 'order-service:8082', methods: ['GET', 'POST', 'PUT'], rateLimit: 200, tenantRouting: true, enabled: true },
  { id: 'r3', path: '/api/v1/ai/*', upstream: 'ai-service:8083', methods: ['POST'], rateLimit: 50, tenantRouting: true, enabled: true },
  { id: 'r4', path: '/api/v1/proofread/*', upstream: 'ai-service:8083', methods: ['POST'], rateLimit: 30, tenantRouting: true, enabled: true },
  { id: 'r5', path: '/health', upstream: 'gateway:8080', methods: ['GET'], rateLimit: null, tenantRouting: false, enabled: true },
  { id: 'r6', path: '/metrics', upstream: 'gateway:8080', methods: ['GET'], rateLimit: null, tenantRouting: false, enabled: true },
  { id: 'r7', path: '/api/v1/auth/*', upstream: 'user-service:8081', methods: ['POST'], rateLimit: 80, tenantRouting: false, enabled: true },
  { id: 'r8', path: '/api/v1/notifications/*', upstream: 'order-service:8082', methods: ['GET', 'POST'], rateLimit: 150, tenantRouting: true, enabled: true },
  { id: 'r9', path: '/api/v1/files/*', upstream: 'minio:9000', methods: ['GET', 'POST', 'DELETE'], rateLimit: 50, tenantRouting: true, enabled: false },
  { id: 'r10', path: '/ws/*', upstream: 'user-service:8081', methods: ['GET'], rateLimit: null, tenantRouting: false, enabled: true },
  { id: 'r11', path: '/api/v1/reports/*', upstream: 'ai-service:8083', methods: ['GET'], rateLimit: 20, tenantRouting: true, enabled: true },
  { id: 'r12', path: '/admin/*', upstream: 'gateway:8080', methods: ['GET', 'POST', 'PUT', 'DELETE'], rateLimit: 30, tenantRouting: false, enabled: true },
])

// ==================== 中间件配置 ====================
const activeTab = ref('routes')

const corsConfig = reactive({
  allowed_origins: ['http://localhost:3000', 'https://app.example.com', 'https://admin.example.com'],
  allowed_methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
  allowed_headers: ['Content-Type', 'Authorization', 'X-Tenant-ID', 'X-Request-ID'],
  max_age: 86400,
})

const jwtConfig = reactive({
  secret: 'jwt-secret-key-2024-***',
  expiry: '24h',
  excluded_paths: ['/health', '/metrics', '/api/v1/public/*'],
  algorithm: 'HS256',
  issuer: 'microservice-gateway',
})

const rateLimitConfig = reactive({
  global_rate: 1000,
  per_tenant_rate: 200,
  burst_size: 50,
  window_size: '1m',
  enabled: true,
})

const tenantRoutingConfig = reactive({
  header_key: 'X-Tenant-ID',
  subdomain_enabled: true,
  subdomain_mapping: '{tenant}.api.example.com',
  default_tenant: 'default',
})

// ==================== 弹窗 ====================
const routeDialogVisible = ref(false)
const editingRoute = ref<RouteRule | null>(null)
const isNewRoute = ref(false)

const routeForm = reactive<RouteRule>({
  id: '', path: '', upstream: '', methods: [], rateLimit: null, tenantRouting: false, enabled: true,
})

function openNewRoute() {
  isNewRoute.value = true
  editingRoute.value = null
  Object.assign(routeForm, { id: 'r' + Date.now(), path: '', upstream: '', methods: [], rateLimit: null, tenantRouting: false, enabled: true })
  routeDialogVisible.value = true
}

function openEditRoute(route: RouteRule) {
  isNewRoute.value = false
  editingRoute.value = route
  Object.assign(routeForm, { ...route })
  routeDialogVisible.value = true
}

function saveRoute() {
  if (isNewRoute.value) {
    routes.value.push({ ...routeForm })
  } else if (editingRoute.value) {
    const idx = routes.value.findIndex(r => r.id === editingRoute.value!.id)
    if (idx !== -1) Object.assign(routes.value[idx], { ...routeForm })
  }
  routeDialogVisible.value = false
}

function deleteRoute(route: RouteRule) {
  routes.value = routes.value.filter(r => r.id !== route.id)
}

function toggleRoute(route: RouteRule) {
  route.enabled = !route.enabled
}

// ==================== 中间件保存 ====================
function saveCors() { /* placeholder */ }
function saveJwt() { /* placeholder */ }
function saveRateLimit() { /* placeholder */ }
function saveTenant() { /* placeholder */ }

onMounted(async () => {
  try {
    const [routeData, mwData] = await Promise.all([
      gatewayApi.listRoutes(),
      gatewayApi.getMiddlewareConfig(),
    ])
    if (routeData && routeData.length > 0) {
      routes.value = routeData.map(r => ({
        id: r.id,
        path: r.path,
        upstream: r.upstream,
        methods: r.methods,
        rateLimit: r.rate_limit,
        tenantRouting: r.tenant_routing,
        enabled: r.status === 'enabled',
      }))
    }
    if (mwData) {
      if (mwData.cors) {
        corsConfig.allowed_origins = mwData.cors.allowed_origins
        corsConfig.allowed_methods = mwData.cors.methods
        corsConfig.allowed_headers = mwData.cors.headers
      }
      if (mwData.jwt) {
        jwtConfig.secret = mwData.jwt.secret
        jwtConfig.expiry = String(mwData.jwt.expiry)
        jwtConfig.excluded_paths = mwData.jwt.excluded_paths
      }
      if (mwData.rate_limit) {
        rateLimitConfig.global_rate = mwData.rate_limit.global_rate
        rateLimitConfig.per_tenant_rate = mwData.rate_limit.per_tenant_rate
        rateLimitConfig.burst_size = mwData.rate_limit.burst_size
      }
      if (mwData.tenant_routing) {
        tenantRoutingConfig.header_key = mwData.tenant_routing.header_key
        tenantRoutingConfig.subdomain_mapping = JSON.stringify(mwData.tenant_routing.subdomain_mapping)
      }
    }
  } catch (e) {
    console.error('Failed to fetch gateway data:', e)
  }
})
</script>

<template>
  <div class="gateway-page">
    <h1 class="page-title">API 网关管理</h1>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div v-for="card in statCards" :key="card.title" class="stat-card">
        <div class="stat-icon" :style="{ background: card.bg, color: card.color }">
          <el-icon :size="22"><component :is="card.icon" /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-label">{{ card.title }}</div>
          <div class="stat-value mono">{{ card.value }}</div>
        </div>
      </div>
    </div>

    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" class="gw-tabs">
      <!-- Tab 1: 路由规则 -->
      <el-tab-pane label="路由规则" name="routes">
        <div class="tab-toolbar">
          <span class="tab-subtitle">共 {{ routes.length }} 条路由规则</span>
          <el-button type="primary" size="small" class="btn-gradient" @click="openNewRoute">
            <el-icon><Plus /></el-icon> 新增路由
          </el-button>
        </div>

        <div class="page-card" style="padding: 0; overflow: hidden">
          <el-table :data="routes" stripe style="width: 100%" :row-style="{ height: '50px' }">
            <el-table-column prop="path" label="路径模式" min-width="180">
              <template #default="{ row }">
                <code class="mono route-path">{{ row.path }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="upstream" label="上游服务" width="170">
              <template #default="{ row }">
                <span class="upstream-tag">{{ row.upstream }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="methods" label="方法" width="150">
              <template #default="{ row }">
                <el-tag
                  v-for="m in row.methods"
                  :key="m"
                  size="small"
                  :type="m === 'GET' ? 'success' : m === 'POST' ? '' : m === 'PUT' ? 'warning' : 'danger'"
                  class="method-tag"
                >{{ m }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="rateLimit" label="限流(QPS)" width="100" align="center">
              <template #default="{ row }">
                <span class="mono" v-if="row.rateLimit">{{ row.rateLimit }}</span>
                <el-tag v-else size="small" type="info">无限制</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="tenantRouting" label="租户路由" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="row.tenantRouting ? 'success' : 'info'">
                  {{ row.tenantRouting ? '是' : '否' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="enabled" label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-switch
                  v-model="row.enabled"
                  size="small"
                  @change="toggleRoute(row)"
                />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="130" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openEditRoute(row)">编辑</el-button>
                <el-button link type="danger" size="small" @click="deleteRoute(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Tab 2: 中间件配置 -->
      <el-tab-pane label="中间件配置" name="middleware">
        <div class="mw-grid">
          <!-- CORS 配置 -->
          <div class="page-card mw-card">
            <div class="page-card-title">CORS 跨域配置</div>
            <el-form label-position="top" size="small">
              <el-form-item label="允许的来源 (Allowed Origins)">
                <div class="tag-input">
                  <el-tag
                    v-for="(origin, idx) in corsConfig.allowed_origins"
                    :key="idx"
                    closable
                    size="small"
                    @close="corsConfig.allowed_origins.splice(idx, 1)"
                  >{{ origin }}</el-tag>
                </div>
              </el-form-item>
              <el-form-item label="允许的方法">
                <div class="tag-input">
                  <el-tag v-for="m in corsConfig.allowed_methods" :key="m" size="small" type="success">{{ m }}</el-tag>
                </div>
              </el-form-item>
              <el-form-item label="允许的请求头">
                <div class="tag-input">
                  <el-tag v-for="h in corsConfig.allowed_headers" :key="h" size="small" type="info">{{ h }}</el-tag>
                </div>
              </el-form-item>
              <el-form-item label="预检缓存时间 (s)">
                <span class="form-value mono">{{ corsConfig.max_age }}</span>
              </el-form-item>
              <el-button type="primary" size="small" @click="saveCors">保存配置</el-button>
            </el-form>
          </div>

          <!-- JWT 鉴权 -->
          <div class="page-card mw-card">
            <div class="page-card-title">JWT 鉴权配置</div>
            <el-form label-position="top" size="small">
              <el-form-item label="密钥 (Secret)">
                <code class="mono form-code">{{ jwtConfig.secret }}</code>
              </el-form-item>
              <el-form-item label="过期时间">
                <span class="form-value mono">{{ jwtConfig.expiry }}</span>
              </el-form-item>
              <el-form-item label="签名算法">
                <span class="form-value mono">{{ jwtConfig.algorithm }}</span>
              </el-form-item>
              <el-form-item label="签发者 (Issuer)">
                <span class="form-value mono">{{ jwtConfig.issuer }}</span>
              </el-form-item>
              <el-form-item label="排除路径">
                <div class="tag-input">
                  <el-tag v-for="p in jwtConfig.excluded_paths" :key="p" size="small" type="warning">{{ p }}</el-tag>
                </div>
              </el-form-item>
              <el-button type="primary" size="small" @click="saveJwt">保存配置</el-button>
            </el-form>
          </div>

          <!-- 限流策略 -->
          <div class="page-card mw-card">
            <div class="page-card-title">限流策略</div>
            <el-form label-position="top" size="small">
              <el-form-item label="全局速率 (QPS)">
                <span class="form-value mono big">{{ rateLimitConfig.global_rate }}</span>
              </el-form-item>
              <el-form-item label="每租户速率 (QPS)">
                <span class="form-value mono big">{{ rateLimitConfig.per_tenant_rate }}</span>
              </el-form-item>
              <el-form-item label="突发容量">
                <span class="form-value mono big">{{ rateLimitConfig.burst_size }}</span>
              </el-form-item>
              <el-form-item label="时间窗口">
                <span class="form-value mono">{{ rateLimitConfig.window_size }}</span>
              </el-form-item>
              <el-form-item label="启用心态">
                <el-switch v-model="rateLimitConfig.enabled" size="small" />
              </el-form-item>
              <el-button type="primary" size="small" @click="saveRateLimit">保存配置</el-button>
            </el-form>
          </div>

          <!-- 租户路由 -->
          <div class="page-card mw-card">
            <div class="page-card-title">租户路由</div>
            <el-form label-position="top" size="small">
              <el-form-item label="租户请求头 Key">
                <code class="mono form-code">{{ tenantRoutingConfig.header_key }}</code>
              </el-form-item>
              <el-form-item label="子域名路由">
                <el-switch v-model="tenantRoutingConfig.subdomain_enabled" size="small" />
              </el-form-item>
              <el-form-item label="子域名映射">
                <code class="mono form-code">{{ tenantRoutingConfig.subdomain_mapping }}</code>
              </el-form-item>
              <el-form-item label="默认租户">
                <span class="form-value mono">{{ tenantRoutingConfig.default_tenant }}</span>
              </el-form-item>
              <el-button type="primary" size="small" @click="saveTenant">保存配置</el-button>
            </el-form>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 路由编辑弹窗 -->
    <el-dialog
      v-model="routeDialogVisible"
      :title="isNewRoute ? '新增路由' : '编辑路由'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form :model="routeForm" label-width="100px" size="small">
        <el-form-item label="路径模式">
          <el-input v-model="routeForm.path" placeholder="/api/v1/users/*" />
        </el-form-item>
        <el-form-item label="上游服务">
          <el-input v-model="routeForm.upstream" placeholder="user-service:8081" />
        </el-form-item>
        <el-form-item label="HTTP 方法">
          <el-select v-model="routeForm.methods" multiple placeholder="选择方法">
            <el-option label="GET" value="GET" />
            <el-option label="POST" value="POST" />
            <el-option label="PUT" value="PUT" />
            <el-option label="DELETE" value="DELETE" />
            <el-option label="PATCH" value="PATCH" />
          </el-select>
        </el-form-item>
        <el-form-item label="限流(QPS)">
          <el-input-number v-model="routeForm.rateLimit" :min="0" :max="10000" placeholder="留空则不限制" />
        </el-form-item>
        <el-form-item label="租户路由">
          <el-switch v-model="routeForm.tenantRouting" size="small" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="routeForm.enabled" size="small" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="routeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRoute">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.gateway-page {
  max-width: 1400px;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: all 0.3s ease;
}

.stat-card:hover {
  background: rgba(255,255,255,0.06);
  transform: translateY(-2px);
}

.stat-icon {
  width: 50px;
  height: 50px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  font-size: 13px;
  color: var(--text-muted);
}

.stat-value {
  font-size: 26px;
  font-weight: 700;
  color: var(--text-primary);
}

.mono {
  font-family: 'Fira Code', 'Cascadia Code', monospace;
}

/* Tab */
.gw-tabs {
  --el-tabs-header-height: 40px;
}

.tab-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.tab-subtitle {
  font-size: 13px;
  color: var(--text-muted);
}

/* 路由表格 */
.route-path {
  font-size: 12px;
  color: var(--accent);
  background: rgba(59,130,246,0.08);
  padding: 2px 8px;
  border-radius: 4px;
}

.upstream-tag {
  font-size: 12px;
  color: var(--text-secondary);
  font-family: 'Fira Code', monospace;
}

.method-tag {
  margin-right: 4px;
  margin-bottom: 2px;
}

/* 中间件配置网格 */
.mw-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.mw-card {
  margin-bottom: 0;
}

.tag-input {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.form-value {
  font-size: 14px;
  color: var(--text-primary);
}

.form-value.big {
  font-size: 22px;
  font-weight: 700;
  color: var(--accent);
}

.form-code {
  font-size: 12px;
  color: var(--accent);
  background: rgba(59,130,246,0.08);
  padding: 3px 10px;
  border-radius: 4px;
}

@media (max-width: 1200px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .mw-grid { grid-template-columns: 1fr; }
}
</style>
