<script setup lang="ts">
import { ref, onMounted } from 'vue'
import client from '../api/client'
import { aiApi } from '../api/ai'
import type { AIUsage } from '../api/ai'
import { observabilityApi, serviceApi } from '@/api/platform'

interface DashboardStats {
  userCount: number
  orderCount: number
  aiTokens: number
  aiCost: number
}

const stats = ref<DashboardStats>({
  userCount: 0,
  orderCount: 0,
  aiTokens: 0,
  aiCost: 0,
})

const loading = ref(true)

const orderTrendData = ref<number[]>([120, 200, 150, 80, 70, 110, 130])
const aiUsageTrend = ref<number[]>([50, 100, 80, 160, 140, 180, 200])
const services = ref<any[]>([])

const statCards = [
  {
    key: 'userCount',
    title: '用户总数',
    icon: 'User',
    color: '#3b82f6',
    bg: 'rgba(59, 130, 246, 0.1)',
    suffix: '',
  },
  {
    key: 'orderCount',
    title: '订单总数',
    icon: 'Tickets',
    color: '#8b5cf6',
    bg: 'rgba(139, 92, 246, 0.1)',
    suffix: '',
  },
  {
    key: 'aiTokens',
    title: 'AI Token 用量',
    icon: 'Cpu',
    color: '#22c55e',
    bg: 'rgba(34, 197, 94, 0.1)',
    suffix: '',
  },
  {
    key: 'aiCost',
    title: 'AI 费用',
    icon: 'Money',
    color: '#f59e0b',
    bg: 'rgba(245, 158, 11, 0.1)',
    suffix: ' USD',
  },
]

// 最近活动
const recentActivities = ref([
  { text: '用户 "张三" 注册了新账户', time: '2 分钟前', type: 'user' },
  { text: '订单 #ORD-20240001 已完成', time: '15 分钟前', type: 'order' },
  { text: 'AI 对话消耗 1,240 tokens', time: '30 分钟前', type: 'ai' },
  { text: '系统设置已更新', time: '1 小时前', type: 'system' },
  { text: '用户 "李四" 登录了系统', time: '2 小时前', type: 'user' },
])

async function fetchStats() {
  loading.value = true
  try {
    const [usersRes, ordersRes, usageRes, metricsRes, servicesRes] = await Promise.allSettled([
      client.get('/users', { params: { page: 1, page_size: 1 } }),
      client.get('/orders', { params: { page: 1, page_size: 1 } }),
      aiApi.getUsage(),
      observabilityApi.getMetrics(),
      serviceApi.list(),
    ])

    if (usersRes.status === 'fulfilled') {
      stats.value.userCount = usersRes.value.total
    }
    if (ordersRes.status === 'fulfilled') {
      stats.value.orderCount = ordersRes.value.total
    }
    if (usageRes.status === 'fulfilled') {
      stats.value.aiTokens = usageRes.value.total_tokens
      stats.value.aiCost = usageRes.value.total_cost
    } else if (metricsRes.status === 'fulfilled' && metricsRes.value) {
      stats.value.aiTokens = metricsRes.value.ai_tokens
    }
    if (metricsRes.status === 'fulfilled' && metricsRes.value) {
      if (metricsRes.value.trend && metricsRes.value.trend.length > 0) {
        aiUsageTrend.value = metricsRes.value.trend
      }
    }
    if (servicesRes.status === 'fulfilled' && servicesRes.value) {
      services.value = servicesRes.value
    }
  } catch {} finally {
    loading.value = false
  }
}

function getActivityIcon(type: string): string {
  const map: Record<string, string> = {
    user: 'User',
    order: 'Tickets',
    ai: 'Cpu',
    system: 'Setting',
  }
  return map[type] || 'InfoFilled'
}

function getActivityColor(type: string): string {
  const map: Record<string, string> = {
    user: '#3b82f6',
    order: '#8b5cf6',
    ai: '#22c55e',
    system: '#f59e0b',
  }
  return map[type] || '#94a3b8'
}

function formatNumber(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n.toString()
}

onMounted(() => fetchStats())
</script>

<template>
  <div class="dashboard-page" v-loading="loading">
    <h1 class="page-title">仪表盘</h1>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div
        v-for="card in statCards"
        :key="card.key"
        class="stat-card"
      >
        <div class="stat-icon" :style="{ background: card.bg, color: card.color }">
          <el-icon :size="22"><component :is="card.icon" /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-label">{{ card.title }}</div>
          <div class="stat-value">
            {{ card.key === 'aiCost' ? '$' + (stats[card.key] as number).toFixed(2) : formatNumber(stats[card.key] as number) }}
          </div>
        </div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-grid">
      <div class="page-card chart-card">
        <div class="page-card-title">订单趋势（近7天）</div>
        <div class="mini-bar-chart">
          <div
            v-for="(val, idx) in orderTrendData"
            :key="idx"
            class="bar-col"
          >
            <div
              class="bar-fill"
              :style="{
                height: (val / 200 * 100) + '%',
                background: 'linear-gradient(180deg, #3b82f6, rgba(59,130,246,0.2))',
              }"
            ></div>
            <span class="bar-label">D{{ idx + 1 }}</span>
          </div>
        </div>
      </div>

      <div class="page-card chart-card">
        <div class="page-card-title">AI Token 消耗趋势（近7天）</div>
        <div class="mini-bar-chart">
          <div
            v-for="(val, idx) in aiUsageTrend"
            :key="idx"
            class="bar-col"
          >
            <div
              class="bar-fill"
              :style="{
                height: (val / 200 * 100) + '%',
                background: 'linear-gradient(180deg, #22c55e, rgba(34,197,94,0.2))',
              }"
            ></div>
            <span class="bar-label">D{{ idx + 1 }}</span>
          </div>
        </div>
      </div>

      <!-- 最近活动 -->
      <div class="page-card" style="grid-column: 1 / -1">
        <div class="page-card-title">最近活动</div>
        <div class="activity-list">
          <div
            v-for="(item, idx) in recentActivities"
            :key="idx"
            class="activity-item"
          >
            <div
              class="activity-dot"
              :style="{ background: getActivityColor(item.type) }"
            >
              <el-icon :size="12"><component :is="getActivityIcon(item.type)" /></el-icon>
            </div>
            <span class="activity-text">{{ item.text }}</span>
            <span class="activity-time">{{ item.time }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-page {
  max-width: 1400px;
}

/* 统计卡片网格 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: all 0.3s ease;
}

.stat-card:hover {
  background: var(--bg-card-hover);
  border-color: rgba(59, 130, 246, 0.2);
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
  font-family: 'Fira Code', monospace;
}

/* 图表网格 */
.charts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.chart-card {
  min-height: 240px;
}

.mini-bar-chart {
  display: flex;
  align-items: flex-end;
  gap: 20px;
  height: 180px;
  padding: 8px 0;
}

.bar-col {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
}

.bar-fill {
  width: 100%;
  max-width: 50px;
  border-radius: 6px 6px 0 0;
  transition: height 0.5s ease;
}

.bar-label {
  font-size: 12px;
  color: var(--text-muted);
}

/* 活动列表 */
.activity-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.activity-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-color);
}

.activity-item:last-child {
  border-bottom: none;
}

.activity-dot {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
}

.activity-text {
  flex: 1;
  font-size: 14px;
  color: var(--text-primary);
}

.activity-time {
  font-size: 12px;
  color: var(--text-muted);
  white-space: nowrap;
}

@media (max-width: 1100px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .charts-grid {
    grid-template-columns: 1fr;
  }
}
</style>
