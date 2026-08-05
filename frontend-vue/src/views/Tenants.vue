<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { tenantApi } from '@/api/platform'
import { ElMessage } from 'element-plus'

// ==================== 统计卡片 ====================
const statCards = [
  { title: '租户总数', value: '4', icon: 'OfficeBuilding', color: '#3b82f6', bg: 'rgba(59,130,246,0.1)' },
  { title: '活跃租户', value: '3', icon: 'CircleCheck', color: '#22c55e', bg: 'rgba(34,197,94,0.1)' },
  { title: '总用户数', value: '156', icon: 'User', color: '#8b5cf6', bg: 'rgba(139,92,246,0.1)' },
  { title: '总AI用量', value: '12.5K', unit: 'tokens', icon: 'Cpu', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)' },
]

// ==================== 租户数据 ====================
interface Tenant {
  id: string
  name: string
  schema: string
  users: number
  quota: number
  used: number
  status: 'active' | 'frozen'
  plan: string
  createdAt: string
  apiCalls: number
  storage: number
  storageQuota: number
  apiKeys: ApiKey[]
  dbTables: number
  dbRecords: number
  redisMemory: number
}

interface ApiKey {
  id: string
  key: string
  createdAt: string
  status: 'active' | 'disabled'
}

const tenants = ref<Tenant[]>([
  {
    id: 'default', name: '默认租户', schema: 'public', users: 45, quota: 50000, used: 12000,
    status: 'active', plan: 'free', createdAt: '2024-01-10 08:00:00', apiCalls: 1000, storage: 512, storageQuota: 1024,
    dbTables: 18, dbRecords: 45200, redisMemory: 24,
    apiKeys: [
      { id: 'k1', key: 'sk-tenant-default-a1b2c3d4', createdAt: '2024-01-10 08:05:00', status: 'active' },
      { id: 'k2', key: 'sk-tenant-default-e5f6g7h8', createdAt: '2024-02-15 10:30:00', status: 'active' },
    ],
  },
  {
    id: 'enterprise-a', name: '企业A', schema: 'tenant_ea', users: 68, quota: 200000, used: 85000,
    status: 'active', plan: 'pro', createdAt: '2024-01-20 09:00:00', apiCalls: 5000, storage: 4096, storageQuota: 8192,
    dbTables: 32, dbRecords: 156800, redisMemory: 86,
    apiKeys: [
      { id: 'k3', key: 'sk-tenant-ea-i9j0k1l2', createdAt: '2024-01-20 09:10:00', status: 'active' },
      { id: 'k4', key: 'sk-tenant-ea-m3n4o5p6', createdAt: '2024-03-01 14:00:00', status: 'disabled' },
    ],
  },
  {
    id: 'enterprise-b', name: '企业B', schema: 'tenant_eb', users: 32, quota: 100000, used: 42000,
    status: 'active', plan: 'standard', createdAt: '2024-02-05 11:00:00', apiCalls: 3000, storage: 2048, storageQuota: 4096,
    dbTables: 24, dbRecords: 88300, redisMemory: 52,
    apiKeys: [
      { id: 'k5', key: 'sk-tenant-eb-q7r8s9t0', createdAt: '2024-02-05 11:15:00', status: 'active' },
    ],
  },
  {
    id: 'test-org', name: '测试组织', schema: 'tenant_test', users: 11, quota: 10000, used: 9500,
    status: 'frozen', plan: 'free', createdAt: '2024-03-01 08:00:00', apiCalls: 200, storage: 256, storageQuota: 512,
    dbTables: 12, dbRecords: 5600, redisMemory: 8,
    apiKeys: [
      { id: 'k6', key: 'sk-tenant-test-u1v2w3x4', createdAt: '2024-03-01 08:10:00', status: 'disabled' },
    ],
  },
])

// ==================== Tab 切换 ====================
const activeTab = ref('list')
const selectedTenant = ref<Tenant | null>(null)

function showDetail(t: Tenant) {
  selectedTenant.value = t
  activeTab.value = 'detail'
}

// ==================== 用量百分比 ====================
function usedPercent(t: Tenant): number {
  return Math.round((t.used / t.quota) * 100)
}

function storagePercent(t: Tenant): number {
  return Math.round((t.storage / t.storageQuota) * 100)
}

// ==================== 新增租户弹窗 ====================
const dialogVisible = ref(false)
const newTenant = reactive({
  id: '', name: '', plan: 'free', quota: 10000, storageQuota: 1024, apiCalls: 1000,
})

function openNewTenant() {
  Object.assign(newTenant, { id: '', name: '', plan: 'free', quota: 10000, storageQuota: 1024, apiCalls: 1000 })
  dialogVisible.value = true
}

function createTenant() {
  if (!newTenant.id || !newTenant.name) {
    ElMessage.warning('请填写租户ID和名称')
    return
  }
  tenants.value.push({
    ...newTenant,
    schema: `tenant_${newTenant.id.replace(/-/g, '_')}`,
    users: 0, used: 0, status: 'active', createdAt: new Date().toLocaleString('zh-CN'),
    apiKeys: [], dbTables: 0, dbRecords: 0, redisMemory: 0, storage: 0,
  })
  dialogVisible.value = false
  ElMessage.success('租户创建成功')
}

// ==================== API Key 操作 ====================
function copyKey(key: string) {
  navigator.clipboard?.writeText(key)
  ElMessage.success('API Key 已复制到剪贴板')
}

function toggleKey(t: Tenant, k: ApiKey) {
  k.status = k.status === 'active' ? 'disabled' : 'active'
  ElMessage.success(`Key 已${k.status === 'active' ? '启用' : '禁用'}`)
}

function deleteKey(t: Tenant, k: ApiKey) {
  t.apiKeys = t.apiKeys.filter(item => item.id !== k.id)
  ElMessage.success('Key 已删除')
}

// ==================== 冻结/解冻 ====================
function toggleFreeze(t: Tenant) {
  t.status = t.status === 'active' ? 'frozen' : 'active'
  ElMessage.success(`租户已${t.status === 'active' ? '解冻' : '冻结'}`)
}

const planTagType: Record<string, string> = { free: 'info', standard: '', pro: 'success' }

onMounted(async () => {
  try {
    const data = await tenantApi.list()
    if (data && data.length > 0) {
      tenants.value = data.map(t => ({
        id: t.id,
        name: t.name,
        schema: t.schema,
        users: t.users,
        quota: t.quota,
        used: t.used,
        status: t.status,
        plan: t.plan,
        createdAt: t.created_at,
        apiCalls: 0,
        storage: 0,
        storageQuota: 0,
        apiKeys: (t.api_keys || []).map((k, idx) => ({
          id: `k-${idx}`,
          key: k.key,
          createdAt: k.created_at,
          status: k.status as 'active' | 'disabled',
        })),
        dbTables: t.db_tables,
        dbRecords: t.db_records,
        redisMemory: 0,
      }))
    }
  } catch (e) {
    console.error('Failed to fetch tenants:', e)
  }
})
</script>

<template>
  <div class="tenants-page">
    <h1 class="page-title">租户管理</h1>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div v-for="card in statCards" :key="card.title" class="stat-card">
        <div class="stat-icon" :style="{ background: card.bg, color: card.color }">
          <el-icon :size="22"><component :is="card.icon" /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-label">{{ card.title }}</div>
          <div class="stat-value mono">{{ card.value }}<span v-if="card.unit" class="stat-unit">{{ card.unit }}</span></div>
        </div>
      </div>
    </div>

    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" class="tenant-tabs">
      <!-- Tab 1: 租户列表 -->
      <el-tab-pane label="租户列表" name="list">
        <div class="tab-toolbar">
          <span class="tab-subtitle">共 {{ tenants.length }} 个租户</span>
          <el-button type="primary" size="small" class="btn-gradient" @click="openNewTenant">
            <el-icon><Plus /></el-icon> 新增租户
          </el-button>
        </div>

        <div class="page-card" style="padding: 0; overflow: hidden">
          <el-table :data="tenants" stripe style="width: 100%" :row-style="{ height: '52px' }">
            <el-table-column prop="id" label="租户ID" width="140">
              <template #default="{ row }">
                <code class="mono tenant-id">{{ row.id }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="租户名称" min-width="120">
              <template #default="{ row }">
                <span class="tenant-name">{{ row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="schema" label="Schema" width="130">
              <template #default="{ row }">
                <code class="mono schema-tag">{{ row.schema }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="users" label="用户数" width="80" align="center" />
            <el-table-column prop="quota" label="AI配额(tokens)" width="130" align="right">
              <template #default="{ row }">
                <span class="mono">{{ row.quota.toLocaleString() }}</span>
              </template>
            </el-table-column>
            <el-table-column label="已用量(%)" width="160">
              <template #default="{ row }">
                <div class="usage-bar-wrap">
                  <el-progress
                    :percentage="usedPercent(row)"
                    :stroke-width="8"
                    :color="usedPercent(row) > 80 ? '#ef4444' : usedPercent(row) > 50 ? '#f59e0b' : '#22c55e'"
                    :show-text="false"
                  />
                  <span class="usage-text mono">{{ usedPercent(row) }}%</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="plan" label="Plan" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="(planTagType[row.plan] as any)" size="small">{{ row.plan.toUpperCase() }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small" effect="dark">
                  {{ row.status === 'active' ? '活跃' : '冻结' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="showDetail(row)">详情</el-button>
                <el-button link type="warning" size="small" @click="showDetail(row)">编辑</el-button>
                <el-button link :type="row.status === 'active' ? 'danger' : 'success'" size="small" @click="toggleFreeze(row)">
                  {{ row.status === 'active' ? '冻结' : '解冻' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Tab 2: 租户详情 -->
      <el-tab-pane label="租户详情" name="detail">
        <template v-if="selectedTenant">
          <!-- 详情顶部：选择器 -->
          <div class="detail-header">
            <span class="tab-subtitle">当前查看：</span>
            <el-select v-model="selectedTenant" value-key="id" size="small" style="width: 200px" @change="(v: any) => selectedTenant = tenants.find(t => t.id === v.id) || null">
              <el-option v-for="t in tenants" :key="t.id" :label="t.name" :value="t" />
            </el-select>
          </div>

          <div class="detail-grid">
            <!-- 基本信息 -->
            <div class="page-card">
              <div class="page-card-title">基本信息</div>
              <div class="info-grid">
                <div class="info-item">
                  <span class="info-label">租户ID</span>
                  <code class="mono info-value accent">{{ selectedTenant.id }}</code>
                </div>
                <div class="info-item">
                  <span class="info-label">租户名称</span>
                  <span class="info-value">{{ selectedTenant.name }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">Plan</span>
                  <el-tag :type="(planTagType[selectedTenant.plan] as any)" size="small">{{ selectedTenant.plan.toUpperCase() }}</el-tag>
                </div>
                <div class="info-item">
                  <span class="info-label">创建时间</span>
                  <span class="info-value mono">{{ selectedTenant.createdAt }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">用户数</span>
                  <span class="info-value mono">{{ selectedTenant.users }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">状态</span>
                  <el-tag :type="selectedTenant.status === 'active' ? 'success' : 'danger'" size="small" effect="dark">
                    {{ selectedTenant.status === 'active' ? '活跃' : '冻结' }}
                  </el-tag>
                </div>
              </div>
            </div>

            <!-- 资源配额 -->
            <div class="page-card">
              <div class="page-card-title">资源配额</div>
              <!-- AI Tokens -->
              <div class="quota-item">
                <div class="quota-header">
                  <span class="quota-label">AI Tokens</span>
                  <span class="mono quota-vals">{{ selectedTenant.used.toLocaleString() }} / {{ selectedTenant.quota.toLocaleString() }}</span>
                </div>
                <el-progress
                  :percentage="usedPercent(selectedTenant)"
                  :stroke-width="10"
                  :color="usedPercent(selectedTenant) > 80 ? '#ef4444' : usedPercent(selectedTenant) > 50 ? '#f59e0b' : '#22c55e'"
                />
              </div>
              <!-- API调用限制 -->
              <div class="quota-item">
                <div class="quota-header">
                  <span class="quota-label">API调用限制</span>
                  <span class="mono quota-vals">{{ selectedTenant.apiCalls.toLocaleString() }} /小时</span>
                </div>
                <el-progress :percentage="35" :stroke-width="10" color="#3b82f6" />
              </div>
              <!-- 存储配额 -->
              <div class="quota-item">
                <div class="quota-header">
                  <span class="quota-label">存储配额</span>
                  <span class="mono quota-vals">{{ selectedTenant.storage }}MB / {{ selectedTenant.storageQuota }}MB</span>
                </div>
                <el-progress :percentage="storagePercent(selectedTenant)" :stroke-width="10" color="#8b5cf6" />
              </div>
            </div>
          </div>

          <!-- API Key 管理 -->
          <div class="page-card">
            <div class="page-card-title">
              API Key 管理
              <el-tag size="small" type="info" class="title-badge">{{ selectedTenant.apiKeys.length }} 个</el-tag>
            </div>
            <div class="isolation-note">
              <el-icon><InfoFilled /></el-icon>
              <span>每个租户使用独立API Key进行鉴权，网关通过 <code class="mono">X-Tenant-ID</code> 请求头与 Key 映射实现租户隔离</span>
            </div>
            <el-table :data="selectedTenant.apiKeys" style="width: 100%" :row-style="{ height: '48px' }">
              <el-table-column prop="key" label="API Key" min-width="280">
                <template #default="{ row }">
                  <code class="mono api-key-display">{{ row.key.slice(0, 12) }}••••••••{{ row.key.slice(-4) }}</code>
                </template>
              </el-table-column>
              <el-table-column prop="createdAt" label="创建时间" width="180">
                <template #default="{ row }">
                  <span class="mono text-muted">{{ row.createdAt }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="90" align="center">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                    {{ row.status === 'active' ? '启用' : '禁用' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="180">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="copyKey(row.key)">
                    <el-icon><CopyDocument /></el-icon> 复制
                  </el-button>
                  <el-button link type="warning" size="small" @click="toggleKey(selectedTenant, row)">
                    {{ row.status === 'active' ? '禁用' : '启用' }}
                  </el-button>
                  <el-button link type="danger" size="small" @click="deleteKey(selectedTenant, row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <!-- 隔离机制展示 -->
          <div class="detail-grid">
            <!-- Redis 隔离 -->
            <div class="page-card">
              <div class="page-card-title">Redis 隔离状态</div>
              <div class="isolation-block">
                <div class="iso-label">Key 前缀</div>
                <code class="mono iso-code">tenant:{{ selectedTenant.id }}:*</code>
                <div class="iso-desc">该租户所有 Redis Key 以 <code class="mono">tenant:{id}:</code> 为前缀，与其他租户数据物理隔离</div>
              </div>
              <div class="iso-metric">
                <div class="iso-metric-item">
                  <span class="iso-metric-label">内存使用</span>
                  <span class="mono iso-metric-value">{{ selectedTenant.redisMemory }} MB</span>
                </div>
                <div class="iso-metric-item">
                  <span class="iso-metric-label">Key数量(估)</span>
                  <span class="mono iso-metric-value">{{ Math.round(selectedTenant.redisMemory * 1.2) }}</span>
                </div>
              </div>
              <el-progress :percentage="Math.min(selectedTenant.redisMemory / 2, 100)" :stroke-width="8" color="#ef4444" />
            </div>

            <!-- 数据库隔离 -->
            <div class="page-card">
              <div class="page-card-title">数据库隔离 (Schema)</div>
              <div class="isolation-block">
                <div class="iso-label">Schema 名称</div>
                <code class="mono iso-code">{{ selectedTenant.schema }}</code>
                <div class="iso-desc">PostgreSQL Schema 级别隔离，每租户独立 Schema，表结构相同数据互不可见</div>
              </div>
              <div class="iso-metric">
                <div class="iso-metric-item">
                  <span class="iso-metric-label">表数量</span>
                  <span class="mono iso-metric-value">{{ selectedTenant.dbTables }}</span>
                </div>
                <div class="iso-metric-item">
                  <span class="iso-metric-label">总记录数</span>
                  <span class="mono iso-metric-value">{{ selectedTenant.dbRecords.toLocaleString() }}</span>
                </div>
              </div>
              <el-progress :percentage="Math.min((selectedTenant.dbRecords / 200000) * 100, 100)" :stroke-width="8" color="#3b82f6" />
            </div>
          </div>
        </template>

        <div v-else class="empty-detail">
          <el-icon :size="48"><OfficeBuilding /></el-icon>
          <p>请从"租户列表"中选择一个租户查看详情</p>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 新增租户弹窗 -->
    <el-dialog v-model="dialogVisible" title="新增租户" width="520px" :close-on-click-modal="false">
      <el-form :model="newTenant" label-width="110px" size="default">
        <el-form-item label="租户ID" required>
          <el-input v-model="newTenant.id" placeholder="如: enterprise-c" />
        </el-form-item>
        <el-form-item label="租户名称" required>
          <el-input v-model="newTenant.name" placeholder="如: 企业C" />
        </el-form-item>
        <el-form-item label="Plan">
          <el-select v-model="newTenant.plan" style="width: 100%">
            <el-option label="Free (免费版)" value="free" />
            <el-option label="Standard (标准版)" value="standard" />
            <el-option label="Pro (专业版)" value="pro" />
          </el-select>
        </el-form-item>
        <el-form-item label="AI配额(tokens)">
          <el-input-number v-model="newTenant.quota" :min="1000" :step="5000" style="width: 100%" />
        </el-form-item>
        <el-form-item label="存储配额(MB)">
          <el-input-number v-model="newTenant.storageQuota" :min="100" :step="512" style="width: 100%" />
        </el-form-item>
        <el-form-item label="API限制/小时">
          <el-input-number v-model="newTenant.apiCalls" :min="100" :step="500" style="width: 100%" />
        </el-form-item>
      </el-form>
      <div class="dialog-hint">
        <el-icon><InfoFilled /></el-icon>
        <span>创建后将自动生成 Schema: <code class="mono">tenant_{{ newTenant.id || 'xxx' }}</code> 和 Redis 前缀: <code class="mono">tenant:{{ newTenant.id || 'xxx' }}:</code></span>
      </div>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createTenant">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.tenants-page { max-width: 1400px; }

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
  display: flex; align-items: center; gap: 16px;
  transition: all 0.3s ease;
}
.stat-card:hover { background: rgba(255,255,255,0.06); transform: translateY(-2px); }
.stat-icon {
  width: 50px; height: 50px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-info { display: flex; flex-direction: column; gap: 4px; }
.stat-label { font-size: 13px; color: var(--text-muted); }
.stat-value { font-size: 26px; font-weight: 700; color: var(--text-primary); }
.stat-unit { font-size: 12px; color: var(--text-muted); margin-left: 4px; font-weight: 400; }
.mono { font-family: 'Fira Code', 'Cascadia Code', monospace; }

/* Tab */
.tenant-tabs { --el-tabs-header-height: 40px; }
.tab-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px;
}
.tab-subtitle { font-size: 13px; color: var(--text-muted); }

/* 列表表格 */
.tenant-id { font-size: 12px; color: var(--accent); background: rgba(59,130,246,0.08); padding: 2px 8px; border-radius: 4px; }
.tenant-name { font-weight: 600; color: var(--text-primary); }
.schema-tag { font-size: 12px; color: var(--text-secondary); }

.usage-bar-wrap { display: flex; align-items: center; gap: 8px; }
.usage-bar-wrap .el-progress { flex: 1; }
.usage-text { font-size: 11px; color: var(--text-secondary); min-width: 36px; }

/* 详情 */
.detail-header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 0;
}
.detail-grid .page-card { margin-bottom: 20px; }

.empty-detail {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 80px 0; color: var(--text-muted); gap: 16px;
}

/* 基本信息 */
.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.info-item { display: flex; flex-direction: column; gap: 4px; }
.info-label { font-size: 12px; color: var(--text-muted); }
.info-value { font-size: 14px; color: var(--text-primary); }
.info-value.accent { color: var(--accent); }

/* 配额 */
.quota-item { margin-bottom: 18px; }
.quota-item:last-child { margin-bottom: 0; }
.quota-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 8px;
}
.quota-label { font-size: 13px; color: var(--text-secondary); font-weight: 500; }
.quota-vals { font-size: 13px; color: var(--text-primary); }
.title-badge { margin-left: 8px; }

/* 隔离提示 */
.isolation-note {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 14px;
  background: rgba(59,130,246,0.06);
  border: 1px solid rgba(59,130,246,0.15);
  border-radius: 8px;
  font-size: 12px; color: var(--text-secondary);
  margin-bottom: 16px;
}
.isolation-note .mono { color: var(--accent); }

.isolation-block {
  margin-bottom: 16px;
}
.iso-label { font-size: 12px; color: var(--text-muted); margin-bottom: 4px; }
.iso-code {
  display: inline-block;
  font-size: 14px; color: var(--success);
  background: rgba(34,197,94,0.08);
  padding: 4px 12px; border-radius: 6px;
  margin-bottom: 8px;
}
.iso-desc { font-size: 12px; color: var(--text-secondary); line-height: 1.5; }
.iso-desc .mono { color: var(--accent); font-size: 11px; }

.iso-metric {
  display: flex; gap: 24px; margin-bottom: 12px;
}
.iso-metric-item { display: flex; flex-direction: column; gap: 2px; }
.iso-metric-label { font-size: 11px; color: var(--text-muted); }
.iso-metric-value { font-size: 18px; font-weight: 700; color: var(--text-primary); }

/* API Key */
.api-key-display { font-size: 12px; color: var(--warning); }
.text-muted { color: var(--text-muted); font-size: 12px; }

/* 弹窗提示 */
.dialog-hint {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 14px;
  background: rgba(34,197,94,0.06);
  border: 1px solid rgba(34,197,94,0.15);
  border-radius: 8px;
  font-size: 12px; color: var(--text-secondary);
  margin-top: 8px;
}
.dialog-hint .mono { color: var(--success); }

@media (max-width: 1100px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .detail-grid { grid-template-columns: 1fr; }
}
</style>
