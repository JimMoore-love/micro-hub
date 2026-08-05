# 微服务项目问题总结报告

> 生成时间：2026-08-03
> 分析范围：api-server / backend-go / backend-laravel / frontend-vue / docker-compose

---

## 一、最严重问题：三套后端并存，架构身份不明

项目里存在 **三个后端目录**，职责重叠、互不兼容，这是所有混乱的根源。

| 目录 | 身份 | 代码状态 | 是否在用 |
|------|------|----------|----------|
| `api-server/` | 单体 Go 后端（Gin+GORM） | 3文件2400行，可编译运行 | ✅ 唯一活跃 |
| `backend-go/` | 微服务架构脚手架（gateway/user/order/ai） | go.mod 缺关键依赖，proto 未生成，**无法编译** | ❌ 半成品 |
| `backend-laravel/` | Laravel BFF 层 | 仅6个PHP文件，无 routes/migrations，对接的是 backend-go 不是 api-server | ❌ 废弃 |

**具体矛盾**：
- `backend-laravel/ApiService.php` 调用 `/api/v1/admin/*`，但 api-server 根本没实现这些端点
- `backend-go` 的数据模型是 User/Order 体系，api-server 是 Service/Tenant 体系，**两套ORM模型并行存在**
- 租户管理在三个目录里各写了一套，互不兼容
- 前端 `vite.config.ts` 代理到 `localhost:8081`（api-server），与 docker-compose 中的微服务群（gateway=8080/user=8081）端口矛盾

---

## 二、后端代码结构问题

### 2.1 handlers.go — 1861行的"上帝文件"
单文件包含 **63个函数**，覆盖 7 大业务域 + 数据访问 + 外部系统集成：

| 模块 | 行号区间 | 问题 |
|------|----------|------|
| 工具/外部抓取 | 21-148 | Prometheus/Jaeger/Loki 抓取逻辑混在业务文件里 |
| 服务治理 | 149-645 | 11个函数，服务发现+注册+健康检查+事件全混一起 |
| API 网关 | 647-786 | 6个函数 |
| 流量管理 | 787-822 | 4个函数 |
| 租户管理 | 823-941 | 8个函数 |
| AI 供应商 | 942-1108 | 10个函数 |
| 校对 API | 1109-1278 | 4个函数 |
| 可观测性 | 1279-1861 | 7个函数 |

**应拆分为**：`handler/service.go`、`handler/gateway.go`、`handler/tenant.go`、`handler/ai.go`、`handler/observability.go` 等独立文件。

### 2.2 无分层架构
- **无 repository 层**：handler 直接操作全局 `db`/`rdb` 变量做 CRUD
- **无 service 层**：业务逻辑全写在 handler 里
- `db.go` 同时包含：GORM 模型定义(17-129) + initDB(140) + initRedis(168) + seedData(185-336，170行硬编码种子数据)

### 2.3 中间件链极弱
`main.go` 仅有 `gin.Default()` + 内联 CORS（29-38行），**无 JWT/鉴权/限流/日志/租户解析中间件**。而 `backend-go/internal/middleware/` 里写了 auth/rate_limit/metrics 但从未启用。

---

## 三、前端架构问题

### 3.1 API 层混乱
- `platform.ts`（401行）是**巨型文件**，把 6 个不相关域（service/gateway/traffic/tenant/aiProvider/proofread/observability）硬塞一起
- `ai.ts:45` 的 `chatStream` 直接用原生 `fetch`，绕过 `client`，导致拦截器/错误处理/token注入全部重复实现
- **4处绕过 API 层**：`Dashboard.vue`、`OrderList.vue`、`Observability.vue` 直接调 `client.get`，orders 域根本没有 API 模块

### 3.2 Store 严重缺失
- 14个视图，**只有 1 个 store**（`user.ts`，43行，只管 token）
- 所有列表数据、表单状态、loading 全部散落在各 .vue 内部 ref，跨视图无法共享

### 3.3 视图过大，零复用组件
- **7/14 视图超过 600 行**：AIProviders(925)、Topology(858)、Services(772)、Proofread(739)、Observability(703)、AIChat(629)、Tenants(617)
- `src/components/` 目录**完全为空**——表格/弹窗/卡片全部在各视图内重复手写
- 仅 1 个 composable（`useTenant.ts`），且用 `window.location.reload()` 切租户，属反模式

### 3.4 路由无守卫
- `router/index.ts` 全文 31 行，**完全无 beforeEach/meta 鉴权**，却存在 token 登录体系，未登录可直接访问所有页
- `OrderList.vue`、`Settings.vue` 作为**孤儿文件未注册路由**

### 3.5 类型定义散落
- 11 个视图在 `<script>` 内 inline 定义 `interface`
- `platform.ts` 接口与视图内联接口字段命名不一致（如 `rateLimit` vs `rate_limit`、`tenantRouting` vs `tenant_routing`）

---

## 四、服务发现机制问题（用户最困惑的点）

### 4.1 声称 Consul，实际是 TCP 端口扫描
- api-server 的 import **不含任何 consul 包**
- `Service.ConsulID`（db.go:30）只是 MySQL 字符串列，值由 `fmt.Sprintf("discovered-%s", svcID)` 硬编码生成
- `backend-go/pkg/consul/registry.go` 存在但 api-server 从未调用，微服务自身也不注册 Consul
- **所谓"Consul 服务发现"实际只是 TCP 端口扫描**

### 4.2 服务发现的真实度
| 机制 | 真实度 | 说明 |
|------|--------|------|
| TCP 端口探测 | ✅ 真实 | `net.DialTimeout("tcp", "127.0.0.1:port", 1500ms)` |
| HTTP 健康检查 | ✅ 真实 | `GET http://address/health`（仅TCP通过后） |
| MySQL 健康检查 | ❌ 假的 | 只是 `if svc.ID == "mysql" && tcpOK { mysqlStatus = "reachable" }`，无协议握手 |
| 服务元信息 | ❌ 硬编码 | 来自 `knownPorts` 映射表（24个端口），非真实探测 |
| 服务事件 | ❌ 写死 | `handleServiceEvents` 完全是硬编码事件列表 |

### 4.3 端口冲突 bug
- `db.go:198-199` 中 `gateway` 与 `user-service` 种子数据**都用 8081 端口**
- 服务发现按 `port + host=127.0.0.1` 查库，无法区分这两个服务

### 4.4 只扫描本机
- `scanPorts` 只扫描 `127.0.0.1`，不扫描子网，无法发现局域网内其他机器的服务

---

## 五、架构声称与实际不符

| 项目声称 | 实际情况 |
|----------|----------|
| Consul 服务发现 | TCP 端口扫描本机 24 个端口 |
| gRPC-Gateway | 只有 Gin REST，无任何 gRPC import |
| 微服务架构 | api-server 是单体，docker-compose 里的微服务群与它平行未对接 |
| 多租户隔离 | 只有 tenant_id 字段传递，无 Schema 级隔离 |
| Prometheus/Jaeger/Loki 可观测性 | 不可达时回退的"模拟数据"是硬编码的固定 trace_id 和随机数 |

### docker-compose 与 api-server 的平行问题
- docker-compose 定义了 user-service(8081)/order-service(8082)/ai-service(8083)/gateway(8080)
- 但 **api-server 从不调用它们**，只扫描本机端口、读写自己的 MySQL
- 两者是平行的两套架构：compose 里的微服务面向 backend-go，前端实际对接的是 api-server

---

## 六、可观测性模拟数据质量低

| 数据源 | 回退模拟质量 | 问题 |
|--------|-------------|------|
| Metrics | 低 | 硬编码 `request_count=12500`，trend 是 `800+rand(500)` 随机填充，**即便 Prometheus 可达，trend 仍是随机** |
| Traces | 低 | 10条写死 trace，固定 trace_id（如 `trace-a1b2c3d4e5f6`），固定 span 结构 |
| Logs | 低 | 15条写死日志，按 keyword/service/level 做内存过滤 |

---

## 七、问题优先级排序

### P0 — 架构身份必须明确
1. **决定 api-server vs backend-go 的去留**：要么完善 backend-go 替换 api-server，要么删除 backend-go 和 backend-laravel，明确 api-server 是唯一后端
2. **docker-compose 与 api-server 对齐**：要么让 api-server 真正调用 compose 里的微服务，要么删除 compose 中的微服务定义

### P1 — 代码结构必须重构
3. **拆分 handlers.go**：1861行按业务域拆成 7 个文件
4. **引入分层架构**：repository → service → handler
5. **拆分 platform.ts**：按域拆成 7 个 API 模块
6. **补充 Pinia store**：每个业务域至少一个 store
7. **抽取公共组件**：表格、弹窗、卡片

### P2 — 功能真实性
8. **服务发现**：要么真对接 Consul，要么文档明确说"本机端口扫描"不叫 Consul
9. **MySQL 健康检查**：实现真正的 mysqladmin ping 或协议握手
10. **路由守卫**：补 beforeEach 鉴权
11. **修复端口冲突**：gateway 和 user-service 不能都用 8081

### P3 — 代码质量
12. 类型定义集中到 `types/` 目录
13. 修复 4 处绕过 API 层的直接 fetch
14. 清理孤儿视图（OrderList/Settings 未注册路由）
