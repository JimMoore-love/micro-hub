package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "microhub-jwt-secret-2026"))

// CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// 请求日志中间件
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		latency := time.Since(start)
		if latency > 100*time.Millisecond {
			log.Printf("[WARN] %s %s %d %v (slow)", c.Request.Method, path, c.Writer.Status(), latency)
		} else {
			log.Printf("[INFO] %s %s %d %v", c.Request.Method, path, c.Writer.Status(), latency)
		}
	}
}

// JWT 认证中间件
func jwtAuthMiddleware() gin.HandlerFunc {
	excludedPaths := map[string]bool{
		"/health": true, "/metrics": true,
		"/api/v1/auth/login": true,
	}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if excludedPaths[path] {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证令牌"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "认证令牌无效或已过期"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("username", claims["username"])
			c.Set("role", claims["role"])
			c.Set("tenant_id", claims["tenant_id"])
		}
		c.Next()
	}
}

// 租户解析中间件 — 从 Header 提取 X-Tenant-ID，注入到 context
func tenantResolver() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "default"
		}
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// 限流中间件 — 基于 Redis 的滑动窗口限流
func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		if tenantID == "" {
			tenantID = "anonymous"
		}

		// Redis 不可用时降级到内存限流
		if rdb == nil {
			key := fmt.Sprintf("%s:%d", tenantID, time.Now().Unix()/60)
			if !memLimiter.allow(key, 500) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code":    429,
					"message": "请求频率超限，每分钟最多 500 次",
				})
				return
			}
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%d", tenantID, time.Now().Unix()/60)
		ctx, cancel := contextWithTimeout(1 * time.Second)
		defer cancel()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, 60*time.Second)
		}

		if count > 500 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求频率超限，每分钟最多 500 次",
			})
			return
		}
		c.Next()
	}
}

// 生成 JWT token
func generateToken(username, role, tenantID string) (string, error) {
	claims := jwt.MapClaims{
		"username":  username,
		"role":      role,
		"tenant_id": tenantID,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 简单的内存限流（Redis 不可用时的降级方案）
type memoryRateLimiter struct {
	mu    sync.Mutex
	store map[string]int
}

var memLimiter = &memoryRateLimiter{store: make(map[string]int)}

func (m *memoryRateLimiter) allow(key string, limit int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key]++
	if m.store[key] == 1 {
		go func() {
			time.Sleep(60 * time.Second)
			m.mu.Lock()
			delete(m.store, key)
			m.mu.Unlock()
		}()
	}
	return m.store[key] <= limit
}
