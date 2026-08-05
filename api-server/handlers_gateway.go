package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func handleListRoutes(c *gin.Context) {
	var routes []GatewayRoute
	db.Find(&routes)
	type RouteOut struct {
		GatewayRoute
		Methods []string `json:"methods"`
	}
	result := make([]RouteOut, len(routes))
	for i, r := range routes {
		result[i] = RouteOut{GatewayRoute: r, Methods: parseCSV(r.Methods)}
	}
	ok(c, result)
}

func handleCreateRoute(c *gin.Context) {
	var input struct {
		Path          string   `json:"path"`
		Upstream      string   `json:"upstream"`
		Methods       []string `json:"methods"`
		RateLimit     *int     `json:"rate_limit"`
		TenantRouting bool     `json:"tenant_routing"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	route := GatewayRoute{
		ID: fmt.Sprintf("route-%03d", time.Now().UnixMilli()%1000), Path: input.Path,
		Upstream: input.Upstream, Methods: strings.Join(input.Methods, ","),
		RateLimit: input.RateLimit, TenantRouting: input.TenantRouting, Status: "enabled",
	}
	db.Create(&route)
	ok(c, gin.H{
		"id": route.ID, "path": route.Path, "upstream": route.Upstream,
		"methods": input.Methods, "rate_limit": route.RateLimit,
		"tenant_routing": route.TenantRouting, "status": route.Status,
	})
}

func handleUpdateRoute(c *gin.Context) {
	id := c.Param("id")
	var route GatewayRoute
	if err := db.First(&route, "id = ?", id).Error; err != nil {
		fail(c, http.StatusNotFound, "路由不存在")
		return
	}
	var input map[string]interface{}
	c.ShouldBindJSON(&input)
	if v, ok := input["path"]; ok {
		route.Path = v.(string)
	}
	if v, ok := input["upstream"]; ok {
		route.Upstream = v.(string)
	}
	if v, ok := input["methods"]; ok {
		methods := v.([]interface{})
		strs := make([]string, len(methods))
		for i, m := range methods {
			strs[i] = m.(string)
		}
		route.Methods = strings.Join(strs, ",")
	}
	if v, ok := input["rate_limit"]; ok {
		if v == nil {
			route.RateLimit = nil
		} else {
			n := int(v.(float64))
			route.RateLimit = &n
		}
	}
	if v, ok := input["tenant_routing"]; ok {
		route.TenantRouting = v.(bool)
	}
	if v, ok := input["status"]; ok {
		route.Status = v.(string)
	}
	db.Save(&route)
	ok(c, gin.H{
		"id": route.ID, "path": route.Path, "upstream": route.Upstream,
		"methods": parseCSV(route.Methods), "rate_limit": route.RateLimit,
		"tenant_routing": route.TenantRouting, "status": route.Status,
	})
}

func handleDeleteRoute(c *gin.Context) {
	db.Delete(&GatewayRoute{}, "id = ?", c.Param("id"))
	okMsg(c, "已删除")
}

func handleGetMiddlewareConfig(c *gin.Context) {
	ok(c, gin.H{
		"cors": gin.H{
			"allowed_origins": []string{"http://localhost:3200", "https://app.microhub.io"},
			"methods":         []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			"headers":         []string{"Content-Type", "Authorization", "X-Tenant-ID", "X-Request-ID"},
		},
		"jwt": gin.H{
			"secret":         "mh-jwt-secret-2026-xxxxx",
			"expiry":         3600,
			"excluded_paths": []string{"/health", "/metrics", "/api/v1/auth/login"},
		},
		"rate_limit": gin.H{"global_rate": 5000, "per_tenant_rate": 500, "burst_size": 100},
		"tenant_routing": gin.H{
			"header_key": "X-Tenant-ID",
			"subdomain_mapping": gin.H{"ea": "enterprise-a", "eb": "enterprise-b", "test": "test-org"},
		},
	})
}

func handleUpdateMiddlewareConfig(c *gin.Context) {
	okMsg(c, "中间件配置已更新")
}

// ==================== 流量管理 ====================

func handleListCircuitBreakers(c *gin.Context) {
	ok(c, []gin.H{
		{"service": "user-service", "state": "closed", "failure_threshold": 5.0, "open_duration": 30, "half_open_probes": 3},
		{"service": "order-service", "state": "closed", "failure_threshold": 5.0, "open_duration": 30, "half_open_probes": 3},
		{"service": "ai-service", "state": "half_open", "failure_threshold": 3.0, "open_duration": 60, "half_open_probes": 2},
		{"service": "gateway", "state": "closed", "failure_threshold": 10.0, "open_duration": 15, "half_open_probes": 5},
	})
}

func handleUpdateCircuitBreaker(c *gin.Context) {
	okMsg(c, "熔断器配置已更新")
}

func handleListDegradation(c *gin.Context) {
	ok(c, []gin.H{
		{"service": "user-service", "condition": "错误率>5%", "response": "返回缓存数据", "enabled": true},
		{"service": "order-service", "condition": "延迟>500ms", "response": "返回稍后重试", "enabled": true},
		{"service": "ai-service", "condition": "超配额", "response": "返回429 Too Many Requests", "enabled": true},
		{"service": "gateway", "condition": "QPS>5000", "response": "启用排队模式", "enabled": false},
	})
}

func handleListRetry(c *gin.Context) {
	ok(c, []gin.H{
		{"service": "user-service", "max_retries": 3, "interval": 100, "retryable_codes": []int{500, 503}},
		{"service": "order-service", "max_retries": 2, "interval": 200, "retryable_codes": []int{500, 502, 503}},
		{"service": "ai-service", "max_retries": 1, "interval": 500, "retryable_codes": []int{429, 503}},
		{"service": "gateway", "max_retries": 2, "interval": 100, "retryable_codes": []int{502, 503}},
	})
}
