package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func handleListProviders(c *gin.Context) {
	var providers []AIProvider
	db.Find(&providers)
	type ProviderOut struct {
		AIProvider
		Models []string `json:"models"`
	}
	result := make([]ProviderOut, len(providers))
	for i, p := range providers {
		result[i] = ProviderOut{AIProvider: p, Models: parseModels(p.Models)}
	}
	ok(c, result)
}

func handleGetProvider(c *gin.Context) {
	var provider AIProvider
	if err := db.First(&provider, "id = ?", c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, "供应商不存在")
		return
	}
	ok(c, gin.H{
		"id": provider.ID, "name": provider.Name, "type": provider.Type, "icon": provider.Icon,
		"status": provider.Status, "requests": provider.Requests, "latency": provider.Latency,
		"cost_per_1k": provider.CostPer1k, "models": parseModels(provider.Models),
		"api_key": maskKey(provider.APIKey), "endpoint": provider.Endpoint,
	})
}

func handleCreateProvider(c *gin.Context) {
	var input AIProvider
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if input.ID == "" {
		input.ID = fmt.Sprintf("provider-%03d", time.Now().UnixMilli()%10000)
	}
	if input.Status == "" {
		input.Status = "disconnected"
	}
	db.Create(&input)
	ok(c, input)
}

func handleUpdateProvider(c *gin.Context) {
	var provider AIProvider
	if err := db.First(&provider, "id = ?", c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, "供应商不存在")
		return
	}
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if v, ok := input["name"].(string); ok {
		provider.Name = v
	}
	if v, ok := input["status"].(string); ok {
		provider.Status = v
	}
	if v, ok := input["endpoint"].(string); ok {
		provider.Endpoint = v
	}
	if v, ok := input["api_key"].(string); ok {
		provider.APIKey = v
	}
	if v, ok := input["cost_per_1k"].(float64); ok {
		provider.CostPer1k = v
	}
	db.Save(&provider)
	ok(c, provider)
}

func handleDeleteProvider(c *gin.Context) {
	db.Delete(&AIProvider{}, "id = ?", c.Param("id"))
	okMsg(c, "供应商已删除")
}

func handleListRoutingRules(c *gin.Context) {
	var rules []RoutingRule
	db.Order("priority asc").Find(&rules)
	ok(c, rules)
}

func handleCreateRoutingRule(c *gin.Context) {
	var input RoutingRule
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if input.ID == "" {
		input.ID = fmt.Sprintf("rr-%03d", time.Now().UnixMilli()%10000)
	}
	db.Create(&input)
	ok(c, input)
}

func handleUpdateRoutingRule(c *gin.Context) {
	var rule RoutingRule
	if err := db.First(&rule, "id = ?", c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, "路由规则不存在")
		return
	}
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if v, ok := input["enabled"].(bool); ok {
		rule.Enabled = v
	}
	if v, ok := input["priority"].(float64); ok {
		rule.Priority = int(v)
	}
	if v, ok := input["provider_id"].(string); ok {
		rule.ProviderID = v
	}
	if v, ok := input["condition"].(string); ok {
		rule.Condition = v
	}
	db.Save(&rule)
	ok(c, rule)
}

func handleProviderUsage(c *gin.Context) {
	var provider AIProvider
	db.First(&provider, "id = ?", c.Param("id"))
	ok(c, gin.H{
		"date": time.Now().Format("2006-01-02"),
		"input_tokens": provider.Requests * 120, "output_tokens": provider.Requests * 80,
		"total_tokens": provider.Requests * 200,
		"cost": float64(provider.Requests) * provider.CostPer1k * 0.2,
		"request_count": provider.Requests,
		"tenant_distribution": []gin.H{
			{"tenant": "default", "percentage": 35}, {"tenant": "enterprise-a", "percentage": 40},
			{"tenant": "enterprise-b", "percentage": 20}, {"tenant": "test-org", "percentage": 5},
		},
	})
}

func handleProviderHealth(c *gin.Context) {
	checks := []gin.H{}
	for i := 0; i < 5; i++ {
		latency := 100 + rand.Intn(300)
		status := "healthy"
		if latency > 400 {
			status = "warning"
		}
		checks = append(checks, gin.H{
			"time": time.Now().Add(-time.Duration(i*5) * time.Minute).Format(time.RFC3339),
			"status": status, "latency": latency,
		})
	}
	ok(c, gin.H{"checks": checks})
}

// ==================== 校对 API ====================

func handleProofreadCheck(c *gin.Context) {
	var input struct {
		Text     string   `json:"text"`
		Language string   `json:"language"`
		Checks   []string `json:"checks"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	if input.Language == "" {
		input.Language = "zh"
	}
	start := time.Now()

	type ErrorItem struct {
		Original   string  `json:"original"`
		Type       string  `json:"type"`
		Suggestion string  `json:"suggestion"`
		Confidence float64 `json:"confidence"`
		Position   [2]int  `json:"position"`
	}

	errors := []ErrorItem{}
	corrected := input.Text
	errorDict := map[string]struct {
		Suggestion string
		Type       string
	}{
		"目地": {"目的", "拼写错误"}, "问提": {"问题", "拼写错误"},
		"做为": {"作为", "用词错误"}, "截止": {"截至", "用词错误"},
		"况且": {"何况", "用词错误"}, "帐号": {"账号", "用词错误"},
		"登陆": {"登录", "用词错误"}, "帐户": {"账户", "用词错误"},
	}

	searchText := input.Text
	for word, fix := range errorDict {
		idx := strings.Index(searchText, word)
		for idx >= 0 {
			if word != fix.Suggestion {
				errors = append(errors, ErrorItem{
					Original: word, Type: fix.Type, Suggestion: fix.Suggestion,
					Confidence: 90 + float64(rand.Intn(10)), Position: [2]int{idx, idx + len(word)},
				})
				corrected = strings.Replace(corrected, word, fix.Suggestion, 1)
			}
			searchText = searchText[idx+len(word):]
			newIdx := strings.Index(searchText, word)
			if newIdx >= 0 {
				idx = idx + len(word) + newIdx
			} else {
				break
			}
		}
	}

	latency := int(time.Since(start).Milliseconds())
	if latency < 50 {
		latency = 100 + rand.Intn(100)
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	logEntry := ProofreadLog{
		ID: fmt.Sprintf("pr-%06d", time.Now().UnixMilli()%1000000), Time: time.Now(),
		TenantID: tenantID, TextLength: len(input.Text), ErrorCount: len(errors),
		Latency: latency, Provider: "proofread-x", Status: "success",
	}
	db.Create(&logEntry)
	redisIncr("metrics:proofread_count")

	ok(c, gin.H{
		"corrected_text": corrected, "errors": errors, "provider": "proofread-x",
		"latency": latency, "tokens": len(input.Text) / 4,
		"cost": float64(len(input.Text)) / 4 * 0.005 / 1000,
	})
}

func handleProofreadConfig(c *gin.Context) {
	var provider AIProvider
	db.First(&provider, "id = ?", "proofread-x")
	var rules []RoutingRule
	db.Where("provider_id = ?", "proofread-x").Find(&rules)
	ok(c, gin.H{
		"provider": gin.H{
			"id": provider.ID, "name": provider.Name, "type": provider.Type, "icon": provider.Icon,
			"status": provider.Status, "requests": provider.Requests, "latency": provider.Latency,
			"cost_per_1k": provider.CostPer1k, "models": parseModels(provider.Models),
			"api_key": maskKey(provider.APIKey), "endpoint": provider.Endpoint,
		},
		"routing": rules,
	})
}

func handleProofreadLogs(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	status := c.Query("status")
	query := db.Model(&ProofreadLog{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var logs []ProofreadLog
	query.Order("time desc").Limit(50).Find(&logs)
	ok(c, logs)
}

func handleProofreadStats(c *gin.Context) {
	var totalLogs, successLogs int64
	db.Model(&ProofreadLog{}).Count(&totalLogs)
	db.Model(&ProofreadLog{}).Where("status = ?", "success").Count(&successLogs)
	var avgLatency float64
	db.Model(&ProofreadLog{}).Select("COALESCE(AVG(latency), 0)").Scan(&avgLatency)
	successRate := 0.0
	if totalLogs > 0 {
		successRate = float64(successLogs) / float64(totalLogs) * 100
	}
	ok(c, gin.H{
		"today_calls": totalLogs, "avg_latency": int(avgLatency),
		"success_rate": successRate, "today_cost": float64(totalLogs) * 0.005 * 0.2,
	})
}
