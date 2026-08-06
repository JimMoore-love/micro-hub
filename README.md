# MicroHub 微服务治理平台

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.4-42b883)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-MIT-blue)](LICENSE)

> 让你**看到、管到、用到**本机所有微服务的真实状态。

MicroHub 是一个**微服务治理平台**，不是商城、不是后台管理面板。它通过**真实端口探测**来发现和监控服务，而不是靠硬编码数据——你本机跑了什么服务，它就能看到什么；服务宕了自动标红，服务上线自动恢复。

## ✨ 核心特性

- 🔍 **服务发现** — 三种注册方式（自动发现 / 手动注册 / 种子数据），TCP 端口探测，无需 Consul/K8s
- 🩺 **三层健康检查** — TCP 端口 → HTTP /health → 综合状态（healthy / warning / critical），30 秒自动轮询
- 🗺️ **服务拓扑** — 可视化依赖关系，节点颜色实时反映健康状态
- 🚦 **API 网关** — 路由规则管理、中间件配置（CORS / JWT / 限流 / 租户路由）
- 🧊 **流量管理** — 熔断器、降级规则、重试策略
- 👥 **多租户隔离** — Schema 隔离 + Redis 前缀 + API Key，租户配额管理
- 🤖 **AI 供应商管理** — 多供应商接入（OpenAI / Claude / DeepSeek）、路由、计费、用量监测、降级兜底
- 📝 **智能校对** — 第三方 API 接入完整案例（接入 → 调用 → 日志 → 用量统计）
- 📊 **可观测性** — 指标 / 链路追踪 / 日志 / 告警四维监控，**模拟数据与真实数据源自动切换**
- 🖥️ **Agent 节点监控** — 本机 Agent 自动上报端口扫描、网段扫描、CPU/内存用量

## 🏗️ 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.22 + Gin + GORM + JWT |
| 前端 | Vue 3.4 + Vite + Pinia + Element Plus + Axios |
| 数据库 | MySQL 8（元数据持久化） |
| 缓存 | Redis 7（实时指标 + 健康状态） |
| 可观测性 | Prometheus + Grafana + Jaeger + Loki |
| 部署 | Docker Compose（开发）/ Podman + Quadlet（生产） |
| 压测 | k6 + Vegeta |

## 📁 项目结构

```
fullstack-app/
├── api-server/              # Go API 服务（核心）
│   ├── main.go              # 入口 + 路由 + 30 秒健康检查轮询
│   ├── auth.go              # JWT 认证（bcrypt + token）
│   ├── handlers_service.go  # 服务管理 / 发现 / 健康检查
│   ├── handlers_tenant.go   # 租户管理
│   ├── handlers_ai.go       # AI 供应商 / 路由规则 / 校对
│   ├── handlers_gateway.go  # 网关路由 / 中间件 / 流量管理
│   ├── handlers_observability.go  # 指标 / 链路 / 日志 / 告警
│   ├── handlers_agent.go    # Agent 上报 / 节点列表 / 网段扫描
│   ├── models.go            # 数据模型
│   └── Dockerfile           # 容器构建
├── agent/                   # Go Agent（节点采集端）
│   └── main.go              # 端口扫描 + 网段扫描 + 资源上报
├── frontend-vue/            # Vue 3 前端
│   ├── src/api/             # Axios 封装（auth / client / node / platform / ai / user）
│   ├── src/views/           # 页面（Dashboard / Services / Topology / Gateway / Traffic / Tenants / AIProviders / Proofread / Observability / Nodes / Alerts / Login ...）
│   └── vite.config.ts       # Vite 代理 /api → 8081
├── observability/           # 可观测性配置
│   ├── prometheus.yml       # Prometheus 采集配置
│   ├── grafana/             # Grafana 仪表盘 + 数据源
│   └── loki/                # Loki 日志配置
├── deployments/             # 部署脚本
├── tests/                   # 负载测试（k6 / Vegeta）
├── docker-compose.yml       # 生产环境编排（MySQL/Redis/Jaeger/otel-lgtm/api-server）
├── PROGRESS.md              # 开发进度追踪
├── ISSUES_ANALYSIS.md       # 问题分析
└── USER_GUIDE.md            # 用户使用手册（详细版）
```

## 🚀 快速开始

### 本地开发（Windows）

```bash
# 1. 启动 API Server（需先启动 MySQL + Redis）
cd api-server
go build -o microhub-api.exe .   # 或直接运行已有二进制
./microhub-api.exe
# → Listening on http://localhost:8081

# 2. 启动前端
cd ../frontend-vue
npm install
npm run dev -- --port 3200
# → http://localhost:3200

# 3. 默认账号
#    用户名: admin
#    密码:   admin123
```

### Docker Compose（生产/一键部署）

```bash
docker compose up -d
# 或 Podman: podman compose up -d

# 查看状态
docker compose ps

# 日志
docker compose logs -f api-server
```

### Agent 节点监控（可选）

```bash
cd agent
go build -o microhub-agent.exe .
./microhub-agent.exe --server http://localhost:8081
# 默认扫描端口: 80,443,3306,5432,6379,8080-8083,8500,4222,9000,9090,16686,11434,3200,8890
```

## 🔑 登录认证

| 项 | 值 |
|----|----|
| 登录接口 | `POST /api/v1/auth/login` |
| 默认账号 | `admin` / `admin123` |
| 认证方式 | JWT Bearer Token |
| 租户传递 | `X-Tenant-ID` Header |

## 📡 API 概览

### 服务治理
| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/services` | 列出所有服务 |
| POST | `/api/v1/services` | 手动注册服务 |
| POST | `/api/v1/services/discover` | 自动发现（TCP 端口扫描） |
| POST | `/api/v1/services/refresh-health` | 批量健康检查 |
| GET | `/api/v1/services/:id/health` | 单服务三层健康检查详情 |
| PUT | `/api/v1/services/:id` | 更新服务 |
| DELETE | `/api/v1/services/:id` | 删除服务 |

### 网关 / 流量
| 方法 | 端点 | 说明 |
|------|------|------|
| GET/POST | `/api/v1/gateway/routes` | 路由列表 / 创建 |
| PUT/DELETE | `/api/v1/gateway/routes/:id` | 更新 / 删除路由 |
| GET/PUT | `/api/v1/gateway/middleware` | 中间件配置 |
| GET | `/api/v1/traffic/circuit-breakers` | 熔断器列表 |

### 租户
| 方法 | 端点 | 说明 |
|------|------|------|
| GET/POST | `/api/v1/tenants` | 租户列表 / 创建 |
| PUT | `/api/v1/tenants/:id/freeze` | 冻结租户 |
| PUT | `/api/v1/tenants/:id/unfreeze` | 解冻租户 |
| POST | `/api/v1/tenants/:id/api-keys` | 创建 API Key |

### AI
| 方法 | 端点 | 说明 |
|------|------|------|
| GET/POST | `/api/v1/ai/providers` | 供应商列表 / 创建 |
| GET | `/api/v1/ai/providers/:id/usage` | 供应商用量统计 |
| GET | `/api/v1/ai/providers/:id/health` | 供应商健康检查 |
| GET/POST | `/api/v1/ai/routing-rules` | AI 路由规则 |
| POST | `/api/v1/proofread` | 智能校对 |

### 可观测性
| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/observability/metrics` | 系统指标 |
| GET | `/api/v1/observability/traces` | 链路追踪 |
| GET | `/api/v1/observability/logs` | 日志搜索 |
| GET/POST | `/api/v1/observability/alerts/rules` | 告警规则 |

### Agent / 节点
| 方法 | 端点 | 说明 |
|------|------|------|
| POST | `/api/v1/agents/report` | Agent 上报（端口/网段/资源） |
| GET | `/api/v1/nodes` | 节点列表 |
| POST | `/api/v1/nodes/scan-subnet` | 触发网段扫描 |

### 基础
| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/metrics` | Prometheus 指标 |

## 🩺 健康状态说明

| 状态 | 颜色 | 含义 |
|------|------|------|
| `healthy` | 🟢 | TCP 可达 + HTTP /health 可达 |
| `warning` | 🟡 | TCP 可达但无 HTTP 端点（如 MySQL/Redis） |
| `critical` | 🔴 | TCP 端口不可达 |
| `unreachable` | ⚪ | 手动注册时端口未开放 |

**三层检查机制**：TCP 端口探测（2s 超时）→ HTTP /health（3s 超时）→ 综合状态判定。后台每 30 秒轮询所有服务。

## 📊 可观测性：模拟 ↔ 真实数据自动切换

API Server 的三个可观测性 handler 自动检测外部服务是否可达：

```
handleGetMetrics()  → 检测 Prometheus (localhost:9090)  → 可达拉真实指标 / 不可达回退模拟
handleListTraces()  → 检测 Jaeger (localhost:16686)     → 可达拉真实链路 / 不可达回退模拟
handleSearchLogs()  → 检测 Loki (localhost:3100)        → 可达查真实日志 / 不可达回退模拟
```

**这意味着**：本地开发（无 Jaeger/Prometheus/Loki）界面照常显示模拟数据；迁移到 Linux 用 `docker compose up` 后自动切换真实数据源，**界面完全不变，无需改代码**。

## 🧪 测试

```bash
# k6 渐进加压测试
cd tests
k6 run load/k6_script.js

# Vegeta 压测
vegeta attack -targets=load/vegeta.txt -rate=100 -duration=30s | vegeta report
```

## 📦 部署到 Linux 服务器

```bash
# 1. 复制项目
scp -r fullstack-app/ user@linux-server:/opt/microhub/

# 2. 安装 Podman（如未安装）
sudo apt-get install -y podman   # Ubuntu/Debian

# 3. 一键启动
cd /opt/microhub/fullstack-app
podman compose up -d

# 4. （可选）Quadlet systemd 自启动
cd deployments/
cp *.container ~/.config/containers/systemd/
systemctl --user daemon-reload
systemctl --user start user-service
```

## 🔧 常见问题

<details>
<summary><b>为什么 MySQL/Redis 显示 warning 而不是 healthy？</b></summary>
端口可达（TCP 检查通过），但 MySQL/Redis 不提供 HTTP `/health` 端点，所以 HTTP 检查失败，综合状态为 warning。这是正常的——服务本身在运行。
</details>

<details>
<summary><b>如何添加新的已知端口映射？</b></summary>
编辑 `api-server/handlers_service.go` 中的 `knownPorts` map，添加新端口映射后重新编译。
</details>

<details>
<summary><b>后台健康检查频率怎么改？</b></summary>
编辑 `api-server/main.go` 中 `healthChecker()` 里的 `time.Sleep(30 * time.Second)`。
</details>

<details>
<summary><b>为什么 Podman 里看不到 fullstack 容器？</b></summary>
本地开发模式所有服务直接跑在 Windows 进程里（ServBay MySQL/Redis + microhub-api.exe + vite），不在容器中。只有 `docker compose up` 部署后才会有容器。
</details>

## 📄 文档

- [用户使用手册（详细版）](USER_GUIDE.md) — 功能详解、使用场景、API 速查、FAQ
- [开发进度追踪](PROGRESS.md) — 模块完成状态、技术栈演进
- [问题分析](ISSUES_ANALYSIS.md) — 历史问题与根因分析

## 📜 License

MIT
