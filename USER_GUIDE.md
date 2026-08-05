# MicroHub 微服务治理平台 — 用户使用手册

> 版本 1.0 | 2026-07-09

---

## 一、平台概述

MicroHub 是一个**微服务治理平台**，不是商城、不是后台管理面板。它的核心定位是：

**让你看到、管到、用到本机所有微服务的真实状态。**

平台通过**真实端口探测**来发现和监控服务，而不是靠硬编码数据。你本机跑了什么服务，它就能看到什么；服务宕了，它会自动标红；服务上线，它会自动恢复。

### 架构原理

```
┌─────────────────────────────────────────────────────────┐
│                    你的本机 (localhost)                    │
│                                                           │
│  ┌─────┐  ┌─────┐  ┌─────┐  ┌──────┐  ┌──────┐         │
│  │MySQL│  │Redis│  │FRP  │  │MinIO │  │你的9000│         │
│  │3306 │  │6379 │  │7000 │  │9000  │  │ 服务  │         │
│  └─────┘  └─────┘  └─────┘  └──────┘  └──────┘         │
│                                                           │
│          ┌───────────────────────────┐                    │
│          │   MicroHub API Server     │                    │
│          │   (Go + Gin, 端口 8084)   │                    │
│          │                           │                    │
│          │  ┌─── TCP 端口探测 ────┐  │                    │
│          │  │  127.0.0.1:3306 ✓  │  │                    │
│          │  │  127.0.0.1:6379 ✓  │  │                    │
│          │  │  127.0.0.1:7000 ✗  │  │                    │
│          │  │  127.0.0.1:9000 ✓  │  │                    │
│          │  └───────────────────┘  │                    │
│          │                           │                    │
│          │  MySQL (microhub 库)      │ ← 存服务元数据      │
│          │  Redis (metrics 计数)     │ ← 存实时指标        │
│          └───────────────────────────┘                    │
│                                                           │
│          ┌───────────────────────────┐                    │
│          │   Vue3 前端 (端口 3200)    │                    │
│          │   通过 /api 代理 → 8084   │                    │
│          └───────────────────────────┘                    │
└─────────────────────────────────────────────────────────┘
```

**关键原理**：
- **不靠 Consul/K8s 注册** — 直接用 TCP 连接探测 `host:port`
- **不靠 HTTP 健康检查为主** — TCP 端口可达就是最基本的健康标准
- **MySQL 存元数据** — 服务名、类型、版本、来源、依赖关系等持久化在 MySQL
- **Redis 存实时数据** — 请求计数、AI Token、活跃连接数等高频变化数据
- **后台 30 秒轮询** — API Server 每 30 秒对所有已注册服务做 TCP 健康检查，自动更新状态

---

## 二、两种运行环境 — 本地开发 vs 线上生产

**这是理解本项目的关键**：项目有完全不同的两种运行方式，取决于你是在 Windows 本地开发还是在 Linux 服务器上生产部署。

### 2.1 本地开发环境（Windows，当前你用的）

**核心区别：不使用 Podman 容器，所有服务直接跑在 Windows 上。**

```
┌───────────────────────────────────────────────────────────┐
│                  Windows 本机（你的电脑）                      │
│                                                             │
│   ServBay MySQL ──── 3306 ──── 直接进程，不在 Podman 中      │
│   ServBay Redis  ──── 6379 ──── 直接进程，不在 Podman 中      │
│   microhub-api.exe ── 8081 ──── Go 编译的二进制，直接运行     │
│   npx vite        ──── 3200 ──── Node.js 开发服务器          │
│                                                             │
│   ❌ Podman 中看不到任何 fullstack 容器                        │
│   ❌ Jaeger/Prometheus/Loki 没有运行                         │
│   ✅ 可观测性数据用模拟回退（界面照常显示）                       │
│                                                             │
│   启动方式：                                                 │
│   1. cd api-server && ./microhub-api.exe                    │
│   2. cd frontend-vue && npx vite --port 3200               │
│   3. 浏览器打开 http://localhost:3200                        │
└───────────────────────────────────────────────────────────┘
```

**为什么不用 Podman？**
- 你本机已经有 ServBay 提供的 MySQL 和 Redis，不需要再在容器里跑一份
- Go API Server 直接编译成 `.exe` 运行，更快、调试更方便
- Vue 前端用 Vite 开发服务器（热更新），不适合容器化
- Podman 在 Windows 上运行容器有性能开销（需要 WSL 虚拟化层）

**你在 Podman 里看不到容器是正常的**，因为当前所有服务都不在容器里运行。

### 2.2 线上生产环境（Linux 服务器）

**核心区别：所有服务都在 Podman 容器中运行，一键部署。**

```
┌───────────────────────────────────────────────────────────┐
│                  Linux 服务器（生产部署）                       │
│                                                             │
│   Podman 容器：                                              │
│   fullstack-mysql      ──── 3306  ──── MySQL 容器           │
│   fullstack-redis      ──── 6379  ──── Redis 容器           │
│   fullstack-api-server ──── 8081  ──── Go API 容器          │
│   fullstack-jaeger     ──── 16686 ──── Jaeger 链路追踪      │
│   fullstack-otel-lgtm  ──── 3000/9090/3100 ── Grafana+     │
│                              Prometheus+Loki+Tempo 四合一     │
│   fullstack-frontend   ──── 80    ──── Vue 前端容器          │
│   ... 更多微服务容器                                         │
│                                                             │
│   ✅ podman ps 能看到 14+ 个 fullstack 容器                   │
│   ✅ Jaeger/Prometheus/Loki 真实运行                          │
│   ✅ 可观测性数据从真实服务拉取（自动切换）                       │
│                                                             │
│   启动方式：                                                 │
│   1. podman compose up -d                                  │
│   2. 浏览器打开 http://服务器IP                              │
│   或                                                        │
│   1. ./deployments/podman/deploy.sh start                  │
└───────────────────────────────────────────────────────────┘
```

**为什么线上用 Podman？**
- Linux 没有 ServBay，MySQL/Redis 等基础设施必须用容器提供
- 容器化部署保证环境一致性
- Podman Rootless 模式安全性更好（不需要 root 权限）
- Quadlet + systemd 实现开机自启和自动重启

### 2.3 两种环境的对比

| 对比项 | 本地开发（Windows） | 线上生产（Linux） |
|-------|-------------------|-----------------|
| **MySQL** | ServBay 本地服务（已装好的） | Podman 容器 `mysql:8-oracle` |
| **Redis** | ServBay 本地服务（已装好的） | Podman 容器 `redis:7-alpine` |
| **API Server** | `microhub-api.exe` 直接运行 | Podman 容器（自定义 Dockerfile 构建） |
| **前端** | `npx vite --port 3200` 开发服务器 | Podman 容器（nginx 托管编译产物） |
| **Jaeger** | ❌ 未部署，可观测性用模拟数据 | ✅ Podman 容器 `jaegertracing/all-in-one` |
| **Prometheus** | ❌ 未部署，可观测性用模拟数据 | ✅ otel-lgtm 内置（端口 9090） |
| **Loki** | ❌ 未部署，可观测性用模拟数据 | ✅ otel-lgtm 内置（端口 3100） |
| **Grafana** | ❌ 未部署 | ✅ otel-lgtm 内置（端口 3000） |
| **Podman 可见容器** | 无 fullstack 容器（只有其他项目容器） | 14+ 个 fullstack-* 容器 |
| **启动命令** | 手动分别启动 2 个进程 | `podman compose up -d` 一键启动 |
| **可观测性数据** | 模拟回退（界面不变） | 真实数据（自动检测切换） |
| **数据源切换** | `data_source: "simulated"` | `data_source: "prometheus"/"jaeger"/"loki"` |
| **开机自启** | 需手动启动 | Quadlet systemd 自动启动 |
| **调试** | 直接改代码，热更新 | 需要重新构建镜像 |

### 2.4 可观测性数据源智能切换

**这是两种环境的核心衔接点**：API Server 的三个可观测性 handler 自动检测外部服务是否可达：

```
handleGetMetrics()     → 检测 Prometheus (localhost:9090)
                         ├─ 可达 → 从 /api/v1/query 拉真实指标
                         └─ 不可达 → 回退到 Redis 模拟数据

handleListTraces()     → 检测 Jaeger (localhost:16686)
                         ├─ 可达 → 从 /api/traces 拉真实链路
                         └─ 不可达 → 回退到硬编码模拟链路

handleSearchLogs()     → 检测 Loki (localhost:3100)
                         ├─ 可达 → 从 /loki/api/v1/query_range 查真实日志
                         └─ 不可达 → 回退到硬编码模拟日志
```

**这意味着**：
- 在 Windows 本地（没装 Jaeger/Prometheus/Loki）→ 界面照常显示，数据来源标注"模拟"
- 迁移到 Linux 后 `podman compose up` → Jaeger/Prometheus/Loki 自动启动 → API Server 检测到端口可达 → 自动切换到真实数据 → 界面**完全不变**，但数据变成真实的

前端可观测性页面有**数据源状态条**，实时显示：
- 🟡 指标：模拟数据（Prometheus 不可达） 或 🟢 指标：真实数据（Prometheus 可达）
- 🟡 链路：模拟数据（Jaeger 不可达）   或 🟢 链路：真实数据（Jaeger 可达）
- 🟡 日志：模拟数据（Loki 不可达）     或 🟢 日志：真实数据（Loki 可达）

### 2.5 如果想在本地也运行 Jaeger/Prometheus？

可以单独拉镜像运行，不需要整个 docker-compose：

```bash
# 只启动 Jaeger（链路追踪）
podman run -d --name jaeger -p 16686:16686 -p 4317:4317 jaegertracing/all-in-one:latest

# 只启动 Prometheus（指标采集）
podman run -d --name prometheus -p 9090:9090 prom/prometheus:latest

# 只启动 Loki（日志聚合）
podman run -d --name loki -p 3100:3100 grafana/loki:latest
```

启动任何一个后，API Server 会在下一次请求时自动检测到端口可达，切换到真实数据源。不需要改任何代码或重启 API Server。

---

## 三、快速启动（本地开发环境）

### 3.1 启动 API Server

```bash
cd F:\MicroserviceManage\fullstack-app\api-server
./microhub-api.exe
```

启动后会看到：

```
========================================
  MicroHub API Server
  Listening on http://localhost:8081
  MySQL: microhub database
  Redis: localhost:6379
========================================
```

**前提条件**：
- MySQL 必须在运行（`root/root`，`localhost:3306`，数据库 `microhub`）
- Redis 必须在运行（`localhost:6379`）
- 首次启动会自动建表 + 写入种子数据

### 3.2 启动前端

```bash
cd F:\MicroserviceManage\fullstack-app\frontend-vue
npx vite --port 3200
```

浏览器打开 **http://localhost:3200**

### 3.3 验证服务是否连通

```bash
curl http://localhost:8081/health
# → {"status":"ok","time":"2026-07-09T14:30:00+08:00"}

curl http://localhost:8081/api/v1/services
# → 返回所有已注册的服务列表
```

---

## 四、核心功能详解

### 4.1 服务管理 — 三种方式注册服务

服务管理是平台的核心入口。每个服务在系统中有一条记录，包含：

| 字段 | 说明 | 示例 |
|------|------|------|
| `id` | 服务唯一标识 | `mysql`, `frp-server-7000` |
| `name` | 显示名称 | `MySQL`, `FRP Server` |
| `type` | 服务类型 | `gateway/service/infra/observability/custom` |
| `host` | 主机地址 | `127.0.0.1` |
| `port` | 端口号 | `3306`, `6379`, `9000` |
| `status` | 健康状态 | `healthy/warning/critical/unreachable` |
| `source` | 来源 | `seed`（初始种子）/ `discovered`（自动发现）/ `manual`（手动注册） |
| `version` | 版本 | `8.x`, `v0.61.0` |
| `last_checked` | 最后检查时间 | 每 30 秒自动更新 |
| `dependencies` | 依赖的其他服务 ID 列表 | `["redis", "mysql"]` |

#### 方式一：自动发现（推荐）

**原理**：API Server 内置了一张"已知端口映射表"，覆盖 20+ 常见服务端口：

```
3306 → MySQL          5432 → PostgreSQL       6379 → Redis
8080 → API Gateway    8500 → Consul           4222 → NATS
9000 → MinIO          16686 → Jaeger          9090 → Prometheus
3200 → Frontend Dev   8081 → User Service     27017 → MongoDB
7000 → FRP Server     7001 → FRP Dashboard    15672 → RabbitMQ Mgmt
... 更多
```

**点击"自动发现"按钮后发生的事**：

```
1. 对所有已知端口做 TCP 连接探测（超时 1.5 秒）
2. 端口可达 → 标记 healthy
3. 端口不可达 → 标记 unreachable（不会注册不可达的新服务）
4. 如果数据库中没有该端口的服务 → 自动注册（source=discovered）
5. 如果数据库中已有该端口的服务 → 只更新状态
6. 弹出扫描结果窗口，展示所有端口详情
```

**前端操作**：服务管理页面 → 点击 **"自动发现"** 按钮 → 等待扫描完成 → 查看弹窗中的结果

**API 调用**：
```bash
# 标准扫描（只扫已知端口）
curl -X POST http://localhost:8081/api/v1/services/discover

# 扫描 + 指定额外端口（如你的 9000 端口不在已知表中）
curl -X POST http://localhost:8081/api/v1/services/discover \
  -H "Content-Type: application/json" \
  -d '{"extra_ports": [9000, 8888, 12345]}'
```

> **为什么你之前看不到 MySQL 和 FRP？**
> 因为旧版平台用硬编码种子数据，写了 PostgreSQL（5432）而不是 MySQL（3306），
> FRP（7000）虽然在种子数据中但没有做真实端口探测。现在改成真实探测后，
> 只要端口在运行，就会被发现并注册。

#### 方式二：手动注册

**原理**：你填一个服务名和端口，系统立即做 TCP 探测判断端口是否可达。

**适用场景**：
- 自定义服务端口不在已知表中
- 远程服务（host 不是 127.0.0.1）
- 你知道服务名但想自定义显示信息

**前端操作**：服务管理页面 → 点击 **"注册服务"** 按钮 → 填写表单 → 提交

表单字段：
- **服务名**（必填）：如 `FRP Server`
- **端口**（必填）：如 `7000`
- **地址**（默认 127.0.0.1）：远程服务填实际 IP
- **类型**（默认 custom）：下拉选择
- **版本**（可选）：如 `v0.61.0`
- **依赖**（可选）：从已有服务列表中选择或输入

**注册时自动做的事**：
1. TCP 连接 `host:port`，2 秒超时
2. 端口可达 → `status = healthy`
3. 端口不可达 → `status = unreachable`（服务仍会注册，但标记为不可达）
4. 不可达的服务在端口恢复后，30 秒内自动变为 healthy

**API 调用**：
```bash
curl -X POST http://localhost:8081/api/v1/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "FRP Server",
    "type": "infra",
    "port": 7000,
    "host": "127.0.0.1",
    "version": "v0.61.0",
    "dependencies": ["redis"]
  }'
```

返回示例：
```json
{
  "code": 0,
  "data": {
    "service": { "id": "frp-server", "name": "FRP Server", "status": "healthy", ... },
    "port_reachable": true,
    "dependencies": ["redis"]
  }
}
```

#### 方式三：种子数据（自动）

API Server 首次启动时，如果 MySQL 的 `services` 表为空，会自动写入一批初始种子数据：

```
gateway       (API Gateway,    8081, healthy)
mysql         (MySQL,          3306, healthy) ← 对你本机的 MySQL
redis         (Redis,          6379, healthy) ← 对你本机的 Redis
minio         (MinIO,          9000, healthy) ← 对 9000 端口
user-service  (User Service,   8081, healthy)
order-service (Order Service,  8082, critical)
ai-service    (AI Service,     8083, critical)
consul        (Consul,         8500, critical)
nats          (NATS,           4222, critical)
jaeger        (Jaeger,         16686, critical)
prometheus    (Prometheus,     9090, critical)
```

> 注意：种子数据只在表为空时写入一次。之后所有状态变化由健康检查自动更新。

### 4.2 健康检查 — 三层检查机制

平台对每个服务做三层健康检查：

```
第一层：TCP 端口探测（最基础）
  └─ net.DialTimeout("tcp", "127.0.0.1:3306", 2秒)
  └─ 成功 → 端口可达
  └─ 失败 → 端口不可达，标记 critical

第二层：HTTP /health 端点（如果端口可达）
  └─ http.Get("http://127.0.0.1:3306/health", 3秒超时)
  └─ 200-399 → healthy
  └─ 其他状态码 → warning
  └─ 连接失败 → warning（端口可达但 HTTP 不通）

第三层：综合判断
  └─ TCP 不可达 → critical（最严重）
  └─ TCP 可达 + HTTP 可达 → healthy
  └─ TCP 可达 + HTTP 不通 → warning（如 MySQL 没有 /health 端点）
```

**各类型服务的典型检查结果**：

| 服务 | TCP | HTTP /health | 综合状态 |
|------|-----|-------------|---------|
| MySQL 3306 | ✓ reachable | ✗ unreachable | **warning**（端口可达但不提供 HTTP） |
| Redis 6379 | ✓ reachable | ✗ unreachable | **warning** |
| Go API 8081 | ✓ reachable | ✓ 200 OK | **healthy** |
| FRP 7000（未启动） | ✗ unreachable | ✗ skip | **critical** |
| MinIO 9000 | ✓ reachable | 视配置 | **healthy/warning** |

**健康检查的三种触发方式**：

1. **后台自动轮询**：每 30 秒对所有已注册服务做 TCP 检查，自动更新 MySQL 和 Redis
2. **手动刷新**：服务管理页面 → 点击 **"刷新健康"** 按钮，立即对所有服务做一轮检查
3. **查看详情**：服务管理页面 → 点击某个服务的 **"详情"** 按钮 → 弹窗中展示三层检查结果

**API 调用**：
```bash
# 批量健康检查
curl -X POST http://localhost:8081/api/v1/services/refresh-health

# 单个服务健康详情
curl http://localhost:8081/api/v1/services/mysql/health
```

单服务健康详情返回示例：
```json
{
  "service_id": "mysql",
  "overall": "warning",
  "checks": [
    {
      "name": "TCP Port Check",
      "status": "healthy",
      "latency_ms": 3,
      "address": "127.0.0.1:3306",
      "last_check": "2026-07-09T14:30:00+08:00"
    },
    {
      "name": "HTTP Health Endpoint",
      "status": "unreachable",
      "latency_ms": -1,
      "url": "http://127.0.0.1:3306/health",
      "last_check": "2026-07-09T14:30:00+08:00"
    },
    {
      "name": "Service Status (DB)",
      "status": "healthy",
      "last_check": "2026-07-09T14:29:30+08:00"
    },
    {
      "name": "MySQL Protocol Check",
      "status": "reachable",
      "address": "127.0.0.1:3306",
      "last_check": "2026-07-09T14:30:00+08:00"
    }
  ]
}
```

### 4.3 服务拓扑 — 可视化依赖关系

拓扑图页面展示所有服务的依赖关系和实时健康状态：

- **节点颜色**：绿色=healthy，黄色=warning，红色=critical
- **连线**：表示依赖关系（如 gateway → user-service → mysql）
- **悬停/点击**：查看服务详情和健康状态
- **自动刷新**：每次加载页面都从 API 拉取最新数据

拓扑图中的服务数据来源与"服务管理"完全相同 — 都是同一个 `/api/v1/services` 端点。

### 4.4 API 网关 — 路由规则管理

网关页面展示和管理所有路由规则：

| 功能 | 说明 |
|------|------|
| 路由列表 | 查看所有路由规则（路径、上游、方法、限流、租户路由） |
| 创建路由 | 新增路由规则 |
| 修改路由 | 更新路由配置 |
| 删除路由 | 移除路由规则 |
| 中间件配置 | 查看 CORS/JWT/限流/租户路由配置 |

### 4.5 流量管理 — 熔断/降级/重试

流量管理页面展示三种策略：

- **熔断器**：服务故障超过阈值时自动切断流量
- **降级规则**：服务不可用时的替代响应策略
- **重试策略**：请求失败时的自动重试配置

### 4.6 租户管理 — 多租户隔离

租户管理页面支持：

| 功能 | 说明 |
|------|------|
| 租户列表 | 查看所有租户（名称/配额/用量/状态） |
| 创建租户 | 新增租户（自动生成 Schema/Redis 前缀） |
| 冻结/解冻 | 禁止/恢复租户访问 |
| API Key 管理 | 为租户创建和删除 API Key |
| 配额管理 | 设置和调整租户用量上限 |

每个租户的隔离维度：
- **Schema 隔离**：数据库层面的租户专属表空间
- **Redis 前缀**：`tenant:{name}:` 前缀隔离缓存数据
- **API Key**：租户专属密钥，请求时通过 `X-Tenant-ID` Header 传递

### 4.7 AI 供应商管理 — 智能路由

AI 供应商管理展示了微服务中"第三方能力接入"的完整案例：

```
供应商接入流程（6 步）：
  1. 注册供应商 → 填写 endpoint、api_key、模型列表
  2. 健康检测 → 自动探测供应商连通性
  3. 配置路由 → 按租户/按成本/按延迟设定分发规则
  4. 计费配置 → cost_per_1k 定价
  5. 用量监测 → 按租户统计 Token 消耗
  6. 降级兜底 → 主供应商故障时自动切换备用
```

### 4.8 智能校对 — 第三方 API 接入完整案例

智能校对页面是一个**完整的第三方服务接入案例**，展示从接入到监测的全流程：

- **接入配置**：供应商信息 + 路由规则
- **实时校对**：输入中文文本，检测拼写/用词/风格错误
- **调用日志**：每次校对的详细记录（时间/租户/延迟/结果）
- **用量统计**：调用次数/平均延迟/成功率/成本

### 4.9 可观测性 — 指标/链路/日志/告警

可观测性页面提供四维监控：

| 维度 | 数据来源 | 说明 |
|------|---------|------|
| 指标 | Redis + 模拟 | 请求计数、P95/P99 延迟、错误率、AI Token |
| 链路追踪 | 模拟数据 | 每个请求经过的服务调用链 + 火焰图 |
| 日志 | 模拟数据 | 按关键字/服务/级别搜索日志 |
| 告警规则 | MySQL | 配置告警规则 + 查看告警历史 |

---

## 五、常见使用场景

### 场景 1：刚装了一个新服务，想监控它

**步骤**：
1. 启动你的服务（假设在端口 8888）
2. 打开平台 → 服务管理 → 点击 **"自动发现"**
   - 如果 8888 不在已知端口表中，点击 **"注册服务"**
3. 填写：服务名=`My App`，端口=`8888`，类型=`custom`
4. 提交后，平台立即做 TCP 探测
5. 之后每 30 秒自动检查，服务宕了自动标红，恢复后自动标绿

### 场景 2：MySQL 服务检测不到

**原因排查**：
1. 检查 MySQL 是否在运行：`netstat -an | grep 3306`
2. 如果端口不在 LISTENING → MySQL 没启动
3. 如果端口在 LISTENING → 点 **"自动发现"** 或 **"刷新健康"**
4. 如果仍然看不到 → 点 **"注册服务"**，手动注册 MySQL（端口 3306）

### 场景 3：FRP 服务检测不到

**原因排查**：
1. FRP 默认端口 7000（Server）+ 7001（Dashboard）
2. 如果 FRP 没启动 → 注册后状态会显示 `unreachable`，启动后 30 秒自动恢复
3. 如果 FRP 用了非标准端口 → 手动注册时填实际端口

### 场景 4：9000 端口服务看不到

**说明**：9000 端口在已知端口表中映射为 **MinIO**。如果你在 9000 端口跑的不是 MinIO：
1. 点 **"自动发现"** → 会被识别为 "MinIO"（已知映射）
2. 或者点 **"注册服务"** → 填你自己的服务名，如 `My Service on 9000`
3. 如果想修改已注册的服务名 → 目前需要通过 API：

```bash
curl -X PUT http://localhost:8081/api/v1/services/minio \
  -H "Content-Type: application/json" \
  -d '{"name": "My Custom Service"}'
```

### 场景 5：服务上线了但状态还是 critical

**原因**：后台健康检查每 30 秒运行一次，可能还没刷新。

**解决**：点击 **"刷新健康"** 按钮，立即触发一轮检查。

### 场景 6：想删除一个不需要的服务

**操作**：服务管理页面 → 找到该服务 → 点击 **"删除"** → 确认

**API**：
```bash
curl -X DELETE http://localhost:8081/api/v1/services/{service_id}
```

### 场景 7：想让拓扑图显示服务间的依赖关系

**操作**：注册/更新服务时指定 `dependencies` 字段。

```bash
# 注册时指定依赖
curl -X POST http://localhost:8081/api/v1/services \
  -d '{"name": "My App", "port": 8888, "dependencies": ["mysql", "redis"]}'

# 更新已有服务的依赖
curl -X PUT http://localhost:8081/api/v1/services/my-app \
  -d '{"dependencies": ["mysql", "redis", "gateway"]}'
```

---

## 六、API 端点速查表

### 服务治理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/services` | 列出所有服务 |
| POST | `/api/v1/services` | 手动注册服务 |
| POST | `/api/v1/services/discover` | 自动发现（扫描端口） |
| POST | `/api/v1/services/refresh-health` | 批量健康检查 |
| GET | `/api/v1/services/events` | 服务注册事件时间线 |
| GET | `/api/v1/services/:id` | 获取单个服务详情 |
| GET | `/api/v1/services/:id/health` | 单服务三层健康检查详情 |
| PUT | `/api/v1/services/:id` | 更新服务信息 |
| DELETE | `/api/v1/services/:id` | 删除服务 |

### API 网关

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/gateway/routes` | 路由列表 |
| POST | `/api/v1/gateway/routes` | 创建路由 |
| PUT | `/api/v1/gateway/routes/:id` | 更新路由 |
| DELETE | `/api/v1/gateway/routes/:id` | 删除路由 |
| GET | `/api/v1/gateway/middleware` | 中间件配置 |
| PUT | `/api/v1/gateway/middleware` | 更新中间件配置 |

### 流量管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/traffic/circuit-breakers` | 熔断器列表 |
| PUT | `/api/v1/traffic/circuit-breakers/:service` | 更新熔断器 |
| GET | `/api/v1/traffic/degradation` | 降级规则列表 |
| GET | `/api/v1/traffic/retry` | 重试策略列表 |

### 租户管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/tenants` | 租户列表 |
| POST | `/api/v1/tenants` | 创建租户 |
| GET | `/api/v1/tenants/:id` | 租户详情 |
| PUT | `/api/v1/tenants/:id` | 更新租户 |
| PUT | `/api/v1/tenants/:id/freeze` | 冻结租户 |
| PUT | `/api/v1/tenants/:id/unfreeze` | 解冻租户 |
| POST | `/api/v1/tenants/:id/api-keys` | 创建 API Key |
| DELETE | `/api/v1/tenants/:id/api-keys/:key` | 删除 API Key |

### AI 供应商

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/ai/providers` | 供应商列表 |
| POST | `/api/v1/ai/providers` | 创建供应商 |
| GET | `/api/v1/ai/providers/:id` | 供应商详情 |
| PUT | `/api/v1/ai/providers/:id` | 更新供应商 |
| DELETE | `/api/v1/ai/providers/:id` | 删除供应商 |
| GET | `/api/v1/ai/providers/:id/usage` | 供应商用量统计 |
| GET | `/api/v1/ai/providers/:id/health` | 供应商健康检查 |
| GET | `/api/v1/ai/routing-rules` | AI 路由规则列表 |
| POST | `/api/v1/ai/routing-rules` | 创建路由规则 |
| PUT | `/api/v1/ai/routing-rules/:id` | 更新路由规则 |

### 智能校对

| 方法 | 端点 | 说明 |
|------|------|------|
| POST | `/api/v1/proofread` | 执行校对 |
| GET | `/api/v1/proofread/config` | 校对配置 |
| PUT | `/api/v1/proofread/config` | 更新配置 |
| GET | `/api/v1/proofread/logs` | 校对日志 |
| GET | `/api/v1/proofread/stats` | 校对统计 |

### 可观测性

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/observability/metrics` | 系统指标 |
| GET | `/api/v1/observability/traces` | 链路追踪 |
| GET | `/api/v1/observability/logs` | 日志搜索 |
| GET | `/api/v1/observability/alerts/rules` | 告警规则列表 |
| POST | `/api/v1/observability/alerts/rules` | 创建告警规则 |
| GET | `/api/v1/observability/alerts/events` | 告警事件历史 |

### 基础

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/health` | API Server 健康检查 |
| GET | `/metrics` | Prometheus 指标端点 |

---

## 七、健康状态说明

| 状态 | 颜色 | 含义 | 触发条件 |
|------|------|------|---------|
| `healthy` | 🟢 绿 | 服务正常运行 | TCP 端口可达 + HTTP /health 可达（或业务指标正常） |
| `warning` | 🟡 黄 | 服务可用但有隐患 | TCP 端口可达但 HTTP /health 不通（如 MySQL/Redis），或延迟偏高 |
| `critical` | 🔴 红 | 服务不可用 | TCP 端口不可达 |
| `unreachable` | ⚪ 灰 | 端口未开放 | 手动注册时端口不可达，标记为初始状态 |

**状态自动流转规则**：
```
unreachable → healthy   （后台检查发现端口可达）
healthy     → critical  （后台检查发现端口不可达）
critical    → healthy   （后台检查发现端口恢复）
warning     → healthy   （HTTP /health 端点恢复可达）
```

---

## 八、已知端口映射表

自动发现扫描的所有端口（可在 `handlers.go` 的 `knownPorts` 中扩展）：

| 端口 | 默认识别名 | 类型 |
|------|-----------|------|
| 3306 | MySQL | infra |
| 5432 | PostgreSQL | infra |
| 6379 | Redis | infra |
| 8080 | API Gateway | gateway |
| 8081 | User Service | service |
| 8082 | Order Service | service |
| 8083 | AI Service | service |
| 8500 | Consul | infra |
| 4222 | NATS | infra |
| 9000 | MinIO | infra |
| 9090 | Prometheus | observability |
| 9100 | Node Exporter | observability |
| 16686 | Jaeger | observability |
| 3200 | Frontend Dev | service |
| 27017 | MongoDB | infra |
| 2379 | etcd | infra |
| 5601 | Kibana | observability |
| 9200 | Elasticsearch | infra |
| 15672 | RabbitMQ Management | infra |
| 5672 | RabbitMQ | infra |
| 7000 | FRP Server | infra |
| 7001 | FRP Dashboard | infra |
| 7500 | FRP Client Port | infra |

**不在表中的端口**：
- 自动发现扫描到但不在表中 → 标记为 `未知服务(端口)`，类型 `custom`
- 也可以通过 `extra_ports` 参数指定额外扫描端口

---

## 九、技术栈参考

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端 | Go 1.22 + Gin + GORM | API Server，端口 8081 |
| 数据库 | MySQL 8 (ServBay) | 元数据持久化，`microhub` 库 |
| 缓存 | Redis 7 (ServBay) | 实时指标计数 + 健康状态缓存 |
| 前端 | Vue 3 + Pinia + Element Plus | 暗色主题治理界面，端口 3200 |
| 代理 | Vite dev proxy | `/api` → `http://localhost:8081` |
| 部署 | Podman + Quadlet | 容器化部署（Linux 生产环境） |

---

## 十、常见问题 FAQ

### Q: 为什么 MySQL 显示 warning 而不是 healthy？
A: MySQL 端口 3306 可达（TCP 检查通过），但 MySQL 不提供 HTTP `/health` 端点，所以 HTTP 检查失败，综合状态为 warning。这是正常的 — MySQL 本身就是在运行的。

### Q: 为什么 Redis 显示 warning？
A: 同 MySQL — Redis 端口可达但没有 HTTP `/health` 端点。

### Q: 我启动了一个 9000 端口的服务，为什么显示为 MinIO？
A: 9000 端口在已知映射表中默认识别为 MinIO。你可以通过 API 更改名称：
```bash
curl -X PUT http://localhost:8081/api/v1/services/minio \
  -d '{"name": "你的真实服务名"}'
```

### Q: 如何添加新的已知端口？
A: 编辑 `api-server/handlers.go` 中的 `knownPorts` map，添加新的端口映射，然后重新编译启动：
```go
var knownPorts = map[int]struct {
    Name    string
    Type    string
    Version string
}{
    // ... 已有映射 ...
    8888: {"My Service", "custom", "v1.0"},  // 新增
}
```

### Q: 后台健康检查的频率可以改吗？
A: 编辑 `api-server/main.go` 中的 `healthChecker()` 函数，修改 `time.Sleep(30 * time.Second)` 的间隔。

### Q: 如何完全重置服务数据？
A: 清空 MySQL services 表后重启 API Server：
```bash
"D:/ServBay/packages/mysql/current/bin/mysql.exe" -uroot -proot -e "USE microhub; TRUNCATE TABLE services;"
# 然后重启 microhub-api.exe
```

### Q: 端口 8080 被占用了怎么办？
A: 当前 API Server 使用 8081 端口。如果你的 8080 被其他服务占用（如 nginx），这是正常的。如果需要改端口，编辑 `main.go` 中的 `r.Run(":8081")` 和 `vite.config.ts` 中的代理 target。

### Q: 前端数据不更新怎么办？
A: 刷新浏览器页面即可。数据每次页面加载时从 API 拉取，不做本地缓存。

### Q: 为什么 Podman 里看不到 fullstack 容器？
A: 因为你当前用的是**本地开发模式**——MySQL/Redis/API Server/前端都是直接在 Windows 上运行的进程，不在 Podman 容器中。这是正常的。只有迁移到 Linux 用 `podman compose up` 部署后，才能在 Podman 里看到容器。

### Q: docker-compose.yml 文件是干嘛的？我现在用不用？
A: docker-compose.yml 是**线上生产环境的部署配置**。本地开发时不需要用它。它定义了 MySQL 容器、Redis 容器、Jaeger 容器、Prometheus 容器等 14+ 个服务，等迁移到 Linux 服务器时 `podman compose up -d` 一键启动所有容器。

### Q: 我想在本地也看到 Jaeger/Prometheus 的真实数据怎么办？
A: 单独启动容器就行，不需要整个 docker-compose：
```bash
podman run -d --name jaeger -p 16686:16686 jaegertracing/all-in-one:latest
podman run -d --name prometheus -p 9090:9090 prom/prometheus:latest
```
启动后 API Server 自动检测到端口可达，可观测性数据自动切换为真实数据源。

---

## 十一、迁移到 Linux 服务器（线上生产环境）

### 从本地开发切换到线上生产的完整流程

**当你在 Windows 本地开发调试完成后，需要把项目部署到 Linux 服务器上。这时运行方式会从"直接跑进程"变为"Podman 容器化部署"。**

### 1. 复制项目
```bash
scp -r fullstack-app/ user@linux-server:/opt/microhub/
```

### 2. 安装 Podman（如果 Linux 上没有）
```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y podman

# CentOS/RHEL/Fedora（通常已预装）
sudo yum install -y podman
```

### 3. 一键启动所有容器
```bash
cd /opt/microhub/fullstack-app
podman compose up -d
```

启动后检查：
```bash
podman ps
# 应该看到 14+ 个 fullstack-* 容器全部在运行
```

### 4. 可观测性自动切换

容器启动后，Jaeger (16686)、Prometheus (9090)、Loki (3100) 都在运行。
API Server 检测到这些端口可达，**自动切换到真实数据源**：

| 数据维度 | 本地（模拟） | 线上（真实） |
|---------|-------------|-------------|
| 指标 | Redis 随机计数 | Prometheus 真实时间序列 |
| 链路 | 硬编码 10 条 trace | Jaeger 实时采集的真实链路 |
| 日志 | 硬编码 15 条日志 | Loki 实时流式日志 |

**不需要改任何代码或配置**，API Server 内置的智能检测会自动完成切换。

### 5. 配置 Quadlet systemd 自启动（推荐）

Podman 容器默认不会开机自启。用 Quadlet 配置 systemd 实现：

```bash
cd /opt/microhub/fullstack-app/deployments/podman/
cp *.container ~/.config/containers/systemd/
systemctl --user daemon-reload
systemctl --user start user-service    # 启动 API Server
systemctl --user status user-service   # 查看状态
```

Quadlet 的优势：
- 开机自动启动所有微服务容器
- 容器崩溃时自动重启（`Restart=always`）
- 不需要 root 权限（Rootless 模式）
- 用 `systemctl --user` 管理，和传统 systemd 服务一样方便

### 6. 修改配置（跨机器部署时）

如果 MySQL/Redis 等不在同一台机器上：

```bash
# 编辑 api-server 的环境变量
export MYSQL_HOST=192.168.1.100
export MYSQL_PORT=3306
export REDIS_HOST=192.168.1.101
export REDIS_PORT=6379
export PROMETHEUS_URL=http://192.168.1.102:9090
export JAEGER_URL=http://192.168.1.102:16686
export LOKI_URL=http://192.168.1.102:3100
```

或修改 docker-compose.yml 中的环境变量配置。

### 7. 验证线上环境

```bash
# 检查所有容器运行状态
podman ps

# 检查 API Server 健康检查
curl http://localhost:8081/health

# 检查数据源状态（应该全部 reachable）
curl http://localhost:8081/api/v1/observability/datasource-status

# 检查服务发现（应该发现所有容器内的服务）
curl -X POST http://localhost:8081/api/v1/services/discover

# 打开浏览器
http://服务器IP:3200
```

---

*本文档最后更新：2026-07-09 | 作者：MicroHub AI Agent*
