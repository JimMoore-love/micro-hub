import client from './client'

// ==================== 服务治理 API ====================

export interface ServiceInfo {
  id: string
  name: string
  type: 'gateway' | 'service' | 'infra' | 'observability' | 'custom'
  port: number
  host: string
  status: 'healthy' | 'warning' | 'critical' | 'unreachable'
  version: string
  qps: number
  p95: number
  error_rate: number
  instances: number
  schema: string
  dependencies: string[]
  consul_id: string
  registered_at: string
  source: 'seed' | 'discovered' | 'manual'
  last_checked: string
  desc_source: string
  start_cmd: string
}

export interface ServiceEvent {
  time: string
  service: string
  action: 'register' | 'deregister' | 'health_change'
  detail: string
}

export interface DiscoverResult {
  scan_results: Array<{
    port: number
    name: string
    expected: string
    type: string
    version: string
    status: string
    latency: number
    address: string
    known: boolean
    source: string
    start_cmd: string
    match_type: string
    process?: {
      pid: number
      process_name: string
      exec_path: string
      is_system_proc: boolean
    }
  }>
  new_registered: ServiceInfo[]
  total_reachable: number
}

export interface RegisterResult {
  service: ServiceInfo
  port_reachable: boolean
  dependencies: string[]
}

export const serviceApi = {
  list() {
    return client.get<any, ServiceInfo[]>('/services')
  },
  get(id: string) {
    return client.get<any, ServiceInfo>(`/services/${id}`)
  },
  register(data: { name: string; type?: string; port: number; host?: string; version?: string; dependencies?: string[] }) {
    return client.post<any, RegisterResult>('/services', data)
  },
  discover(extraPorts?: number[]) {
    return client.post<any, DiscoverResult>('/services/discover', { extra_ports: extraPorts || [] })
  },
  refreshHealth() {
    return client.post<any, { checked: number; updates: Array<{ id: string; name: string; port: number; old_status: string; new_status: string; latency: number }> }>('/services/refresh-health')
  },
  update(id: string, data: Partial<ServiceInfo>) {
    return client.put<any, ServiceInfo>(`/services/${id}`, data)
  },
  delete(id: string) {
    return client.delete(`/services/${id}`)
  },
  health(id: string) {
    return client.get<any, { service_id: string; overall: string; checks: Array<{ name: string; status: string; latency_ms?: number; address?: string; url?: string; last_check: string }> }>(`/services/${id}/health`)
  },
  events() {
    return client.get<any, ServiceEvent[]>('/services/events')
  },
}

// ==================== API 网关 API ====================

export interface RouteRule {
  id: string
  path: string
  upstream: string
  methods: string[]
  rate_limit: number | null
  tenant_routing: boolean
  status: 'enabled' | 'disabled'
}

export interface MiddlewareConfig {
  cors: { allowed_origins: string[]; methods: string[]; headers: string[] }
  jwt: { secret: string; expiry: number; excluded_paths: string[] }
  rate_limit: { global_rate: number; per_tenant_rate: number; burst_size: number }
  tenant_routing: { header_key: string; subdomain_mapping: Record<string, string> }
}

export const gatewayApi = {
  listRoutes() {
    return client.get<any, RouteRule[]>('/gateway/routes')
  },
  createRoute(data: Partial<RouteRule>) {
    return client.post<any, RouteRule>('/gateway/routes', data)
  },
  updateRoute(id: string, data: Partial<RouteRule>) {
    return client.put<any, RouteRule>(`/gateway/routes/${id}`, data)
  },
  deleteRoute(id: string) {
    return client.delete(`/gateway/routes/${id}`)
  },
  getMiddlewareConfig() {
    return client.get<any, MiddlewareConfig>('/gateway/middleware')
  },
  updateMiddlewareConfig(data: Partial<MiddlewareConfig>) {
    return client.put<any, MiddlewareConfig>('/gateway/middleware', data)
  },
}

// ==================== 流量管理 API ====================

export interface CircuitBreaker {
  service: string
  state: 'closed' | 'open' | 'half_open'
  failure_threshold: number
  open_duration: number
  half_open_probes: number
}

export interface DegradationRule {
  service: string
  condition: string
  response: string
  enabled: boolean
}

export interface RetryPolicy {
  service: string
  max_retries: number
  interval: number
  retryable_codes: number[]
}

export const trafficApi = {
  listCircuitBreakers() {
    return client.get<any, CircuitBreaker[]>('/traffic/circuit-breakers')
  },
  updateCircuitBreaker(service: string, data: Partial<CircuitBreaker>) {
    return client.put<any, CircuitBreaker>(`/traffic/circuit-breakers/${service}`, data)
  },
  listDegradationRules() {
    return client.get<any, DegradationRule[]>('/traffic/degradation')
  },
  listRetryPolicies() {
    return client.get<any, RetryPolicy[]>('/traffic/retry')
  },
}

// ==================== 租户管理 API ====================

export interface TenantInfo {
  id: string
  name: string
  schema: string
  users: number
  quota: number
  used: number
  status: 'active' | 'frozen'
  plan: 'free' | 'standard' | 'pro'
  created_at: string
  api_keys: Array<{ key: string; created_at: string; status: string }>
  redis_prefix: string
  db_tables: number
  db_records: number
}

export const tenantApi = {
  list() {
    return client.get<any, TenantInfo[]>('/tenants')
  },
  get(id: string) {
    return client.get<any, TenantInfo>(`/tenants/${id}`)
  },
  create(data: Partial<TenantInfo>) {
    return client.post<any, TenantInfo>('/tenants', data)
  },
  update(id: string, data: Partial<TenantInfo>) {
    return client.put<any, TenantInfo>(`/tenants/${id}`, data)
  },
  freeze(id: string) {
    return client.put<any, TenantInfo>(`/tenants/${id}/freeze`)
  },
  unfreeze(id: string) {
    return client.put<any, TenantInfo>(`/tenants/${id}/unfreeze`)
  },
  createApiKey(id: string) {
    return client.post<any, { key: string }>(`/tenants/${id}/api-keys`)
  },
  deleteApiKey(id: string, key: string) {
    return client.delete(`/tenants/${id}/api-keys/${key}`)
  },
}

// ==================== AI 供应商管理 API ====================

export interface AIProvider {
  id: string
  name: string
  type: 'llm' | 'proofread' | 'translate' | 'image'
  icon: string
  status: 'connected' | 'disconnected' | 'testing'
  requests: number
  latency: number
  cost_per_1k: number
  models: string[]
  api_key: string
  endpoint: string
}

export interface RoutingRule {
  id: string
  tenant_id: string
  provider_id: string
  priority: number
  condition: string
  enabled: boolean
}

export interface AIUsageDetail {
  date: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  cost: number
  request_count: number
  tenant_distribution: Array<{ tenant: string; percentage: number }>
}

export const aiProviderApi = {
  listProviders() {
    return client.get<any, AIProvider[]>('/ai/providers')
  },
  getProvider(id: string) {
    return client.get<any, AIProvider>(`/ai/providers/${id}`)
  },
  createProvider(data: Partial<AIProvider>) {
    return client.post<any, AIProvider>('/ai/providers', data)
  },
  updateProvider(id: string, data: Partial<AIProvider>) {
    return client.put<any, AIProvider>(`/ai/providers/${id}`, data)
  },
  deleteProvider(id: string) {
    return client.delete(`/ai/providers/${id}`)
  },
  listRoutingRules() {
    return client.get<any, RoutingRule[]>('/ai/routing-rules')
  },
  createRoutingRule(data: Partial<RoutingRule>) {
    return client.post<any, RoutingRule>('/ai/routing-rules', data)
  },
  updateRoutingRule(id: string, data: Partial<RoutingRule>) {
    return client.put<any, RoutingRule>(`/ai/routing-rules/${id}`, data)
  },
  getProviderUsage(id: string) {
    return client.get<any, AIUsageDetail>(`/ai/providers/${id}/usage`)
  },
  healthCheck(id: string) {
    return client.get<any, { checks: Array<{ time: string; status: string; latency: number }> }>(`/ai/providers/${id}/health`)
  },
}

// ==================== 校对 API ====================

export interface ProofreadRequest {
  text: string
  language?: string
  checks?: string[]
}

export interface ProofreadError {
  original: string
  type: string
  suggestion: string
  confidence: number
  position: [number, number]
}

export interface ProofreadResult {
  corrected_text: string
  errors: ProofreadError[]
  provider: string
  latency: number
  tokens: number
  cost: number
}

export interface ProofreadLog {
  id: string
  time: string
  tenant_id: string
  text_length: number
  error_count: number
  latency: number
  provider: string
  status: 'success' | 'timeout' | 'rate_limit' | 'error'
}

export const proofreadApi = {
  check(data: ProofreadRequest) {
    return client.post<any, ProofreadResult>('/proofread', data)
  },
  getConfig() {
    return client.get<any, { provider: AIProvider; routing: RoutingRule[] }>('/proofread/config')
  },
  updateConfig(data: any) {
    return client.put<any, any>('/proofread/config', data)
  },
  listLogs(params?: { tenant_id?: string; status?: string }) {
    return client.get<any, ProofreadLog[]>('/proofread/logs', { params })
  },
  getStats() {
    return client.get<any, { today_calls: number; avg_latency: number; success_rate: number; today_cost: number }>('/proofread/stats')
  },
}

// ==================== 可观测性 API ====================

export interface MetricData {
  request_count: number
  p95_latency: number
  p99_latency: number
  error_rate: number
  ai_tokens: number
  active_connections: number
  trend: number[]
}

export interface TraceInfo {
  trace_id: string
  path: string
  tenant_id: string
  total_latency: number
  services: number
  status: 'success' | 'error'
  spans: Array<{ service: string; duration: number; start: number }>
}

export interface LogEntry {
  timestamp: string
  level: 'ERROR' | 'WARN' | 'INFO'
  service: string
  message: string
  trace_id: string
}

export interface AlertRule {
  id: string
  name: string
  metric: string
  condition: string
  threshold: string
  duration: string
  notify: string[]
  status: 'enabled' | 'disabled'
}

export interface AlertEvent {
  id: string
  time: string
  rule_name: string
  level: 'critical' | 'warning' | 'info'
  trigger_value: string
  threshold: string
  duration: string
  status: 'firing' | 'resolved'
  handler: string
}

export const observabilityApi = {
  getMetrics() {
    return client.get<any, MetricData>('/observability/metrics')
  },
  listTraces(params?: { service?: string; limit?: number }) {
    return client.get<any, TraceInfo[]>('/observability/traces', { params })
  },
  searchLogs(params: { keyword?: string; service?: string; level?: string }) {
    return client.get<any, LogEntry[]>('/observability/logs', { params })
  },
  listAlertRules() {
    return client.get<any, AlertRule[]>('/observability/alerts/rules')
  },
  createAlertRule(data: Partial<AlertRule>) {
    return client.post<any, AlertRule>('/observability/alerts/rules', data)
  },
  listAlertEvents() {
    return client.get<any, AlertEvent[]>('/observability/alerts/events')
  },
}
