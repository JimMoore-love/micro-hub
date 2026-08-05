<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, watch, computed } from 'vue'
import { serviceApi } from '@/api/platform'

// ==================== 类型定义 ====================
interface ServiceNode {
  id: string
  name: string
  type: 'gateway' | 'service' | 'infra' | 'observability' | 'custom'
  port: number
  host: string
  status: 'healthy' | 'warning' | 'critical' | 'unreachable'
  version?: string
  qps?: number
  p95?: number
  errorRate?: number
  instances?: number
  registeredAt?: string
  source?: string
  lastChecked?: string
  dependencies?: string[]
}

interface EdgeLine {
  id: string
  from: string
  to: string
  avgLatency: number
  qps: number
}

interface NodePosition {
  x: number
  y: number
}

// ==================== 数据 ====================
const services = ref<ServiceNode[]>([
  { id: 'gateway', name: 'API Gateway', type: 'gateway', port: 8080, status: 'healthy', version: 'v1.2.0', qps: 1250, p95: 45, errorRate: 0.2, instances: 2, registeredAt: '2024-01-15 08:00:00', healthChecks: [{ name: 'HTTP Check', status: 'passing', output: 'HTTP GET /health 200 OK', time: '2s ago' }, { name: 'Memory Usage', status: 'passing', output: '45% used', time: '5s ago' }] },
  { id: 'user-service', name: 'User Service', type: 'service', port: 8081, status: 'healthy', version: 'v1.0.3', qps: 380, p95: 12, errorRate: 0.1, instances: 3, registeredAt: '2024-01-15 08:05:00', healthChecks: [{ name: 'HTTP Check', status: 'passing', output: 'HTTP GET /health 200 OK', time: '1s ago' }, { name: 'DB Connection', status: 'passing', output: 'Connected', time: '3s ago' }] },
  { id: 'order-service', name: 'Order Service', type: 'service', port: 8082, status: 'healthy', version: 'v0.9.1', qps: 220, p95: 18, errorRate: 0.3, instances: 2, registeredAt: '2024-01-15 08:10:00', healthChecks: [{ name: 'HTTP Check', status: 'passing', output: 'HTTP GET /health 200 OK', time: '2s ago' }, { name: 'NATS Connection', status: 'passing', output: 'Connected to nats://localhost:4222', time: '4s ago' }] },
  { id: 'ai-service', name: 'AI Service', type: 'service', port: 8083, status: 'warning', version: 'v0.8.0', qps: 85, p95: 320, errorRate: 1.5, instances: 1, registeredAt: '2024-01-15 08:15:00', healthChecks: [{ name: 'HTTP Check', status: 'warning', output: 'HTTP GET /health 200 OK (slow: 2.1s)', time: '1s ago' }, { name: 'GPU Memory', status: 'passing', output: '68% used', time: '5s ago' }] },
  { id: 'postgres', name: 'PostgreSQL', type: 'infra', port: 5432, status: 'healthy', registeredAt: '2024-01-10 06:00:00', healthChecks: [{ name: 'TCP Check', status: 'passing', output: 'TCP 5432 connected', time: '1s ago' }, { name: 'Replication Lag', status: 'passing', output: '0ms lag', time: '2s ago' }] },
  { id: 'redis', name: 'Redis', type: 'infra', port: 6379, status: 'healthy', registeredAt: '2024-01-10 06:05:00', healthChecks: [{ name: 'TCP Check', status: 'passing', output: 'TCP 6379 connected', time: '1s ago' }, { name: 'Memory Usage', status: 'passing', output: '1.2GB / 4GB', time: '3s ago' }] },
  { id: 'consul', name: 'Consul', type: 'infra', port: 8500, status: 'healthy', registeredAt: '2024-01-10 05:00:00', healthChecks: [{ name: 'HTTP Check', status: 'passing', output: '200 OK', time: '1s ago' }, { name: 'Leader Status', status: 'passing', output: 'Leader elected', time: '5s ago' }] },
  { id: 'nats', name: 'NATS', type: 'infra', port: 4222, status: 'warning', registeredAt: '2024-01-10 06:10:00', healthChecks: [{ name: 'TCP Check', status: 'warning', output: 'TCP 4222 connected (high latency)', time: '1s ago' }, { name: 'Connections', status: 'passing', output: '342 active', time: '2s ago' }] },
  { id: 'minio', name: 'MinIO', type: 'infra', port: 9000, status: 'healthy', registeredAt: '2024-01-10 06:15:00', healthChecks: [{ name: 'HTTP Check', status: 'passing', output: '200 OK', time: '1s ago' }, { name: 'Disk Usage', status: 'passing', output: '45GB / 200GB', time: '4s ago' }] },
  { id: 'jaeger', name: 'Jaeger', type: 'observability', port: 16686, status: 'healthy', registeredAt: '2024-01-12 09:00:00', healthChecks: [{ name: 'HTTP Check', status: 'passing', output: '200 OK', time: '1s ago' }, { name: 'Storage', status: 'passing', output: 'Elasticsearch connected', time: '3s ago' }] },
  { id: 'prometheus', name: 'Prometheus', type: 'observability', port: 9090, status: 'healthy', registeredAt: '2024-01-12 09:05:00', healthChecks: [{ name: 'HTTP Check', status: 'passing', output: '200 OK', time: '1s ago' }, { name: 'Targets', status: 'passing', output: '12/12 up', time: '5s ago' }] },
])

const edges = ref<EdgeLine[]>([
  { id: 'e1', from: 'gateway', to: 'user-service', avgLatency: 8, qps: 380 },
  { id: 'e2', from: 'gateway', to: 'order-service', avgLatency: 12, qps: 220 },
  { id: 'e3', from: 'gateway', to: 'ai-service', avgLatency: 280, qps: 85 },
  { id: 'e4', from: 'user-service', to: 'postgres', avgLatency: 3, qps: 150 },
  { id: 'e5', from: 'user-service', to: 'redis', avgLatency: 1, qps: 520 },
  { id: 'e6', from: 'user-service', to: 'consul', avgLatency: 5, qps: 10 },
  { id: 'e7', from: 'order-service', to: 'postgres', avgLatency: 4, qps: 180 },
  { id: 'e8', from: 'order-service', to: 'redis', avgLatency: 1, qps: 340 },
  { id: 'e9', from: 'order-service', to: 'nats', avgLatency: 6, qps: 80 },
  { id: 'e10', from: 'ai-service', to: 'redis', avgLatency: 2, qps: 200 },
  { id: 'e11', from: 'ai-service', to: 'consul', avgLatency: 8, qps: 10 },
  { id: 'e12', from: 'ai-service', to: 'nats', avgLatency: 15, qps: 45 },
  { id: 'e13', from: 'user-service', to: 'jaeger', avgLatency: 2, qps: 50 },
  { id: 'e14', from: 'order-service', to: 'jaeger', avgLatency: 2, qps: 40 },
  { id: 'e15', from: 'ai-service', to: 'jaeger', avgLatency: 3, qps: 30 },
  { id: 'e16', from: 'gateway', to: 'jaeger', avgLatency: 3, qps: 60 },
])

// 初始节点位置
const nodePositions = reactive<Record<string, NodePosition>>({
  'gateway': { x: 480, y: 70 },
  'user-service': { x: 200, y: 220 },
  'order-service': { x: 480, y: 220 },
  'ai-service': { x: 760, y: 220 },
  'postgres': { x: 100, y: 400 },
  'redis': { x: 330, y: 400 },
  'consul': { x: 500, y: 400 },
  'nats': { x: 670, y: 400 },
  'minio': { x: 860, y: 400 },
  'jaeger': { x: 830, y: 70 },
  'prometheus': { x: 970, y: 70 },
})

// ==================== 交互状态 ====================
const selectedNode = ref<ServiceNode | null>(null)
const hoveredEdge = ref<EdgeLine | null>(null)
const hoveredEdgePos = ref({ x: 0, y: 0 })

// 拖拽
const dragging = ref<string | null>(null)
const dragOffset = ref({ x: 0, y: 0 })
const svgContainer = ref<HTMLElement | null>(null)

// 流量动画 - 小点在连线上的位置 (0-1)
const flowPositions = reactive<Record<string, number>>({})
const frameId = ref(0)
const timeRef = ref(0)

// ==================== 方法 ====================
function getNodeById(id: string): ServiceNode | undefined {
  return services.value.find(s => s.id === id)
}

function getNodeColor(type: string): string {
  const map: Record<string, string> = 'url(#grad-gateway)'
  if (type === 'gateway') return 'url(#grad-gateway)'
  if (type === 'service') return 'url(#grad-service)'
  if (type === 'infra') return 'url(#grad-infra)'
  if (type === 'observability') return 'url(#grad-observability)'
  return '#555'
}

function getNodeSolidColor(type: string): string {
  const map: Record<string, string> = { gateway: '#6366f1', service: '#22c55e', infra: '#f59e0b', observability: '#06b6d4' }
  return map[type] || '#555'
}

function getStatusIcon(status: string): string {
  if (status === 'healthy') return '\u2713'
  if (status === 'warning') return '\u26A0'
  return '\u2717'
}

function getStatusColor(status: string): string {
  if (status === 'healthy') return 'var(--success)'
  if (status === 'warning') return 'var(--warning)'
  if (status === 'critical') return 'var(--danger)'
  if (status === 'unreachable') return '#64748b'
  return 'var(--danger)'
}

function selectNode(node: ServiceNode) {
  selectedNode.value = node
}

function closePanel() {
  selectedNode.value = null
}

function getUpstreamServices(nodeId: string): ServiceNode[] {
  return edges.value
    .filter(e => e.to === nodeId)
    .map(e => getNodeById(e.from))
    .filter(Boolean) as ServiceNode[]
}

function getDownstreamServices(nodeId: string): ServiceNode[] {
  return edges.value
    .filter(e => e.from === nodeId)
    .map(e => getNodeById(e.to))
    .filter(Boolean) as ServiceNode[]
}

// ==================== 拖拽逻辑 ====================
function onNodeMouseDown(e: MouseEvent, nodeId: string) {
  if (e.button !== 0) return
  e.stopPropagation()
  dragging.value = nodeId
  const pos = nodePositions[nodeId]
  const rect = (e.currentTarget as SVGGElement).closest('svg')!.getBoundingClientRect()
  dragOffset.value = {
    x: e.clientX - rect.left - pos.x,
    y: e.clientY - rect.top - pos.y,
  }
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function onMouseMove(e: MouseEvent) {
  if (!dragging.value || !svgContainer.value) return
  const rect = svgContainer.value.getBoundingClientRect()
  nodePositions[dragging.value] = {
    x: Math.max(20, Math.min(rect.width - 100, e.clientX - rect.left - dragOffset.value.x)),
    y: Math.max(20, Math.min(rect.height - 60, e.clientY - rect.top - dragOffset.value.y)),
  }
}

function onMouseUp() {
  dragging.value = null
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
}

// ==================== 连线hover ====================
function onEdgeMouseEnter(e: MouseEvent, edge: EdgeLine) {
  hoveredEdge.value = edge
  const rect = (e.currentTarget as SVGElement).closest('svg')!.getBoundingClientRect()
  hoveredEdgePos.value = { x: e.clientX - rect.left + 12, y: e.clientY - rect.top - 10 }
}

function onEdgeMouseLeave() {
  hoveredEdge.value = null
}

// ==================== 流量动画循环 ====================
function animateFlow() {
  timeRef.value += 0.003
  edges.value.forEach(edge => {
    flowPositions[edge.id] = (Math.sin(timeRef.value * (1 + edge.id.charCodeAt(1) * 0.1)) + 1) / 2
  })
  frameId.value = requestAnimationFrame(animateFlow)
}

onMounted(async () => {
  frameId.value = requestAnimationFrame(animateFlow)
  try {
    const data = await serviceApi.list()
    if (data && data.length > 0) {
      services.value = data.map(s => ({
        id: s.id,
        name: s.name,
        type: s.type,
        port: s.port,
        host: s.host || '127.0.0.1',
        status: s.status,
        version: s.version,
        qps: s.qps,
        p95: s.p95,
        errorRate: s.error_rate,
        instances: s.instances,
        registeredAt: s.registered_at,
        source: s.source,
        lastChecked: s.last_checked,
        dependencies: s.dependencies,
      }))
      // Generate edges from dependencies
      const newEdges: EdgeLine[] = []
      data.forEach(svc => {
        if (svc.dependencies) {
          svc.dependencies.forEach(dep => {
            newEdges.push({
              id: `e-${svc.id}-${dep}`,
              from: dep,
              to: svc.id,
              avgLatency: 0,
              qps: 0,
            })
          })
        }
      })
      if (newEdges.length > 0) {
        edges.value = newEdges
      }
      // Ensure all nodes have positions
      services.value.forEach((svc, idx) => {
        if (!nodePositions[svc.id]) {
          const col = idx % 4
          const row = Math.floor(idx / 4)
          nodePositions[svc.id] = { x: 100 + col * 250, y: 100 + row * 180 }
        }
      })
    }
  } catch (e) {
    console.error('Failed to fetch services:', e)
  }
})

onUnmounted(() => {
  cancelAnimationFrame(frameId.value)
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
})
</script>

<template>
  <div class="topology-page">
    <h1 class="page-title">服务拓扑</h1>

    <div class="topology-layout">
      <!-- 左侧：拓扑图 -->
      <div class="topo-graph-panel">
        <div class="graph-toolbar">
          <span class="graph-subtitle">Service Mesh Topology</span>
          <div class="legend">
            <span class="legend-item"><span class="legend-dot gateway"></span>Gateway</span>
            <span class="legend-item"><span class="legend-dot service"></span>Service</span>
            <span class="legend-item"><span class="legend-dot infra"></span>Infrastructure</span>
            <span class="legend-item"><span class="legend-dot observability"></span>Observability</span>
          </div>
        </div>

        <div class="svg-wrapper" ref="svgContainer">
          <svg width="100%" height="100%" viewBox="0 0 1100 520" preserveAspectRatio="xMidYMid meet">
            <defs>
              <!-- 渐变 -->
              <linearGradient id="grad-gateway" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#6366f1" />
                <stop offset="70%" stop-color="#3b82f6" />
                <stop offset="100%" stop-color="#8b5cf6" />
              </linearGradient>
              <linearGradient id="grad-service" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#22c55e" />
                <stop offset="100%" stop-color="#16a34a" />
              </linearGradient>
              <linearGradient id="grad-infra" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#f59e0b" />
                <stop offset="100%" stop-color="#d97706" />
              </linearGradient>
              <linearGradient id="grad-observability" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#06b6d4" />
                <stop offset="100%" stop-color="#0891b2" />
              </linearGradient>
              <!-- 发光 -->
              <filter id="glow">
                <feGaussianBlur stdDeviation="3" result="blur" />
                <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
              </filter>
              <!-- 图案 -->
              <pattern id="grid" width="30" height="30" patternUnits="userSpaceOnUse">
                <path d="M 30 0 L 0 0 0 30" fill="none" stroke="rgba(255,255,255,0.025)" stroke-width="0.5" />
              </pattern>
            </defs>

            <!-- 背景网格 -->
            <rect width="1100" height="520" fill="url(#grid)" />

            <!-- 连线 -->
            <g v-for="edge in edges" :key="edge.id">
              <!-- 连线本体 -->
              <line
                :x1="nodePositions[edge.from].x + 55"
                :y1="nodePositions[edge.from].y + 35"
                :x2="nodePositions[edge.to].x + 55"
                :y2="nodePositions[edge.to].y + 35"
                :stroke="hoveredEdge?.id === edge.id ? 'rgba(255,255,255,0.5)' : 'rgba(255,255,255,0.12)'"
                :stroke-width="hoveredEdge?.id === edge.id ? 2.5 : 1.5"
                stroke-dasharray="6 3"
                class="edge-line"
                @mouseenter="(e: MouseEvent) => onEdgeMouseEnter(e, edge)"
                @mouseleave="onEdgeMouseLeave"
              />
              <!-- 流流动画小点 -->
              <circle
                :cx="nodePositions[edge.from].x + 55 + (nodePositions[edge.to].x - nodePositions[edge.from].x) * (flowPositions[edge.id] || 0)"
                :cy="nodePositions[edge.from].y + 35 + (nodePositions[edge.to].y - nodePositions[edge.from].y) * (flowPositions[edge.id] || 0)"
                r="3"
                :fill="getNodeSolidColor(getNodeById(edge.from)?.type || 'service')"
                opacity="0.8"
              />
            </g>

            <!-- 节点 -->
            <g
              v-for="svc in services"
              :key="svc.id"
              :transform="`translate(${nodePositions[svc.id].x}, ${nodePositions[svc.id].y})`"
              class="topo-node"
              :class="{ selected: selectedNode?.id === svc.id }"
              @mousedown="(e: MouseEvent) => onNodeMouseDown(e, svc.id)"
              @click.stop="selectNode(svc)"
              style="cursor: grab;"
            >
              <!-- 节点主体 -->
              <rect
                x="0" y="0"
                width="110" height="70"
                rx="12" ry="12"
                :fill="getNodeColor(svc.type)"
                :opacity="selectedNode?.id === svc.id ? 0.25 : 0.15"
                :stroke="selectedNode?.id === svc.id ? getNodeSolidColor(svc.type) : 'transparent'"
                :stroke-width="selectedNode?.id === svc.id ? 1.5 : 0"
              />
              <!-- 状态指示点 -->
              <circle :cx="10" :cy="10" r="5" :fill="getStatusColor(svc.status)" />
              <!-- 服务名 -->
              <text x="55" y="28" text-anchor="middle" fill="var(--text-primary)" font-size="11" font-weight="600">{{ svc.name }}</text>
              <!-- 端口 -->
              <text x="55" y="44" text-anchor="middle" fill="var(--text-muted)" font-size="10">{{ ':' + svc.port }}</text>
              <!-- QPS / Latency -->
              <text x="55" y="60" text-anchor="middle" fill="var(--text-secondary)" font-size="9" v-if="svc.qps">
                {{ svc.qps }} QPS / {{ svc.p95 }}ms
              </text>
            </g>
          </svg>

          <!-- 连线hover提示 -->
          <div
            v-if="hoveredEdge"
            class="edge-tooltip"
            :style="{ left: hoveredEdgePos.x + 'px', top: hoveredEdgePos.y + 'px' }"
          >
            <div class="tt-row">
              <span>{{ getNodeById(hoveredEdge.from)?.name }}</span>
              <span class="tt-arrow">&rarr;</span>
              <span>{{ getNodeById(hoveredEdge.to)?.name }}</span>
            </div>
            <div class="tt-meta">
              <span>延迟: {{ hoveredEdge.avgLatency }}ms</span>
              <span>QPS: {{ hoveredEdge.qps }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧：详情面板 -->
      <div class="detail-panel" :class="{ open: selectedNode }">
        <template v-if="selectedNode">
          <div class="panel-header">
            <div class="panel-service-name">
              <span class="status-dot" :class="selectedNode.status"></span>
              <span class="panel-name">{{ selectedNode.name }}</span>
              <el-tag
                :type="selectedNode.status === 'healthy' ? 'success' : selectedNode.status === 'warning' ? 'warning' : 'danger'"
                size="small"
              >
                {{ selectedNode.status === 'healthy' ? '正常' : selectedNode.status === 'warning' ? '告警' : '故障' }}
              </el-tag>
            </div>
            <el-button :icon="'Close'" text size="small" @click="closePanel" />
          </div>

          <div class="panel-body">
            <!-- 基本信息 -->
            <div class="info-card">
              <div class="info-card-title">基本信息</div>
              <div class="info-grid">
                <div class="info-item">
                  <span class="info-label">类型</span>
                  <el-tag size="small" :type="selectedNode.type === 'gateway' ? '' : selectedNode.type === 'service' ? 'success' : selectedNode.type === 'infra' ? 'warning' : 'info'">
                    {{ selectedNode.type }}
                  </el-tag>
                </div>
                <div class="info-item"><span class="info-label">版本</span><span class="info-value">{{ selectedNode.version || '-' }}</span></div>
                <div class="info-item"><span class="info-label">端口</span><span class="info-value mono">{{ selectedNode.port }}</span></div>
                <div class="info-item"><span class="info-label">实例数</span><span class="info-value mono">{{ selectedNode.instances || '-' }}</span></div>
                <div class="info-item"><span class="info-label">注册时间</span><span class="info-value">{{ selectedNode.registeredAt || '-' }}</span></div>
              </div>
            </div>

            <!-- 实时指标 -->
            <div class="info-card" v-if="selectedNode.qps !== undefined">
              <div class="info-card-title">实时指标</div>
              <div class="metrics-grid">
                <div class="metric-item">
                  <div class="metric-value mono" style="color: var(--accent)">{{ selectedNode.qps?.toLocaleString() }}</div>
                  <div class="metric-label">QPS</div>
                </div>
                <div class="metric-item">
                  <div class="metric-value mono" :style="{ color: (selectedNode.p95 || 0) > 100 ? 'var(--warning)' : 'var(--success)' }">{{ selectedNode.p95 }}ms</div>
                  <div class="metric-label">P95 延迟</div>
                </div>
                <div class="metric-item">
                  <div class="metric-value mono" :style="{ color: (selectedNode.errorRate || 0) > 1 ? 'var(--danger)' : 'var(--success)' }">{{ selectedNode.errorRate }}%</div>
                  <div class="metric-label">错误率</div>
                </div>
              </div>
            </div>

            <!-- 依赖关系 -->
            <div class="info-card">
              <div class="info-card-title">依赖关系</div>
              <div class="dep-section">
                <div class="dep-title">上游服务</div>
                <div class="dep-list" v-if="getUpstreamServices(selectedNode.id).length">
                  <span v-for="up in getUpstreamServices(selectedNode.id)" :key="up.id" class="dep-tag" :style="{ borderColor: getNodeSolidColor(up.type) }">
                    {{ up.name }}
                  </span>
                </div>
                <span v-else class="no-data">无</span>
              </div>
              <div class="dep-section">
                <div class="dep-title">下游服务</div>
                <div class="dep-list" v-if="getDownstreamServices(selectedNode.id).length">
                  <span v-for="down in getDownstreamServices(selectedNode.id)" :key="down.id" class="dep-tag" :style="{ borderColor: getNodeSolidColor(down.type) }">
                    {{ down.name }}
                  </span>
                </div>
                <span v-else class="no-data">无</span>
              </div>
            </div>

            <!-- 健康检查 -->
            <div class="info-card">
              <div class="info-card-title">TCP 健康检查</div>
              <div class="hc-summary" v-if="selectedNode.lastChecked">
                <div class="hc-row">
                  <span class="hc-dot" :class="selectedNode.status === 'healthy' ? 'passing' : selectedNode.status === 'warning' ? 'warning' : 'critical'"></span>
                  <span>端口 {{ selectedNode.host }}:{{ selectedNode.port }} — {{ selectedNode.status === 'healthy' ? '可达' : selectedNode.status === 'warning' ? '告警' : selectedNode.status === 'critical' ? '不可达' : '未知' }}</span>
                </div>
                <div class="hc-row">
                  <span class="hc-label">最后检查:</span>
                  <span>{{ selectedNode.lastChecked ? selectedNode.lastChecked.substring(0, 19).replace('T', ' ') : '-' }}</span>
                </div>
                <div class="hc-row">
                  <span class="hc-label">来源:</span>
                  <span>{{ selectedNode.source === 'seed' ? '初始数据' : selectedNode.source === 'discovered' ? '自动发现' : selectedNode.source === 'manual' ? '手动注册' : selectedNode.source }}</span>
                </div>
              </div>
              <span v-else class="no-data">未检查</span>
            </div>
          </div>
        </template>

        <div v-else class="panel-empty">
          <div class="empty-icon">
            <svg width="60" height="60" viewBox="0 0 60 60" fill="none">
              <rect x="10" y="10" width="40" height="40" rx="8" stroke="rgba(255,255,255,0.1)" stroke-width="1.5" stroke-dasharray="4 2" />
              <circle cx="30" cy="30" r="6" stroke="rgba(255,255,255,0.15)" stroke-width="1" fill="none" />
              <line x1="18" y1="42" x2="28" y2="32" stroke="rgba(255,255,255,0.08)" stroke-width="1" />
              <line x1="42" y1="42" x2="32" y2="32" stroke="rgba(255,255,255,0.08)" stroke-width="1" />
            </svg>
          </div>
          <p>点击左侧节点<br/>查看服务详情</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.topology-page {
  height: calc(100vh - 100px);
  display: flex;
  flex-direction: column;
}

.topology-layout {
  display: flex;
  gap: 16px;
  flex: 1;
  min-height: 0;
}

/* ===== 拓扑图区域 ===== */
.topo-graph-panel {
  flex: 7;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.graph-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-bottom: 1px solid var(--border-color);
}

.graph-subtitle {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.legend {
  display: flex;
  gap: 18px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
}

.legend-dot.gateway { background: linear-gradient(135deg, #6366f1, #3b82f6); }
.legend-dot.service { background: linear-gradient(135deg, #22c55e, #16a34a); }
.legend-dot.infra { background: #f59e0b; }
.legend-dot.observability { background: #06b6d4; }

.svg-wrapper {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.svg-wrapper svg {
  width: 100%;
  height: 100%;
}

.topo-node {
  transition: filter 0.2s ease;
}

.topo-node:hover {
  filter: url(#glow);
}

.topo-node.selected rect {
  filter: url(#glow);
}

.edge-line {
  transition: all 0.2s ease;
  cursor: pointer;
}

/* 连线tooltip */
.edge-tooltip {
  position: absolute;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius);
  padding: 10px 14px;
  pointer-events: none;
  z-index: 10;
  box-shadow: 0 8px 24px rgba(0,0,0,0.4);
}

.tt-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.tt-arrow {
  color: var(--accent);
  font-weight: 700;
}

.tt-meta {
  display: flex;
  gap: 12px;
  font-size: 11px;
  color: var(--text-secondary);
}

/* ===== 右侧详情面板 ===== */
.detail-panel {
  flex: 3;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 320px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.panel-service-name {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.healthy { background: var(--success); box-shadow: 0 0 8px var(--success); }
.status-dot.warning { background: var(--warning); box-shadow: 0 0 8px var(--warning); }
.status-dot.critical { background: var(--danger); box-shadow: 0 0 8px var(--danger); }

.panel-name {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  gap: 16px;
}

.empty-icon {
  opacity: 0.4;
}

.panel-empty p {
  font-size: 14px;
  text-align: center;
  line-height: 1.6;
}

/* 信息卡片 */
.info-card {
  background: rgba(255,255,255,0.03);
  border: 1px solid var(--border-color);
  border-radius: var(--radius);
  padding: 14px;
}

.info-card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.info-card-title::before {
  content: '';
  width: 3px;
  height: 14px;
  background: var(--accent);
  border-radius: 1.5px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-label {
  font-size: 11px;
  color: var(--text-muted);
}

.info-value {
  font-size: 13px;
  color: var(--text-primary);
}

.mono {
  font-family: 'Fira Code', 'Cascadia Code', monospace;
}

/* 指标 */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.metric-item {
  text-align: center;
  padding: 10px 6px;
  background: rgba(255,255,255,0.02);
  border-radius: var(--radius);
}

.metric-value {
  font-size: 20px;
  font-weight: 700;
}

.metric-label {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
}

/* 依赖 */
.dep-section {
  margin-bottom: 10px;
}

.dep-section:last-child {
  margin-bottom: 0;
}

.dep-title {
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 6px;
}

.dep-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.dep-tag {
  font-size: 11px;
  padding: 3px 10px;
  border: 1px solid;
  border-radius: 12px;
  color: var(--text-primary);
}

.no-data {
  font-size: 12px;
  color: var(--text-muted);
  font-style: italic;
}

/* 健康检查 */
.hc-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.hc-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-primary);
}

.hc-label {
  color: var(--text-muted);
  min-width: 70px;
}

.hc-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.hc-dot.passing { background: var(--success); }
.hc-dot.warning { background: var(--warning); }
.hc-dot.critical { background: var(--danger); }

@media (max-width: 1200px) {
  .topology-layout {
    flex-direction: column;
  }
  .topo-graph-panel { flex: 1; min-height: 400px; }
  .detail-panel { flex: none; max-height: 400px; min-width: 0; }
}
</style>
