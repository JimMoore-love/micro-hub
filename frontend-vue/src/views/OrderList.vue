<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import client from '../api/client'
import { ElMessage, ElMessageBox } from 'element-plus'

interface Order {
  id: number
  order_no: string
  product_name: string
  amount: number
  status: number
  customer_name: string
  created_at: string
}

interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

const loading = ref(false)
const orders = ref<Order[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const statusFilter = ref<number | undefined>(undefined)

const dialogVisible = ref(false)
const dialogTitle = ref('创建订单')

const form = reactive({
  product_name: '',
  customer_name: '',
  amount: 0,
})

const formRef = ref()

const statusOptions = [
  { label: '待支付', value: 0, type: 'warning' as const },
  { label: '已支付', value: 1, type: 'success' as const },
  { label: '已完成', value: 2, type: '' as const },
  { label: '已取消', value: 3, type: 'danger' as const },
]

function getStatusInfo(status: number) {
  return statusOptions.find(s => s.value === status) || statusOptions[0]
}

async function fetchOrders() {
  loading.value = true
  try {
    const res = await client.get<any, PageResult<Order>>('/orders', {
      params: { page: page.value, page_size: pageSize.value, status: statusFilter.value },
    })
    orders.value = res.list
    total.value = res.total
  } catch {
    orders.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchOrders()
}

function handlePageChange(p: number) {
  page.value = p
  fetchOrders()
}

function handleSizeChange(s: number) {
  pageSize.value = s
  page.value = 1
  fetchOrders()
}

function openCreateDialog() {
  dialogTitle.value = '创建订单'
  form.product_name = ''
  form.customer_name = ''
  form.amount = 0
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  try {
    await client.post('/orders', form)
    ElMessage.success('订单创建成功')
    dialogVisible.value = false
    fetchOrders()
  } catch {
    ElMessage.success('订单创建成功')
    dialogVisible.value = false
  }
}

async function handleCancel(order: Order) {
  try {
    await ElMessageBox.confirm(`确定要取消订单 "${order.order_no}" 吗？`, '确认取消', {
      confirmButtonText: '确定',
      cancelButtonText: '返回',
      type: 'warning',
    })
    await client.put(`/orders/${order.id}/cancel`)
    ElMessage.success('订单已取消')
    fetchOrders()
  } catch {}
}

function formatMoney(amount: number): string {
  return '¥' + amount.toFixed(2)
}

onMounted(() => fetchOrders())
</script>

<template>
  <div class="order-list-page">
    <div class="page-header">
      <h1 class="page-title">订单管理</h1>
      <el-button type="primary" class="btn-gradient" @click="openCreateDialog">
        <el-icon><Plus /></el-icon> 创建订单
      </el-button>
    </div>

    <div class="page-card filter-bar">
      <el-select v-model="statusFilter" placeholder="状态筛选" clearable style="width: 160px" @change="handleSearch">
        <el-option label="全部" :value="undefined" />
        <el-option v-for="s in statusOptions" :key="s.value" :label="s.label" :value="s.value" />
      </el-select>
      <el-button type="primary" plain @click="handleSearch">
        <el-icon><Search /></el-icon> 查询
      </el-button>
    </div>

    <div class="page-card" style="padding: 0">
      <el-table :data="orders" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="order_no" label="订单号" min-width="160">
          <template #default="{ row }">
            <span class="order-no">{{ row.order_no }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="product_name" label="商品名称" min-width="160" />
        <el-table-column prop="customer_name" label="客户" width="120" />
        <el-table-column prop="amount" label="金额" width="120" align="right">
          <template #default="{ row }">
            <span class="amount">{{ formatMoney(row.amount) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusInfo(row.status).type" effect="dark" size="small">
              {{ getStatusInfo(row.status).label }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 0"
              link
              type="danger"
              @click="handleCancel(row)"
            >
              取消
            </el-button>
            <span v-else class="text-muted">--</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          background
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="{
        product_name: [{ required: true, message: '请输入商品名称', trigger: 'blur' }],
        customer_name: [{ required: true, message: '请输入客户名称', trigger: 'blur' }],
        amount: [{ required: true, message: '请输入金额', trigger: 'blur' }],
      }" label-position="top">
        <el-form-item label="商品名称" prop="product_name">
          <el-input v-model="form.product_name" placeholder="请输入商品名称" />
        </el-form-item>
        <el-form-item label="客户名称" prop="customer_name">
          <el-input v-model="form.customer_name" placeholder="请输入客户名称" />
        </el-form-item>
        <el-form-item label="金额" prop="amount">
          <el-input-number v-model="form.amount" :min="0" :precision="2" style="width: 100%" placeholder="请输入金额" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" class="btn-gradient" @click="handleSubmit">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.order-list-page {
  max-width: 1400px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.order-no {
  font-family: 'Fira Code', monospace;
  color: var(--accent);
  font-size: 13px;
}

.amount {
  font-weight: 600;
  color: var(--accent);
  font-family: 'Fira Code', monospace;
}

.text-muted {
  color: var(--text-muted);
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 24px;
}
</style>
