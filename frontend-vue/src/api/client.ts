import axios from 'axios'
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse, AxiosError } from 'axios'
import { ElMessage } from 'element-plus'

const client: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器：注入 X-Tenant-ID 和 Authorization
// 直接从 localStorage 读取，避免与 Pinia store 的循环依赖
client.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const tenantId = localStorage.getItem('tenantId')
    const token = localStorage.getItem('token')

    if (tenantId) {
      config.headers['X-Tenant-ID'] = tenantId
    }
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`
    }

    return config
  },
  (error: AxiosError) => {
    return Promise.reject(error)
  }
)

// 响应拦截器：统一错误处理 + 提取 data
client.interceptors.response.use(
  (response: AxiosResponse) => {
    const body = response.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code !== 0 && body.code !== 200) {
        ElMessage.error(body.message || '请求失败')
        return Promise.reject(new Error(body.message || '请求失败'))
      }
      return body.data ?? body
    }
    return body
  },
  (error: AxiosError) => {
    const status = error.response?.status
    let message = '网络错误，请稍后重试'

    switch (status) {
      case 400:
        message = '请求参数错误'
        break
      case 401:
        message = '登录已过期，请重新登录'
        localStorage.removeItem('token')
        localStorage.removeItem('username')
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
        break
      case 403:
        message = '没有权限访问'
        break
      case 404:
        message = '请求的资源不存在'
        break
      case 500:
        message = '服务器内部错误'
        break
    }

    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export default client
