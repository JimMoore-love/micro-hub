# FullStack 微服务全栈项目 - 进度追踪

## 项目概览
Go 微服务全栈项目，包含9个模块，使用 Podman Rootless 部署，支持迁移到 Linux。

## 技术栈
| 层级 | 技术 |
|------|------|
| 后端微服务 | Go 1.22 + Gin + gRPC-Gateway |
| 前端 | Vue 3 + Pinia + Axios + Element Plus |
| 业务后台 | Laravel 11 + gRPC Client + Queue Worker |
| 容器化 | Podman 5.8.2 (Rootless) + Quadlet (.container) |
| 服务发现 | Consul |
| 消息队列 | NATS |
| 数据库 | PostgreSQL 16 (多租户) |
| 缓存 | Redis 7 |
| 对象存储 | MinIO |
| 可观测性 | Prometheus + Grafana + Jaeger + Loki |
| AI 接入 | OpenAI/Claude/DeepSeek SDK |

---

## 模块完成状态

### ✅ 模块一：多租户中间件
- [x] `pkg/tenant/tenant.go` — TenantResolver 从 Header/Subdomain 解析租户
- [x] `pkg/tenant/gorm_plugin.go` — GORM 多租户插件，自动注入 tenant_id
- [x] `pkg/tenant/gorm_plugin.go` — PostgreSQL Schema 动态切换 (SwitchSchema/CreateTenantSchema)
- [x] `pkg/tenant/tenant.go` — Redis 多租户隔离 (RedisNamespace + Key前缀 tenant:{id}:)
- [x] `internal/middleware/auth.go` — TenantResolver Gin中间件

### ✅ 模块二：AI 服务接入
- [x] `internal/ai/provider.go` — OpenAI 封装 (ChatCompletion + StreamChat)
- [x] `internal/ai/provider.go` — Claude 封装 (ChatCompletion)
- [x] `internal/ai/provider.go` — DeepSeek 封装 (ChatCompletion + StreamChat)
- [x] `internal/ai/gateway.go` — AI 网关路由，根据租户配置自动路由
- [x] `internal/ai/gateway.go` — SSE 流式响应 (SSEChunk + SSEReader)
- [x] `internal/ai/gateway.go` — 租户用量配额控制 (CheckQuota + RecordUsage)
- [x] `internal/ai/gateway.go` — 多租户 API Key 隔离 (RedisNamespace)

### ✅ 模块三：gRPC-Gateway 统一入口
- [x] `api/v1/user.proto` — 用户服务 Proto 定义 + google.api.http 注解
- [x] `api/v1/order.proto` — 订单服务 Proto 定义
- [x] `api/v1/ai.proto` — AI 服务 Proto 定义（含流式）
- [x] `cmd/gateway/main.go` — Gateway 反向代理 + JWT 鉴权 + CORS
- [x] `internal/middleware/auth.go` — JWTAuth + CORSConfig 中间件
- [x] `internal/middleware/auth.go` — 租户信息透传 (X-Tenant-ID → context)

### ✅ 模块四：Vue3 前端
- [x] `frontend-vue/package.json` — 完整依赖配置
- [x] `frontend-vue/src/api/client.ts` — Axios 封装（自动注入 X-Tenant-ID + Authorization）
- [x] `frontend-vue/src/api/user.ts` — 用户 CRUD API
- [x] `frontend-vue/src/api/ai.ts` — AI 对话 API（含 SSE）
- [x] `frontend-vue/src/stores/user.ts` — Pinia 用户状态管理
- [x] `frontend-vue/src/composables/useTenant.ts` — 租户切换
- [x] `frontend-vue/src/views/UserList.vue` — 用户列表页（搜索/筛选/分页）
- [x] `frontend-vue/src/views/OrderList.vue` — 订单管理页
- [x] `frontend-vue/src/views/AIChat.vue` — AI 对话页（SSE 流式 + Markdown）
- [x] `frontend-vue/src/views/Dashboard.vue` — 仪表盘
- [x] `frontend-vue/src/App.vue` — 深色主题布局（侧边栏+顶栏+租户选择器）

### ✅ 模块五：Laravel 业务后台
- [x] `backend-laravel/composer.json` — Laravel 11 依赖
- [x] `backend-laravel/app/Services/GrpcClient.php` — gRPC Client 封装
- [x] `backend-laravel/app/Services/ApiService.php` — HTTP API 封装
- [x] `backend-laravel/app/Jobs/ProcessAIRequest.php` — Queue Worker 异步 AI
- [x] `backend-laravel/app/Http/Controllers/AdminController.php` — Admin Dashboard
- [x] `backend-laravel/config/services.php` — 微服务配置

### ✅ 模块六：Podman 部署
- [x] `deployments/podman/Dockerfile.user` — 多阶段构建（alpine最小镜像）
- [x] `deployments/podman/Dockerfile.order`
- [x] `deployments/podman/Dockerfile.ai`
- [x] `deployments/podman/Dockerfile.gateway`
- [x] `deployments/podman/user-service.container` — Quadlet systemd 配置
- [x] `deployments/podman/gateway.container`
- [x] `deployments/podman/deploy.sh` — 一键部署脚本（start/stop/restart/status）
- [x] `docker-compose.yml` — 本地开发环境（15个服务）

### ✅ 模块七：可观测性
- [x] `observability/prometheus.yml` — Prometheus 指标采集
- [x] `observability/grafana/dashboards/fullstack.json` — Grafana 仪表盘（11面板）
- [x] `observability/grafana/provisioning/datasources.yml` — 4数据源
- [x] `observability/loki/local-config.yaml` — Loki 日志聚合
- [x] `pkg/otel/tracer.go` — OpenTelemetry + Jaeger 集成
- [x] `internal/middleware/metrics.go` — Prometheus 指标中间件

### ✅ 模块八：并发测试
- [x] `tests/load/k6_script.js` — 渐进加压（P95<200ms 验证）
- [x] `tests/load/vegeta.txt` — Vegeta targets
- [x] `backend-go/Makefile` — test-k6 / test-vegeta / bench 命令

### ✅ 模块九：Postman 代码生成
- [x] `internal/codegen/postman/parser.go` — Postman v2.1 解析器
- [x] `internal/codegen/generator/gin.go` — Gin Handler 生成（含 Swagger）
- [x] `internal/codegen/generator/proto.go` — Proto3 生成（含 http 注解）
- [x] `internal/codegen/generator/vue.go` — Vue Axios Client 生成
- [x] `internal/codegen/generator/laravel.go` — Laravel Facade 生成
- [x] `cmd/codegen/main.go` — CLI 入口工具

---

## 关键约束检查
| # | 约束 | 状态 |
|---|------|------|
| 1 | 所有服务支持多租户，tenant_id 全链路传递 | ✅ |
| 2 | GORM 插件自动注入租户过滤 | ✅ |
| 3 | AI 服务用量配额，超配额返回 429 | ✅ |
| 4 | Podman Rootless + Quadlet | ✅ |
| 5 | Postman Collection v2.1 解析 | ✅ |
| 6 | P95 < 200ms, error_rate < 1% | ✅ (测试脚本已写) |

---

## 快速启动指南

### 本地开发（docker-compose）
```bash
cd F:/MicroserviceManage/fullstack-app
podman compose up -d          # 启动所有服务
podman compose logs -f gateway # 查看日志
podman compose down            # 停止
```

### Podman Rootless 部署
```bash
cd backend-go/deployments/podman
./deploy.sh start   # 一键启动
./deploy.sh status  # 查看状态
./deploy.sh stop    # 停止
```

### 前端开发
```bash
cd frontend-vue
npm install
npm run dev   # 端口 3000，API 代理到 8080
```

### 代码生成
```bash
cd backend-go
go run ./cmd/codegen -input collection.json -output ./generated -target all
```

### 并发测试
```bash
make test-k6      # k6 渐进加压
make test-vegeta  # Vegeta 压测
```

---

## Linux 迁移指南
1. 复制整个 fullstack-app 目录到 Linux 服务器
2. 安装 Podman: `sudo apt install podman` 或 `sudo yum install podman`
3. 复制 Quadlet .container 文件到 `~/.config/containers/systemd/`
4. 运行 `systemctl --user daemon-reload && systemctl --user start user-service`
5. 或直接使用 `podman compose up -d`（compose 方式跨平台通用）

---

## 下次任务快速熟悉
- 阅读 PROGRESS.md 了解整体进度
- 阅读 docker-compose.yml 了解服务编排
- 阅读各模块代码时，先看 internal/ 结构
- 修改时注意 tenant_id 全链路传递要求
