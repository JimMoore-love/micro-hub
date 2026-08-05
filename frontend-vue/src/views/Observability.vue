<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { observabilityApi, serviceApi } from '@/api/platform'
import client from '@/api/client'

// ==================== 数据源状态 ====================
interface DataSourceStatus {
  prometheus: { url: string; reachable: boolean; data_source: string }
  jaeger: { url: string; reachable: boolean; data_source: string }
  loki: { url: string; reachable: boolean; data_source: string }
}

const dataSourceStatus = ref<DataSourceStatus | null>(null)

async function fetchDataSourceStatus() {
  try {
    const resp = await client.get('/observability/datasource-status')
    dataSourceStatus.value = resp.data.data
  } catch (e) {
    console.error('Failed to fetch datasource status:', e)
  }
}

function dataSourceBadge(source: string): string {
  if (source === 'prometheus' || source === 'jaeger' || source === 'loki') return '🟢 真实数据'
  return '🟡 模拟数据'
}

function dataSourceDetail(name: string): string {
  if (!dataSourceStatus.value) return ''
  const ds = dataSourceStatus.value[name as keyof DataSourceStatus]
  if (!ds) return ''
  if (ds.reachable) return `${ds.url} · 真实数据`
  return `${ds.url} · 不可达 → 模拟数据回退`
}

// ==================== Section 1: 实时指标 ====================
const metrics = ref([
  { title: '请求总量', value: '12.5K', unit: '/min', icon: 'DataLine', color: '#3b82f6', bg: 'rgba(59,130,246,0.1)', trend: [60, 75, 65, 90] },
  { title: 'P95 延迟', value: '45', unit: 'ms', icon: 'Timer', color: '#22c55e', bg: 'rgba(34,197,94,0.1)', trend: [40, 55, 48, 45] },
  { title: 'P99 延迟', value: '120', unit: 'ms', icon: 'Timer', color: '#8b5cf6', bg: 'rgba(139,92,246,0.1)', trend: [100, 130, 115, 120] },
  { title: '错误率', value: '0.2', unit: '%', icon: 'WarningFilled', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)', trend: [15, 25, 20, 18] },
  { title: 'AI Token 用量', value: '8.5K', unit: '/min', icon: 'Cpu', color: '#ef4444', bg: 'rgba(239,68,68,0.1)', trend: [50, 70, 65, 85] },
  { title: '活跃连接', value: '156', unit: '', icon: 'Connection', color: '#06b6d4', bg: 'rgba(6,182,212,0.1)', trend: [120, 140, 135, 156] },
])

// ==================== Section 2: 链路追踪 ====================
interface TraceSpan {
  service: string
  duration: number
  offset: number
  status: 'ok' | 'error'
}

interface Trace {
  id: string
  path: string
  tenant: string
  totalLatency: number
  serviceCount: number
  status: 'success' | 'failed'
  spans: TraceSpan[]
}

const traces = ref<Trace[]>([
  {
    id: 'trace-001', path: '/api/v1/users', tenant: 'default', totalLatency: 45, serviceCount: 3, status: 'success',
    spans: [
      { service: 'gateway', duration: 5, offset: 0, status: 'ok' },
      { service: 'user-service', duration: 35, offset: 5, status: 'ok' },
      { service: 'postgres', duration: 5, offset: 40, status: 'ok' },
    ],
  },
  {
    id: 'trace-002', path: '/api/v1/ai/chat', tenant: 'enterprise-a', totalLatency: 320, serviceCount: 5, status: 'success',
    spans: [
      { service: 'gateway', duration: 8, offset: 0, status: 'ok' },
      { service: 'ai-service', duration: 15, offset: 8, status: 'ok' },
      { service: 'openai', duration: 280, offset: 23, status: 'ok' },
      { service: 'redis', duration: 5, offset: 303, status: 'ok' },
      { service: 'postgres', duration: 12, offset: 308, status: 'ok' },
    ],
  },
  {
    id: 'trace-003', path: '/api/v1/proofread', tenant: 'enterprise-b', totalLatency: 150, serviceCount: 4, status: 'success',
    spans: [
      { service: 'gateway', duration: 6, offset: 0, status: 'ok' },
      { service: 'ai-service', duration: 10, offset: 6, status: 'ok' },
      { service: 'proofread-x', duration: 125, offset: 16, status: 'ok' },
      { service: 'redis', duration: 9, offset: 141, status: 'ok' },
    ],
  },
  {
    id: 'trace-004', path: '/api/v1/orders', tenant: 'default', totalLatency: 18, serviceCount: 2, status: 'failed',
    spans: [
      { service: 'gateway', duration: 5, offset: 0, status: 'ok' },
      { service: 'order-service', duration: 13, offset: 5, status: 'error' },
    ],
  },
  {
    id: 'trace-005', path: '/api/v1/users/profile', tenant: 'enterprise-a', totalLatency: 32, serviceCount: 3, status: 'success',
    spans: [
      { service: 'gateway', duration: 4, offset: 0, status: 'ok' },
      { service: 'user-service', duration: 20, offset: 4, status: 'ok' },
      { service: 'redis', duration: 8, offset: 24, status: 'ok' },
    ],
  },
  {
    id: 'trace-006', path: '/api/v1/ai/chat', tenant: 'default', totalLatency: 195, serviceCount: 5, status: 'success',
    spans: [
      { service: 'gateway', duration: 6, offset: 0, status: 'ok' },
      { service: 'ai-service', duration: 12, offset: 6, status: 'ok' },
      { service: 'deepseek', duration: 165, offset: 18, status: 'ok' },
      { service: 'redis', duration: 5, offset: 183, status: 'ok' },
      { service: 'postgres', duration: 7, offset: 188, status: 'ok' },
    ],
  },
  {
    id: 'trace-007', path: '/api/v1/orders/create', tenant: 'enterprise-b', totalLatency: 85, serviceCount: 4, status: 'success',
    spans: [
      { service: 'gateway', duration: 5, offset: 0, status: 'ok' },
      { service: 'order-service', duration: 30, offset: 5, status: 'ok' },
      { service: 'postgres', duration: 40, offset: 35, status: 'ok' },
      { service: 'nats', duration: 10, offset: 75, status: 'ok' },
    ],
  },
  {
    id: 'trace-008', path: '/api/v1/proofread', tenant: 'default', totalLatency: 138, serviceCount: 4, status: 'success',
    spans: [
      { service: 'gateway', duration: 5, offset: 0, status: 'ok' },
      { service: 'ai-service', duration: 8, offset: 5, status: 'ok' },
      { service: 'proofread-x', duration: 115, offset: 13, status: 'ok' },
      { service: 'redis', duration: 10, offset: 128, status: 'ok' },
    ],
  },
  {
    id: 'trace-009', path: '/api/v1/reports', tenant: 'enterprise-a', totalLatency: 210, serviceCount: 3, status: 'success',
    spans: [
      { service: 'gateway', duration: 6, offset: 0, status: 'ok' },
      { service: 'ai-service', duration: 195, offset: 6, status: 'ok' },
      { service: 'postgres', duration: 9, offset: 201, status: 'ok' },
    ],
  },
  {
    id: 'trace-010', path: '/api/v1/auth/login', tenant: 'default', totalLatency: 28, serviceCount: 3, status: 'success',
    spans: [
      { service: 'gateway', duration: 4, offset: 0, status: 'ok' },
      { service: 'user-service', duration: 18, offset: 4, status: 'ok' },
      { service: 'redis', duration: 6, offset: 22, status: 'ok' },
    ],
  },
])

const expandedTraces = ref<Set<string>>(new Set())

function toggleTrace(id: string) {
  if (expandedTraces.value.has(id)) expandedTraces.value.delete(id)
  else expandedTraces.value.add(id)
}

const spanColors: Record<string, string> = {
  gateway: '#3b82f6',
  'user-service': '#22c55e',
  'order-service': '#8b5cf6',
  'ai-service': '#f59e0b',
  postgres: '#06b6d4',
  redis: '#ef4444',
  nats: '#ec4899',
  openai: '#10b981',
  deepseek: '#6366f1',
  'proofread-x': '#f97316',
}

function getSpanColor(service: string): string {
  return spanColors[service] || '#64748b'
}

// ==================== Section 3: 日志搜索 ====================
interface LogEntry {
  time: string
  level: 'ERROR' | 'WARN' | 'INFO'
  service: string
  message: string
  traceId: string
}

const allLogs = ref<LogEntry[]>([
  { time: '14:32:15.123', level: 'INFO', service: 'gateway', message: 'Request received: POST /api/v1/ai/chat from tenant=enterprise-a', traceId: 'trace-002' },
  { time: '14:32:14.890', level: 'INFO', service: 'ai-service', message: 'Routing to provider: openai (quality strategy)', traceId: 'trace-002' },
  { time: '14:32:10.456', level: 'WARN', service: 'ai-service', message: 'OpenAI response latency 280ms exceeds threshold 200ms', traceId: 'trace-002' },
  { time: '14:31:55.234', level: 'ERROR', service: 'order-service', message: 'Database query failed: connection timeout after 5s', traceId: 'trace-004' },
  { time: '14:31:50.012', level: 'INFO', service: 'gateway', message: 'Request received: GET /api/v1/orders from tenant=default', traceId: 'trace-004' },
  { time: '14:30:30.789', level: 'INFO', service: 'ai-service', message: 'Proofread API call completed in 125ms', traceId: 'trace-003' },
  { time: '14:30:25.456', level: 'INFO', service: 'gateway', message: 'Request received: POST /api/v1/proofread from tenant=enterprise-b', traceId: 'trace-003' },
  { time: '14:29:15.123', level: 'WARN', service: 'nats', message: 'High connection latency detected: 45ms', traceId: 'trace-007' },
  { time: '14:28:50.890', level: 'INFO', service: 'user-service', message: 'User profile cache hit: user_id=4521', traceId: 'trace-005' },
  { time: '14:28:30.234', level: 'INFO', service: 'gateway', message: 'JWT validation passed for tenant=enterprise-a', traceId: 'trace-005' },
  { time: '14:27:45.567', level: 'ERROR', service: 'ai-service', message: 'Translate API returned 502 Bad Gateway', traceId: 'trace-009' },
  { time: '14:27:40.123', level: 'INFO', service: 'gateway', message: 'Rate limit check passed: tenant=default, 45/200 QPS', traceId: 'trace-001' },
  { time: '14:26:30.890', level: 'INFO', service: 'user-service', message: 'Redis SET: session:user:8890 (TTL=3600)', traceId: 'trace-001' },
  { time: '14:25:15.456', level: 'WARN', service: 'gateway', message: 'CORS preflight from unknown origin: https://unknown.com', traceId: '' },
  { time: '14:24:50.234', level: 'INFO', service: 'order-service', message: 'Order created: ORD-20240308-0042, amount=$128.50', traceId: 'trace-007' },
])

const logFilters = reactive({
  keyword: '',
  service: '',
  level: '',
})

const filteredLogs = ref<LogEntry[]>(allLogs.value)

function searchLogs() {
  filteredLogs.value = allLogs.value.filter(log => {
    if (logFilters.keyword && !log.message.toLowerCase().includes(logFilters.keyword.toLowerCase())) return false
    if (logFilters.service && log.service !== logFilters.service) return false
    if (logFilters.level && log.level !== logFilters.level) return false
    return true
  })
}

const levelColors: Record<string, string> = { ERROR: '#ef4444', WARN: '#f59e0b', INFO: '#3b82f6' }

// ==================== Section 4: 服务健康热力图 ====================
const healthServices = ref([
  { name: 'Gateway', latency: 28, status: 'healthy' },
  { name: 'User Service', latency: 12, status: 'healthy' },
  { name: 'Order Service', latency: 18, status: 'healthy' },
  { name: 'AI Service', latency: 320, status: 'critical' },
  { name: 'PostgreSQL', latency: 8, status: 'healthy' },
  { name: 'Redis', latency: 3, status: 'healthy' },
  { name: 'Consul', latency: 15, status: 'healthy' },
  { name: 'NATS', latency: 85, status: 'warning' },
  { name: 'MinIO', latency: 22, status: 'healthy' },
  { name: 'Jaeger', latency: 35, status: 'healthy' },
  { name: 'Prometheus', latency: 40, status: 'healthy' },
  { name: 'OpenAI', latency: 320, status: 'warning' },
])

function healthColor(s: string): string {
  if (s === 'healthy') return '#22c55e'
  if (s === 'warning') return '#f59e0b'
  return '#ef4444'
}

function healthLabel(s: string): string {
  if (s === 'healthy') return '健康'
  if (s === 'warning') return '注意'
  return '告警'
}

function formatMetric(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return String(n)
}

onMounted(async () => {
  // 先获取数据源状态
  await fetchDataSourceStatus()

  try {
    const data = await observabilityApi.getMetrics()
    if (data) {
      const trend = data.trend && data.trend.length > 0 ? data.trend : [60, 75, 65, 90]
      metrics.value = [
        { title: '请求总量', value: formatMetric(data.request_count), unit: '/min', icon: 'DataLine', color: '#3b82f6', bg: 'rgba(59,130,246,0.1)', trend },
        { title: 'P95 延迟', value: String(data.p95_latency), unit: 'ms', icon: 'Timer', color: '#22c55e', bg: 'rgba(34,197,94,0.1)', trend },
        { title: 'P99 延迟', value: String(data.p99_latency), unit: 'ms', icon: 'Timer', color: '#8b5cf6', bg: 'rgba(139,92,246,0.1)', trend },
        { title: '错误率', value: String(data.error_rate), unit: '%', icon: 'WarningFilled', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)', trend },
        { title: 'AI Token 用量', value: formatMetric(data.ai_tokens), unit: '/min', icon: 'Cpu', color: '#ef4444', bg: 'rgba(239,68,68,0.1)', trend },
        { title: '活跃连接', value: String(data.active_connections), unit: '', icon: 'Connection', color: '#06b6d4', bg: 'rgba(6,182,212,0.1)', trend },
      ]
    }
  } catch (e) {
    console.error('Failed to fetch metrics:', e)
  }
  try {
    const data = await observabilityApi.listTraces()
    if (data && data.length > 0) {
      traces.value = data.map(t => ({
        id: t.trace_id,
        path: t.path,
        tenant: t.tenant_id,
        totalLatency: t.total_latency,
        serviceCount: t.services,
        status: t.status === 'success' ? 'success' : 'failed',
        spans: t.spans.map(s => ({
          service: s.service,
          duration: s.duration,
          offset: s.start,
          status: 'ok' as const,
        })),
      }))
    }
  } catch (e) {
    console.error('Failed to fetch traces:', e)
  }
  try {
    const data = await observabilityApi.searchLogs({})
    if (data && data.length > 0) {
      allLogs.value = data.map(l => ({
        time: l.timestamp,
        level: l.level,
        service: l.service,
        message: l.message,
        traceId: l.trace_id,
      }))
      searchLogs()
    }
  } catch (e) {
    console.error('Failed to fetch logs:', e)
  }
  try {
    const data = await serviceApi.list()
    if (data && data.length > 0) {
      healthServices.value = data.map(s => ({
        name: s.name,
        latency: s.p95,
        status: s.status,
      }))
    }
  } catch (e) {
    console.error('Failed to fetch service health:', e)
  }
})
</script>

<template>
  <div class="observability-page">
    <h1 class="page-title">可观测性中心</h1>

    <!-- 数据源状态条 -->
    <div class="datasource-bar" v-if="dataSourceStatus">
      <div class="ds-item" :class="{ reachable: dataSourceStatus.prometheus.reachable }">
        <span class="ds-dot"></span>
        <span class="ds-label">指标</span>
        <span class="ds-source">{{ dataSourceBadge(dataSourceStatus.prometheus.data_source) }}</span>
        <span class="ds-detail">{{ dataSourceDetail('prometheus') }}</span>
      </div>
      <div class="ds-item" :class="{ reachable: dataSourceStatus.jaeger.reachable }">
        <span class="ds-dot"></span>
        <span class="ds-label">链路</span>
        <span class="ds-source">{{ dataSourceBadge(dataSourceStatus.jaeger.data_source) }}</span>
        <span class="ds-detail">{{ dataSourceDetail('jaeger') }}</span>
      </div>
      <div class="ds-item" :class="{ reachable: dataSourceStatus.loki.reachable }">
        <span class="ds-dot"></span>
        <span class="ds-label">日志</span>
        <span class="ds-source">{{ dataSourceBadge(dataSourceStatus.loki.data_source) }}</span>
        <span class="ds-detail">{{ dataSourceDetail('loki') }}</span>
      </div>
    </div>

    <!-- ==================== Section 1: 实时指标 ==================== -->
    <div class="section-header">
      <h2 class="section-title"><el-icon><DataLine /></el-icon> 实时指标</h2>
      <span class="section-sub">每分钟采样 · 自动刷新</span>
    </div>
    <div class="metrics-grid">
      <div v-for="m in metrics" :key="m.title" class="metric-card">
        <div class="metric-top">
          <div class="metric-icon" :style="{ background: m.bg, color: m.color }">
            <el-icon :size="20"><component :is="m.icon" /></el-icon>
          </div>
          <div class="metric-info">
            <div class="metric-label">{{ m.title }}</div>
            <div class="metric-value mono">{{ m.value }}<span class="metric-unit">{{ m.unit }}</span></div>
          </div>
        </div>
        <div class="metric-trend">
          <div v-for="(val, idx) in m.trend" :key="idx" class="trend-bar-mini"
            :style="{ height: (val / Math.max(...m.trend) * 100) + '%', background: m.color }"
          ></div>
        </div>
      </div>
    </div>

    <!-- ==================== Section 2: 链路追踪 ==================== -->
    <div class="section-header">
      <h2 class="section-title"><el-icon><Share /></el-icon> 链路追踪</h2>
      <span class="section-sub">最近 10 条 Trace · 点击展开调用链</span>
    </div>
    <div class="page-card" style="padding: 0; overflow: hidden">
      <el-table :data="traces" stripe style="width: 100%" :row-style="{ height: '48px' }" size="small">
        <el-table-column type="expand" width="30">
          <template #default="{ row }">
            <div class="trace-expand">
              <div class="waterfall">
                <div v-for="(span, idx) in row.spans" :key="idx" class="waterfall-row">
                  <span class="wf-service" :style="{ color: getSpanColor(span.service) }">{{ span.service }}</span>
                  <div class="wf-bar-area">
                    <div
                      class="wf-bar"
                      :style="{
                        marginLeft: (span.offset / row.totalLatency * 100) + '%',
                        width: Math.max(span.duration / row.totalLatency * 100, 2) + '%',
                        background: getSpanColor(span.service),
                        opacity: span.status === 'error' ? 1 : 0.7,
                      }"
                      :class="{ error: span.status === 'error' }"
                    >
                      <span class="wf-duration mono">{{ span.duration }}ms</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="id" label="Trace ID" width="120">
          <template #default="{ row }">
            <code class="mono trace-id">{{ row.id }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="请求路径" min-width="180">
          <template #default="{ row }">
            <code class="mono trace-path">{{ row.path }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="tenant" label="租户" width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.tenant }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="totalLatency" label="总延迟" width="100" align="center">
          <template #default="{ row }">
            <span class="mono" :class="{ 'text-warn': row.totalLatency > 200 }">{{ row.totalLatency }}ms</span>
          </template>
        </el-table-column>
        <el-table-column prop="serviceCount" label="服务数" width="90" align="center">
          <template #default="{ row }"><span class="mono">{{ row.serviceCount }}</span></template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90" align="center">
          <template #default="{ row }">
            <span class="trace-status" :class="row.status">
              <span class="ts-dot"></span>
              {{ row.status === 'success' ? '成功' : '失败' }}
            </span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ==================== Section 3: 日志搜索 ==================== -->
    <div class="section-header">
      <h2 class="section-title"><el-icon><Document /></el-icon> 日志搜索</h2>
      <span class="section-sub">多维度筛选 · Trace ID 关联</span>
    </div>
    <div class="page-card">
      <!-- 搜索栏 -->
      <div class="log-search-bar">
        <el-input v-model="logFilters.keyword" placeholder="搜索关键词..." clearable class="log-search-input" @input="searchLogs">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="logFilters.service" placeholder="服务" clearable @change="searchLogs" style="width: 160px">
          <el-option label="gateway" value="gateway" />
          <el-option label="user-service" value="user-service" />
          <el-option label="order-service" value="order-service" />
          <el-option label="ai-service" value="ai-service" />
          <el-option label="nats" value="nats" />
        </el-select>
        <el-select v-model="logFilters.level" placeholder="级别" clearable @change="searchLogs" style="width: 120px">
          <el-option label="ERROR" value="ERROR" />
          <el-option label="WARN" value="WARN" />
          <el-option label="INFO" value="INFO" />
        </el-select>
        <span class="log-count">{{ filteredLogs.length }} 条日志</span>
      </div>

      <!-- 日志列表 -->
      <div class="log-list">
        <div v-for="(log, idx) in filteredLogs" :key="idx" class="log-item">
          <span class="mono log-time">{{ log.time }}</span>
          <span class="log-level" :style="{ color: levelColors[log.level], background: levelColors[log.level] + '15' }">{{ log.level }}</span>
          <span class="log-service">{{ log.service }}</span>
          <span class="log-message">{{ log.message }}</span>
          <a v-if="log.traceId" class="mono log-trace" @click="toggleTrace(log.traceId)">{{ log.traceId }}</a>
        </div>
      </div>
    </div>

    <!-- ==================== Section 4: 服务健康热力图 ==================== -->
    <div class="section-header">
      <h2 class="section-title"><el-icon><Monitor /></el-icon> 服务健康热力图</h2>
      <span class="section-sub">12 个服务 · 实时延迟监控</span>
    </div>
    <div class="page-card">
      <div class="health-legend">
        <div class="legend-item"><span class="legend-dot green"></span> 健康 (&lt;50ms)</div>
        <div class="legend-item"><span class="legend-dot yellow"></span> 注意 (50-200ms)</div>
        <div class="legend-item"><span class="legend-dot red"></span> 告警 (&gt;200ms)</div>
      </div>
      <div class="health-grid">
        <div
          v-for="svc in healthServices"
          :key="svc.name"
          class="health-block"
          :style="{ borderColor: healthColor(svc.status) + '40', background: healthColor(svc.status) + '0d' }"
        >
          <div class="hb-header">
            <span class="hb-dot" :style="{ background: healthColor(svc.status), boxShadow: '0 0 8px ' + healthColor(svc.status) }"></span>
            <span class="hb-name">{{ svc.name }}</span>
          </div>
          <div class="hb-latency">
            <span class="mono hb-value" :style="{ color: healthColor(svc.status) }">{{ svc.latency }}ms</span>
            <span class="hb-status" :style="{ color: healthColor(svc.status) }">{{ healthLabel(svc.status) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.observability-page { max-width: 1400px; }
.mono { font-family: 'Fira Code', 'Cascadia Code', monospace; }

/* 数据源状态条 */
.datasource-bar {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
  padding: 12px 16px;
  background: rgba(0,0,0,0.3);
  border-radius: 10px;
  border: 1px solid rgba(255,255,255,0.08);
}
.ds-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: rgba(0,0,0,0.2);
  border-radius: 6px;
  font-size: 13px;
}
.ds-item.reachable {
  border: 1px solid rgba(34,197,94,0.3);
}
.ds-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #f59e0b;
}
.ds-item.reachable .ds-dot {
  background: #22c55e;
}
.ds-label {
  font-weight: 600;
  color: var(--text-primary);
}
.ds-source {
  color: var(--text-secondary);
}
.ds-detail {
  font-size: 11px;
  color: var(--text-muted);
  margin-left: 4px;
}

/* Section 标题 */
.section-header {
  display: flex; align-items: baseline; gap: 12px;
  margin: 28px 0 16px;
}
.section-header:first-of-type { margin-top: 0; }
.section-title {
  font-size: 18px; font-weight: 700; color: var(--text-primary);
  display: flex; align-items: center; gap: 8px; margin: 0;
}
.section-title .el-icon { color: var(--accent); }
.section-sub { font-size: 12px; color: var(--text-muted); }

/* 实时指标 */
.metrics-grid {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px;
}
.metric-card {
  background: var(--bg-card); border: 1px solid var(--border-color);
  border-radius: var(--radius-lg); padding: 20px;
  display: flex; flex-direction: column; gap: 16px;
  transition: all 0.3s ease;
}
.metric-card:hover { background: var(--bg-card-hover); transform: translateY(-2px); }
.metric-top { display: flex; align-items: center; gap: 14px; }
.metric-icon {
  width: 44px; height: 44px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.metric-info { display: flex; flex-direction: column; gap: 2px; }
.metric-label { font-size: 12px; color: var(--text-muted); }
.metric-value { font-size: 24px; font-weight: 700; color: var(--text-primary); }
.metric-unit { font-size: 12px; color: var(--text-muted); margin-left: 4px; font-weight: 400; }

.metric-trend {
  display: flex; align-items: flex-end; gap: 6px; height: 40px;
}
.trend-bar-mini {
  flex: 1; border-radius: 3px 3px 0 0; min-height: 4px;
  transition: height 0.4s ease;
}

/* 链路追踪 */
.trace-id { font-size: 12px; color: var(--accent); }
.trace-path { font-size: 12px; color: var(--text-secondary); }
.text-warn { color: var(--warning) !important; }

.trace-status {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12px; font-weight: 500;
}
.trace-status.success { color: var(--success); }
.trace-status.failed { color: var(--danger); }
.ts-dot { width: 6px; height: 6px; border-radius: 50%; }
.trace-status.success .ts-dot { background: var(--success); }
.trace-status.failed .ts-dot { background: var(--danger); }

.trace-expand { padding: 16px 24px; background: var(--bg-input); }
.waterfall { display: flex; flex-direction: column; gap: 8px; }
.waterfall-row { display: flex; align-items: center; gap: 12px; }
.wf-service {
  font-size: 12px; font-weight: 600; min-width: 120px; font-family: 'Fira Code', monospace;
}
.wf-bar-area {
  flex: 1; position: relative; height: 24px;
  background: rgba(255,255,255,0.03); border-radius: 4px;
}
.wf-bar {
  height: 100%; border-radius: 4px;
  display: flex; align-items: center; justify-content: center;
  min-width: 30px; position: relative;
}
.wf-bar.error { border: 1px solid var(--danger); }
.wf-duration { font-size: 10px; color: #fff; white-space: nowrap; }

/* 日志搜索 */
.log-search-bar {
  display: flex; align-items: center; gap: 12px; margin-bottom: 16px;
}
.log-search-input { width: 300px; }
.log-count { margin-left: auto; font-size: 12px; color: var(--text-muted); }

.log-list {
  display: flex; flex-direction: column; gap: 2px;
  max-height: 480px; overflow-y: auto;
}
.log-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 12px; border-radius: 4px;
  font-size: 13px; transition: background 0.15s;
}
.log-item:hover { background: rgba(59,130,246,0.06); }
.log-time { font-size: 12px; color: var(--text-muted); white-space: nowrap; min-width: 110px; }
.log-level {
  font-size: 10px; font-weight: 700; padding: 1px 6px; border-radius: 3px; min-width: 50px; text-align: center;
}
.log-service {
  font-size: 12px; color: var(--accent); min-width: 110px;
  font-family: 'Fira Code', monospace;
}
.log-message { flex: 1; color: var(--text-secondary); font-size: 13px; }
.log-trace {
  font-size: 11px; color: var(--warning); cursor: pointer; text-decoration: underline;
  white-space: nowrap;
}
.log-trace:hover { color: var(--accent); }

/* 健康热力图 */
.health-legend {
  display: flex; gap: 20px; margin-bottom: 16px; font-size: 12px; color: var(--text-secondary);
}
.legend-item { display: flex; align-items: center; gap: 6px; }
.legend-dot { width: 10px; height: 10px; border-radius: 3px; }
.legend-dot.green { background: #22c55e; }
.legend-dot.yellow { background: #f59e0b; }
.legend-dot.red { background: #ef4444; }

.health-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px;
}
.health-block {
  padding: 16px; border: 1px solid; border-radius: 10px;
  transition: all 0.25s ease; cursor: pointer;
}
.health-block:hover { transform: translateY(-2px); }
.hb-header { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.hb-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.hb-name { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.hb-latency { display: flex; align-items: baseline; justify-content: space-between; }
.hb-value { font-size: 22px; font-weight: 700; }
.hb-status { font-size: 12px; font-weight: 500; }

@media (max-width: 1100px) {
  .metrics-grid { grid-template-columns: repeat(2, 1fr); }
  .health-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 768px) {
  .metrics-grid { grid-template-columns: 1fr; }
  .health-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
