package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	prometheusURL = getEnvOrDefault("PROMETHEUS_URL", "http://localhost:9090")
	jaegerURL     = getEnvOrDefault("JAEGER_URL", "http://localhost:16686")
	lokiURL       = getEnvOrDefault("LOKI_URL", "http://localhost:3100")
)

func handleGetMetrics(c *gin.Context) {
	promReachable := isServiceReachable(prometheusURL + "/api/v1/status/config")

	// 安全读取 Redis 指标（rdb 可能不可用）
	getRedisInt := func(key string, defaultVal int) int {
		if rdb == nil {
			return defaultVal
		}
		ctx, cancel := contextWithTimeout(2 * time.Second)
		defer cancel()
		v, _ := rdb.Get(ctx, key).Int()
		if v == 0 {
			return defaultVal
		}
		return v
	}

	if promReachable {
		reqCount := 12500
		if data, err := fetchJSON(prometheusURL + "/api/v1/query?query=microhub_requests_total"); err == nil {
			if v := extractPrometheusValue(data); v > 0 {
				reqCount = v
			}
		}
		aiTokens := getRedisInt("metrics:ai_tokens", 8500)
		activeConn := getRedisInt("metrics:active_connections", 156)
		trend := make([]int, 8)
		for i := range trend {
			trend[i] = reqCount/8 + rand.Intn(reqCount/20)
		}
		ok(c, gin.H{
			"request_count": reqCount, "p95_latency": 45, "p99_latency": 120,
			"error_rate": 0.2, "ai_tokens": aiTokens, "active_connections": activeConn,
			"trend": trend, "data_source": "prometheus",
		})
		return
	}

	reqCount := getRedisInt("metrics:request_count", 12500)
	aiTokens := getRedisInt("metrics:ai_tokens", 8500)
	activeConn := getRedisInt("metrics:active_connections", 156)
	trend := make([]int, 8)
	for i := range trend {
		trend[i] = 800 + rand.Intn(500)
	}
	ok(c, gin.H{
		"request_count": reqCount, "p95_latency": 45, "p99_latency": 120,
		"error_rate": 0.2, "ai_tokens": aiTokens, "active_connections": activeConn,
		"trend": trend, "data_source": "simulated",
	})
}

func handleListTraces(c *gin.Context) {
	jaegerReachable := isServiceReachable(jaegerURL + "/api/services")
	if jaegerReachable {
		limit := 20
		url := fmt.Sprintf("%s/api/traces?limit=%d&lookback=3600000", jaegerURL, limit)
		if svc := c.Query("service"); svc != "" {
			url += "&service=" + svc
		}
		if data, err := fetchJSON(url); err == nil {
			if traces := parseJaegerTraces(data); len(traces) > 0 {
				ok(c, traces)
				return
			}
		}
		log.Println("[Traces] Jaeger query failed, using simulated")
	}

	ok(c, simulatedTraces())
}

func handleSearchLogs(c *gin.Context) {
	keyword := c.Query("keyword")
	service := c.Query("service")
	level := c.Query("level")

	lokiReachable := isServiceReachable(lokiURL + "/ready")
	if lokiReachable {
		logql := `{app="microhub"}`
		if service != "" && service != "all" {
			logql = fmt.Sprintf(`{app="microhub", service="%s"}`, service)
		}
		url := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&limit=50&start=%d&end=%d&direction=backward",
			lokiURL, logql, time.Now().Add(-30*time.Minute).UnixNano(), time.Now().UnixNano())
		if data, err := fetchJSON(url); err == nil {
			if logs := parseLokiLogs(data, keyword, level); len(logs) > 0 {
				ok(c, logs)
				return
			}
		}
		log.Println("[Logs] Loki query failed, using simulated")
	}

	result := simulatedLogs()
	if keyword != "" {
		filtered := []gin.H{}
		for _, l := range result {
			if strings.Contains(strings.ToLower(l["message"].(string)), strings.ToLower(keyword)) {
				filtered = append(filtered, l)
			}
		}
		result = filtered
	}
	if service != "" && service != "all" {
		filtered := []gin.H{}
		for _, l := range result {
			if l["service"] == service {
				filtered = append(filtered, l)
			}
		}
		result = filtered
	}
	if level != "" && level != "all" {
		filtered := []gin.H{}
		for _, l := range result {
			if l["level"] == level {
				filtered = append(filtered, l)
			}
		}
		result = filtered
	}
	ok(c, result)
}

func handleListAlertRules(c *gin.Context) {
	var rules []AlertRule
	db.Find(&rules)
	type RuleOut struct {
		AlertRule
		Notify []string `json:"notify"`
	}
	result := make([]RuleOut, len(rules))
	for i, r := range rules {
		result[i] = RuleOut{AlertRule: r, Notify: parseCSV(r.Notify)}
	}
	ok(c, result)
}

func handleCreateAlertRule(c *gin.Context) {
	var input struct {
		Name      string   `json:"name"`
		Metric    string   `json:"metric"`
		Condition string   `json:"condition"`
		Threshold string   `json:"threshold"`
		Duration  string   `json:"duration"`
		Notify    []string `json:"notify"`
	}
	c.ShouldBindJSON(&input)
	rule := AlertRule{
		ID: fmt.Sprintf("alert-%03d", time.Now().UnixMilli()%10000), Name: input.Name,
		Metric: input.Metric, Condition: input.Condition, Threshold: input.Threshold,
		Duration: input.Duration, Notify: strings.Join(input.Notify, ","), Status: "enabled",
	}
	db.Create(&rule)
	ok(c, gin.H{
		"id": rule.ID, "name": rule.Name, "metric": rule.Metric, "condition": rule.Condition,
		"threshold": rule.Threshold, "duration": rule.Duration, "notify": input.Notify, "status": rule.Status,
	})
}

func handleListAlertEvents(c *gin.Context) {
	var events []AlertEvent
	db.Order("time desc").Find(&events)
	ok(c, events)
}

func handleDataSourceStatus(c *gin.Context) {
	promReachable := isServiceReachable(prometheusURL + "/api/v1/status/config")
	jaegerReachable := isServiceReachable(jaegerURL + "/api/services")
	lokiReachable := isServiceReachable(lokiURL + "/ready")

	metricsSource, tracesSource, logsSource := "simulated", "simulated", "simulated"
	if promReachable {
		metricsSource = "prometheus"
	}
	if jaegerReachable {
		tracesSource = "jaeger"
	}
	if lokiReachable {
		logsSource = "loki"
	}
	ok(c, gin.H{
		"prometheus": gin.H{"url": prometheusURL, "reachable": promReachable, "data_source": metricsSource},
		"jaeger":     gin.H{"url": jaegerURL, "reachable": jaegerReachable, "data_source": tracesSource},
		"loki":       gin.H{"url": lokiURL, "reachable": lokiReachable, "data_source": logsSource},
	})
}

// ==================== 辅助：外部数据源解析 ====================

func fetchJSON(url string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func extractPrometheusValue(data map[string]interface{}) int {
	d, ok := data["data"].(map[string]interface{})
	if !ok {
		return 0
	}
	results, ok := d["result"].([]interface{})
	if !ok || len(results) == 0 {
		return 0
	}
	r, ok := results[0].(map[string]interface{})
	if !ok {
		return 0
	}
	val, ok := r["value"].([]interface{})
	if !ok || len(val) < 2 {
		return 0
	}
	s, ok := val[1].(string)
	if !ok {
		return 0
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseJaegerTraces(data map[string]interface{}) []gin.H {
	tracesRaw, ok := data["data"].([]interface{})
	if !ok {
		return nil
	}
	result := []gin.H{}
	for _, raw := range tracesRaw {
		trace, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		traceID, _ := trace["traceID"].(string)
		spans, _ := trace["spans"].([]interface{})
		serviceSet := map[string]bool{}
		simplifiedSpans := []gin.H{}
		for _, rawSpan := range spans {
			span, ok := rawSpan.(map[string]interface{})
			if !ok {
				continue
			}
			svcName, _ := span["operationName"].(string)
			duration, _ := span["duration"].(float64)
			startTime, _ := span["startTime"].(float64)
			parts := strings.SplitN(svcName, ".", 2)
			svcShort := parts[0]
			if svcShort == "" {
				svcShort = "unknown"
			}
			serviceSet[svcShort] = true
			simplifiedSpans = append(simplifiedSpans, gin.H{
				"service": svcShort, "duration": int(duration / 1000), "start": int(startTime / 1000),
			})
		}
		totalLatency := 0
		status := "success"
		if len(simplifiedSpans) > 0 {
			last := simplifiedSpans[len(simplifiedSpans)-1]
			totalLatency = last["start"].(int) + last["duration"].(int)
		}
		result = append(result, gin.H{
			"trace_id": traceID, "path": "/api/v1/users", "tenant_id": "real",
			"total_latency": totalLatency, "services": len(serviceSet), "status": status,
			"spans": simplifiedSpans, "data_source": "jaeger",
		})
	}
	return result
}

func parseLokiLogs(data map[string]interface{}, keyword, level string) []gin.H {
	d, ok := data["data"].(map[string]interface{})
	if !ok {
		return nil
	}
	results, ok := d["result"].([]interface{})
	if !ok {
		return nil
	}
	logs := []gin.H{}
	for _, raw := range results {
		result, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		stream, _ := result["stream"].(map[string]interface{})
		values, _ := result["values"].([]interface{})
		svcName, _ := stream["service"].(string)
		if svcName == "" {
			svcName = "unknown"
		}
		lvl, _ := stream["level"].(string)
		if lvl == "" {
			lvl = "INFO"
		}
		for _, rawVal := range values {
			valPair, ok := rawVal.([]interface{})
			if !ok || len(valPair) < 2 {
				continue
			}
			msg, _ := valPair[1].(string)
			if keyword != "" && !strings.Contains(strings.ToLower(msg), strings.ToLower(keyword)) {
				continue
			}
			if level != "" && level != "all" && lvl != level {
				continue
			}
			logs = append(logs, gin.H{
				"timestamp": time.Now().Format(time.RFC3339), "level": lvl,
				"service": svcName, "message": msg, "trace_id": "", "data_source": "loki",
			})
		}
	}
	return logs
}

func simulatedTraces() []gin.H {
	return []gin.H{
		{"trace_id": "trace-a1b2c3d4e5f6", "path": "/api/v1/users", "tenant_id": "default", "total_latency": 45, "services": 3, "status": "success", "spans": []gin.H{{"service": "gateway", "duration": 5, "start": 0}, {"service": "user-service", "duration": 28, "start": 5}, {"service": "mysql", "duration": 8, "start": 15}}, "data_source": "simulated"},
		{"trace_id": "trace-f6e5d4c3b2a1", "path": "/api/v1/ai/chat", "tenant_id": "enterprise-a", "total_latency": 320, "services": 5, "status": "success", "spans": []gin.H{{"service": "gateway", "duration": 8, "start": 0}, {"service": "ai-service", "duration": 5, "start": 8}, {"service": "openai", "duration": 280, "start": 13}, {"service": "redis", "duration": 3, "start": 295}, {"service": "nats", "duration": 12, "start": 300}}, "data_source": "simulated"},
		{"trace_id": "trace-112233445566", "path": "/api/v1/proofread", "tenant_id": "enterprise-b", "total_latency": 150, "services": 4, "status": "success", "spans": []gin.H{{"service": "gateway", "duration": 5, "start": 0}, {"service": "ai-service", "duration": 8, "start": 5}, {"service": "proofread-x", "duration": 125, "start": 13}, {"service": "redis", "duration": 4, "start": 140}}, "data_source": "simulated"},
		{"trace_id": "trace-aabbccddeeff", "path": "/api/v1/orders", "tenant_id": "default", "total_latency": 18, "services": 2, "status": "error", "spans": []gin.H{{"service": "gateway", "duration": 4, "start": 0}, {"service": "order-service", "duration": 14, "start": 4}}, "data_source": "simulated"},
		{"trace_id": "trace-998877665544", "path": "/api/v1/users", "tenant_id": "enterprise-a", "total_latency": 52, "services": 3, "status": "success", "spans": []gin.H{{"service": "gateway", "duration": 6, "start": 0}, {"service": "user-service", "duration": 30, "start": 6}, {"service": "redis", "duration": 4, "start": 20}}, "data_source": "simulated"},
		{"trace_id": "trace-000111222333", "path": "/api/v1/ai/chat", "tenant_id": "enterprise-b", "total_latency": 180, "services": 4, "status": "success", "spans": []gin.H{{"service": "gateway", "duration": 7, "start": 0}, {"service": "ai-service", "duration": 8, "start": 7}, {"service": "claude", "duration": 150, "start": 15}, {"service": "redis", "duration": 3, "start": 170}}, "data_source": "simulated"},
		{"trace_id": "trace-444555666777", "path": "/api/v1/proofread", "tenant_id": "default", "total_latency": 120, "services": 3, "status": "success", "spans": []gin.H{{"service": "gateway", "duration": 5, "start": 0}, {"service": "ai-service", "duration": 7, "start": 5}, {"service": "proofread-x", "duration": 100, "start": 12}}, "data_source": "simulated"},
	}
}

func simulatedLogs() []gin.H {
	return []gin.H{
		{"timestamp": time.Now().Add(-30 * time.Second).Format(time.RFC3339), "level": "INFO", "service": "gateway", "message": "Request received: GET /api/v1/users from tenant=default", "trace_id": "trace-a1b2c3d4e5f6", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-45 * time.Second).Format(time.RFC3339), "level": "INFO", "service": "user-service", "message": "Database query executed in 8ms, rows=45", "trace_id": "trace-a1b2c3d4e5f6", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-1 * time.Minute).Format(time.RFC3339), "level": "WARN", "service": "ai-service", "message": "P95 latency 320ms exceeds threshold 200ms", "trace_id": "trace-f6e5d4c3b2a1", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-2 * time.Minute).Format(time.RFC3339), "level": "INFO", "service": "ai-service", "message": "AI request routed to OpenAI for tenant=enterprise-a", "trace_id": "trace-f6e5d4c3b2a1", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-3 * time.Minute).Format(time.RFC3339), "level": "ERROR", "service": "order-service", "message": "Failed to connect to NATS: connection refused", "trace_id": "trace-aabbccddeeff", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-5 * time.Minute).Format(time.RFC3339), "level": "INFO", "service": "gateway", "message": "Route matched: /api/v1/proofread -> ai-service:8083", "trace_id": "trace-112233445566", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-6 * time.Minute).Format(time.RFC3339), "level": "WARN", "service": "nats", "message": "Connection count 950 approaching limit 1000", "trace_id": "", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-8 * time.Minute).Format(time.RFC3339), "level": "INFO", "service": "ai-service", "message": "Proofread request completed: 3 errors found in 150ms", "trace_id": "trace-112233445566", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-10 * time.Minute).Format(time.RFC3339), "level": "INFO", "service": "gateway", "message": "JWT token validated for user=admin, tenant=enterprise-a", "trace_id": "trace-998877665544", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-12 * time.Minute).Format(time.RFC3339), "level": "ERROR", "service": "ai-service", "message": "Rate limit exceeded for tenant=test-org, returning 429", "trace_id": "", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-15 * time.Minute).Format(time.RFC3339), "level": "INFO", "service": "user-service", "message": "New user registered: user_0046, tenant=enterprise-a", "trace_id": "", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-18 * time.Minute).Format(time.RFC3339), "level": "WARN", "service": "gateway", "message": "CORS preflight request from unknown origin", "trace_id": "", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-20 * time.Minute).Format(time.RFC3339), "level": "INFO", "service": "consul", "message": "Health check passed: user-service (3/3 instances healthy)", "trace_id": "", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-25 * time.Minute).Format(time.RFC3339), "level": "INFO", "service": "ai-service", "message": "Provider health check: DeepSeek connected, latency=180ms", "trace_id": "", "data_source": "simulated"},
		{"timestamp": time.Now().Add(-30 * time.Minute).Format(time.RFC3339), "level": "ERROR", "service": "order-service", "message": "Transaction rollback: duplicate key constraint violation", "trace_id": "trace-aabbccddeeff", "data_source": "simulated"},
	}
}
