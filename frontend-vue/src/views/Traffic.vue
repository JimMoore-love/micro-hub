<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { trafficApi } from '@/api/platform'

// ==================== 熔断器状态 ====================
interface CircuitBreaker {
  serviceId: string
  serviceName: string
  state: 'closed' | 'open' | 'half-open'
  failureThreshold: number
  timeout: number
  halfOpenProbes: number
  currentFailures: number
  lastStateChange: string
}

const breakers = ref<CircuitBreaker[]>([
  { serviceId: 'user-service', serviceName: 'User Service', state: 'closed', failureThreshold: 5, timeout: 30, halfOpenProbes: 3, currentFailures: 0, lastStateChange: '2024-03-07 16:00:00' },
  { serviceId: 'order-service', serviceName: 'Order Service', state: 'closed', failureThreshold: 5, timeout: 30, halfOpenProbes: 3, currentFailures: 1, lastStateChange: '2024-03-08 10:30:00' },
  { serviceId: 'ai-service', serviceName: 'AI Service', state: 'half-open', failureThreshold: 3, timeout: 60, halfOpenProbes: 2, currentFailures: 4, lastStateChange: '2024-03-08 14:15:00' },
])

// ==================== 降级策略 ====================
interface DegradeRule {
  id: string
  serviceName: string
  condition: string
  fallback: string
  configured: boolean
}

const degradeRules = ref<DegradeRule[]>([
  { id: 'd1', serviceName: 'User Service', condition: '错误率 > 5%', fallback: '返回缓存数据', configured: true },
  { id: 'd2', serviceName: 'Order Service', condition: '延迟 > 500ms', fallback: '返回"系统繁忙稍后重试"', configured: true },
  { id: 'd3', serviceName: 'AI Service', condition: '配额超限', fallback: '返回 HTTP 429', configured: true },
])

// ==================== 重试策略 ====================
interface RetryPolicy {
  id: string
  serviceName: string
  maxRetries: number
  retryInterval: number
  retryableCodes: string[]
  enabled: boolean
}

const retryPolicies = ref<RetryPolicy[]>([
  { id: 'rp1', serviceName: 'User Service', maxRetries: 3, retryInterval: 100, retryableCodes: ['500', '503'], enabled: true },
  { id: 'rp2', serviceName: 'Order Service', maxRetries: 2, retryInterval: 200, retryableCodes: ['500', '502', '503'], enabled: true },
  { id: 'rp3', serviceName: 'AI Service', maxRetries: 1, retryInterval: 500, retryableCodes: ['429', '503'], enabled: true },
])

// ==================== 弹窗 ====================
const breakerDialog = ref(false)
const editingBreaker = ref<CircuitBreaker | null>(null)
const breakerForm = reactive({ failureThreshold: 5, timeout: 30, halfOpenProbes: 3 })

function openBreakerEdit(cb: CircuitBreaker) {
  editingBreaker.value = cb
  breakerForm.failureThreshold = cb.failureThreshold
  breakerForm.timeout = cb.timeout
  breakerForm.halfOpenProbes = cb.halfOpenProbes
  breakerDialog.value = true
}

function saveBreaker() {
  if (editingBreaker.value) {
    editingBreaker.value.failureThreshold = breakerForm.failureThreshold
    editingBreaker.value.timeout = breakerForm.timeout
    editingBreaker.value.halfOpenProbes = breakerForm.halfOpenProbes
  }
  breakerDialog.value = false
}

const degradeDialog = ref(false)
const editingDegrade = ref<DegradeRule | null>(null)
const degradeForm = reactive({ condition: '', fallback: '' })

function openDegradeEdit(rule: DegradeRule) {
  editingDegrade.value = rule
  degradeForm.condition = rule.condition
  degradeForm.fallback = rule.fallback
  degradeDialog.value = true
}

function saveDegrade() {
  if (editingDegrade.value) {
    editingDegrade.value.condition = degradeForm.condition
    editingDegrade.value.fallback = degradeForm.fallback
  }
  degradeDialog.value = false
}

const retryDialog = ref(false)
const editingRetry = ref<RetryPolicy | null>(null)
const retryForm = reactive({ maxRetries: 3, retryInterval: 100 })

function openRetryEdit(policy: RetryPolicy) {
  editingRetry.value = policy
  retryForm.maxRetries = policy.maxRetries
  retryForm.retryInterval = policy.retryInterval
  retryDialog.value = true
}

function saveRetry() {
  if (editingRetry.value) {
    editingRetry.value.maxRetries = retryForm.maxRetries
    editingRetry.value.retryInterval = retryForm.retryInterval
  }
  retryDialog.value = false
}

// ==================== 工具函数 ====================
function getBreakerStateLabel(state: string) {
  const map: Record<string, string> = { closed: '已关闭 (Closed)', open: '已熔断 (Open)', 'half-open': '半开探测 (Half-Open)' }
  return map[state] || state
}

function getBreakerStateType(state: string) {
  const map: Record<string, string> = { closed: 'success', open: 'danger', 'half-open': 'warning' }
  return map[state] || 'info'
}

function getBreakerStateColor(state: string) {
  const map: Record<string, string> = { closed: 'var(--success)', open: 'var(--danger)', 'half-open': 'var(--warning)' }
  return map[state] || 'var(--text-muted)'
}

onMounted(async () => {
  try {
    const [breakerData, degradeData, retryData] = await Promise.all([
      trafficApi.listCircuitBreakers(),
      trafficApi.listDegradationRules(),
      trafficApi.listRetryPolicies(),
    ])
    if (breakerData && breakerData.length > 0) {
      breakers.value = breakerData.map(b => ({
        serviceId: b.service,
        serviceName: b.service,
        state: b.state,
        failureThreshold: b.failure_threshold,
        timeout: b.open_duration,
        halfOpenProbes: b.half_open_probes,
        currentFailures: 0,
        lastStateChange: '',
      }))
    }
    if (degradeData && degradeData.length > 0) {
      degradeRules.value = degradeData.map((d, idx) => ({
        id: `d-${idx}`,
        serviceName: d.service,
        condition: d.condition,
        fallback: d.response,
        configured: d.enabled,
      }))
    }
    if (retryData && retryData.length > 0) {
      retryPolicies.value = retryData.map((r, idx) => ({
        id: `rp-${idx}`,
        serviceName: r.service,
        maxRetries: r.max_retries,
        retryInterval: r.interval,
        retryableCodes: r.retryable_codes.map(String),
        enabled: true,
      }))
    }
  } catch (e) {
    console.error('Failed to fetch traffic data:', e)
  }
})
</script>

<template>
  <div class="traffic-page">
    <h1 class="page-title">流量治理</h1>

    <div class="traffic-grid">
      <!-- ===== 卡片1: 熔断器状态 ===== -->
      <div class="page-card circuit-card">
        <div class="page-card-title">熔断器状态</div>

        <div class="circuit-list">
          <div
            v-for="cb in breakers"
            :key="cb.serviceId"
            class="circuit-item"
          >
            <div class="circuit-header">
              <span class="circuit-dot" :style="{ background: getBreakerStateColor(cb.state) }"></span>
              <span class="circuit-name">{{ cb.serviceName }}</span>
              <el-tag :type="getBreakerStateType(cb.state)" size="small" effect="dark">
                {{ getBreakerStateLabel(cb.state) }}
              </el-tag>
            </div>

            <div class="circuit-body">
              <div class="circuit-config">
                <div class="config-row">
                  <span class="config-label">失败阈值</span>
                  <span class="config-value mono">{{ cb.failureThreshold }}%</span>
                </div>
                <div class="config-row">
                  <span class="config-label">熔断时间</span>
                  <span class="config-value mono">{{ cb.timeout }}s</span>
                </div>
                <div class="config-row">
                  <span class="config-label">半开探测</span>
                  <span class="config-value mono">{{ cb.halfOpenProbes }} 次</span>
                </div>
              </div>

              <div class="circuit-stats">
                <div class="stat-item">
                  <span class="stat-num mono" :style="{ color: cb.currentFailures > cb.failureThreshold ? 'var(--danger)' : 'var(--success)' }">
                    {{ cb.currentFailures }}
                  </span>
                  <span class="stat-label">当前失败数</span>
                </div>
                <div class="stat-item">
                  <span class="stat-time">{{ cb.lastStateChange }}</span>
                  <span class="stat-label">上次状态变更</span>
                </div>
              </div>
            </div>

            <el-button size="small" class="cb-edit-btn" @click="openBreakerEdit(cb)">
              编辑配置
            </el-button>
          </div>
        </div>
      </div>

      <!-- ===== 卡片2: 降级策略 ===== -->
      <div class="page-card">
        <div class="page-card-title">降级策略</div>

        <el-table :data="degradeRules" stripe style="width: 100%" :row-style="{ height: '48px' }">
          <el-table-column prop="serviceName" label="服务名" width="140" />
          <el-table-column prop="condition" label="降级条件" min-width="150">
            <template #default="{ row }">
              <span class="condition-text">{{ row.condition }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="fallback" label="降级响应" min-width="180">
            <template #default="{ row }">
              <code class="mono fallback-code">{{ row.fallback }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="configured" label="状态" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="row.configured ? 'success' : 'info'" size="small">
                {{ row.configured ? '已配置' : '未配置' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openDegradeEdit(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- ===== 卡片3: 重试策略 ===== -->
      <div class="page-card">
        <div class="page-card-title">重试策略</div>

        <el-table :data="retryPolicies" stripe style="width: 100%" :row-style="{ height: '48px' }">
          <el-table-column prop="serviceName" label="服务名" width="140" />
          <el-table-column prop="maxRetries" label="最大重试" width="90" align="center">
            <template #default="{ row }">
              <span class="mono retry-count">{{ row.maxRetries }} 次</span>
            </template>
          </el-table-column>
          <el-table-column prop="retryInterval" label="重试间隔" width="100" align="center">
            <template #default="{ row }">
              <span class="mono">{{ row.retryInterval }}ms</span>
            </template>
          </el-table-column>
          <el-table-column prop="retryableCodes" label="可重试错误码" min-width="200">
            <template #default="{ row }">
              <el-tag
                v-for="code in row.retryableCodes"
                :key="code"
                size="small"
                :type="code === '429' ? 'warning' : 'danger'"
                class="code-tag"
              >{{ code }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openRetryEdit(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- ===== 弹窗：编辑熔断器 ===== -->
    <el-dialog v-model="breakerDialog" title="编辑熔断器 - {{ editingBreaker?.serviceName }}" width="440px">
      <el-form :model="breakerForm" label-width="120px" size="small">
        <el-form-item label="失败阈值 (%)">
          <el-input-number v-model="breakerForm.failureThreshold" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="熔断时间 (秒)">
          <el-input-number v-model="breakerForm.timeout" :min="5" :max="300" :step="5" />
        </el-form-item>
        <el-form-item label="半开探测次数">
          <el-input-number v-model="breakerForm.halfOpenProbes" :min="1" :max="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="breakerDialog = false">取消</el-button>
        <el-button type="primary" @click="saveBreaker">保存</el-button>
      </template>
    </el-dialog>

    <!-- ===== 弹窗：编辑降级策略 ===== -->
    <el-dialog v-model="degradeDialog" title="编辑降级策略 - {{ editingDegrade?.serviceName }}" width="460px">
      <el-form :model="degradeForm" label-width="100px" size="small">
        <el-form-item label="降级条件">
          <el-input v-model="degradeForm.condition" placeholder="如: 错误率 > 5%" />
        </el-form-item>
        <el-form-item label="降级响应">
          <el-input v-model="degradeForm.fallback" placeholder="如: 返回缓存数据" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="degradeDialog = false">取消</el-button>
        <el-button type="primary" @click="saveDegrade">保存</el-button>
      </template>
    </el-dialog>

    <!-- ===== 弹窗：编辑重试策略 ===== -->
    <el-dialog v-model="retryDialog" title="编辑重试策略 - {{ editingRetry?.serviceName }}" width="440px">
      <el-form :model="retryForm" label-width="120px" size="small">
        <el-form-item label="最大重试次数">
          <el-input-number v-model="retryForm.maxRetries" :min="0" :max="10" />
        </el-form-item>
        <el-form-item label="重试间隔 (ms)">
          <el-input-number v-model="retryForm.retryInterval" :min="50" :max="5000" :step="50" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="retryDialog = false">取消</el-button>
        <el-button type="primary" @click="saveRetry">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.traffic-page {
  max-width: 1400px;
}

.traffic-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* ===== 熔断器卡片 ===== */
.circuit-list {
  display: grid;
  gap: 16px;
}

.circuit-item {
  background: rgba(255,255,255,0.03);
  border: 1px solid var(--border-color);
  border-radius: var(--radius);
  padding: 16px;
  position: relative;
  transition: all 0.3s ease;
}

.circuit-item:hover {
  border-color: rgba(255,255,255,0.15);
  background: rgba(255,255,255,0.05);
}

.circuit-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.circuit-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
  box-shadow: 0 0 10px currentColor;
}

.circuit-name {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  flex: 1;
}

.circuit-body {
  display: flex;
  gap: 24px;
  margin-bottom: 12px;
}

.circuit-config {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.config-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.config-label {
  font-size: 12px;
  color: var(--text-muted);
}

.config-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.circuit-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.stat-num {
  font-size: 18px;
  font-weight: 700;
}

.stat-time {
  font-size: 11px;
  color: var(--text-secondary);
  font-family: 'Fira Code', monospace;
}

.stat-label {
  font-size: 11px;
  color: var(--text-muted);
}

.cb-edit-btn {
  position: absolute;
  top: 16px;
  right: 16px;
}

.mono {
  font-family: 'Fira Code', 'Cascadia Code', monospace;
}

/* ===== 降级策略表格 ===== */
.condition-text {
  font-size: 13px;
  color: var(--text-primary);
}

.fallback-code {
  font-size: 12px;
  color: var(--accent);
  background: rgba(59,130,246,0.08);
  padding: 2px 8px;
  border-radius: 4px;
}

/* ===== 重试策略表格 ===== */
.retry-count {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.code-tag {
  margin-right: 4px;
}

@media (max-width: 768px) {
  .circuit-body {
    flex-direction: column;
    gap: 12px;
  }
}
</style>
