// ============================================================
// k6 负载测试 - FullStack Microservice
// ============================================================
//
// 使用方式:
//   k6 run tests/load/k6_script.js
//   k6 run --out json=results.json tests/load/k6_script.js
//   k6 run -e BASE_URL=http://localhost:8080 tests/load/k6_script.js
//
// 阶段:
//   1. 预热: 10 VU, 30s
//   2. 正常: 50 VU, 60s
//   3. 高峰: 100 VU, 30s
//   4. 压力: 200 VU, 30s
// ============================================================

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Trend, Rate, Counter } from "k6/metrics";
import { randomIntBetween, randomString } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

// ------------------------------------------------------
// 配置
// ------------------------------------------------------
const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

// 租户ID列表（模拟多租户）
const TENANT_IDS = [
  "tenant-001",
  "tenant-002",
  "tenant-003",
  "tenant-004",
  "tenant-005",
];

// ------------------------------------------------------
// 自定义指标
// ------------------------------------------------------
const userReqDuration = new Trend("user_request_duration", true);
const orderReqDuration = new Trend("order_request_duration", true);
const aiReqDuration = new Trend("ai_request_duration", true);
const errorRate = new Rate("error_rate");
const userRequests = new Counter("user_requests");
const orderRequests = new Counter("order_requests");
const aiRequests = new Counter("ai_requests");

// ------------------------------------------------------
// 配置阶段
// ------------------------------------------------------
export const options = {
  stages: [
    { duration: "30s", target: 10 },   // Phase 1: 预热
    { duration: "60s", target: 50 },   // Phase 2: 正常负载
    { duration: "30s", target: 100 },  // Phase 3: 高峰负载
    { duration: "30s", target: 200 },  // Phase 4: 压力测试
  ],
  thresholds: {
    // P95 延迟 < 200ms
    "http_req_duration{name:/api/v1/users}": ["p(95)<200"],
    "http_req_duration{name:/api/v1/orders}": ["p(95)<200"],
    "http_req_duration{name:/api/v1/ai/chat}": ["p(95)<200"],
    // 错误率 < 1%
    error_rate: ["rate<0.01"],
    "http_req_failed": ["rate<0.01"],
    // 用户服务指标
    user_request_duration: ["p(95)<200"],
    // 订单服务指标
    order_request_duration: ["p(95)<200"],
    // AI服务指标（允许更宽松）
    ai_request_duration: ["p(95)<500"],
  },
  // 优雅终止
  gracefulStop: "10s",
};

// ------------------------------------------------------
// 获取随机租户ID
// ------------------------------------------------------
function getRandomTenant() {
  return TENANT_IDS[Math.floor(Math.random() * TENANT_IDS.length)];
}

// ------------------------------------------------------
// 通用Headers（包含租户ID）
// ------------------------------------------------------
function getHeaders() {
  return {
    "Content-Type": "application/json",
    "X-Tenant-ID": getRandomTenant(),
    "Accept": "application/json",
  };
}

// ------------------------------------------------------
// 初始化（每个VU只执行一次）
// ------------------------------------------------------
export function setup() {
  console.log(`=== FullStack k6 Load Test ===`);
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Tenants: ${TENANT_IDS.join(", ")}`);
  console.log(`Stages: ${JSON.stringify(options.stages)}`);

  // 健康检查
  const healthResp = http.get(`${BASE_URL}/health`, {
    timeout: "10s",
  });
  check(healthResp, {
    "Gateway is healthy": (r) => r.status === 200,
  });

  if (healthResp.status !== 200) {
    throw new Error(`Gateway is not healthy. Status: ${healthResp.status}`);
  }

  return {
    baseUrl: BASE_URL,
    testStartTime: new Date().toISOString(),
  };
}

// ------------------------------------------------------
// 测试组
// ------------------------------------------------------
export default function (data) {
  const baseUrl = data.baseUrl;

  // 75% 概率执行用户接口
  if (Math.random() < 0.75) {
    group("User API", () => {
      testGetUsers(baseUrl);
    });
  }

  // 50% 概率执行订单接口
  if (Math.random() < 0.5) {
    group("Order API", () => {
      testGetOrders(baseUrl);
    });
  }

  // 20% 概率执行AI接口
  if (Math.random() < 0.2) {
    group("AI API", () => {
      testAIChat(baseUrl);
    });
  }

  // 随机睡眠（模拟真实用户行为）
  sleep(randomIntBetween(1, 5));
}

// ------------------------------------------------------
// 用户 API 测试
// ------------------------------------------------------
function testGetUsers(baseUrl) {
  const headers = getHeaders();
  const tenantId = headers["X-Tenant-ID"];

  // GET /api/v1/users - 获取用户列表
  const listRes = http.get(`${baseUrl}/api/v1/users?page=1&limit=20`, {
    headers: headers,
    tags: { name: "/api/v1/users", tenant: tenantId },
  });

  userReqDuration.add(listRes.timings.duration);
  userRequests.add(1);

  check(listRes, {
    "[Users] status is 200": (r) => r.status === 200,
    "[Users] response has data": (r) => {
      try {
        const body = JSON.parse(r.body);
        return body && (body.data || body.users || Array.isArray(body));
      } catch {
        return false;
      }
    },
  }) || errorRate.add(1);

  // GET /api/v1/users/:id - 获取单个用户（如果是列表响应）
  if (listRes.status === 200) {
    try {
      const body = JSON.parse(listRes.body);
      const users = body.data || body.users || body;
      if (Array.isArray(users) && users.length > 0) {
        const userId = users[0].id || users[0].ID || 1;
        const singleRes = http.get(`${baseUrl}/api/v1/users/${userId}`, {
          headers: headers,
          tags: { name: "/api/v1/users/:id", tenant: tenantId },
        });
        userReqDuration.add(singleRes.timings.duration);
        userRequests.add(1);

        check(singleRes, {
          "[Users] single user status is 200": (r) => r.status === 200,
        }) || errorRate.add(1);
      }
    } catch {
      // 忽略解析错误（可能响应格式不同）
    }
  }
}

// ------------------------------------------------------
// 订单 API 测试
// ------------------------------------------------------
function testGetOrders(baseUrl) {
  const headers = getHeaders();
  const tenantId = headers["X-Tenant-ID"];

  // GET /api/v1/orders - 获取订单列表
  const listRes = http.get(`${baseUrl}/api/v1/orders?page=1&limit=20`, {
    headers: headers,
    tags: { name: "/api/v1/orders", tenant: tenantId },
  });

  orderReqDuration.add(listRes.timings.duration);
  orderRequests.add(1);

  check(listRes, {
    "[Orders] status is 200": (r) => r.status === 200,
    "[Orders] response has data": (r) => {
      try {
        const body = JSON.parse(r.body);
        return body && (body.data || body.orders || Array.isArray(body));
      } catch {
        return false;
      }
    },
  }) || errorRate.add(1);

  // POST /api/v1/orders - 创建订单（低频）
  if (Math.random() < 0.3 && listRes.status === 200) {
    const orderPayload = JSON.stringify({
      user_id: randomIntBetween(1, 100),
      product: `Product-${randomIntBetween(1, 1000)}`,
      quantity: randomIntBetween(1, 5),
      price: parseFloat((Math.random() * 1000).toFixed(2)),
    });

    const createRes = http.post(`${baseUrl}/api/v1/orders`, orderPayload, {
      headers: headers,
      tags: { name: "/api/v1/orders (POST)", tenant: tenantId },
    });

    orderReqDuration.add(createRes.timings.duration);
    orderRequests.add(1);

    check(createRes, {
      "[Orders] create status is 20x": (r) =>
        r.status === 200 || r.status === 201,
    }) || errorRate.add(1);
  }
}

// ------------------------------------------------------
// AI API 测试
// ------------------------------------------------------
function testAIChat(baseUrl) {
  const headers = getHeaders();
  const tenantId = headers["X-Tenant-ID"];

  // POST /api/v1/ai/chat - AI 对话
  const chatPayload = JSON.stringify({
    message: "Hello, introduce yourself briefly.",
    context: {
      session_id: `test-session-${randomString(8)}`,
    },
  });

  const chatRes = http.post(`${baseUrl}/api/v1/ai/chat`, chatPayload, {
    headers: headers,
    tags: { name: "/api/v1/ai/chat", tenant: tenantId },
    timeout: "30s", // AI响应可能较慢
  });

  aiReqDuration.add(chatRes.timings.duration);
  aiRequests.add(1);

  check(chatRes, {
    "[AI] status is 200": (r) => r.status === 200,
    "[AI] response has reply": (r) => {
      try {
        const body = JSON.parse(r.body);
        return body && (body.reply || body.message || body.data);
      } catch {
        return false;
      }
    },
    "[AI] response time < 5s": (r) => r.timings.duration < 5000,
  }) || errorRate.add(1);
}

// ------------------------------------------------------
// 汇总报告
// ------------------------------------------------------
export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    test_duration_seconds: data.state.testRunDurationMs / 1000,
    total_requests: data.metrics.http_reqs?.values?.count || 0,
    failed_requests: data.metrics.http_req_failed?.values?.rate || 0,
    error_rate: data.metrics.error_rate?.values?.rate || 0,

    latency: {
      avg_ms: data.metrics.http_req_duration?.values?.avg?.toFixed(2),
      p50_ms: data.metrics.http_req_duration?.values?.p(50)?.toFixed(2),
      p90_ms: data.metrics.http_req_duration?.values?.p(90)?.toFixed(2),
      p95_ms: data.metrics.http_req_duration?.values?.p(95)?.toFixed(2),
      p99_ms: data.metrics.http_req_duration?.values?.p(99)?.toFixed(2),
      max_ms: data.metrics.http_req_duration?.values?.max?.toFixed(2),
    },

    user_service: {
      requests: data.metrics.user_requests?.values?.count || 0,
      p95_ms: data.metrics.user_request_duration?.values?.p(95)?.toFixed(2),
    },
    order_service: {
      requests: data.metrics.order_requests?.values?.count || 0,
      p95_ms: data.metrics.order_request_duration?.values?.p(95)?.toFixed(2),
    },
    ai_service: {
      requests: data.metrics.ai_requests?.values?.count || 0,
      p95_ms: data.metrics.ai_request_duration?.values?.p(95)?.toFixed(2),
    },

    thresholds_passed: data.metrics.http_req_duration?.thresholds
      ? Object.entries(data.metrics.http_req_duration.thresholds)
          .every(([, v]) => v.ok)
      : false,
  };

  return {
    "tests/load/results/summary.json": JSON.stringify(summary, null, 2),
    stdout: `
==========================================================
  FullStack k6 Load Test Summary
==========================================================
  Duration:     ${summary.test_duration_seconds}s
  Requests:     ${summary.total_requests}
  Error Rate:   ${(summary.error_rate * 100).toFixed(2)}%
  Failed:       ${(summary.failed_requests * 100).toFixed(2)}%
----------------------------------------------------------
  Latency (ms):
    Avg: ${summary.latency.avg_ms}  P50: ${summary.latency.p50_ms}
    P90: ${summary.latency.p90_ms}  P95: ${summary.latency.p95_ms}
    P99: ${summary.latency.p99_ms}  Max: ${summary.latency.max_ms}
----------------------------------------------------------
  User Service:  ${summary.user_service.requests} reqs, P95=${summary.user_service.p95_ms}ms
  Order Service: ${summary.order_service.requests} reqs, P95=${summary.order_service.p95_ms}ms
  AI Service:    ${summary.ai_service.requests} reqs, P95=${summary.ai_service.p95_ms}ms
----------------------------------------------------------
  Thresholds: ${summary.thresholds_passed ? "PASSED" : "FAILED"}
==========================================================
`,
  };
}
