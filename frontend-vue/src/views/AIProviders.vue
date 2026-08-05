<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { aiProviderApi } from '@/api/platform'

// ==================== 供应商数据 ====================
interface Provider {
  id: string
  name: string
  type: 'llm' | 'proofread' | 'translate' | 'image'
  icon: string
  status: 'connected' | 'disconnected' | 'testing'
  requests: number
  latency: number
  costPer1k: number
  models: string[]
  apiKey: string
  endpoint: string
  healthCheckUrl: string
  lastError: string
  todayCost: number
  monthCost: number
  inputTokens: number
  outputTokens: number
  quota: number
  healthChecks: HealthCheck[]
  errorTrend: ErrorDay[]
  rateLimitHits: number
  rateLimit429: number
}

interface HealthCheck { time: string; status: 'ok' | 'fail'; latency: number }
interface ErrorDay { date: string; errors: number; rate: number }

const providers = ref<Provider[]>([
  {
    id: 'openai', name: 'OpenAI', type: 'llm', icon: '🤖', status: 'connected',
    requests: 850, latency: 320, costPer1k: 0.03, models: ['gpt-4o', 'gpt-4o-mini'],
    apiKey: 'sk-proj-aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890', endpoint: 'https://api.openai.com/v1',
    healthCheckUrl: 'https://api.openai.com/v1/models', lastError: '',
    todayCost: 25.5, monthCost: 680.2, inputTokens: 320000, outputTokens: 180000, quota: 1000000,
    healthChecks: [
      { time: '14:30:01', status: 'ok', latency: 310 },
      { time: '14:25:01', status: 'ok', latency: 295 },
      { time: '14:20:01', status: 'ok', latency: 340 },
      { time: '14:15:01', status: 'ok', latency: 280 },
      { time: '14:10:01', status: 'fail', latency: 0 },
    ],
    errorTrend: [
      { date: '03-02', errors: 2, rate: 0.3 }, { date: '03-03', errors: 0, rate: 0 },
      { date: '03-04', errors: 5, rate: 0.8 }, { date: '03-05', errors: 1, rate: 0.1 },
      { date: '03-06', errors: 0, rate: 0 }, { date: '03-07', errors: 3, rate: 0.4 },
      { date: '03-08', errors: 0, rate: 0 },
    ],
    rateLimitHits: 3, rateLimit429: 2,
  },
  {
    id: 'claude', name: 'Claude', type: 'llm', icon: '🧠', status: 'connected',
    requests: 220, latency: 450, costPer1k: 0.015, models: ['claude-3.5-sonnet'],
    apiKey: 'sk-ant-api03-xYzAbCdEfGhIjKlMnOpQrStUvWxYz0987654321', endpoint: 'https://api.anthropic.com/v1',
    healthCheckUrl: 'https://api.anthropic.com/v1/models', lastError: '',
    todayCost: 6.6, monthCost: 180.5, inputTokens: 120000, outputTokens: 80000, quota: 500000,
    healthChecks: [
      { time: '14:30:00', status: 'ok', latency: 440 },
      { time: '14:25:00', status: 'ok', latency: 460 },
      { time: '14:20:00', status: 'ok', latency: 430 },
      { time: '14:15:00', status: 'ok', latency: 450 },
      { time: '14:10:00', status: 'ok', latency: 420 },
    ],
    errorTrend: [
      { date: '03-02', errors: 0, rate: 0 }, { date: '03-03', errors: 1, rate: 0.5 },
      { date: '03-04', errors: 0, rate: 0 }, { date: '03-05', errors: 2, rate: 1.0 },
      { date: '03-06', errors: 0, rate: 0 }, { date: '03-07', errors: 0, rate: 0 },
      { date: '03-08', errors: 0, rate: 0 },
    ],
    rateLimitHits: 1, rateLimit429: 1,
  },
  {
    id: 'deepseek', name: 'DeepSeek', type: 'llm', icon: '🔍', status: 'connected',
    requests: 580, latency: 180, costPer1k: 0.001, models: ['deepseek-chat', 'deepseek-coder'],
    apiKey: 'sk-ds-aB1cD2eF3gH4iJ5kL6mN7oP8qR9sT0u', endpoint: 'https://api.deepseek.com/v1',
    healthCheckUrl: 'https://api.deepseek.com/v1/models', lastError: '',
    todayCost: 1.74, monthCost: 45.2, inputTokens: 580000, outputTokens: 320000, quota: 2000000,
    healthChecks: [
      { time: '14:30:02', status: 'ok', latency: 170 },
      { time: '14:25:02', status: 'ok', latency: 185 },
      { time: '14:20:02', status: 'ok', latency: 165 },
      { time: '14:15:02', status: 'ok', latency: 190 },
      { time: '14:10:02', status: 'ok', latency: 175 },
    ],
    errorTrend: [
      { date: '03-02', errors: 0, rate: 0 }, { date: '03-03', errors: 0, rate: 0 },
      { date: '03-04', errors: 0, rate: 0 }, { date: '03-05', errors: 1, rate: 0.2 },
      { date: '03-06', errors: 0, rate: 0 }, { date: '03-07', errors: 0, rate: 0 },
      { date: '03-08', errors: 0, rate: 0 },
    ],
    rateLimitHits: 0, rateLimit429: 0,
  },
  {
    id: 'proofread-x', name: '校对厂商X', type: 'proofread', icon: '✍️', status: 'connected',
    requests: 340, latency: 150, costPer1k: 0.005, models: ['proofread-v2', 'grammar-check-v1'],
    apiKey: 'pk-pr-XmNbVcXzLkJhGfDsAwQeRtYuIoP123456', endpoint: 'https://api.proofread-x.com/v2',
    healthCheckUrl: 'https://api.proofread-x.com/v2/health', lastError: '',
    todayCost: 1.7, monthCost: 42.3, inputTokens: 180000, outputTokens: 90000, quota: 500000,
    healthChecks: [
      { time: '14:30:03', status: 'ok', latency: 145 },
      { time: '14:25:03', status: 'ok', latency: 155 },
      { time: '14:20:03', status: 'ok', latency: 140 },
      { time: '14:15:03', status: 'ok', latency: 160 },
      { time: '14:10:03', status: 'ok', latency: 150 },
    ],
    errorTrend: [
      { date: '03-02', errors: 1, rate: 0.3 }, { date: '03-03', errors: 0, rate: 0 },
      { date: '03-04', errors: 0, rate: 0 }, { date: '03-05', errors: 3, rate: 1.5 },
      { date: '03-06', errors: 0, rate: 0 }, { date: '03-07', errors: 1, rate: 0.3 },
      { date: '03-08', errors: 0, rate: 0 },
    ],
    rateLimitHits: 2, rateLimit429: 1,
  },
  {
    id: 'translate-y', name: '翻译服务Y', type: 'translate', icon: '🌐', status: 'testing',
    requests: 45, latency: 200, costPer1k: 0.002, models: ['translate-en-zh'],
    apiKey: '', endpoint: 'https://api.translate-y.com/v1',
    healthCheckUrl: 'https://api.translate-y.com/v1/health', lastError: 'API Key 未配置',
    todayCost: 0.09, monthCost: 2.1, inputTokens: 20000, outputTokens: 12000, quota: 100000,
    healthChecks: [
      { time: '14:30:05', status: 'fail', latency: 0 },
      { time: '14:25:05', status: 'ok', latency: 210 },
      { time: '14:20:05', status: 'fail', latency: 0 },
      { time: '14:15:05', status: 'ok', latency: 195 },
      { time: '14:10:05', status: 'fail', latency: 0 },
    ],
    errorTrend: [
      { date: '03-02', errors: 0, rate: 0 }, { date: '03-03', errors: 5, rate: 15 },
      { date: '03-04', errors: 8, rate: 20 }, { date: '03-05', errors: 3, rate: 8 },
      { date: '03-06', errors: 6, rate: 12 }, { date: '03-07', errors: 4, rate: 10 },
      { date: '03-08', errors: 2, rate: 5 },
    ],
    rateLimitHits: 5, rateLimit429: 3,
  },
])

const selectedProvider = ref<Provider | null>(providers.value[0])
const detailTab = ref('info')

function selectProvider(p: Provider) {
  selectedProvider.value = p
  aiProviderApi.getProvider(p.id).then(detail => {
    if (detail) {
      p.name = detail.name
      p.type = detail.type
      p.icon = detail.icon || p.icon
      p.status = detail.status
      p.requests = detail.requests
      p.latency = detail.latency
      p.costPer1k = detail.cost_per_1k
      p.models = detail.models
      p.apiKey = detail.api_key
      p.endpoint = detail.endpoint
    }
  }).catch(e => console.error('Failed to fetch provider detail:', e))
}

// ==================== 类型/状态标签 ====================
const typeLabels: Record<string, string> = { llm: 'LLM', proofread: '校对', translate: '翻译', image: '图像' }
const typeColors: Record<string, string> = { llm: '#3b82f6', proofread: '#f59e0b', translate: '#22c55e', image: '#8b5cf6' }

const statusLabels: Record<string, string> = { connected: '已连接', disconnected: '未连接', testing: '测试中' }
const statusColors: Record<string, string> = { connected: '#22c55e', disconnected: '#ef4444', testing: '#f59e0b' }

// ==================== 路由策略 ====================
interface RouteRule {
  id: string
  condition: string
  target: string
  strategy: string
  enabled: boolean
}

const routeRules = ref<RouteRule[]>([
  { id: 'rr1', condition: '租户 default → (默认)', target: 'DeepSeek', strategy: '成本优先', enabled: true },
  { id: 'rr2', condition: '租户 enterprise-a → (高质量)', target: 'OpenAI', strategy: '质量优先', enabled: true },
  { id: 'rr3', condition: '租户 enterprise-b → (均衡)', target: 'Claude', strategy: '均衡模式', enabled: true },
  { id: 'rr4', condition: '请求类型 = 校对', target: '校对厂商X', strategy: '类型路由', enabled: true },
  { id: 'rr5', condition: '请求类型 = 翻译', target: '翻译服务Y', strategy: '能力路由', enabled: false },
])

const routingPriority = [
  { level: 1, label: '按租户配置', desc: '优先匹配租户级供应商绑定' },
  { level: 2, label: '按请求类型', desc: '校对/翻译/LLM 类型分发' },
  { level: 3, label: '按成本', desc: '选择单价最低的可用供应商' },
  { level: 4, label: '按延迟', desc: '选择响应最快的可用供应商' },
]

function toggleRule(rule: RouteRule) {
  rule.enabled = !rule.enabled
  ElMessage.success(`路由规则已${rule.enabled ? '启用' : '禁用'}`)
}

// ==================== API Key 显示/隐藏 ====================
const showApiKey = ref(false)
function copyApiKey() {
  if (selectedProvider.value?.apiKey) {
    navigator.clipboard?.writeText(selectedProvider.value.apiKey)
    ElMessage.success('API Key 已复制')
  }
}

// ==================== 用量趋势 (mock 8小时) ====================
const requestTrend = computed(() => {
  if (!selectedProvider.value) return []
  const base = selectedProvider.value.requests
  return [0.4, 0.6, 0.8, 0.5, 0.7, 0.9, 0.85, 1.0].map((ratio, i) => ({
    hour: `${8 + i}h`,
    value: Math.round(base * ratio),
  }))
})

// ==================== 租户用量分布 ====================
const tenantUsage = computed(() => {
  if (!selectedProvider.value) return []
  return [
    { tenant: 'default', count: Math.round(selectedProvider.value.requests * 0.3), color: '#3b82f6' },
    { tenant: 'enterprise-a', count: Math.round(selectedProvider.value.requests * 0.45), color: '#8b5cf6' },
    { tenant: 'enterprise-b', count: Math.round(selectedProvider.value.requests * 0.2), color: '#22c55e' },
    { tenant: 'test-org', count: Math.round(selectedProvider.value.requests * 0.05), color: '#f59e0b' },
  ]
})

const totalTenantUsage = computed(() => tenantUsage.value.reduce((sum, t) => sum + t.count, 0))

// ==================== 编辑供应商弹窗 ====================
const editDialogVisible = ref(false)
const editForm = reactive({ name: '', type: 'llm', endpoint: '', apiKey: '', healthCheckUrl: '' })

function openEdit() {
  if (!selectedProvider.value) return
  Object.assign(editForm, {
    name: selectedProvider.value.name,
    type: selectedProvider.value.type,
    endpoint: selectedProvider.value.endpoint,
    apiKey: selectedProvider.value.apiKey,
    healthCheckUrl: selectedProvider.value.healthCheckUrl,
  })
  editDialogVisible.value = true
}

function saveProvider() {
  if (selectedProvider.value) {
    selectedProvider.value.name = editForm.name
    selectedProvider.value.type = editForm.type as any
    selectedProvider.value.endpoint = editForm.endpoint
    selectedProvider.value.apiKey = editForm.apiKey
    selectedProvider.value.healthCheckUrl = editForm.healthCheckUrl
  }
  editDialogVisible.value = false
  ElMessage.success('供应商配置已保存')
}

// ==================== 费用预估 ====================
function estimateMonthCost(p: Provider): number {
  return Math.round(p.todayCost * 30 * 100) / 100
}

function quotaPercent(p: Provider): number {
  return Math.round(((p.inputTokens + p.outputTokens) / p.quota) * 100)
}

onMounted(async () => {
  try {
    const data = await aiProviderApi.listProviders()
    if (data && data.length > 0) {
      const oldProviders = [...providers.value]
      providers.value = data.map(p => {
        const fb = oldProviders.find(fp => fp.id === p.id)
        return {
          id: p.id,
          name: p.name,
          type: p.type,
          icon: p.icon || fb?.icon || '🤖',
          status: p.status,
          requests: p.requests,
          latency: p.latency,
          costPer1k: p.cost_per_1k,
          models: p.models,
          apiKey: p.api_key,
          endpoint: p.endpoint,
          healthCheckUrl: fb?.healthCheckUrl || '',
          lastError: fb?.lastError || '',
          todayCost: fb?.todayCost || 0,
          monthCost: fb?.monthCost || 0,
          inputTokens: fb?.inputTokens || 0,
          outputTokens: fb?.outputTokens || 0,
          quota: fb?.quota || 0,
          healthChecks: fb?.healthChecks || [],
          errorTrend: fb?.errorTrend || [],
          rateLimitHits: fb?.rateLimitHits || 0,
          rateLimit429: fb?.rateLimit429 || 0,
        }
      })
      selectedProvider.value = providers.value[0]
    }
  } catch (e) {
    console.error('Failed to fetch AI providers:', e)
  }
})

watch(detailTab, async (tab) => {
  if (!selectedProvider.value) return
  if (tab === 'routing') {
    try {
      const data = await aiProviderApi.listRoutingRules()
      if (data && data.length > 0) {
        routeRules.value = data.map(r => {
          const provider = providers.value.find(p => p.id === r.provider_id)
          return {
            id: r.id,
            condition: r.condition || `租户 ${r.tenant_id}`,
            target: provider?.name || r.provider_id,
            strategy: r.priority <= 1 ? '成本优先' : '质量优先',
            enabled: r.enabled,
          }
        })
      }
    } catch (e) {
      console.error('Failed to fetch routing rules:', e)
    }
  } else if (tab === 'usage') {
    try {
      const data = await aiProviderApi.getProviderUsage(selectedProvider.value.id)
      if (data) {
        selectedProvider.value.inputTokens = data.input_tokens
        selectedProvider.value.outputTokens = data.output_tokens
        selectedProvider.value.todayCost = data.cost
        selectedProvider.value.requests = data.request_count
      }
    } catch (e) {
      console.error('Failed to fetch provider usage:', e)
    }
  } else if (tab === 'health') {
    try {
      const data = await aiProviderApi.healthCheck(selectedProvider.value.id)
      if (data && data.checks && data.checks.length > 0) {
        selectedProvider.value.healthChecks = data.checks.map(c => ({
          time: c.time,
          status: c.status === 'ok' ? 'ok' : 'fail',
          latency: c.latency,
        }))
      }
    } catch (e) {
      console.error('Failed to fetch provider health:', e)
    }
  }
})
</script>

<template>
  <div class="providers-page">
    <h1 class="page-title">AI 供应商接入管理</h1>

    <div class="providers-layout">
      <!-- 左侧供应商列表 -->
      <div class="provider-list-panel">
        <div class="panel-header">
          <span class="panel-title">供应商列表</span>
          <el-button type="primary" size="small" class="btn-gradient" @click="editDialogVisible = true; Object.assign(editForm, { name: '', type: 'llm', endpoint: '', apiKey: '', healthCheckUrl: '' })">
            <el-icon><Plus /></el-icon>
          </el-button>
        </div>

        <div class="provider-cards">
          <div
            v-for="p in providers"
            :key="p.id"
            class="provider-card"
            :class="{ active: selectedProvider?.id === p.id }"
            @click="selectProvider(p)"
          >
            <div class="pc-header">
              <span class="pc-icon">{{ p.icon }}</span>
              <div class="pc-name-area">
                <span class="pc-name">{{ p.name }}</span>
                <span class="pc-type" :style="{ color: typeColors[p.type], background: typeColors[p.type] + '15' }">{{ typeLabels[p.type] }}</span>
              </div>
              <span class="pc-status-dot" :style="{ background: statusColors[p.status] }"></span>
            </div>
            <div class="pc-metrics">
              <div class="pc-metric">
                <span class="pc-metric-label">今日请求</span>
                <span class="mono pc-metric-value">{{ p.requests }}</span>
              </div>
              <div class="pc-metric">
                <span class="pc-metric-label">平均延迟</span>
                <span class="mono pc-metric-value">{{ p.latency }}ms</span>
              </div>
            </div>
            <div class="pc-status-text" :style="{ color: statusColors[p.status] }">{{ statusLabels[p.status] }}</div>
          </div>
        </div>
      </div>

      <!-- 右侧详情面板 -->
      <div class="provider-detail-panel">
        <template v-if="selectedProvider">
          <el-tabs v-model="detailTab" class="detail-tabs">
            <!-- Tab 1: 基本信息 -->
            <el-tab-pane label="基本信息" name="info">
              <div class="detail-grid">
                <div class="page-card">
                  <div class="page-card-title">连接信息</div>
                  <div class="info-rows">
                    <div class="info-row">
                      <span class="info-label">供应商名称</span>
                      <span class="info-value">{{ selectedProvider.icon }} {{ selectedProvider.name }}</span>
                    </div>
                    <div class="info-row">
                      <span class="info-label">类型</span>
                      <el-tag size="small" :style="{ color: typeColors[selectedProvider.type], background: typeColors[selectedProvider.type] + '15', border: 'none' }">{{ typeLabels[selectedProvider.type] }}</el-tag>
                    </div>
                    <div class="info-row">
                      <span class="info-label">连接状态</span>
                      <span class="status-badge" :style="{ color: statusColors[selectedProvider.status], background: statusColors[selectedProvider.status] + '15' }">
                        <span class="sb-dot" :style="{ background: statusColors[selectedProvider.status] }"></span>
                        {{ statusLabels[selectedProvider.status] }}
                      </span>
                    </div>
                    <div class="info-row">
                      <span class="info-label">Endpoint URL</span>
                      <code class="mono info-code">{{ selectedProvider.endpoint }}</code>
                    </div>
                    <div class="info-row">
                      <span class="info-label">健康检查URL</span>
                      <code class="mono info-code">{{ selectedProvider.healthCheckUrl }}</code>
                    </div>
                  </div>
                </div>

                <div class="page-card">
                  <div class="page-card-title">API Key</div>
                  <div class="api-key-area">
                    <div class="api-key-display">
                      <code class="mono">{{ showApiKey ? selectedProvider.apiKey : (selectedProvider.apiKey ? selectedProvider.apiKey.slice(0, 8) + '••••••••••••••••••••' : '未配置') }}</code>
                    </div>
                    <div class="api-key-actions">
                      <el-button size="small" @click="showApiKey = !showApiKey">
                        <el-icon><component :is="showApiKey ? 'Hide' : 'View'" /></el-icon>
                        {{ showApiKey ? '隐藏' : '显示' }}
                      </el-button>
                      <el-button size="small" type="primary" @click="copyApiKey" :disabled="!selectedProvider.apiKey">
                        <el-icon><CopyDocument /></el-icon> 复制
                      </el-button>
                    </div>
                  </div>
                  <div class="info-row" v-if="selectedProvider.lastError">
                    <span class="info-label">最近错误</span>
                    <span class="error-text">{{ selectedProvider.lastError }}</span>
                  </div>
                  <div class="info-row">
                    <span class="info-label">支持模型</span>
                    <div class="model-tags">
                      <el-tag v-for="m in selectedProvider.models" :key="m" size="small" type="info">{{ m }}</el-tag>
                    </div>
                  </div>
                  <el-button type="primary" size="small" class="btn-gradient" @click="openEdit" style="margin-top: 8px">
                    <el-icon><Edit /></el-icon> 编辑供应商
                  </el-button>
                </div>
              </div>
            </el-tab-pane>

            <!-- Tab 2: 路由策略 -->
            <el-tab-pane label="路由策略" name="routing">
              <div class="page-card">
                <div class="page-card-title">路由优先级</div>
                <div class="priority-flow">
                  <div v-for="p in routingPriority" :key="p.level" class="priority-step">
                    <div class="ps-num">{{ p.level }}</div>
                    <div class="ps-content">
                      <span class="ps-label">{{ p.label }}</span>
                      <span class="ps-desc">{{ p.desc }}</span>
                    </div>
                    <el-icon v-if="p.level < 4" class="ps-arrow"><ArrowRight /></el-icon>
                  </div>
                </div>
              </div>

              <div class="page-card">
                <div class="page-card-title">路由规则列表</div>
                <div class="route-rule-list">
                  <div v-for="rule in routeRules" :key="rule.id" class="route-rule-item">
                    <div class="rr-left">
                      <el-switch v-model="rule.enabled" size="small" @change="toggleRule(rule)" />
                    </div>
                    <div class="rr-condition">
                      <code class="mono">{{ rule.condition }}</code>
                    </div>
                    <el-icon class="rr-arrow"><Right /></el-icon>
                    <div class="rr-target">
                      <span class="rr-target-name">{{ rule.target }}</span>
                      <el-tag size="small" type="info">{{ rule.strategy }}</el-tag>
                    </div>
                  </div>
                </div>
                <div class="routing-hint">
                  <el-icon><InfoFilled /></el-icon>
                  <span>AI网关按优先级依次匹配路由规则，命中后即分发请求到对应供应商。未命中任何规则时走默认路由。</span>
                </div>
              </div>
            </el-tab-pane>

            <!-- Tab 3: 用量与计费 -->
            <el-tab-pane label="用量与计费" name="usage">
              <div class="detail-grid">
                <!-- 请求趋势 -->
                <div class="page-card">
                  <div class="page-card-title">今日请求趋势</div>
                  <div class="trend-chart">
                    <div v-for="item in requestTrend" :key="item.hour" class="trend-bar-col">
                      <div class="trend-bar" :style="{ height: (item.value / (selectedProvider!.requests) * 100) + '%', background: 'linear-gradient(180deg, ' + typeColors[selectedProvider!.type] + ', ' + typeColors[selectedProvider!.type] + '30)' }"></div>
                      <span class="trend-label">{{ item.hour }}</span>
                    </div>
                  </div>
                </div>

                <!-- Token用量 -->
                <div class="page-card">
                  <div class="page-card-title">Token 用量</div>
                  <div class="token-stats">
                    <div class="token-stat-item">
                      <span class="ts-label">输入 Tokens</span>
                      <span class="mono ts-value" style="color: #3b82f6">{{ selectedProvider.inputTokens.toLocaleString() }}</span>
                    </div>
                    <div class="token-stat-item">
                      <span class="ts-label">输出 Tokens</span>
                      <span class="mono ts-value" style="color: #22c55e">{{ selectedProvider.outputTokens.toLocaleString() }}</span>
                    </div>
                    <div class="token-stat-item total">
                      <span class="ts-label">总计</span>
                      <span class="mono ts-value">{{ (selectedProvider.inputTokens + selectedProvider.outputTokens).toLocaleString() }}</span>
                    </div>
                  </div>
                  <div class="quota-area">
                    <div class="quota-header">
                      <span class="quota-label">配额使用</span>
                      <span class="mono quota-vals">{{ (selectedProvider.inputTokens + selectedProvider.outputTokens).toLocaleString() }} / {{ selectedProvider.quota.toLocaleString() }}</span>
                    </div>
                    <el-progress :percentage="quotaPercent(selectedProvider)" :stroke-width="10" :color="quotaPercent(selectedProvider) > 80 ? '#ef4444' : '#3b82f6'" />
                  </div>
                </div>
              </div>

              <!-- 费用统计 -->
              <div class="page-card">
                <div class="page-card-title">费用统计</div>
                <div class="cost-grid">
                  <div class="cost-item">
                    <span class="cost-label">今日费用</span>
                    <span class="mono cost-value">${{ selectedProvider.todayCost.toFixed(2) }}</span>
                  </div>
                  <div class="cost-item">
                    <span class="cost-label">本月累计</span>
                    <span class="mono cost-value">${{ selectedProvider.monthCost.toFixed(2) }}</span>
                  </div>
                  <div class="cost-item">
                    <span class="cost-label">预估月费</span>
                    <span class="mono cost-value warn">${{ estimateMonthCost(selectedProvider).toFixed(2) }}</span>
                  </div>
                  <div class="cost-item">
                    <span class="cost-label">单价/1K tokens</span>
                    <span class="mono cost-value">${{ selectedProvider.costPer1k.toFixed(3) }}</span>
                  </div>
                </div>
              </div>

              <!-- 租户分布 -->
              <div class="page-card">
                <div class="page-card-title">租户使用分布</div>
                <div class="tenant-usage-bar">
                  <div
                    v-for="t in tenantUsage"
                    :key="t.tenant"
                    class="tub-segment"
                    :style="{ width: (t.count / totalTenantUsage * 100) + '%', background: t.color }"
                  ></div>
                </div>
                <div class="tenant-usage-legend">
                  <div v-for="t in tenantUsage" :key="t.tenant" class="tul-item">
                    <span class="tul-dot" :style="{ background: t.color }"></span>
                    <span class="tul-name">{{ t.tenant }}</span>
                    <span class="mono tul-count">{{ t.count }}</span>
                  </div>
                </div>
              </div>
            </el-tab-pane>

            <!-- Tab 4: 健康监测 -->
            <el-tab-pane label="健康监测" name="health">
              <div class="detail-grid">
                <!-- 健康检查 -->
                <div class="page-card">
                  <div class="page-card-title">最近5次健康检查</div>
                  <div class="health-check-list">
                    <div v-for="(hc, idx) in selectedProvider.healthChecks" :key="idx" class="hc-item">
                      <span class="mono hc-time">{{ hc.time }}</span>
                      <span class="hc-status" :class="hc.status">
                        <span class="hc-dot"></span>
                        {{ hc.status === 'ok' ? '正常' : '失败' }}
                      </span>
                      <span class="mono hc-latency" :class="{ 'text-muted': hc.status === 'fail' }">
                        {{ hc.status === 'ok' ? hc.latency + 'ms' : '-' }}
                      </span>
                    </div>
                  </div>
                </div>

                <!-- 限流记录 -->
                <div class="page-card">
                  <div class="page-card-title">限流记录</div>
                  <div class="ratelimit-stats">
                    <div class="rl-stat">
                      <span class="rl-label">被限流次数</span>
                      <span class="mono rl-value" :class="{ danger: selectedProvider.rateLimitHits > 0 }">{{ selectedProvider.rateLimitHits }}</span>
                    </div>
                    <div class="rl-stat">
                      <span class="rl-label">429返回</span>
                      <span class="mono rl-value" :class="{ danger: selectedProvider.rateLimit429 > 0 }">{{ selectedProvider.rateLimit429 }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 错误率趋势 -->
              <div class="page-card">
                <div class="page-card-title">错误率趋势 (近7天)</div>
                <div class="error-trend-chart">
                  <div v-for="day in selectedProvider.errorTrend" :key="day.date" class="et-col">
                    <div class="et-bar-area">
                      <div
                        class="et-bar"
                        :style="{ height: Math.max(day.rate / 20 * 100, 2) + '%', background: day.errors === 0 ? '#22c55e40' : day.rate > 10 ? '#ef4444' : '#f59e0b' }"
                      ></div>
                    </div>
                    <span class="et-label">{{ day.date }}</span>
                    <span class="mono et-value" :class="{ danger: day.errors > 0 }">{{ day.errors }}</span>
                  </div>
                </div>
              </div>
            </el-tab-pane>
          </el-tabs>
        </template>
      </div>
    </div>

    <!-- 编辑供应商弹窗 -->
    <el-dialog v-model="editDialogVisible" title="编辑供应商" width="520px" :close-on-click-modal="false">
      <el-form :model="editForm" label-width="120px" size="default">
        <el-form-item label="供应商名称">
          <el-input v-model="editForm.name" placeholder="如: OpenAI" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editForm.type" style="width: 100%">
            <el-option label="LLM 大语言模型" value="llm" />
            <el-option label="校对服务" value="proofread" />
            <el-option label="翻译服务" value="translate" />
            <el-option label="图像生成" value="image" />
          </el-select>
        </el-form-item>
        <el-form-item label="Endpoint URL">
          <el-input v-model="editForm.endpoint" placeholder="https://api.example.com/v1" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="editForm.apiKey" placeholder="sk-xxx..." show-password />
        </el-form-item>
        <el-form-item label="健康检查URL">
          <el-input v-model="editForm.healthCheckUrl" placeholder="https://api.example.com/v1/health" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveProvider">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.providers-page { max-width: 1400px; }
.mono { font-family: 'Fira Code', 'Cascadia Code', monospace; }

.providers-layout {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

/* 左侧列表 */
.provider-list-panel {
  width: 30%;
  min-width: 280px;
}
.panel-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px;
}
.panel-title { font-size: 14px; font-weight: 600; color: var(--text-primary); }

.provider-cards { display: flex; flex-direction: column; gap: 12px; }

.provider-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 16px;
  cursor: pointer;
  transition: all 0.25s ease;
}
.provider-card:hover { background: var(--bg-card-hover); border-color: rgba(59,130,246,0.3); }
.provider-card.active {
  border-color: var(--accent);
  background: rgba(59,130,246,0.08);
  box-shadow: 0 0 0 1px var(--accent);
}

.pc-header { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.pc-icon { font-size: 24px; }
.pc-name-area { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.pc-name { font-size: 14px; font-weight: 600; color: var(--text-primary); }
.pc-type { font-size: 10px; padding: 1px 6px; border-radius: 4px; font-weight: 500; width: fit-content; }
.pc-status-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }

.pc-metrics { display: flex; gap: 16px; margin-bottom: 8px; }
.pc-metric { display: flex; flex-direction: column; gap: 2px; }
.pc-metric-label { font-size: 11px; color: var(--text-muted); }
.pc-metric-value { font-size: 14px; color: var(--text-primary); font-weight: 600; }

.pc-status-text { font-size: 12px; font-weight: 500; }

/* 右侧详情 */
.provider-detail-panel { flex: 1; min-width: 0; }
.detail-tabs { --el-tabs-header-height: 40px; }

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 0;
}
.detail-grid .page-card { margin-bottom: 20px; }

/* 信息行 */
.info-rows { display: flex; flex-direction: column; gap: 14px; }
.info-row {
  display: flex; align-items: center; justify-content: space-between;
  gap: 12px;
}
.info-label { font-size: 13px; color: var(--text-muted); white-space: nowrap; }
.info-value { font-size: 14px; color: var(--text-primary); }
.info-code { font-size: 12px; color: var(--accent); background: rgba(59,130,246,0.08); padding: 2px 8px; border-radius: 4px; word-break: break-all; text-align: right; }

.status-badge {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 2px 10px; border-radius: 6px; font-size: 12px; font-weight: 500;
}
.sb-dot { width: 6px; height: 6px; border-radius: 50%; }

/* API Key */
.api-key-area { margin-bottom: 16px; }
.api-key-display {
  padding: 12px;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  margin-bottom: 8px;
  word-break: break-all;
}
.api-key-display .mono { font-size: 13px; color: var(--warning); }
.api-key-actions { display: flex; gap: 8px; }

.error-text { font-size: 12px; color: var(--danger); }
.model-tags { display: flex; flex-wrap: wrap; gap: 4px; }

/* 路由策略 */
.priority-flow {
  display: flex; align-items: center; gap: 4px; flex-wrap: wrap;
}
.priority-step {
  display: flex; align-items: center; gap: 10px;
}
.ps-num {
  width: 28px; height: 28px; border-radius: 50%;
  background: linear-gradient(135deg, var(--accent), #8b5cf6);
  color: #fff; display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700; flex-shrink: 0;
}
.ps-content { display: flex; flex-direction: column; }
.ps-label { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.ps-desc { font-size: 11px; color: var(--text-muted); }
.ps-arrow { color: var(--text-muted); margin: 0 8px; }

.route-rule-list { display: flex; flex-direction: column; gap: 10px; }
.route-rule-item {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  transition: all 0.2s;
}
.route-rule-item:hover { background: var(--bg-card-hover); }
.rr-left { flex-shrink: 0; }
.rr-condition { flex: 1; }
.rr-condition .mono { font-size: 13px; color: var(--text-primary); }
.rr-arrow { color: var(--text-muted); }
.rr-target { display: flex; align-items: center; gap: 8px; min-width: 160px; }
.rr-target-name { font-size: 13px; font-weight: 600; color: var(--accent); }

.routing-hint {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 14px; margin-top: 12px;
  background: rgba(59,130,246,0.06);
  border: 1px solid rgba(59,130,246,0.15);
  border-radius: 8px;
  font-size: 12px; color: var(--text-secondary);
}

/* 用量趋势图 */
.trend-chart {
  display: flex; align-items: flex-end; gap: 12px;
  height: 160px; padding: 8px 0;
}
.trend-bar-col {
  flex: 1; height: 100%;
  display: flex; flex-direction: column; justify-content: flex-end; align-items: center; gap: 6px;
}
.trend-bar {
  width: 100%; max-width: 40px; border-radius: 4px 4px 0 0;
  transition: height 0.4s ease;
}
.trend-label { font-size: 11px; color: var(--text-muted); }

/* Token 统计 */
.token-stats {
  display: flex; gap: 20px; margin-bottom: 20px;
}
.token-stat-item { flex: 1; display: flex; flex-direction: column; gap: 4px; }
.token-stat-item.total { padding-left: 20px; border-left: 1px solid var(--border-color); }
.ts-label { font-size: 12px; color: var(--text-muted); }
.ts-value { font-size: 20px; font-weight: 700; color: var(--text-primary); }

.quota-area { }
.quota-header { display: flex; justify-content: space-between; margin-bottom: 8px; }
.quota-label { font-size: 13px; color: var(--text-secondary); }
.quota-vals { font-size: 13px; color: var(--text-primary); }

/* 费用 */
.cost-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px;
}
.cost-item {
  display: flex; flex-direction: column; gap: 4px;
  padding: 16px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 8px;
}
.cost-label { font-size: 12px; color: var(--text-muted); }
.cost-value { font-size: 20px; font-weight: 700; color: var(--text-primary); }
.cost-value.warn { color: var(--warning); }

/* 租户分布 */
.tenant-usage-bar {
  display: flex; height: 24px; border-radius: 6px; overflow: hidden; margin-bottom: 12px;
}
.tub-segment { transition: width 0.3s ease; }
.tenant-usage-legend { display: flex; flex-wrap: wrap; gap: 16px; }
.tul-item { display: flex; align-items: center; gap: 6px; }
.tul-dot { width: 10px; height: 10px; border-radius: 3px; }
.tul-name { font-size: 12px; color: var(--text-secondary); }
.tul-count { font-size: 12px; color: var(--text-primary); font-weight: 600; }

/* 健康检查 */
.health-check-list { display: flex; flex-direction: column; gap: 10px; }
.hc-item {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 10px 14px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 8px;
}
.hc-time { font-size: 13px; color: var(--text-secondary); }
.hc-status {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 500;
}
.hc-status.ok { color: var(--success); }
.hc-status.fail { color: var(--danger); }
.hc-dot { width: 8px; height: 8px; border-radius: 50%; }
.hc-status.ok .hc-dot { background: var(--success); box-shadow: 0 0 6px var(--success); }
.hc-status.fail .hc-dot { background: var(--danger); }
.hc-latency { font-size: 13px; color: var(--text-primary); }
.text-muted { color: var(--text-muted) !important; }

/* 限流 */
.ratelimit-stats { display: flex; gap: 24px; }
.rl-stat { display: flex; flex-direction: column; gap: 4px; }
.rl-label { font-size: 12px; color: var(--text-muted); }
.rl-value { font-size: 28px; font-weight: 700; color: var(--success); }
.rl-value.danger { color: var(--danger); }

/* 错误趋势 */
.error-trend-chart {
  display: flex; align-items: flex-end; gap: 16px;
  height: 140px; padding: 8px 0;
}
.et-col {
  flex: 1; height: 100%;
  display: flex; flex-direction: column; justify-content: flex-end; align-items: center; gap: 4px;
}
.et-bar-area { width: 100%; height: 100%; display: flex; align-items: flex-end; justify-content: center; }
.et-bar { width: 100%; max-width: 30px; border-radius: 4px 4px 0 0; min-height: 2px; }
.et-label { font-size: 11px; color: var(--text-muted); }
.et-value { font-size: 11px; color: var(--success); }
.et-value.danger { color: var(--danger); }

@media (max-width: 1100px) {
  .providers-layout { flex-direction: column; }
  .provider-list-panel { width: 100%; }
  .detail-grid { grid-template-columns: 1fr; }
  .cost-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
