package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db  *gorm.DB
	rdb *redis.Client
)

func initDB() {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "3306")
	dbUser := getEnvOrDefault("DB_USER", "root")
	dbPass := getEnvOrDefault("DB_PASS", "root")
	dbName := getEnvOrDefault("DB_NAME", "microhub")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPass, dbHost, dbPort)
	tmpDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}
	sqlDB, _ := tmpDB.DB()
	tmpDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName))
	sqlDB.Close()

	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		log.Fatalf("连接 %s 数据库失败: %v", dbName, err)
	}

	db.AutoMigrate(
		&Service{}, &Tenant{}, &APIKey{}, &AIProvider{}, &RoutingRule{},
		&GatewayRoute{}, &ProofreadLog{}, &AlertRule{}, &AlertEvent{}, &User{}, &Node{},
	)
	log.Println("[DB] MySQL 连接成功，表结构已迁移")
}

func initRedis() {
	redisAddr := getEnvOrDefault("REDIS_ADDR", "127.0.0.1:6379")
	redisPwd := getEnvOrDefault("REDIS_PASSWORD", "")
	rdb = redis.NewClient(&redis.Options{Addr: redisAddr, Password: redisPwd, DB: 0})
	ctx, cancel := contextWithTimeout(3 * time.Second)
	defer cancel()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("[Redis] 连接失败（降级运行，无实时指标）: %v", err)
		rdb = nil
		return
	}
	log.Println("[Redis] 连接成功")
}

func seedData() {
	var count int64
	db.Model(&Service{}).Count(&count)
	if count > 0 {
		log.Println("[Seed] 数据已存在，跳过")
		return
	}
	log.Println("[Seed] 开始写入种子数据...")

	db.Create(&[]Service{
		{ID: "gateway", Name: "API Gateway", Type: "gateway", Port: 8080, Host: "127.0.0.1", Status: "healthy", Version: "v1.2.0", QPS: 1250, P95: 45, ErrorRate: 0.2, Instances: 2, Dependencies: "user-service,order-service,ai-service", ConsulID: "svc-gateway-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "user-service", Name: "User Service", Type: "service", Port: 8081, Host: "127.0.0.1", Status: "healthy", Version: "v1.0.3", QPS: 380, P95: 12, ErrorRate: 0.1, Instances: 3, Dependencies: "mysql,redis", ConsulID: "svc-user-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "order-service", Name: "Order Service", Type: "service", Port: 8082, Host: "127.0.0.1", Status: "healthy", Version: "v0.9.1", QPS: 220, P95: 18, ErrorRate: 0.3, Instances: 2, Dependencies: "mysql,redis,nats", ConsulID: "svc-order-001", RegisteredAt: time.Now().Add(-48 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "ai-service", Name: "AI Service", Type: "service", Port: 8083, Host: "127.0.0.1", Status: "warning", Version: "v0.8.0", QPS: 85, P95: 320, ErrorRate: 1.5, Instances: 1, Dependencies: "redis,nats", ConsulID: "svc-ai-001", RegisteredAt: time.Now().Add(-24 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "mysql", Name: "MySQL", Type: "infra", Port: 3306, Host: "127.0.0.1", Status: "healthy", Version: "8.x", Instances: 1, ConsulID: "infra-mysql-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "redis", Name: "Redis", Type: "infra", Port: 6379, Host: "127.0.0.1", Status: "healthy", Version: "7.2", Instances: 1, ConsulID: "infra-redis-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "consul", Name: "Consul", Type: "infra", Port: 8500, Host: "127.0.0.1", Status: "healthy", Version: "1.29", Instances: 1, ConsulID: "infra-consul-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "nats", Name: "NATS", Type: "infra", Port: 4222, Host: "127.0.0.1", Status: "warning", Version: "2.10", Instances: 1, ConsulID: "infra-nats-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "minio", Name: "MinIO", Type: "infra", Port: 9000, Host: "127.0.0.1", Status: "healthy", Version: "RELEASE.2024", Instances: 1, ConsulID: "infra-minio-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "jaeger", Name: "Jaeger", Type: "observability", Port: 16686, Host: "127.0.0.1", Status: "healthy", Version: "1.54", Instances: 1, ConsulID: "obs-jaeger-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
		{ID: "prometheus", Name: "Prometheus", Type: "observability", Port: 9090, Host: "127.0.0.1", Status: "healthy", Version: "2.51", Instances: 1, ConsulID: "obs-prom-001", RegisteredAt: time.Now().Add(-72 * time.Hour), Source: "seed", LastChecked: time.Now()},
	})

	db.Create(&[]Tenant{
		{ID: "default", Name: "默认租户", Schema: "public", Users: 45, Quota: 50000, Used: 12000, Status: "active", Plan: "free", CreatedAt: time.Now().Add(-90 * 24 * time.Hour), RedisPrefix: "tenant:default:", DBTables: 12, DBRecords: 3450},
		{ID: "enterprise-a", Name: "企业A", Schema: "tenant_ea", Users: 68, Quota: 200000, Used: 85000, Status: "active", Plan: "pro", CreatedAt: time.Now().Add(-60 * 24 * time.Hour), RedisPrefix: "tenant:enterprise-a:", DBTables: 18, DBRecords: 8920},
		{ID: "enterprise-b", Name: "企业B", Schema: "tenant_eb", Users: 32, Quota: 100000, Used: 42000, Status: "active", Plan: "standard", CreatedAt: time.Now().Add(-30 * 24 * time.Hour), RedisPrefix: "tenant:enterprise-b:", DBTables: 15, DBRecords: 5210},
		{ID: "test-org", Name: "测试组织", Schema: "tenant_test", Users: 11, Quota: 10000, Used: 9500, Status: "frozen", Plan: "free", CreatedAt: time.Now().Add(-15 * 24 * time.Hour), RedisPrefix: "tenant:test-org:", DBTables: 8, DBRecords: 420},
	})

	db.Create(&[]APIKey{
		{Key: "mh-default-sk-2026a7b3", TenantID: "default", Status: "active", CreatedAt: time.Now().Add(-90 * 24 * time.Hour)},
		{Key: "mh-default-sk-2026f2c8", TenantID: "default", Status: "active", CreatedAt: time.Now().Add(-30 * 24 * time.Hour)},
		{Key: "mh-ea-sk-20260115xyz", TenantID: "enterprise-a", Status: "active", CreatedAt: time.Now().Add(-60 * 24 * time.Hour)},
		{Key: "mh-ea-sk-20260320abc", TenantID: "enterprise-a", Status: "disabled", CreatedAt: time.Now().Add(-10 * 24 * time.Hour)},
		{Key: "mh-eb-sk-20260218def", TenantID: "enterprise-b", Status: "active", CreatedAt: time.Now().Add(-30 * 24 * time.Hour)},
		{Key: "mh-test-sk-20260601ghi", TenantID: "test-org", Status: "active", CreatedAt: time.Now().Add(-15 * 24 * time.Hour)},
	})

	db.Create(&[]AIProvider{
		{ID: "openai", Name: "OpenAI", Type: "llm", Icon: "robot", Status: "connected", Requests: 850, Latency: 320, CostPer1k: 0.03, Models: `["gpt-4o","gpt-4o-mini"]`, APIKey: "sk-proj-xxxxxxxxxxxxxxxx", Endpoint: "https://api.openai.com/v1"},
		{ID: "claude", Name: "Claude", Type: "llm", Icon: "brain", Status: "connected", Requests: 220, Latency: 450, CostPer1k: 0.015, Models: `["claude-3.5-sonnet","claude-3-opus"]`, APIKey: "sk-ant-xxxxxxxxxxxxx", Endpoint: "https://api.anthropic.com/v1"},
		{ID: "deepseek", Name: "DeepSeek", Type: "llm", Icon: "search", Status: "connected", Requests: 580, Latency: 180, CostPer1k: 0.001, Models: `["deepseek-chat","deepseek-coder"]`, APIKey: "sk-ds-xxxxxxxxxxxxx", Endpoint: "https://api.deepseek.com/v1"},
		{ID: "proofread-x", Name: "校对厂商X", Type: "proofread", Icon: "edit", Status: "connected", Requests: 340, Latency: 150, CostPer1k: 0.005, Models: `["proofread-v2","grammar-check-v1"]`, APIKey: "pk-xxxxxxxxxxxxx", Endpoint: "https://api.proofread-x.com/v2"},
		{ID: "translate-y", Name: "翻译服务Y", Type: "translate", Icon: "globe", Status: "testing", Requests: 45, Latency: 200, CostPer1k: 0.002, Models: `["translate-en-zh","translate-zh-en"]`, Endpoint: "https://api.translate-y.com/v1"},
	})

	db.Create(&[]RoutingRule{
		{ID: "rr-001", TenantID: "default", ProviderID: "deepseek", Priority: 1, Condition: "成本优先", Enabled: true},
		{ID: "rr-002", TenantID: "enterprise-a", ProviderID: "openai", Priority: 1, Condition: "质量优先", Enabled: true},
		{ID: "rr-003", TenantID: "enterprise-b", ProviderID: "claude", Priority: 1, Condition: "均衡", Enabled: true},
		{ID: "rr-004", TenantID: "*", ProviderID: "proofread-x", Priority: 2, Condition: "类型路由:校对", Enabled: true},
		{ID: "rr-005", TenantID: "*", ProviderID: "translate-y", Priority: 2, Condition: "类型路由:翻译", Enabled: true},
	})

	rl1, rl2, rl3, rl4 := 100, 200, 50, 30
	db.Create(&[]GatewayRoute{
		{ID: "route-001", Path: "/api/v1/users/*", Upstream: "user-service:8081", Methods: "GET,POST,PUT,DELETE", RateLimit: &rl1, TenantRouting: true, Status: "enabled"},
		{ID: "route-002", Path: "/api/v1/orders/*", Upstream: "order-service:8082", Methods: "GET,POST,PUT", RateLimit: &rl2, TenantRouting: true, Status: "enabled"},
		{ID: "route-003", Path: "/api/v1/ai/*", Upstream: "ai-service:8083", Methods: "POST", RateLimit: &rl3, TenantRouting: true, Status: "enabled"},
		{ID: "route-004", Path: "/api/v1/proofread/*", Upstream: "ai-service:8083", Methods: "POST", RateLimit: &rl4, TenantRouting: true, Status: "enabled"},
		{ID: "route-005", Path: "/health", Upstream: "gateway:8080", Methods: "GET", Status: "enabled"},
		{ID: "route-006", Path: "/metrics", Upstream: "gateway:8080", Methods: "GET", Status: "enabled"},
	})

	for i := 0; i < 20; i++ {
		tenants := []string{"default", "enterprise-a", "enterprise-b", "test-org"}
		statuses := []string{"success", "success", "success", "success", "success", "timeout", "rate_limit", "error"}
		db.Create(&ProofreadLog{
			ID: fmt.Sprintf("pr-%04d", i+1), Time: time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
			TenantID: tenants[rand.Intn(len(tenants))], TextLength: 50 + rand.Intn(500), ErrorCount: rand.Intn(5),
			Latency: 100 + rand.Intn(200), Provider: "proofread-x", Status: statuses[rand.Intn(len(statuses))],
		})
	}

	db.Create(&[]AlertRule{
		{ID: "alert-001", Name: "AI服务延迟过高", Metric: "P95延迟", Condition: ">", Threshold: "200ms", Duration: "5min", Notify: "dingtalk,email", Status: "enabled"},
		{ID: "alert-002", Name: "网关错误率超限", Metric: "error_rate", Condition: ">", Threshold: "1%", Duration: "3min", Notify: "dingtalk", Status: "enabled"},
		{ID: "alert-003", Name: "NATS连接断开", Metric: "health", Condition: "==", Threshold: "down", Duration: "0min", Notify: "dingtalk,sms", Status: "enabled"},
		{ID: "alert-004", Name: "Redis内存超限", Metric: "memory_usage", Condition: ">", Threshold: "80%", Duration: "15min", Notify: "email", Status: "disabled"},
		{ID: "alert-005", Name: "服务实例下线", Metric: "instances", Condition: "<", Threshold: "1", Duration: "1min", Notify: "dingtalk,sms,email", Status: "enabled"},
	})

	db.Create(&[]AlertEvent{
		{ID: "ev-001", Time: time.Now().Add(-2 * time.Hour), RuleName: "AI服务延迟过高", Level: "warning", TriggerValue: "320ms", Threshold: "200ms", Duration: "8min", Status: "resolved", Handler: "auto-scaler"},
		{ID: "ev-002", Time: time.Now().Add(-5 * time.Hour), RuleName: "NATS连接断开", Level: "critical", TriggerValue: "down", Threshold: "down", Duration: "3min", Status: "resolved", Handler: "admin"},
		{ID: "ev-003", Time: time.Now().Add(-30 * time.Minute), RuleName: "AI配额超80%", Level: "warning", TriggerValue: "95%", Threshold: "80%", Duration: "30min", Status: "firing", Handler: ""},
	})

	// 默认管理员账号
	db.Create(&User{Username: "admin", Password: hashPassword("admin123"), Role: "admin", TenantID: "default", CreatedAt: time.Now()})

	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()
	if rdb != nil {
		rdb.Set(ctx, "metrics:request_count", 12500, 0)
		rdb.Set(ctx, "metrics:ai_tokens", 8500, 0)
		rdb.Set(ctx, "metrics:active_connections", 156, 0)
		rdb.Set(ctx, "tenant:default:usage", 12000, 0)
		rdb.Set(ctx, "tenant:enterprise-a:usage", 85000, 0)
		rdb.Set(ctx, "tenant:enterprise-b:usage", 42000, 0)
		rdb.Set(ctx, "tenant:test-org:usage", 9500, 0)
		rdb.HSet(ctx, "health:services", "gateway", "healthy", "user-service", "healthy", "ai-service", "warning", "nats", "warning")
	}

	log.Println("[Seed] 种子数据写入完成")
}

// ensureAdminUser 确保管理员账号存在（即使种子数据已跳过）
func ensureAdminUser() {
	var count int64
	db.Model(&User{}).Count(&count)
	if count > 0 {
		return
	}
	db.Create(&User{Username: "admin", Password: hashPassword("admin123"), Role: "admin", TenantID: "default", CreatedAt: time.Now()})
	log.Println("[Seed] 管理员账号已创建 (admin / admin123)")
}
