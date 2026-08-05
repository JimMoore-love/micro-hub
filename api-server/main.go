package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	initDB()
	initRedis()
	seedData()
	ensureAdminUser()

	go metricsSimulator()
	go healthChecker()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 中间件链（顺序很重要）
	r.Use(corsMiddleware())
	r.Use(requestLogger())

	api := r.Group("/api/v1")

	// 认证路由（不需要 JWT）
	auth := api.Group("/auth")
	{
		auth.POST("/login", handleLogin)
		auth.GET("/profile", jwtAuthMiddleware(), handleProfile)
	}

	// 业务路由组（JWT 认证 + 租户解析 + 限流）
	protected := api.Group("")
	protected.Use(jwtAuthMiddleware())
	protected.Use(tenantResolver())
	protected.Use(rateLimitMiddleware())

	// ==================== 服务治理 ====================
	svc := protected.Group("/services")
	{
		svc.GET("", handleListServices)
		svc.POST("", handleRegisterService)
		svc.POST("/discover", handleDiscoverServices)
		svc.POST("/refresh-health", handleRefreshHealth)
		svc.GET("/events", handleServiceEvents)
		svc.GET("/:id", handleGetService)
		svc.GET("/:id/health", handleServiceHealth)
		svc.PUT("/:id", handleUpdateService)
		svc.DELETE("/:id", handleDeleteService)
	}

	// ==================== API 网关 ====================
	gw := protected.Group("/gateway")
	{
		gw.GET("/routes", handleListRoutes)
		gw.POST("/routes", handleCreateRoute)
		gw.PUT("/routes/:id", handleUpdateRoute)
		gw.DELETE("/routes/:id", handleDeleteRoute)
		gw.GET("/middleware", handleGetMiddlewareConfig)
		gw.PUT("/middleware", handleUpdateMiddlewareConfig)
	}

	// ==================== 流量管理 ====================
	tf := protected.Group("/traffic")
	{
		tf.GET("/circuit-breakers", handleListCircuitBreakers)
		tf.PUT("/circuit-breakers/:service", handleUpdateCircuitBreaker)
		tf.GET("/degradation", handleListDegradation)
		tf.GET("/retry", handleListRetry)
	}

	// ==================== 租户管理 ====================
	tn := protected.Group("/tenants")
	{
		tn.GET("", handleListTenants)
		tn.POST("", handleCreateTenant)
		tn.GET("/:id", handleGetTenant)
		tn.PUT("/:id", handleUpdateTenant)
		tn.PUT("/:id/freeze", handleFreezeTenant)
		tn.PUT("/:id/unfreeze", handleUnfreezeTenant)
		tn.POST("/:id/api-keys", handleCreateApiKey)
		tn.DELETE("/:id/api-keys/:key", handleDeleteApiKey)
	}

	// ==================== AI 供应商 ====================
	ai := protected.Group("/ai")
	{
		ai.GET("/providers", handleListProviders)
		ai.POST("/providers", handleCreateProvider)
		ai.GET("/providers/:id", handleGetProvider)
		ai.PUT("/providers/:id", handleUpdateProvider)
		ai.DELETE("/providers/:id", handleDeleteProvider)
		ai.GET("/providers/:id/usage", handleProviderUsage)
		ai.GET("/providers/:id/health", handleProviderHealth)
		ai.GET("/routing-rules", handleListRoutingRules)
		ai.POST("/routing-rules", handleCreateRoutingRule)
		ai.PUT("/routing-rules/:id", handleUpdateRoutingRule)
	}

	// ==================== 校对 API ====================
	pr := protected.Group("/proofread")
	{
		pr.POST("", handleProofreadCheck)
		pr.GET("/config", handleProofreadConfig)
		pr.PUT("/config", handleUpdateMiddlewareConfig)
		pr.GET("/logs", handleProofreadLogs)
		pr.GET("/stats", handleProofreadStats)
	}

	// ==================== 可观测性 ====================
	ob := protected.Group("/observability")
	{
		ob.GET("/metrics", handleGetMetrics)
		ob.GET("/traces", handleListTraces)
		ob.GET("/logs", handleSearchLogs)
		ob.GET("/alerts/rules", handleListAlertRules)
		ob.POST("/alerts/rules", handleCreateAlertRule)
		ob.GET("/alerts/events", handleListAlertEvents)
		ob.GET("/datasource-status", handleDataSourceStatus)
	}

	// ==================== 节点管理 ====================
	nd := protected.Group("/nodes")
	{
		nd.GET("", handleListNodes)
		nd.POST("/scan-subnet", handleScanSubnet)
	}

	// ==================== Agent 上报（不需要 JWT，用 token 认证） ====================
	api.POST("/agents/report", handleAgentReport)

	// 健康检查（公开）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	r.GET("/metrics", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(200, "# HELP microhub_requests_total Total requests\n# TYPE microhub_requests_total counter\nmicrohub_requests_total 12500\n# HELP microhub_latency_p95 P95 latency\n# TYPE microhub_latency_p95 gauge\nmicrohub_latency_p95 45\n")
	})

	log.Println("========================================")
	log.Println("  MicroHub API Server")
	log.Println("  Listening on http://localhost:8081")
	log.Println("  MySQL: microhub database")
	log.Println("  Redis: localhost:6379")
	log.Println("  JWT: enabled (login: admin / admin123)")
	log.Println("========================================")

	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func metricsSimulator() {
	for {
		time.Sleep(5 * time.Second)
		if rdb == nil {
			continue
		}
		ctx, cancel := contextWithTimeout(2 * time.Second)
		rdb.IncrBy(ctx, "metrics:request_count", int64(10+rand.Intn(50)))
		rdb.IncrBy(ctx, "metrics:ai_tokens", int64(5+rand.Intn(30)))
		rdb.Set(ctx, "metrics:active_connections", 140+rand.Intn(40), 0)
		cancel()
	}
}

func healthChecker() {
	for {
		time.Sleep(30 * time.Second)
		var services []Service
		db.Find(&services)
		for _, svc := range services {
			address := fmt.Sprintf("%s:%d", svc.Host, svc.Port)
			conn, err := net.DialTimeout("tcp", address, 2*time.Second)
			oldStatus := svc.Status
			if err == nil {
				conn.Close()
				if svc.Status == "unreachable" || svc.Status == "critical" {
					svc.Status = "healthy"
				}
			} else {
				svc.Status = "critical"
			}
			svc.LastChecked = time.Now()
			// 只在状态变化时写库，减少无效 UPDATE
			if oldStatus != svc.Status {
				log.Printf("[HealthCheck] %s (%s:%d) %s -> %s", svc.ID, svc.Host, svc.Port, oldStatus, svc.Status)
				db.Save(&svc)
			} else {
				// 状态未变只更新 LastChecked 字段
				db.Model(&Service{}).Where("id = ?", svc.ID).Update("last_checked", svc.LastChecked)
			}
			if rdb != nil {
				ctx, cancel := contextWithTimeout(2 * time.Second)
				rdb.HSet(ctx, "health:services", svc.ID, svc.Status)
				cancel()
			}
		}
	}
}
