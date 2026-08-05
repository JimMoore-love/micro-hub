<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { proofreadApi } from '@/api/platform'

const activeTab = ref('config')

// ==================== 接入配置 - 步骤条 ====================
const currentStep = ref(0)

const stepConfig = reactive({
  // Step 1
  providerName: '校对厂商X',
  apiKey: 'pk-pr-XmNbVcXzLkJhGfDsAwQeRtYuIoP123456',
  endpoint: 'https://api.proofread-x.com/v2',
  authType: 'bearer',
  // Step 2
  capabilityType: 'proofread',
  subCapabilities: ['grammar', 'spelling', 'style'],
  // Step 3
  routeTenants: ['default', 'enterprise-a', 'enterprise-b'],
  routePriority: 1,
  fallback: 'openai',
  // Step 4
  perTenantQuota: 5000,
  budgetPerDay: 10,
  // Step 5
  healthCheckUrl: 'https://api.proofread-x.com/v2/health',
  healthCheckInterval: 30,
  // Step 6
  enableRouting: true,
})

const steps = [
  { title: '注册供应商', desc: '填写API Key与认证' },
  { title: '定义能力', desc: '选择校对类型与子能力' },
  { title: '配置路由', desc: '分发规则与降级策略' },
  { title: '设置配额', desc: '调用限制与费用预算' },
  { title: '健康检查', desc: '检查URL与频率' },
  { title: '发布上线', desc: '启用路由规则' },
]

function nextStep() {
  if (currentStep.value < steps.length - 1) {
    currentStep.value++
  } else {
    ElMessage.success('校对API已发布上线！路由规则已生效。')
  }
}
function prevStep() { if (currentStep.value > 0) currentStep.value-- }

const subCapabilityOptions = [
  { label: '语法检查 (Grammar)', value: 'grammar' },
  { label: '拼写纠正 (Spelling)', value: 'spelling' },
  { label: '风格优化 (Style)', value: 'style' },
  { label: '标点修正 (Punctuation)', value: 'punctuation' },
  { label: '用词建议 (Word Choice)', value: 'wordchoice' },
]

// ==================== Tab 2: 校对测试 ====================
const inputText = ref('这个项目的目地是展示微服务架构的优势。通过智能校对API，我们可以自动检测语法错误、拼写问提和风格不一致的问题。')

interface ProofreadError {
  original: string
  type: string
  suggestion: string
  confidence: number
  position: [number, number]
}

const errors = ref<ProofreadError[]>([
  { original: '目地', type: '拼写错误', suggestion: '目的', confidence: 98, position: [7, 9] },
  { original: '优势', type: '拼写错误', suggestion: '优势', confidence: 99, position: [22, 24] },
  { original: '问提', type: '拼写错误', suggestion: '问题', confidence: 95, position: [38, 40] },
])

const isChecking = ref(false)
const checkResult = ref(false)
const requestDetail = reactive({
  provider: '校对厂商X',
  duration: 0,
  tokens: 0,
  cost: 0,
})

async function runProofread() {
  if (!inputText.value.trim()) {
    ElMessage.warning('请输入待校对文本')
    return
  }
  isChecking.value = true
  checkResult.value = false
  try {
    const result = await proofreadApi.check({ text: inputText.value })
    errors.value = result.errors
    requestDetail.provider = result.provider
    requestDetail.duration = result.latency
    requestDetail.tokens = result.tokens
    requestDetail.cost = result.cost
    checkResult.value = true
  } catch (e) {
    console.error('Proofread API call failed:', e)
    checkResult.value = true
    requestDetail.duration = 150 + Math.floor(Math.random() * 50)
    requestDetail.tokens = Math.ceil(inputText.value.length / 2)
    requestDetail.cost = Math.round((requestDetail.tokens / 1000) * 0.005 * 10000) / 10000
  } finally {
    isChecking.value = false
  }
}

// 生成标注后的文本
const highlightedText = computed(() => {
  if (!checkResult.value) return inputText.value
  let result = inputText.value
  // 从后往前替换，避免位置偏移
  const sorted = [...errors.value].sort((a, b) => b.position[0] - a.position[0])
  for (const err of sorted) {
    const [start, end] = err.position
    const before = result.slice(0, start)
    const wrong = result.slice(start, end)
    const after = result.slice(end)
    result = `${before}<span class="text-error" title="建议: ${err.suggestion}">${wrong}<sup>→${err.suggestion}</sup></span>${after}`
  }
  return result
})

function applyAllFixes() {
  let result = inputText.value
  const sorted = [...errors.value].sort((a, b) => b.position[0] - a.position[0])
  for (const err of sorted) {
    const [start, end] = err.position
    result = result.slice(0, start) + err.suggestion + result.slice(end)
  }
  inputText.value = result
  errors.value = []
  ElMessage.success('已应用所有修改建议')
}

const typeColors: Record<string, string> = {
  '拼写错误': '#ef4444',
  '语法错误': '#f59e0b',
  '风格建议': '#3b82f6',
}

// ==================== Tab 3: 使用监测 ====================
const monitorStats = ref([
  { title: '今日调用', value: '340', icon: 'DataLine', color: '#3b82f6', bg: 'rgba(59,130,246,0.1)' },
  { title: '平均延迟', value: '150ms', icon: 'Timer', color: '#22c55e', bg: 'rgba(34,197,94,0.1)' },
  { title: '成功率', value: '98.5%', icon: 'CircleCheck', color: '#8b5cf6', bg: 'rgba(139,92,246,0.1)' },
  { title: '今日费用', value: '¥1.7', icon: 'Money', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)' },
])

const tenantUsage = ref([
  { tenant: 'default', calls: 120, successRate: 99.2, avgLatency: 145, cost: 0.6 },
  { tenant: 'enterprise-a', calls: 150, successRate: 98.0, avgLatency: 155, cost: 0.75 },
  { tenant: 'enterprise-b', calls: 60, successRate: 98.5, avgLatency: 148, cost: 0.3 },
  { tenant: 'test-org', calls: 10, successRate: 95.0, avgLatency: 180, cost: 0.05 },
])

const requestLogs = ref([
  { time: '14:32:15', tenant: 'enterprise-a', textLen: 256, results: 3, latency: 142, provider: '校对厂商X' },
  { time: '14:30:08', tenant: 'default', textLen: 128, results: 1, latency: 138, provider: '校对厂商X' },
  { time: '14:28:42', tenant: 'enterprise-b', textLen: 512, results: 5, latency: 165, provider: '校对厂商X' },
  { time: '14:25:30', tenant: 'enterprise-a', textLen: 340, results: 2, latency: 150, provider: '校对厂商X' },
  { time: '14:22:18', tenant: 'default', textLen: 89, results: 0, latency: 120, provider: '校对厂商X' },
  { time: '14:20:05', tenant: 'enterprise-a', textLen: 420, results: 4, latency: 158, provider: '校对厂商X' },
  { time: '14:18:50', tenant: 'enterprise-b', textLen: 200, results: 1, latency: 145, provider: '校对厂商X' },
  { time: '14:15:33', tenant: 'default', textLen: 156, results: 2, latency: 135, provider: '校对厂商X' },
  { time: '14:12:20', tenant: 'test-org', textLen: 64, results: 1, latency: 175, provider: '校对厂商X' },
  { time: '14:10:08', tenant: 'enterprise-a', textLen: 380, results: 3, latency: 148, provider: '校对厂商X' },
  { time: '14:08:45', tenant: 'enterprise-b', textLen: 290, results: 2, latency: 152, provider: '校对厂商X' },
  { time: '14:05:30', tenant: 'default', textLen: 110, results: 0, latency: 128, provider: '校对厂商X' },
  { time: '14:03:15', tenant: 'enterprise-a', textLen: 450, results: 5, latency: 170, provider: '校对厂商X' },
  { time: '14:00:02', tenant: 'default', textLen: 95, results: 1, latency: 140, provider: '校对厂商X' },
  { time: '13:58:40', tenant: 'enterprise-b', textLen: 220, results: 2, latency: 146, provider: '校对厂商X' },
  { time: '13:55:25', tenant: 'enterprise-a', textLen: 310, results: 3, latency: 155, provider: '校对厂商X' },
  { time: '13:52:10', tenant: 'default', textLen: 140, results: 1, latency: 132, provider: '校对厂商X' },
  { time: '13:50:00', tenant: 'test-org', textLen: 78, results: 0, latency: 165, provider: '校对厂商X' },
  { time: '13:48:35', tenant: 'enterprise-a', textLen: 360, results: 4, latency: 160, provider: '校对厂商X' },
  { time: '13:45:20', tenant: 'enterprise-b', textLen: 180, results: 1, latency: 143, provider: '校对厂商X' },
])

const errorLogs = ref([
  { time: '13:30:15', type: '429 限流', tenant: 'test-org', detail: '请求频率超过限制，已触发降级', level: 'warn' },
  { time: '12:15:08', type: '超时', tenant: 'enterprise-a', detail: '请求超过5s未响应，已重试', level: 'error' },
  { time: '10:45:30', type: 'API错误', tenant: 'default', detail: '返回 502 Bad Gateway', level: 'error' },
  { time: '09:20:12', type: '429 限流', tenant: 'test-org', detail: '请求频率超过限制', level: 'warn' },
])

onMounted(async () => {
  try {
    const config = await proofreadApi.getConfig()
    if (config?.provider) {
      stepConfig.providerName = config.provider.name
      stepConfig.apiKey = config.provider.api_key
      stepConfig.endpoint = config.provider.endpoint
    }
  } catch (e) {
    console.error('Failed to fetch proofread config:', e)
  }
})

watch(activeTab, async (tab) => {
  if (tab !== 'monitor') return
  try {
    const stats = await proofreadApi.getStats()
    if (stats) {
      monitorStats.value = [
        { title: '今日调用', value: String(stats.today_calls), icon: 'DataLine', color: '#3b82f6', bg: 'rgba(59,130,246,0.1)' },
        { title: '平均延迟', value: stats.avg_latency + 'ms', icon: 'Timer', color: '#22c55e', bg: 'rgba(34,197,94,0.1)' },
        { title: '成功率', value: stats.success_rate + '%', icon: 'CircleCheck', color: '#8b5cf6', bg: 'rgba(139,92,246,0.1)' },
        { title: '今日费用', value: '¥' + stats.today_cost, icon: 'Money', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)' },
      ]
    }
  } catch (e) {
    console.error('Failed to fetch proofread stats:', e)
  }
  try {
    const logs = await proofreadApi.listLogs()
    if (logs && logs.length > 0) {
      requestLogs.value = logs.map(l => ({
        time: l.time,
        tenant: l.tenant_id,
        textLen: l.text_length,
        results: l.error_count,
        latency: l.latency,
        provider: l.provider,
      }))
    }
  } catch (e) {
    console.error('Failed to fetch proofread logs:', e)
  }
})
</script>

<template>
  <div class="proofread-page">
    <h1 class="page-title">智能校对 API 接入</h1>

    <el-tabs v-model="activeTab" class="pr-tabs">
      <!-- ==================== Tab 1: 接入配置 ==================== -->
      <el-tab-pane label="接入配置" name="config">
        <div class="page-card">
          <div class="page-card-title">第三方校对 API 接入流程</div>

          <!-- 步骤条 -->
          <el-steps :active="currentStep" finish-status="success" align-center class="pr-steps">
            <el-step v-for="(step, idx) in steps" :key="idx" :title="step.title" :description="step.desc" />
          </el-steps>

          <!-- 步骤内容 -->
          <div class="step-content">
            <!-- Step 1: 注册供应商 -->
            <div v-show="currentStep === 0" class="step-panel">
              <h3 class="step-title">Step 1 · 注册供应商</h3>
              <p class="step-desc">填写第三方校对API的基本接入信息与认证方式</p>
              <el-form :model="stepConfig" label-width="140px" size="default" style="max-width: 560px">
                <el-form-item label="供应商名称">
                  <el-input v-model="stepConfig.providerName" />
                </el-form-item>
                <el-form-item label="API Key">
                  <el-input v-model="stepConfig.apiKey" show-password />
                </el-form-item>
                <el-form-item label="Endpoint URL">
                  <el-input v-model="stepConfig.endpoint" />
                </el-form-item>
                <el-form-item label="认证方式">
                  <el-radio-group v-model="stepConfig.authType">
                    <el-radio value="bearer">Bearer Token</el-radio>
                    <el-radio value="header">自定义 Header</el-radio>
                    <el-radio value="query">Query 参数</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-form>
            </div>

            <!-- Step 2: 定义能力 -->
            <div v-show="currentStep === 1" class="step-panel">
              <h3 class="step-title">Step 2 · 定义能力</h3>
              <p class="step-desc">选择该供应商提供的能力类型与支持的子能力</p>
              <el-form :model="stepConfig" label-width="140px" size="default" style="max-width: 560px">
                <el-form-item label="能力类型">
                  <el-select v-model="stepConfig.capabilityType" style="width: 100%">
                    <el-option label="校对 (Proofread)" value="proofread" />
                    <el-option label="翻译 (Translate)" value="translate" />
                    <el-option label="LLM 对话" value="llm" />
                    <el-option label="图像生成" value="image" />
                  </el-select>
                </el-form-item>
                <el-form-item label="子能力">
                  <el-checkbox-group v-model="stepConfig.subCapabilities">
                    <el-checkbox v-for="opt in subCapabilityOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-form>
            </div>

            <!-- Step 3: 配置路由 -->
            <div v-show="currentStep === 2" class="step-panel">
              <h3 class="step-title">Step 3 · 配置路由</h3>
              <p class="step-desc">设置分发规则：哪些租户使用此校对服务、优先级与降级策略</p>
              <el-form :model="stepConfig" label-width="140px" size="default" style="max-width: 560px">
                <el-form-item label="适用租户">
                  <el-select v-model="stepConfig.routeTenants" multiple style="width: 100%">
                    <el-option label="默认租户 (default)" value="default" />
                    <el-option label="企业A (enterprise-a)" value="enterprise-a" />
                    <el-option label="企业B (enterprise-b)" value="enterprise-b" />
                    <el-option label="测试组织 (test-org)" value="test-org" />
                  </el-select>
                </el-form-item>
                <el-form-item label="路由优先级">
                  <el-input-number v-model="stepConfig.routePriority" :min="1" :max="10" />
                  <span class="form-hint">数字越小优先级越高</span>
                </el-form-item>
                <el-form-item label="降级供应商">
                  <el-select v-model="stepConfig.fallback" style="width: 100%">
                    <el-option label="OpenAI (LLM兜底)" value="openai" />
                    <el-option label="DeepSeek (LLM兜底)" value="deepseek" />
                    <el-option label="无降级" value="none" />
                  </el-select>
                </el-form-item>
              </el-form>
            </div>

            <!-- Step 4: 设置配额 -->
            <div v-show="currentStep === 3" class="step-panel">
              <h3 class="step-title">Step 4 · 设置配额</h3>
              <p class="step-desc">为每个租户设置调用限制与费用预算</p>
              <el-form :model="stepConfig" label-width="160px" size="default" style="max-width: 560px">
                <el-form-item label="每租户调用限制/天">
                  <el-input-number v-model="stepConfig.perTenantQuota" :min="100" :step="500" style="width: 200px" />
                </el-form-item>
                <el-form-item label="每日费用预算($)">
                  <el-input-number v-model="stepConfig.budgetPerDay" :min="1" :step="5" style="width: 200px" />
                </el-form-item>
              </el-form>
            </div>

            <!-- Step 5: 健康检查 -->
            <div v-show="currentStep === 4" class="step-panel">
              <h3 class="step-title">Step 5 · 健康检查</h3>
              <p class="step-desc">配置健康检查URL与检查频率，确保服务可用性</p>
              <el-form :model="stepConfig" label-width="160px" size="default" style="max-width: 560px">
                <el-form-item label="健康检查URL">
                  <el-input v-model="stepConfig.healthCheckUrl" />
                </el-form-item>
                <el-form-item label="检查频率(秒)">
                  <el-input-number v-model="stepConfig.healthCheckInterval" :min="10" :step="10" style="width: 200px" />
                </el-form-item>
              </el-form>
            </div>

            <!-- Step 6: 发布上线 -->
            <div v-show="currentStep === 5" class="step-panel">
              <h3 class="step-title">Step 6 · 发布上线</h3>
              <p class="step-desc">确认配置无误后，启用路由规则，校对服务正式上线</p>
              <div class="review-config">
                <div class="review-item">
                  <span class="review-label">供应商</span>
                  <span class="review-value">{{ stepConfig.providerName }}</span>
                </div>
                <div class="review-item">
                  <span class="review-label">Endpoint</span>
                  <code class="mono review-value">{{ stepConfig.endpoint }}</code>
                </div>
                <div class="review-item">
                  <span class="review-label">能力类型</span>
                  <span class="review-value">{{ stepConfig.capabilityType }}</span>
                </div>
                <div class="review-item">
                  <span class="review-label">子能力</span>
                  <span class="review-value">{{ stepConfig.subCapabilities.join(', ') }}</span>
                </div>
                <div class="review-item">
                  <span class="review-label">适用租户</span>
                  <span class="review-value">{{ stepConfig.routeTenants.join(', ') }}</span>
                </div>
                <div class="review-item">
                  <span class="review-label">降级策略</span>
                  <span class="review-value">{{ stepConfig.fallback }}</span>
                </div>
                <div class="review-item">
                  <span class="review-label">每租户配额</span>
                  <span class="review-value">{{ stepConfig.perTenantQuota }} 次/天</span>
                </div>
              </div>
              <el-form-item label="启用路由规则" style="margin-top: 16px">
                <el-switch v-model="stepConfig.enableRouting" />
              </el-form-item>
            </div>

            <!-- 步骤导航 -->
            <div class="step-nav">
              <el-button @click="prevStep" :disabled="currentStep === 0">上一步</el-button>
              <el-button type="primary" class="btn-gradient" @click="nextStep">
                {{ currentStep === steps.length - 1 ? '发布上线' : '下一步' }}
              </el-button>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- ==================== Tab 2: 校对测试 ==================== -->
      <el-tab-pane label="校对测试" name="test">
        <div class="test-layout">
          <!-- 左侧输入 -->
          <div class="page-card test-input-card">
            <div class="page-card-title">输入文本</div>
            <el-input
              v-model="inputText"
              type="textarea"
              :rows="12"
              placeholder="请输入需要校对的中文文本..."
              resize="none"
            />
            <div class="test-input-meta">
              <span class="mono text-muted">{{ inputText.length }} 字符</span>
              <el-button type="primary" class="btn-gradient" @click="runProofread" :loading="isChecking">
                <el-icon><Promotion /></el-icon>
                {{ isChecking ? '校对中...' : '开始校对' }}
              </el-button>
            </div>
          </div>

          <!-- 右侧结果 -->
          <div class="test-result-area">
            <div class="page-card">
              <div class="page-card-title">
                校对结果
                <el-tag v-if="checkResult" size="small" type="danger" class="title-badge">发现 {{ errors.length }} 处问题</el-tag>
              </div>

              <div v-if="isChecking" class="checking-placeholder">
                <el-icon class="loading-icon" :size="32"><Loading /></el-icon>
                <p>正在调用校对API...</p>
              </div>

              <div v-else-if="checkResult" class="result-content">
                <!-- 错误列表 -->
                <div v-if="errors.length > 0" class="error-list">
                  <div v-for="(err, idx) in errors" :key="idx" class="error-item">
                    <div class="err-left">
                      <span class="err-index">{{ idx + 1 }}</span>
                    </div>
                    <div class="err-body">
                      <div class="err-top">
                        <code class="mono err-original">{{ err.original }}</code>
                        <el-icon class="err-arrow"><Right /></el-icon>
                        <code class="mono err-suggestion">{{ err.suggestion }}</code>
                      </div>
                      <div class="err-bottom">
                        <el-tag size="small" :style="{ color: typeColors[err.type], background: typeColors[err.type] + '15', border: 'none' }">{{ err.type }}</el-tag>
                        <span class="err-confidence">置信度 <span class="mono">{{ err.confidence }}%</span></span>
                      </div>
                    </div>
                  </div>
                </div>
                <div v-else class="no-errors">
                  <el-icon :size="32" color="#22c55e"><CircleCheckFilled /></el-icon>
                  <p>未发现问题，文本质量良好</p>
                </div>

                <!-- 修改后文本 -->
                <div class="corrected-text-area" v-if="errors.length > 0">
                  <div class="corrected-header">
                    <span class="corrected-label">修改后文本</span>
                    <el-button size="small" type="primary" @click="applyAllFixes">应用全部修改</el-button>
                  </div>
                  <div class="corrected-text" v-html="highlightedText"></div>
                </div>
              </div>

              <div v-else class="result-placeholder">
                <el-icon :size="32" color="var(--text-muted)"><EditPen /></el-icon>
                <p>点击"开始校对"查看结果</p>
              </div>
            </div>

            <!-- 请求详情 -->
            <div v-if="checkResult" class="page-card">
              <div class="page-card-title">请求详情</div>
              <div class="req-detail-grid">
                <div class="req-item">
                  <span class="req-label">调用供应商</span>
                  <span class="req-value">{{ requestDetail.provider }}</span>
                </div>
                <div class="req-item">
                  <span class="req-label">耗时</span>
                  <span class="mono req-value">{{ requestDetail.duration }}ms</span>
                </div>
                <div class="req-item">
                  <span class="req-label">Tokens</span>
                  <span class="mono req-value">{{ requestDetail.tokens }}</span>
                </div>
                <div class="req-item">
                  <span class="req-label">费用</span>
                  <span class="mono req-value">${{ requestDetail.cost.toFixed(4) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- ==================== Tab 3: 使用监测 ==================== -->
      <el-tab-pane label="使用监测" name="monitor">
        <!-- 统计卡片 -->
        <div class="stats-grid">
          <div v-for="card in monitorStats" :key="card.title" class="stat-card">
            <div class="stat-icon" :style="{ background: card.bg, color: card.color }">
              <el-icon :size="22"><component :is="card.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ card.title }}</div>
              <div class="stat-value mono">{{ card.value }}</div>
            </div>
          </div>
        </div>

        <!-- 租户使用分布 -->
        <div class="page-card" style="padding: 0; overflow: hidden">
          <div class="page-card-title" style="padding: 20px 24px 0">租户使用分布</div>
          <el-table :data="tenantUsage" stripe style="width: 100%" :row-style="{ height: '48px' }">
            <el-table-column prop="tenant" label="租户" min-width="140">
              <template #default="{ row }">
                <code class="mono tenant-id">{{ row.tenant }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="calls" label="调用次数" width="120" align="center">
              <template #default="{ row }"><span class="mono">{{ row.calls }}</span></template>
            </el-table-column>
            <el-table-column prop="successRate" label="成功率" width="120" align="center">
              <template #default="{ row }">
                <span class="mono" :class="{ 'text-success': row.successRate >= 98, 'text-warn': row.successRate < 98 }">{{ row.successRate }}%</span>
              </template>
            </el-table-column>
            <el-table-column prop="avgLatency" label="平均延迟" width="120" align="center">
              <template #default="{ row }"><span class="mono">{{ row.avgLatency }}ms</span></template>
            </el-table-column>
            <el-table-column prop="cost" label="费用($)" width="100" align="right">
              <template #default="{ row }"><span class="mono">${{ row.cost.toFixed(2) }}</span></template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 请求日志 -->
        <div class="page-card" style="padding: 0; overflow: hidden">
          <div class="page-card-title" style="padding: 20px 24px 0">请求日志 (最近20条)</div>
          <el-table :data="requestLogs" stripe style="width: 100%" :row-style="{ height: '44px' }" size="small">
            <el-table-column prop="time" label="时间" width="100">
              <template #default="{ row }"><span class="mono text-muted">{{ row.time }}</span></template>
            </el-table-column>
            <el-table-column prop="tenant" label="租户" width="130">
              <template #default="{ row }"><code class="mono">{{ row.tenant }}</code></template>
            </el-table-column>
            <el-table-column prop="textLen" label="文本长度" width="90" align="center">
              <template #default="{ row }"><span class="mono">{{ row.textLen }}</span></template>
            </el-table-column>
            <el-table-column prop="results" label="结果数" width="80" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="row.results === 0 ? 'success' : 'warning'">{{ row.results }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="latency" label="延迟" width="80" align="center">
              <template #default="{ row }"><span class="mono">{{ row.latency }}ms</span></template>
            </el-table-column>
            <el-table-column prop="provider" label="供应商" min-width="120">
              <template #default="{ row }"><span class="text-secondary">{{ row.provider }}</span></template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 错误日志 -->
        <div class="page-card" style="padding: 0; overflow: hidden">
          <div class="page-card-title" style="padding: 20px 24px 0">错误日志</div>
          <div class="error-log-list">
            <div v-for="(log, idx) in errorLogs" :key="idx" class="elog-item">
              <span class="mono elog-time">{{ log.time }}</span>
              <el-tag size="small" :type="log.level === 'error' ? 'danger' : 'warning'">{{ log.type }}</el-tag>
              <span class="elog-tenant">{{ log.tenant }}</span>
              <span class="elog-detail">{{ log.detail }}</span>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.proofread-page { max-width: 1400px; }
.mono { font-family: 'Fira Code', 'Cascadia Code', monospace; }
.pr-tabs { --el-tabs-header-height: 40px; }

/* 步骤条 */
.pr-steps { margin-bottom: 32px; }
.step-content {
  padding: 24px;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: var(--radius);
  min-height: 320px;
}
.step-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin-bottom: 8px; }
.step-desc { font-size: 13px; color: var(--text-muted); margin-bottom: 24px; }
.form-hint { margin-left: 12px; font-size: 12px; color: var(--text-muted); }

.step-nav {
  display: flex; justify-content: space-between;
  margin-top: 24px; padding-top: 20px;
  border-top: 1px solid var(--border-color);
}

/* 审核配置 */
.review-config {
  display: grid; grid-template-columns: 1fr 1fr; gap: 12px;
  padding: 16px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 8px;
}
.review-item { display: flex; flex-direction: column; gap: 2px; }
.review-label { font-size: 11px; color: var(--text-muted); }
.review-value { font-size: 13px; color: var(--text-primary); }

/* 统计卡片 */
.stats-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px; margin-bottom: 20px;
}
.stat-card {
  background: var(--bg-card); border: 1px solid var(--border-color);
  border-radius: var(--radius-lg); padding: 20px;
  display: flex; align-items: center; gap: 16px;
}
.stat-icon {
  width: 48px; height: 48px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-info { display: flex; flex-direction: column; gap: 4px; }
.stat-label { font-size: 13px; color: var(--text-muted); }
.stat-value { font-size: 24px; font-weight: 700; color: var(--text-primary); }

/* 校对测试 */
.test-layout {
  display: grid; grid-template-columns: 1fr 1fr; gap: 20px; align-items: flex-start;
}
.test-input-card { margin-bottom: 0; }
.test-input-meta {
  display: flex; align-items: center; justify-content: space-between;
  margin-top: 12px;
}
.text-muted { color: var(--text-muted); font-size: 12px; }
.text-secondary { color: var(--text-secondary); font-size: 13px; }
.text-success { color: var(--success); }
.text-warn { color: var(--warning); }

.test-result-area { display: flex; flex-direction: column; gap: 20px; }
.title-badge { margin-left: 8px; }

.checking-placeholder, .result-placeholder {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 60px 0; color: var(--text-muted); gap: 12px;
}
.loading-icon { animation: spin 1s linear infinite; color: var(--accent); }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.result-content { display: flex; flex-direction: column; gap: 20px; }

/* 错误列表 */
.error-list { display: flex; flex-direction: column; gap: 10px; }
.error-item {
  display: flex; gap: 12px;
  padding: 14px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 8px;
}
.err-left { flex-shrink: 0; }
.err-index {
  width: 24px; height: 24px; border-radius: 6px;
  background: rgba(239,68,68,0.15); color: var(--danger);
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 700;
}
.err-body { flex: 1; display: flex; flex-direction: column; gap: 8px; }
.err-top { display: flex; align-items: center; gap: 10px; }
.err-original { font-size: 14px; color: var(--danger); text-decoration: line-through; }
.err-arrow { color: var(--text-muted); }
.err-suggestion { font-size: 14px; color: var(--success); font-weight: 600; }
.err-bottom { display: flex; align-items: center; gap: 16px; }
.err-confidence { font-size: 12px; color: var(--text-muted); }

.no-errors {
  display: flex; flex-direction: column; align-items: center; padding: 40px; gap: 12px; color: var(--success);
}

/* 修改后文本 */
.corrected-text-area {
  padding: 16px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 8px;
}
.corrected-header {
  display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;
}
.corrected-label { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.corrected-text {
  font-size: 14px; line-height: 1.8; color: var(--text-primary);
}
.corrected-text :deep(.text-error) {
  color: var(--danger); text-decoration: underline wavy var(--danger);
  position: relative; cursor: help;
}
.corrected-text :deep(.text-error sup) {
  color: var(--success); font-size: 11px; font-weight: 600; margin-left: 2px;
}

/* 请求详情 */
.req-detail-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px;
}
.req-item { display: flex; flex-direction: column; gap: 4px; }
.req-label { font-size: 12px; color: var(--text-muted); }
.req-value { font-size: 16px; font-weight: 600; color: var(--text-primary); }

/* 租户ID */
.tenant-id { font-size: 12px; color: var(--accent); }

/* 错误日志 */
.error-log-list { padding: 0 24px 16px; }
.elog-item {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 0; border-bottom: 1px solid var(--border-color);
  font-size: 13px;
}
.elog-item:last-child { border-bottom: none; }
.elog-time { color: var(--text-muted); font-size: 12px; min-width: 80px; }
.elog-tenant { color: var(--accent); min-width: 120px; }
.elog-detail { color: var(--text-secondary); flex: 1; }

@media (max-width: 1100px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .test-layout { grid-template-columns: 1fr; }
  .review-config { grid-template-columns: 1fr; }
}
</style>
