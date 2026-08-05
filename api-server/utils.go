package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": data})
}

func okMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"code": code, "message": msg})
}

func parseCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func parseModels(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(strings.Trim(s, "[]"), ",")
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return defaultVal
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func isServiceReachable(url string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func redisIncr(key string) {
	if rdb == nil {
		return
	}
	ctx, cancel := contextWithTimeout(2 * time.Second)
	defer cancel()
	rdb.Incr(ctx, key)
}

// readSymlink 读取符号链接目标（用于 Linux /proc/PID/exe）
func readSymlink(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// maskKey 脱敏 API Key，只保留前4位和后4位
func maskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
