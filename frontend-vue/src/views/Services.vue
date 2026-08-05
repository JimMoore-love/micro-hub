<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { serviceApi, type ServiceInfo, type DiscoverResult } from '@/api/platform'

// ==================== 数据 ====================
const services = ref<any[]>([])
const events = ref<any[]>([])
const loading = ref(false)
const discoverLoading = ref(false)
const registerVisible = ref(false)
const discoverResultVisible = ref(false)
const discoverResult = ref<DiscoverResult | null>(null)

// 注册表单
const registerForm = ref({
  name: '',
  type: 'custom',
  port: 0,
  host: '127.0.0.1',
  version: '',
  desc_source: '',
  start_cmd: '',
  dependencies: [] as string[],
})

// 自定义端口输入
const extraPortsInput = ref('')

// ==================== 筛选/搜索 ====================
const searchKeyword = ref('')
const filterStatus = ref('')
const filterType = ref('')
const filterHost = ref('')        // 新增：IP/地址筛选
const filterSource = ref('')      // 新增：来源筛选
const filterNode = ref('')        // 新增：节点筛选
const groupByHost = ref(false)    // 新增：按 IP 分组

// 动态获取筛选选项
const hostOptions = computed(() => {
  const hosts = [...new Set(services.value.map(s => s.host).filter(Boolean))]
  return hosts.sort()
})
const nodeOptions = computed(() => {
  const nodes = [...new Set(services.value.map(s => s.node).filter(Boolean))]
  return nodes.sort()
})

const filteredServices = computed(() => {
  return services.value.filter(s => {
    if (searchKeyword.value && !s.name.toLowerCase().includes(searchKeyword.value.toLowerCase())) return false
    if (filterStatus.value && s.status !== filterStatus.value) return false
    if (filterType.value && s.type !== filterType.value) return false
    if (filterHost.value && s.host !== filterHost.value) return false
    if (filterSource.value && s.source !== filterSource.value) return false
    if (filterNode.value && s.node !== filterNode.value) return false
    return true
  })
})

// 按分组结构化数据
const groupedServices = computed(() => {
  if (!groupByHost.value) return null
  const groups: Record<string, any[]> = {}
  for (const s of filteredServices.value) {
    const key = s.host || 'unknown'
    if (!groups[key]) groups[key] = []
    groups[key].push(s)
  }
  return Object.entries(groups).map(([host, items]) => ({
    host,
    items,
    total: items.length,
    healthy: items.filter((s: any) => s.status === 'healthy').length,
    warning: items.filter((s: any) => s.status === 'warning').length,
    critical: items.filter((s: any) => s.status === 'critical').length,
  }))
})

// 统计卡片
const stats = computed(() => {
  const list = filteredServices.value
  return {
    total: list.length,
    healthy: list.filter(s => s.status === 'healthy').length,
    warning: list.filter(s => s.status === 'warning').length,
    critical: list.filter(s => s.status === 'critical' || s.status === 'unreachable').length,
  }
})

// 清除所有筛选
function clearFilters() {
  searchKeyword.value = ''
  filterStatus.value = ''
  filterType.value = ''
  filterHost.value = ''
  filterSource.value = ''
  filterNode.value = ''
}

const hasActiveFilter = computed(() =>
  !!(searchKeyword.value || filterStatus.value || filterType.value || filterHost.value || filterSource.value || filterNode.value)
)

// ==================== 详情弹窗 ====================
const detailVisible = ref(false)
const detailService = ref<any>(null)
const healthDetail = ref<any>(null)
const healthLoading = ref(false)

async function showDetail(svc: any) {
  detailService.value = svc
  detailVisible.value = true
  healthLoading.value = true
  healthDetail.value = null
  try {
    const data = await serviceApi.health(svc.id)
    healthDetail.value = data
  } catch (e) {
    console.error('Health check failed:', e)
  }
  healthLoading.value = false
}

// ==================== 自动发现 ====================
async function handleDiscover() {
  discoverLoading.value = true
  try {
    const extraPorts: number[] = []
    if (extraPortsInput.value.trim()) {
      for (const part of extraPortsInput.value.split(',')) {
        const range = part.trim().split('-')
        if (range.length === 1) {
          const p = parseInt(range[0])
          if (p > 0) extraPorts.push(p)
        } else if (range.length === 2) {
          const start = parseInt(range[0])
          const end = parseInt(range[1])
          for (let p = start; p <= end && p > 0; p++) extraPorts.push(p)
        }
      }
    }
    const data = await serviceApi.discover(extraPorts)
    discoverResult.value = data
    discoverResultVisible.value = true
    await fetchServices()
    ElMessage.success(`扫描完成：发现 ${data.total_reachable} 个可达服务，新注册 ${data.new_registered.length} 个`)
  } catch (e) {
    ElMessage.error('自动发现失败')
    console.error(e)
  }
  discoverLoading.value = false
}

// ==================== 手动注册 ====================
async function handleRegister() {
  if (!registerForm.value.name || !registerForm.value.port) {
    ElMessage.warning('服务名和端口不能为空')
    return
  }
  loading.value = true
  try {
    const data = await serviceApi.register({
      name: registerForm.value.name,
      type: registerForm.value.type,
      port: registerForm.value.port,
      host: registerForm.value.host || '127.0.0.1',
      version: registerForm.value.version,
      desc_source: registerForm.value.desc_source,
      start_cmd: registerForm.value.start_cmd,
      dependencies: registerForm.value.dependencies,
    } as any)
    registerVisible.value = false
    if (data.port_reachable) {
      ElMessage.success(`服务 ${data.service.name} 注册成功，端口可达`)
    } else {
      ElMessage.warning(`服务 ${data.service.name} 注册成功，但端口不可达（状态标记为 unreachable）`)
    }
    registerForm.value = { name: '', type: 'custom', port: 0, host: '127.0.0.1', version: '', desc_source: '', start_cmd: '', dependencies: [] }
    await fetchServices()
  } catch (e) {
    ElMessage.error('注册失败')
    console.error(e)
  }
  loading.value = false
}

// ==================== 刷新健康状态 ====================
async function handleRefreshHealth() {
  loading.value = true
  try {
    const data = await serviceApi.refreshHealth()
    await fetchServices()
    const changes = data.updates.filter((u: any) => u.old_status !== u.new_status)
    if (changes.length > 0) {
      ElMessage.success(`健康检查完成：${data.checked} 个服务已检查，${changes.length} 个状态变更`)
    } else {
      ElMessage.success(`健康检查完成：${data.checked} 个服务全部正常`)
    }
  } catch (e) {
    ElMessage.error('健康检查失败')
  }
  loading.value = false
}

// ==================== 删除服务 ====================
async function handleDelete(svc: any) {
  try {
    await ElMessageBox.confirm(`确定要删除服务 ${svc.name} (端口 ${svc.port})？`, '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await serviceApi.delete(svc.id)
    ElMessage.success(`已删除 ${svc.name}`)
    await fetchServices()
  } catch {
    // 用户取消
  }
}

// ==================== 编辑服务 ====================
const editVisible = ref(false)
const editForm = ref({
  id: '',
  name: '',
  type: 'custom',
  port: 0,
  host: '127.0.0.1',
  version: '',
  instances: 1,
  desc_source: '',
  start_cmd: '',
  dependencies: [] as string[],
})

function showEdit(svc: any) {
  editForm.value = {
    id: svc.id,
    name: svc.name,
    type: svc.type || 'custom',
    port: svc.port,
    host: svc.host || '127.0.0.1',
    version: svc.version || '',
    instances: svc.instances || 1,
    desc_source: svc.desc_source || '',
    start_cmd: svc.start_cmd || '',
    dependencies: svc.dependencies || [],
  }
  editVisible.value = true
}

async function handleSaveEdit() {
  if (!editForm.value.name) {
    ElMessage.warning('服务名不能为空')
    return
  }
  loading.value = true
  try {
    await serviceApi.update(editForm.value.id, {
      name: editForm.value.name,
      type: editForm.value.type,
      port: editForm.value.port,
      host: editForm.value.host,
      version: editForm.value.version,
      instances: editForm.value.instances,
      desc_source: editForm.value.desc_source,
      start_cmd: editForm.value.start_cmd,
      dependencies: editForm.value.dependencies,
    })
    ElMessage.success(`服务 ${editForm.value.name} 已更新`)
    editVisible.value = false
    await fetchServices()
  } catch (e) {
    ElMessage.error('更新失败')
    console.error(e)
  }
  loading.value = false
}

// ==================== 数据加载 ====================
async function fetchServices() {
  try {
    const [serviceData, eventData] = await Promise.all([
      serviceApi.list(),
      serviceApi.events(),
    ])
    if (serviceData && serviceData.length > 0) {
      services.value = serviceData
    }
    if (eventData && eventData.length > 0) {
      events.value = eventData.map((e: any, idx: number) => ({
        id: `evt-${idx}`,
        serviceName: e.service,
        action: e.action,
        detail: e.detail,
        time: e.time,
      }))
    }
  } catch (e) {
    console.error('Failed to fetch services data:', e)
  }
}

function getTypeTag(type: string) {
  const map: Record<string, string> = { gateway: '', service: 'success', infra: 'warning', observability: 'info', custom: '' }
  return map[type] || 'info'
}

function getTypeLabel(type: string) {
  const map: Record<string, string> = { gateway: '网关', service: '业务服务', infra: '基础设施', observability: '可观测性', custom: '自定义' }
  return map[type] || type
}

function getStatusTag(status: string) {
  const map: Record<string, string> = { healthy: 'success', warning: 'warning', critical: 'danger', unreachable: 'info' }
  return map[status] || 'info'
}

function getStatusLabel(status: string) {
  const map: Record<string, string> = { healthy: '正常', warning: '告警', critical: '故障', unreachable: '不可达' }
  return map[status] || status
}

function getSourceLabel(source: string) {
  const map: Record<string, string> = { seed: '初始', discovered: '自动发现', manual: '手动注册', agent: 'Agent上报', subnet_scan: '网段扫描' }
  return map[source] || source
}

function getSourceTag(source: string) {
  const map: Record<string, string> = { seed: 'info', discovered: 'success', manual: '', agent: 'success', subnet_scan: 'warning' }
  return map[source] || 'info'
}

function getEventIcon(action: string) {
  const map: Record<string, string> = { register: 'CirclePlus', deregister: 'Remove', health_change: 'Warning', config_update: 'Setting' }
  return map[action] || 'InfoFilled'
}

function getEventColor(action: string) {
  const map: Record<string, string> = { register: '#22c55e', deregister: '#94a3b8', health_change: '#f59e0b', config_update: '#3b82f6' }
  return map[action] || '#64748b'
}

function getEventLabel(action: string) {
  const map: Record<string, string> = { register: '服务注册', deregister: '服务注销', health_change: '健康变更', config_update: '配置更新' }
  return map[action] || action
}

function getHealthCheckTag(status: string) {
  if (status === 'healthy') return 'success'
  if (status === 'warning') return 'warning'
  return 'danger'
}

function discoverRowClass({ row }: { row: any }) {
  if (row.match_type === 'conflict') return 'row-conflict'
  if (row.match_type === 'matched') return 'row-matched'
  return ''
}

onMounted(fetchServices)
</script>

<template>
  <div class="services-page">
    <div class="page-header-row">
      <h1 class="page-title">服务管理</h1>
      <div class="header-actions">
        <el-button type="success" :icon="'Plus'" @click="registerVisible = true">注册服务</el-button>
        <el-button type="warning" :icon="'Refresh'" :loading="loading" @click="handleRefreshHealth">刷新健康</el-button>
        <el-button type="primary" :icon="'Compass'" :loading="discoverLoading" @click="handleDiscover">自动发现</el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card" :class="{ active: !filterStatus }" @click="filterStatus = ''">
        <div class="stat-card-value">{{ stats.total }}</div>
        <div class="stat-card-label">全部服务</div>
      </div>
      <div class="stat-card stat-healthy" :class="{ active: filterStatus === 'healthy' }" @click="filterStatus = filterStatus === 'healthy' ? '' : 'healthy'">
        <div class="stat-card-value">{{ stats.healthy }}</div>
        <div class="stat-card-label">正常</div>
      </div>
      <div class="stat-card stat-warning" :class="{ active: filterStatus === 'warning' }" @click="filterStatus = filterStatus === 'warning' ? '' : 'warning'">
        <div class="stat-card-value">{{ stats.warning }}</div>
        <div class="stat-card-label">告警</div>
      </div>
      <div class="stat-card stat-critical" :class="{ active: filterStatus === 'critical' || filterStatus === 'unreachable' }" @click="filterStatus = (filterStatus === 'critical') ? '' : 'critical'">
        <div class="stat-card-value">{{ stats.critical }}</div>
        <div class="stat-card-label">异常</div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-row">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索服务名称..."
          clearable
          class="search-input"
          :prefix-icon="'Search'"
        />
        <el-select v-model="filterHost" placeholder="IP / 地址" clearable filterable class="filter-select-wide">
          <el-option v-for="h in hostOptions" :key="h" :label="h" :value="h" />
        </el-select>
        <el-select v-model="filterStatus" placeholder="健康状态" clearable class="filter-select">
          <el-option label="正常" value="healthy" />
          <el-option label="告警" value="warning" />
          <el-option label="故障" value="critical" />
          <el-option label="不可达" value="unreachable" />
        </el-select>
        <el-select v-model="filterType" placeholder="服务类型" clearable class="filter-select">
          <el-option label="网关" value="gateway" />
          <el-option label="业务服务" value="service" />
          <el-option label="基础设施" value="infra" />
          <el-option label="可观测性" value="observability" />
          <el-option label="自定义" value="custom" />
        </el-select>
        <el-select v-model="filterSource" placeholder="来源" clearable class="filter-select">
          <el-option label="初始" value="seed" />
          <el-option label="自动发现" value="discovered" />
          <el-option label="手动注册" value="manual" />
          <el-option label="Agent上报" value="agent" />
          <el-option label="网段扫描" value="subnet_scan" />
        </el-select>
        <el-select v-if="nodeOptions.length > 0" v-model="filterNode" placeholder="节点" clearable filterable class="filter-select">
          <el-option v-for="n in nodeOptions" :key="n" :label="n" :value="n" />
        </el-select>
        <el-button v-if="hasActiveFilter" link type="info" @click="clearFilters">清除筛选</el-button>
      </div>
      <div class="filter-row second-row">
        <el-switch v-model="groupByHost" active-text="按 IP 分组" inactive-text="平铺列表" />
        <span class="result-count">共 {{ filteredServices.length }} 个服务</span>
        <el-input
          v-model="extraPortsInput"
          placeholder="额外扫描端口: 8896,9001"
          clearable
          style="width: 220px; margin-left: auto"
          :prefix-icon="'Search'"
        />
      </div>
    </div>

    <!-- 服务列表 — 平铺模式 -->
    <div v-if="!groupByHost" class="page-card" style="padding: 0; overflow: hidden">
      <el-table :data="filteredServices" stripe style="width: 100%" :row-style="{ height: '48px' }" v-loading="loading" empty-text="暂无服务数据，点击「自动发现」扫描本机端口">
        <el-table-column prop="name" label="服务名" min-width="160">
          <template #default="{ row }">
            <div class="svc-name-cell">
              <span class="svc-status-dot" :class="row.status"></span>
              <span class="svc-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="host" label="IP 地址" width="130">
          <template #default="{ row }">
            <code class="mono">{{ row.host }}:{{ row.port }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="source" label="来源" width="90">
          <template #default="{ row }">
            <el-tag :type="getSourceTag(row.source)" size="small">{{ getSourceLabel(row.source) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="90">
          <template #default="{ row }">
            <el-tag :type="getTypeTag(row.type)" size="small">{{ getTypeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="node" label="节点" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.node" size="small" :type="row.source === 'subnet_scan' ? 'warning' : row.source === 'agent' ? 'success' : 'info'">
              {{ row.node }}
            </el-tag>
            <span v-else class="text-muted">本机</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="健康状态" width="90">
          <template #default="{ row }">
            <el-tag :type="getStatusTag(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="qps" label="QPS" width="80" align="right">
          <template #default="{ row }">
            <span class="mono" v-if="row.qps > 0">{{ row.qps }}</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="last_checked" label="最后检查" width="150">
          <template #default="{ row }">
            <span class="text-sm">{{ row.last_checked ? row.last_checked.substring(0, 19).replace('T', ' ') : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showDetail(row)">详情</el-button>
            <el-button link type="warning" size="small" @click="showEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 服务列表 — 分组模式 -->
    <div v-else class="grouped-list">
      <div v-for="group in groupedServices" :key="group.host" class="group-block">
        <div class="group-header">
          <code class="mono group-host">{{ group.host }}</code>
          <span class="group-stats">
            共 {{ group.total }} 个 ·
            <span class="gs-healthy">{{ group.healthy }} 正常</span> ·
            <span v-if="group.warning" class="gs-warning">{{ group.warning }} 告警</span>
            <span v-if="group.warning"> · </span>
            <span v-if="group.critical" class="gs-critical">{{ group.critical }} 异常</span>
          </span>
        </div>
        <el-table :data="group.items" stripe size="small" style="width: 100%" :row-style="{ height: '42px' }">
          <el-table-column prop="name" label="服务名" min-width="160">
            <template #default="{ row }">
              <div class="svc-name-cell">
                <span class="svc-status-dot" :class="row.status"></span>
                <span class="svc-name">{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="port" label="端口" width="80">
            <template #default="{ row }">
              <code class="mono">{{ row.port }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="90">
            <template #default="{ row }">
              <el-tag :type="getSourceTag(row.source)" size="small">{{ getSourceLabel(row.source) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="type" label="类型" width="90">
            <template #default="{ row }">
              <el-tag :type="getTypeTag(row.type)" size="small">{{ getTypeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="getStatusTag(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="showDetail(row)">详情</el-button>
              <el-button link type="warning" size="small" @click="showEdit(row)">编辑</el-button>
              <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 服务注册事件时间线 -->
    <div class="page-card">
      <div class="page-card-title">服务注册事件</div>
      <div class="timeline">
        <div v-for="evt in events" :key="evt.id" class="tl-item">
          <div class="tl-dot" :style="{ background: getEventColor(evt.action) }">
            <el-icon :size="12"><component :is="getEventIcon(evt.action)" /></el-icon>
          </div>
          <div class="tl-content">
            <div class="tl-header">
              <el-tag :type="evt.action === 'register' ? 'success' : evt.action === 'deregister' ? 'info' : evt.action === 'health_change' ? 'warning' : ''" size="small">{{ getEventLabel(evt.action) }}</el-tag>
              <span class="tl-service">{{ evt.serviceName }}</span>
            </div>
            <div class="tl-detail">{{ evt.detail }}</div>
            <div class="tl-time">{{ evt.time }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 手动注册弹窗 -->
    <el-dialog v-model="registerVisible" title="注册新服务" width="500px" :close-on-click-modal="false">
      <el-form :model="registerForm" label-width="80px">
        <el-form-item label="服务名" required>
          <el-input v-model="registerForm.name" placeholder="如：FRP Server" />
        </el-form-item>
        <el-form-item label="端口" required>
          <el-input-number v-model="registerForm.port" :min="1" :max="65535" placeholder="如：7000" />
        </el-form-item>
        <el-form-item label="地址">
          <el-input v-model="registerForm.host" placeholder="默认 127.0.0.1" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="registerForm.type">
            <el-option label="网关" value="gateway" />
            <el-option label="业务服务" value="service" />
            <el-option label="基础设施" value="infra" />
            <el-option label="可观测性" value="observability" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="版本">
          <el-input v-model="registerForm.version" placeholder="如：v0.61.0" />
        </el-form-item>
        <el-form-item label="来源">
          <el-input v-model="registerForm.desc_source" placeholder="如：ServBay、Podman、手动安装" />
        </el-form-item>
        <el-form-item label="启动命令">
          <el-input v-model="registerForm.start_cmd" placeholder="如：D:\ServBay\bin\nginx-server.cmd" />
        </el-form-item>
        <el-form-item label="依赖">
          <el-select v-model="registerForm.dependencies" multiple filterable allow-create placeholder="选择或输入依赖服务">
            <el-option v-for="s in services" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="register-tip">
        <el-icon :size="14" color="#f59e0b"><Warning /></el-icon>
        注册时会自动检测端口是否可达。不可达的服务状态会标记为 unreachable。
      </div>
      <template #footer>
        <el-button @click="registerVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="handleRegister">注册</el-button>
      </template>
    </el-dialog>

    <!-- 编辑服务弹窗 -->
    <el-dialog v-model="editVisible" title="编辑服务" width="500px" :close-on-click-modal="false">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="服务名" required>
          <el-input v-model="editForm.name" placeholder="修改服务名称" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editForm.type">
            <el-option label="网关" value="gateway" />
            <el-option label="业务服务" value="service" />
            <el-option label="基础设施" value="infra" />
            <el-option label="可观测性" value="observability" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="editForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="地址">
          <el-input v-model="editForm.host" placeholder="如：127.0.0.1" />
        </el-form-item>
        <el-form-item label="版本">
          <el-input v-model="editForm.version" placeholder="如：v1.2.0" />
        </el-form-item>
        <el-form-item label="实例数">
          <el-input-number v-model="editForm.instances" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="来源">
          <el-input v-model="editForm.desc_source" placeholder="如：ServBay、Podman、手动安装" />
        </el-form-item>
        <el-form-item label="启动命令">
          <el-input v-model="editForm.start_cmd" placeholder="如：D:\ServBay\bin\nginx-server.cmd" />
        </el-form-item>
        <el-form-item label="依赖">
          <el-select v-model="editForm.dependencies" multiple filterable allow-create placeholder="选择或输入依赖服务">
            <el-option v-for="s in services" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="handleSaveEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 自动发现结果弹窗 -->
    <el-dialog v-model="discoverResultVisible" title="自动发现结果" width="700px">
      <template v-if="discoverResult">
        <div class="discover-summary">
          <div class="discover-stat">
            <span class="stat-value">{{ discoverResult.scan_results.length }}</span>
            <span class="stat-label">扫描端口</span>
          </div>
          <div class="discover-stat">
            <span class="stat-value" style="color: var(--success)">{{ discoverResult.total_reachable }}</span>
            <span class="stat-label">可达服务</span>
          </div>
          <div class="discover-stat">
            <span class="stat-value" style="color: var(--accent)">{{ discoverResult.new_registered.length }}</span>
            <span class="stat-label">新注册</span>
          </div>
        </div>

        <div class="discover-table-title">扫描详情</div>
        <el-table :data="discoverResult.scan_results" stripe max-height="380" :row-class-name="discoverRowClass">
          <el-table-column prop="name" label="服务名" width="150">
            <template #default="{ row }">
              <span>{{ row.name }}</span>
              <el-tag v-if="row.match_type === 'conflict'" type="danger" size="small" style="margin-left: 4px">冲突</el-tag>
              <el-tag v-else-if="row.match_type === 'matched'" type="success" size="small" style="margin-left: 4px">匹配</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="port" label="端口" width="65">
            <template #default="{ row }">
              <code class="mono">{{ row.port }}</code>
            </template>
          </el-table-column>
          <el-table-column label="进程名" width="170">
            <template #default="{ row }">
              <div v-if="row.process">
                <code class="mono text-sm">{{ row.process.process_name }}</code>
                <el-tag v-if="row.process.is_system_proc" type="warning" size="small" style="margin-left: 4px">系统</el-tag>
              </div>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
          <el-table-column label="PID" width="65">
            <template #default="{ row }">
              <span class="mono text-sm" v-if="row.process">{{ row.process.pid }}</span>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="70">
            <template #default="{ row }">
              <el-tag :type="getStatusTag(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" min-width="180">
            <template #default="{ row }">
              <span class="text-sm">{{ row.source || '-' }}</span>
            </template>
          </el-table-column>
        </el-table>

        <div v-if="discoverResult.new_registered.length > 0" class="discover-table-title" style="margin-top: 16px">
          新注册的服务
        </div>
        <el-table v-if="discoverResult.new_registered.length > 0" :data="discoverResult.new_registered" stripe max-height="200">
          <el-table-column prop="name" label="服务名" />
          <el-table-column prop="port" label="端口" width="80" />
          <el-table-column prop="status" label="状态" width="80" />
        </el-table>
      </template>
      <template #footer>
        <el-button type="primary" @click="discoverResultVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" :title="detailService?.name" width="560px" :close-on-click-modal="false">
      <template v-if="detailService">
        <div class="detail-grid">
          <div class="detail-item">
            <span class="detail-label">来源</span>
            <el-tag :type="getSourceTag(detailService.source)" size="small">{{ getSourceLabel(detailService.source) }}</el-tag>
          </div>
          <div class="detail-item">
            <span class="detail-label">类型</span>
            <el-tag :type="getTypeTag(detailService.type)" size="small">{{ getTypeLabel(detailService.type) }}</el-tag>
          </div>
          <div class="detail-item">
            <span class="detail-label">地址</span>
            <code class="mono detail-value">{{ detailService.host }}:{{ detailService.port }}</code>
          </div>
          <div class="detail-item">
            <span class="detail-label">版本</span>
            <span class="detail-value">{{ detailService.version || '-' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">实例数</span>
            <span class="detail-value">{{ detailService.instances }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">注册时间</span>
            <span class="detail-value">{{ detailService.registered_at }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">最后检查</span>
            <span class="detail-value">{{ detailService.last_checked ? detailService.last_checked.substring(0, 19).replace('T', ' ') : '-' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">QPS</span>
            <span class="detail-value mono">{{ detailService.qps || '-' }}</span>
          </div>
        </div>

        <!-- 健康检查详情 -->
        <div class="health-section" v-if="healthDetail" style="margin-top: 16px">
          <div class="section-title">
            <span class="status-dot" :class="healthDetail.overall"></span>
            健康检查结果 — 综合：{{ getStatusLabel(healthDetail.overall) }}
          </div>
          <div class="health-checks-list">
            <div v-for="check in healthDetail.checks" :key="check.name" class="hc-item2">
              <div class="hc-header2">
                <span class="hc-dot2" :class="check.status === 'healthy' ? 'passing' : check.status === 'warning' ? 'warning' : 'critical'"></span>
                <span class="hc-name2">{{ check.name }}</span>
                <el-tag :type="getHealthCheckTag(check.status)" size="small">{{ check.status }}</el-tag>
              </div>
              <div class="hc-meta" v-if="check.latency_ms !== undefined">
                <span v-if="check.latency_ms >= 0">延迟: {{ check.latency_ms }}ms</span>
                <span v-if="check.address">{{ check.address }}</span>
                <span v-if="check.url">{{ check.url }}</span>
              </div>
              <div class="hc-time2">{{ check.last_check }}</div>
            </div>
          </div>
        </div>
        <div v-else-if="healthLoading" style="text-align: center; margin-top: 16px">
          <el-icon class="is-loading" :size="20"><Loading /></el-icon> 正在检查...
        </div>
      </template>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button type="danger" @click="handleDelete(detailService); detailVisible = false">删除</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.services-page {
  max-width: 1400px;
}

/* 页头 */
.page-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.header-actions {
  display: flex;
  gap: 8px;
}

/* 统计卡片 */
.stats-row {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.stat-card {
  flex: 1;
  padding: 16px 20px;
  background: rgba(255,255,255,0.04);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
}
.stat-card:hover {
  border-color: var(--accent);
  background: rgba(59, 130, 246, 0.06);
}
.stat-card.active {
  border-color: var(--accent);
  background: rgba(59, 130, 246, 0.1);
}
.stat-card-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}
.stat-card-label {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
}
.stat-healthy .stat-card-value { color: #22c55e; }
.stat-warning .stat-card-value { color: #f59e0b; }
.stat-critical .stat-card-value { color: #ef4444; }
.stat-healthy.active { border-color: #22c55e; background: rgba(34, 197, 94, 0.08); }
.stat-warning.active { border-color: #f59e0b; background: rgba(245, 158, 11, 0.08); }
.stat-critical.active { border-color: #ef4444; background: rgba(239, 68, 68, 0.08); }

/* 筛选栏 */
.filter-bar {
  margin-bottom: 16px;
}
.filter-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.filter-row.second-row {
  padding-top: 2px;
}
.search-input { width: 220px; }
.filter-select { width: 120px; }
.filter-select-wide { width: 150px; }
.result-count {
  font-size: 13px;
  color: var(--text-muted);
  margin-left: 12px;
}

/* 表格 */
.svc-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
.svc-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.svc-status-dot.healthy { background: var(--success); box-shadow: 0 0 6px var(--success); }
.svc-status-dot.warning { background: var(--warning); box-shadow: 0 0 6px var(--warning); }
.svc-status-dot.critical { background: var(--danger); box-shadow: 0 0 6px var(--danger); }
.svc-status-dot.unreachable { background: #64748b; box-shadow: 0 0 4px #64748b; }
.svc-name { font-weight: 600; color: var(--text-primary); }
.mono { font-family: 'Fira Code', 'Cascadia Code', monospace; font-size: 12px; color: var(--text-primary); }
.text-muted { color: var(--text-muted); font-size: 12px; }
.text-sm { font-size: 12px; color: var(--text-secondary); }

/* 分组模式 */
.grouped-list { display: flex; flex-direction: column; gap: 16px; }
.group-block {
  background: rgba(255,255,255,0.02);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  overflow: hidden;
}
.group-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background: rgba(0,0,0,0.15);
  border-bottom: 1px solid var(--border-color);
}
.group-host { font-size: 14px; font-weight: 600; }
.group-stats { font-size: 12px; color: var(--text-secondary); }
.gs-healthy { color: #22c55e; }
.gs-warning { color: #f59e0b; }
.gs-critical { color: #ef4444; }

/* 注册提示 */
.register-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-secondary);
}

/* 发现统计 */
.discover-summary { display: flex; gap: 24px; margin-bottom: 16px; }
.discover-stat { text-align: center; }
.stat-value { font-size: 28px; font-weight: 700; color: var(--text-primary); display: block; }
.stat-label { font-size: 12px; color: var(--text-muted); }
.discover-table-title { font-size: 13px; font-weight: 600; color: var(--text-secondary); margin-bottom: 8px; }

/* 时间线 */
.timeline { position: relative; padding-left: 4px; }
.timeline::before { content: ''; position: absolute; left: 11px; top: 6px; bottom: 6px; width: 1px; background: var(--border-color); }
.tl-item { display: flex; gap: 14px; padding: 10px 0; position: relative; }
.tl-dot { width: 24px; height: 24px; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: #fff; flex-shrink: 0; z-index: 1; }
.tl-content { flex: 1; min-width: 0; }
.tl-header { display: flex; align-items: center; gap: 8px; margin-bottom: 2px; }
.tl-service { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.tl-detail { font-size: 12px; color: var(--text-secondary); margin-bottom: 2px; }
.tl-time { font-size: 11px; color: var(--text-muted); }

/* 详情弹窗 */
.detail-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.detail-item { display: flex; flex-direction: column; gap: 4px; }
.detail-label { font-size: 12px; color: var(--text-muted); }
.detail-value { font-size: 14px; color: var(--text-primary); }

/* 健康检查 */
.health-section { background: rgba(255,255,255,0.03); border: 1px solid var(--border-color); border-radius: var(--radius); padding: 14px; }
.section-title { font-size: 13px; font-weight: 600; color: var(--text-secondary); margin-bottom: 10px; display: flex; align-items: center; gap: 6px; }
.status-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.status-dot.healthy { background: var(--success); box-shadow: 0 0 6px var(--success); }
.status-dot.warning { background: var(--warning); box-shadow: 0 0 6px var(--warning); }
.status-dot.critical { background: var(--danger); box-shadow: 0 0 6px var(--danger); }
.health-checks-list { display: flex; flex-direction: column; gap: 8px; }
.hc-item2 { padding: 8px 10px; background: rgba(0,0,0,0.2); border-radius: 6px; }
.hc-header2 { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.hc-dot2 { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.hc-dot2.passing { background: var(--success); }
.hc-dot2.warning { background: var(--warning); }
.hc-dot2.critical { background: var(--danger); }
.hc-name2 { font-size: 12px; font-weight: 600; color: var(--text-primary); flex: 1; }
.hc-meta { font-size: 11px; color: var(--text-secondary); display: flex; gap: 12px; }
.hc-time2 { font-size: 10px; color: var(--text-muted); }

:deep(.row-conflict) { background-color: rgba(239, 68, 68, 0.08) !important; }
:deep(.row-matched) { background-color: rgba(34, 197, 94, 0.08) !important; }
</style>
