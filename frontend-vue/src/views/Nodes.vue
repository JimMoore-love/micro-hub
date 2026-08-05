<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { nodeApi, type NodeInfo, type SubnetScanResult } from '@/api/node'

const nodes = ref<NodeInfo[]>([])
const loading = ref(false)
const scanLoading = ref(false)
const scanVisible = ref(false)
const scanResult = ref<SubnetScanResult | null>(null)
const scanForm = ref({ subnet: '', ports: '80,3306,6379,8080,8081,22,9090,16686' })

// 筛选
const searchKeyword = ref('')
const filterOS = ref('')
const filterStatus = ref('')
const filterIP = ref('')

const ipOptions = computed(() => {
  return [...new Set(nodes.value.map(n => n.ip).filter(Boolean))].sort()
})

const filteredNodes = computed(() => {
  return nodes.value.filter(n => {
    if (searchKeyword.value) {
      const kw = searchKeyword.value.toLowerCase()
      if (!n.name?.toLowerCase().includes(kw) && !n.hostname?.toLowerCase().includes(kw)) return false
    }
    if (filterOS.value && n.os !== filterOS.value) return false
    if (filterStatus.value && n.status !== filterStatus.value) return false
    if (filterIP.value && n.ip !== filterIP.value) return false
    return true
  })
})

const stats = computed(() => {
  return {
    total: filteredNodes.value.length,
    online: filteredNodes.value.filter(n => n.status === 'online').length,
    offline: filteredNodes.value.filter(n => n.status !== 'online').length,
    windows: filteredNodes.value.filter(n => n.os === 'windows').length,
    linux: filteredNodes.value.filter(n => n.os === 'linux').length,
  }
})

function clearFilters() {
  searchKeyword.value = ''
  filterOS.value = ''
  filterStatus.value = ''
  filterIP.value = ''
}

const hasActiveFilter = computed(() =>
  !!(searchKeyword.value || filterOS.value || filterStatus.value || filterIP.value)
)

async function fetchNodes() {
  loading.value = true
  try {
    const data = await nodeApi.list()
    nodes.value = data || []
  } catch (e) {
    console.error(e)
  }
  loading.value = false
}

async function handleScanSubnet() {
  if (!scanForm.value.subnet) {
    ElMessage.warning('请输入网段，如 192.168.1.0/24')
    return
  }
  scanLoading.value = true
  try {
    const ports = scanForm.value.ports.split(',').map(p => parseInt(p.trim())).filter(p => p > 0)
    const data = await nodeApi.scanSubnet(scanForm.value.subnet, ports)
    scanResult.value = data
    ElMessage.success(`扫描完成：${data.total} 个可达端口`)
    await fetchNodes()
  } catch (e) {
    ElMessage.error('网段扫描失败')
    console.error(e)
  }
  scanLoading.value = false
}

function getStatusTag(status: string) {
  return status === 'online' ? 'success' : 'danger'
}

function getStatusLabel(status: string) {
  return status === 'online' ? '在线' : '离线'
}

function getOSTag(os: string) {
  if (os === 'windows') return 'Windows'
  if (os === 'linux') return 'Linux'
  return os
}

function formatTime(t: string) {
  if (!t) return '-'
  return t.substring(0, 19).replace('T', ' ')
}

function formatDuration(t: string) {
  if (!t) return '-'
  const diff = Date.now() - new Date(t).getTime()
  const sec = Math.floor(diff / 1000)
  if (sec < 60) return `${sec}秒前`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}分钟前`
  const hr = Math.floor(min / 60)
  return `${hr}小时前`
}

onMounted(fetchNodes)
</script>

<template>
  <div class="nodes-page">
    <div class="page-header-row">
      <h1 class="page-title">节点管理</h1>
      <div class="header-actions">
        <el-button type="primary" :icon="'Connection'" @click="scanVisible = true">网段扫描</el-button>
        <el-button type="warning" :icon="'Refresh'" :loading="loading" @click="fetchNodes">刷新</el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card" :class="{ active: !filterStatus }" @click="filterStatus = ''">
        <div class="stat-card-value">{{ stats.total }}</div>
        <div class="stat-card-label">全部节点</div>
      </div>
      <div class="stat-card stat-online" :class="{ active: filterStatus === 'online' }" @click="filterStatus = filterStatus === 'online' ? '' : 'online'">
        <div class="stat-card-value">{{ stats.online }}</div>
        <div class="stat-card-label">在线</div>
      </div>
      <div class="stat-card stat-offline" :class="{ active: filterStatus && filterStatus !== 'online' }" @click="filterStatus = filterStatus === 'offline' ? '' : 'offline'">
        <div class="stat-card-value">{{ stats.offline }}</div>
        <div class="stat-card-label">离线</div>
      </div>
      <div class="stat-card stat-info" :class="{ active: filterOS === 'windows' }" @click="filterOS = filterOS === 'windows' ? '' : 'windows'">
        <div class="stat-card-value">{{ stats.windows }}</div>
        <div class="stat-card-label">Windows</div>
      </div>
      <div class="stat-card stat-info" :class="{ active: filterOS === 'linux' }" @click="filterOS = filterOS === 'linux' ? '' : 'linux'">
        <div class="stat-card-value">{{ stats.linux }}</div>
        <div class="stat-card-label">Linux</div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索节点名 / 主机名..."
        clearable
        class="search-input"
        :prefix-icon="'Search'"
      />
      <el-select v-model="filterIP" placeholder="IP 地址" clearable filterable class="filter-select">
        <el-option v-for="ip in ipOptions" :key="ip" :label="ip" :value="ip" />
      </el-select>
      <el-select v-model="filterOS" placeholder="操作系统" clearable class="filter-select">
        <el-option label="Windows" value="windows" />
        <el-option label="Linux" value="linux" />
      </el-select>
      <el-select v-model="filterStatus" placeholder="状态" clearable class="filter-select">
        <el-option label="在线" value="online" />
        <el-option label="离线" value="offline" />
      </el-select>
      <el-button v-if="hasActiveFilter" link type="info" @click="clearFilters">清除筛选</el-button>
      <span class="result-count">共 {{ filteredNodes.length }} 个节点</span>
    </div>

    <div class="page-card" style="padding: 0; overflow: hidden">
      <el-table :data="filteredNodes" stripe style="width: 100%" v-loading="loading" empty-text="暂无节点数据，请启动 Agent 或扫描网段">
        <el-table-column prop="name" label="节点名称" min-width="140">
          <template #default="{ row }">
            <div class="svc-name-cell">
              <span class="svc-status-dot" :class="row.status"></span>
              <span class="svc-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP 地址" width="130">
          <template #default="{ row }">
            <code class="mono">{{ row.ip }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="hostname" label="主机名" width="140" />
        <el-table-column prop="os" label="系统" width="90">
          <template #default="{ row }">
            <el-tag size="small">{{ getOSTag(row.os) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="getStatusTag(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="agent_version" label="Agent版本" width="100" />
        <el-table-column prop="service_count" label="服务数" width="80" align="right" />
        <el-table-column prop="last_seen" label="最后心跳" min-width="140">
          <template #default="{ row }">
            <span class="text-sm">{{ formatDuration(row.last_seen) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="page-card">
      <div class="page-card-title">Agent 部署说明</div>
      <div class="deploy-guide">
        <p><strong>1. 编译 Agent（在开发机执行）</strong></p>
        <pre>cd agent
# Windows
go build -o microhub-agent.exe .
# Linux 交叉编译
GOOS=linux GOARCH=amd64 go build -o microhub-agent .</pre>
        <p><strong>2. 分发到远程服务器</strong></p>
        <pre>scp microhub-agent user@server:/opt/microhub/
ssh user@server "chmod +x /opt/microhub/microhub-agent"</pre>
        <p><strong>3. 启动 Agent</strong></p>
        <pre># 基础启动
./microhub-agent -server http://API_SERVER_IP:8081 -name node-1

# 带网段扫描
./microhub-agent -server http://API_SERVER_IP:8081 -name node-1 -subnet 192.168.1.0/24

# 带认证
./microhub-agent -server http://API_SERVER_IP:8081 -token YOUR_JWT_TOKEN

# 自定义端口和间隔
./microhub-agent -server http://API_SERVER_IP:8081 -ports 80,3306,6379,8080 -interval 15</pre>
      </div>
    </div>

    <el-dialog v-model="scanVisible" title="网段扫描" width="600px">
      <el-form label-width="80px">
        <el-form-item label="网段">
          <el-input v-model="scanForm.subnet" placeholder="如：192.168.1.0/24 或 10.0.0.0/24" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input v-model="scanForm.ports" placeholder="逗号分隔端口" />
        </el-form-item>
      </el-form>
      <div class="discover-tip-bar">
        <span>扫描整个 /24 网段约需 1-2 分钟。仅扫描可达端口并自动注册。</span>
      </div>
      <template v-if="scanResult">
        <el-divider />
        <div class="scan-summary">
          <span>扫描 {{ scanResult.scanned_ips }} 个 IP x {{ scanResult.scanned_ports }} 端口，发现 {{ scanResult.total }} 个可达</span>
        </div>
        <el-table :data="scanResult.discovered" stripe max-height="300">
          <el-table-column prop="ip" label="IP" width="130" />
          <el-table-column prop="port" label="端口" width="70" />
          <el-table-column prop="service" label="服务" width="120" />
          <el-table-column prop="latency" label="延迟" width="80">
            <template #default="{ row }">{{ row.latency }}ms</template>
          </el-table-column>
        </el-table>
      </template>
      <template #footer>
        <el-button @click="scanVisible = false">关闭</el-button>
        <el-button type="primary" :loading="scanLoading" @click="handleScanSubnet">开始扫描</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.nodes-page { max-width: 1400px; }
.page-header-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.header-actions { display: flex; gap: 8px; }

/* 统计卡片 */
.stats-row { display: flex; gap: 12px; margin-bottom: 16px; }
.stat-card { flex: 1; padding: 16px 20px; background: rgba(255,255,255,0.04); border: 1px solid var(--border-color); border-radius: 10px; cursor: pointer; transition: all 0.2s; text-align: center; }
.stat-card:hover { border-color: var(--accent); background: rgba(59, 130, 246, 0.06); }
.stat-card.active { border-color: var(--accent); background: rgba(59, 130, 246, 0.1); }
.stat-card-value { font-size: 28px; font-weight: 700; color: var(--text-primary); line-height: 1.2; }
.stat-card-label { font-size: 12px; color: var(--text-muted); margin-top: 4px; }
.stat-online .stat-card-value { color: #22c55e; }
.stat-offline .stat-card-value { color: #ef4444; }
.stat-online.active { border-color: #22c55e; background: rgba(34, 197, 94, 0.08); }
.stat-offline.active { border-color: #ef4444; background: rgba(239, 68, 68, 0.08); }

/* 筛选栏 */
.filter-bar { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; }
.search-input { width: 220px; }
.filter-select { width: 120px; }
.result-count { margin-left: auto; font-size: 13px; color: var(--text-muted); }

/* 表格 */
.svc-name-cell { display: flex; align-items: center; gap: 8px; }
.svc-status-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.svc-status-dot.online { background: #22c55e; box-shadow: 0 0 6px #22c55e; }
.svc-status-dot.offline { background: #ef4444; box-shadow: 0 0 6px #ef4444; }
.svc-name { font-weight: 600; color: var(--text-primary); }
.mono { font-family: 'Fira Code', 'Cascadia Code', monospace; font-size: 12px; color: var(--text-primary); }
.text-sm { font-size: 12px; color: var(--text-secondary); }

.deploy-guide { font-size: 13px; line-height: 1.8; }
.deploy-guide pre { background: rgba(0, 0, 0, 0.05); padding: 12px; border-radius: 8px; font-size: 12px; overflow-x: auto; margin: 8px 0 16px; }
.scan-summary { margin-bottom: 12px; font-size: 13px; color: var(--el-color-primary); }
.discover-tip-bar { padding: 8px 12px; background: rgba(59, 130, 246, 0.08); border-radius: 8px; font-size: 12px; color: #64748b; margin-bottom: 12px; }
</style>
