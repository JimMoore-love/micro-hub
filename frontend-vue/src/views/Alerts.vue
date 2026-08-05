<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { observabilityApi } from '@/api/platform'
import { ElMessage } from 'element-plus'

// ==================== 统计卡片 ====================
const statCards = [
  { title: '活跃规则', value: '8', icon: 'Bell', color: '#3b82f6', bg: 'rgba(59,130,246,0.1)' },
  { title: '今日告警', value: '3', icon: 'Warning', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)' },
  { title: '未处理', value: '1', icon: 'CircleClose', color: '#ef4444', bg: 'rgba(239,68,68,0.1)' },
  { title: '已恢复', value: '2', icon: 'CircleCheck', color: '#22c55e', bg: 'rgba(34,197,94,0.1)' },
]

// ==================== 告警规则 ====================
interface AlertRule {
  id: string
  name: string
  metric: string
  operator: string
  threshold: string
  duration: string
  notify: string[]
  enabled: boolean
}

const rules = ref<AlertRule[]>([
  { id: 'r1', name: 'ai-service延迟>200ms', metric: 'P95延迟', operator: '>', threshold: '200ms', duration: '5min', notify: ['钉钉', '邮件'], enabled: true },
  { id: 'r2', name: 'gateway错误率>1%', metric: 'error_rate', operator: '>', threshold: '1%', duration: '3min', notify: ['钉钉'], enabled: true },
  { id: 'r3', name: 'NATS连接断开', metric: 'health', operator: '==', threshold: 'down', duration: '0min', notify: ['钉钉', '短信'], enabled: true },
  { id: 'r4', name: 'postgres慢查询>500ms', metric: 'query_time', operator: '>', threshold: '500ms', duration: '10min', notify: ['邮件'], enabled: true },
  { id: 'r5', name: 'AI配额超80%', metric: 'usage_percent', operator: '>', threshold: '80%', duration: '30min', notify: ['钉钉'], enabled: true },
  { id: 'r6', name: 'Redis内存>90%', metric: 'memory_usage', operator: '>', threshold: '90%', duration: '15min', notify: ['钉钉', '邮件'], enabled: true },
  { id: 'r7', name: 'Consul节点离线', metric: 'node_count', operator: '<', threshold: '3', duration: '0min', notify: ['钉钉', '短信'], enabled: true },
  { id: 'r8', name: '订单服务QPS异常', metric: 'qps', operator: '>', threshold: '500', duration: '5min', notify: ['邮件'], enabled: false },
])

function toggleRule(rule: AlertRule) {
  rule.enabled = !rule.enabled
  ElMessage.success(`规则已${rule.enabled ? '启用' : '禁用'}`)
}

// ==================== 告警历史 ====================
interface AlertHistory {
  id: string
  time: string
  ruleName: string
  level: 'critical' | 'warning' | 'info'
  triggerValue: string
  threshold: string
  duration: string
  status: 'firing' | 'resolved'
  handler: string
}

const history = ref<AlertHistory[]>([
  { id: 'a1', time: '2024-03-08 14:32:10', ruleName: 'ai-service延迟>200ms', level: 'warning', triggerValue: '245ms', threshold: '200ms', duration: '5min', status: 'firing', handler: '张运维' },
  { id: 'a2', time: '2024-03-08 13:15:00', ruleName: 'gateway错误率>1%', level: 'critical', triggerValue: '1.5%', threshold: '1%', duration: '3min', status: 'resolved', handler: '李开发' },
  { id: 'a3', time: '2024-03-08 11:08:22', ruleName: 'AI配额超80%', level: 'warning', triggerValue: '85%', threshold: '80%', duration: '30min', status: 'resolved', handler: '王运维' },
  { id: 'a4', time: '2024-03-08 09:45:30', ruleName: 'NATS连接断开', level: 'critical', triggerValue: 'down', threshold: 'down', duration: '0min', status: 'resolved', handler: '张运维' },
  { id: 'a5', time: '2024-03-07 18:20:15', ruleName: 'postgres慢查询>500ms', level: 'warning', triggerValue: '680ms', threshold: '500ms', duration: '10min', status: 'resolved', handler: '李开发' },
  { id: 'a6', time: '2024-03-07 16:10:00', ruleName: 'ai-service延迟>200ms', level: 'warning', triggerValue: '280ms', threshold: '200ms', duration: '5min', status: 'resolved', handler: '张运维' },
  { id: 'a7', time: '2024-03-07 14:55:08', ruleName: 'Redis内存>90%', level: 'critical', triggerValue: '92%', threshold: '90%', duration: '15min', status: 'resolved', handler: '王运维' },
  { id: 'a8', time: '2024-03-07 10:30:00', ruleName: 'gateway错误率>1%', level: 'critical', triggerValue: '2.1%', threshold: '1%', duration: '3min', status: 'resolved', handler: '李开发' },
])

const levelColors: Record<string, string> = { critical: '#ef4444', warning: '#f59e0b', info: '#3b82f6' }
const levelLabels: Record<string, string> = { critical: '严重', warning: '警告', info: '信息' }

// ==================== Tab ====================
const activeTab = ref('rules')

// ==================== 新增规则弹窗 ====================
const dialogVisible = ref(false)
const newRule = reactive({
  name: '', metric: '', operator: '>', threshold: '', duration: '5min',
  notify: [] as string[], enabled: true,
})

function openNewRule() {
  Object.assign(newRule, { name: '', metric: '', operator: '>', threshold: '', duration: '5min', notify: [], enabled: true })
  dialogVisible.value = true
}

function createRule() {
  if (!newRule.name || !newRule.metric || !newRule.threshold) {
    ElMessage.warning('请填写完整的告警规则')
    return
  }
  rules.value.push({ id: 'r' + Date.now(), ...newRule })
  dialogVisible.value = false
  ElMessage.success('告警规则已创建')
}

const notifyOptions = ['钉钉', '邮件', '短信', '企业微信']

onMounted(async () => {
  try {
    const [ruleData, eventData] = await Promise.all([
      observabilityApi.listAlertRules(),
      observabilityApi.listAlertEvents(),
    ])
    if (ruleData && ruleData.length > 0) {
      rules.value = ruleData.map(r => ({
        id: r.id,
        name: r.name,
        metric: r.metric,
        operator: r.condition,
        threshold: r.threshold,
        duration: r.duration,
        notify: r.notify,
        enabled: r.status === 'enabled',
      }))
    }
    if (eventData && eventData.length > 0) {
      history.value = eventData.map(e => ({
        id: e.id,
        time: e.time,
        ruleName: e.rule_name,
        level: e.level,
        triggerValue: e.trigger_value,
        threshold: e.threshold,
        duration: e.duration,
        status: e.status,
        handler: e.handler,
      }))
    }
  } catch (e) {
    console.error('Failed to fetch alerts data:', e)
  }
})
</script>

<template>
  <div class="alerts-page">
    <h1 class="page-title">告警规则</h1>

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

    <!-- Tab -->
    <el-tabs v-model="activeTab" class="alert-tabs">
      <!-- Tab 1: 告警规则 -->
      <el-tab-pane label="告警规则" name="rules">
        <div class="tab-toolbar">
          <span class="tab-subtitle">共 {{ rules.length }} 条规则</span>
          <el-button type="primary" size="small" class="btn-gradient" @click="openNewRule">
            <el-icon><Plus /></el-icon> 新增规则
          </el-button>
        </div>

        <div class="page-card" style="padding: 0; overflow: hidden">
          <el-table :data="rules" stripe style="width: 100%" :row-style="{ height: '52px' }">
            <el-table-column prop="name" label="规则名" min-width="200">
              <template #default="{ row }">
                <span class="rule-name">{{ row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="metric" label="指标" width="130">
              <template #default="{ row }">
                <code class="mono metric-tag">{{ row.metric }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="operator" label="条件" width="70" align="center">
              <template #default="{ row }">
                <span class="mono operator">{{ row.operator }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="threshold" label="阈值" width="100" align="center">
              <template #default="{ row }">
                <span class="mono threshold">{{ row.threshold }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="duration" label="持续时间" width="90" align="center">
              <template #default="{ row }">
                <span class="mono text-muted">{{ row.duration }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="notify" label="通知方式" width="160">
              <template #default="{ row }">
                <div class="notify-tags">
                  <el-tag v-for="n in row.notify" :key="n" size="small" type="info" class="notify-tag">{{ n }}</el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="enabled" label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" size="small" @change="toggleRule(row)" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small">编辑</el-button>
                <el-button link type="danger" size="small">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Tab 2: 告警历史 -->
      <el-tab-pane label="告警历史" name="history">
        <div class="tab-toolbar">
          <span class="tab-subtitle">共 {{ history.length }} 条记录</span>
        </div>

        <div class="page-card" style="padding: 0; overflow: hidden">
          <el-table :data="history" stripe style="width: 100%" :row-style="{ height: '52px' }">
            <el-table-column prop="time" label="时间" width="170">
              <template #default="{ row }">
                <span class="mono text-muted">{{ row.time }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="ruleName" label="规则名" min-width="200">
              <template #default="{ row }">
                <span class="rule-name">{{ row.ruleName }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="level" label="级别" width="90" align="center">
              <template #default="{ row }">
                <span class="level-badge" :style="{ color: levelColors[row.level], background: levelColors[row.level] + '15' }">
                  {{ levelLabels[row.level] }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="触发值 / 阈值" width="160" align="center">
              <template #default="{ row }">
                <span class="mono trigger-value" :style="{ color: levelColors[row.level] }">{{ row.triggerValue }}</span>
                <span class="mono text-muted"> / {{ row.threshold }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="duration" label="持续时间" width="90" align="center">
              <template #default="{ row }">
                <span class="mono text-muted">{{ row.duration }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100" align="center">
              <template #default="{ row }">
                <span class="alert-status" :class="row.status">
                  <span class="as-dot" :class="row.status"></span>
                  {{ row.status === 'firing' ? '触发中' : '已恢复' }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="handler" label="处理人" width="100">
              <template #default="{ row }">
                <span class="handler">{{ row.handler }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 新增规则弹窗 -->
    <el-dialog v-model="dialogVisible" title="新增告警规则" width="520px" :close-on-click-modal="false">
      <el-form :model="newRule" label-width="100px" size="default">
        <el-form-item label="规则名" required>
          <el-input v-model="newRule.name" placeholder="如: ai-service延迟>200ms" />
        </el-form-item>
        <el-form-item label="指标" required>
          <el-input v-model="newRule.metric" placeholder="如: P95延迟" />
        </el-form-item>
        <el-form-item label="条件">
          <el-select v-model="newRule.operator" style="width: 100%">
            <el-option label="> 大于" value=">" />
            <el-option label="< 小于" value="<" />
            <el-option label="== 等于" value="==" />
            <el-option label=">= 大于等于" value=">=" />
            <el-option label="<= 小于等于" value="<=" />
          </el-select>
        </el-form-item>
        <el-form-item label="阈值" required>
          <el-input v-model="newRule.threshold" placeholder="如: 200ms" />
        </el-form-item>
        <el-form-item label="持续时间">
          <el-select v-model="newRule.duration" style="width: 100%">
            <el-option label="立即 (0min)" value="0min" />
            <el-option label="3 分钟" value="3min" />
            <el-option label="5 分钟" value="5min" />
            <el-option label="10 分钟" value="10min" />
            <el-option label="15 分钟" value="15min" />
            <el-option label="30 分钟" value="30min" />
          </el-select>
        </el-form-item>
        <el-form-item label="通知方式">
          <el-checkbox-group v-model="newRule.notify">
            <el-checkbox v-for="opt in notifyOptions" :key="opt" :value="opt">{{ opt }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="newRule.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createRule">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.alerts-page { max-width: 1400px; }
.mono { font-family: 'Fira Code', 'Cascadia Code', monospace; }

/* 统计卡片 */
.stats-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px; margin-bottom: 24px;
}
.stat-card {
  background: var(--bg-card); border: 1px solid var(--border-color);
  border-radius: var(--radius-lg); padding: 24px;
  display: flex; align-items: center; gap: 16px;
  transition: all 0.3s ease;
}
.stat-card:hover { background: var(--bg-card-hover); transform: translateY(-2px); }
.stat-icon {
  width: 50px; height: 50px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-info { display: flex; flex-direction: column; gap: 4px; }
.stat-label { font-size: 13px; color: var(--text-muted); }
.stat-value { font-size: 26px; font-weight: 700; color: var(--text-primary); }

/* Tab */
.alert-tabs { --el-tabs-header-height: 40px; }
.tab-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px;
}
.tab-subtitle { font-size: 13px; color: var(--text-muted); }

/* 规则表格 */
.rule-name { font-weight: 600; color: var(--text-primary); }
.metric-tag { font-size: 12px; color: var(--accent); background: rgba(59,130,246,0.08); padding: 2px 8px; border-radius: 4px; }
.operator { font-size: 14px; color: var(--warning); font-weight: 700; }
.threshold { font-size: 13px; color: var(--text-primary); font-weight: 600; }
.text-muted { color: var(--text-muted); font-size: 12px; }

.notify-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.notify-tag { margin: 0; }

/* 告警历史 */
.level-badge {
  display: inline-block; font-size: 12px; font-weight: 600;
  padding: 2px 10px; border-radius: 4px;
}
.trigger-value { font-size: 13px; font-weight: 600; }

.alert-status {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 500;
}
.alert-status.firing { color: var(--danger); }
.alert-status.resolved { color: var(--success); }
.as-dot { width: 8px; height: 8px; border-radius: 50%; }
.as-dot.firing { background: var(--danger); box-shadow: 0 0 6px var(--danger); animation: pulse 1.5s ease infinite; }
.as-dot.resolved { background: var(--success); }

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.handler { font-size: 13px; color: var(--text-secondary); }

@media (max-width: 1100px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
