package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func handleListServices(c *gin.Context) {
	var services []Service
	db.Find(&services)

	type ServiceOut struct {
		Service
		Dependencies []string `json:"dependencies"`
	}
	result := make([]ServiceOut, len(services))
	for i, s := range services {
		result[i] = ServiceOut{Service: s, Dependencies: parseCSV(s.Dependencies)}
	}
	ok(c, result)
}

func handleGetService(c *gin.Context) {
	id := c.Param("id")
	var svc Service
	if err := db.First(&svc, "id = ?", id).Error; err != nil {
		fail(c, http.StatusNotFound, "服务不存在")
		return
	}
	ok(c, gin.H{
		"id": svc.ID, "name": svc.Name, "type": svc.Type, "port": svc.Port,
		"host": svc.Host, "status": svc.Status, "version": svc.Version,
		"qps": svc.QPS, "p95": svc.P95, "error_rate": svc.ErrorRate,
		"instances": svc.Instances, "dependencies": parseCSV(svc.Dependencies),
		"consul_id": svc.ConsulID, "registered_at": svc.RegisteredAt,
		"source": svc.Source, "last_checked": svc.LastChecked,
	})
}

func handleServiceHealth(c *gin.Context) {
	id := c.Param("id")
	var svc Service
	if err := db.First(&svc, "id = ?", id).Error; err != nil {
		fail(c, http.StatusNotFound, "服务不存在")
		return
	}

	now := time.Now()
	tcpStatus, httpStatus, tcpLatency := checkServiceHealth(svc)

	checks := []gin.H{
		{"name": "TCP Port Check", "status": tcpStatus, "latency_ms": tcpLatency, "address": fmt.Sprintf("%s:%d", svc.Host, svc.Port), "last_check": now.Format(time.RFC3339)},
		{"name": "HTTP Health Endpoint", "status": httpStatus, "url": fmt.Sprintf("http://%s:%d/health", svc.Host, svc.Port), "last_check": now.Format(time.RFC3339)},
		{"name": "Service Status (DB)", "status": svc.Status, "last_check": svc.LastChecked.Format(time.RFC3339)},
	}

	// MySQL 专用协议检查
	if svc.ID == "mysql" {
		mysqlOK := "unreachable"
		if tcpStatus == "healthy" && checkMySQLProtocol(svc.Host, svc.Port) {
			mysqlOK = "healthy"
		}
		checks = append(checks, gin.H{"name": "MySQL Protocol Check", "status": mysqlOK, "address": fmt.Sprintf("%s:%d", svc.Host, svc.Port), "last_check": now.Format(time.RFC3339)})
	}

	overall := svc.Status
	if tcpStatus == "unreachable" {
		overall = "critical"
	} else if httpStatus == "unreachable" {
		overall = "warning"
	}
	ok(c, gin.H{"service_id": svc.ID, "overall": overall, "checks": checks})
}

func handleDiscoverServices(c *gin.Context) {
	allPorts := []int{}
	for p := range knownPorts {
		allPorts = append(allPorts, p)
	}
	var input struct {
		ExtraPorts []int `json:"extra_ports"`
	}
	c.ShouldBindJSON(&input)
	for _, p := range input.ExtraPorts {
		found := false
		for _, ex := range allPorts {
			if ex == p {
				found = true
				break
			}
		}
		if !found {
			allPorts = append(allPorts, p)
		}
	}

	results := scanPorts(allPorts)
	discovered := []Service{}
	for _, r := range results {
		if r["status"] != "healthy" {
			continue
		}
		port := r["port"].(int)
		var existing Service
		if err := db.First(&existing, "port = ? AND host = ?", port, "127.0.0.1").Error; err == nil {
			if existing.Status == "unreachable" || existing.Status == "critical" {
				existing.Status = "healthy"
				existing.LastChecked = time.Now()
				db.Save(&existing)
			}
			continue
		}
		svcID := strings.ToLower(strings.ReplaceAll(r["name"].(string), " ", "-"))
		var count int64
		db.Model(&Service{}).Where("id = ?", svcID).Count(&count)
		if count > 0 {
			svcID = fmt.Sprintf("%s-%d", svcID, port)
		}
		svc := Service{
			ID: svcID, Name: r["name"].(string), Type: r["type"].(string), Port: port,
			Host: "127.0.0.1", Status: "healthy", Version: r["version"].(string),
			Instances: 1, ConsulID: fmt.Sprintf("discovered-%s", svcID),
			RegisteredAt: time.Now(), Source: "discovered", LastChecked: time.Now(),
			DescSource: r["source"].(string), StartCmd: r["start_cmd"].(string),
		}
		db.Create(&svc)
		discovered = append(discovered, svc)
	}

	reachable := 0
	for _, r := range results {
		if r["status"] == "healthy" {
			reachable++
		}
	}
	ok(c, gin.H{"scan_results": results, "new_registered": discovered, "total_reachable": reachable})
}

func handleRegisterService(c *gin.Context) {
	var input struct {
		Name         string   `json:"name"`
		Type         string   `json:"type"`
		Port         int      `json:"port"`
		Host         string   `json:"host"`
		Version      string   `json:"version"`
		DescSource   string   `json:"desc_source"`
		StartCmd     string   `json:"start_cmd"`
		Dependencies []string `json:"dependencies"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Port == 0 {
		fail(c, http.StatusBadRequest, "参数错误：需要 name, port")
		return
	}
	if input.Host == "" {
		input.Host = "127.0.0.1"
	}
	if input.Type == "" {
		input.Type = "custom"
	}
	if input.Version == "" {
		input.Version = "unknown"
	}

	address := fmt.Sprintf("%s:%d", input.Host, input.Port)
	status := "unreachable"
	if conn, err := net.DialTimeout("tcp", address, 2*time.Second); err == nil {
		conn.Close()
		status = "healthy"
	}

	svcID := strings.ToLower(strings.ReplaceAll(input.Name, " ", "-"))
	if svcID == "" {
		svcID = fmt.Sprintf("svc-%d", input.Port)
	}
	var count int64
	db.Model(&Service{}).Where("id = ?", svcID).Count(&count)
	if count > 0 {
		svcID = fmt.Sprintf("%s-%d", svcID, time.Now().UnixMilli()%1000)
	}

	svc := Service{
		ID: svcID, Name: input.Name, Type: input.Type, Port: input.Port, Host: input.Host,
		Status: status, Version: input.Version, Instances: 1,
		Dependencies: strings.Join(input.Dependencies, ","),
		ConsulID: fmt.Sprintf("manual-%s", svcID), RegisteredAt: time.Now(),
		Source: "manual", LastChecked: time.Now(),
		DescSource: input.DescSource, StartCmd: input.StartCmd,
	}
	db.Create(&svc)
	ok(c, gin.H{"service": svc, "port_reachable": status == "healthy", "dependencies": input.Dependencies})
}

func handleUpdateService(c *gin.Context) {
	id := c.Param("id")
	var svc Service
	if err := db.First(&svc, "id = ?", id).Error; err != nil {
		fail(c, http.StatusNotFound, "服务不存在")
		return
	}
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if v, ok := input["name"].(string); ok {
		svc.Name = v
	}
	if v, ok := input["type"].(string); ok {
		svc.Type = v
	}
	if v, ok := input["port"].(float64); ok {
		svc.Port = int(v)
	}
	if v, ok := input["host"].(string); ok {
		svc.Host = v
	}
	if v, ok := input["version"].(string); ok {
		svc.Version = v
	}
	if v, ok := input["instances"].(float64); ok {
		svc.Instances = int(v)
	}
	if v, ok := input["desc_source"].(string); ok {
		svc.DescSource = v
	}
	if v, ok := input["start_cmd"].(string); ok {
		svc.StartCmd = v
	}
	if deps, ok := input["dependencies"].([]interface{}); ok {
		strs := make([]string, len(deps))
		for i, d := range deps {
			if s, ok2 := d.(string); ok2 {
				strs[i] = s
			}
		}
		svc.Dependencies = strings.Join(strs, ",")
	}
	db.Save(&svc)
	ok(c, svc)
}

func handleDeleteService(c *gin.Context) {
	id := c.Param("id")
	var svc Service
	if err := db.First(&svc, "id = ?", id).Error; err != nil {
		fail(c, http.StatusNotFound, "服务不存在")
		return
	}
	db.Delete(&svc)
	okMsg(c, fmt.Sprintf("服务 %s 已删除", svc.Name))
}

func handleRefreshHealth(c *gin.Context) {
	var services []Service
	db.Find(&services)
	updated := []gin.H{}
	for _, svc := range services {
		tcpStatus, _, latency := checkServiceHealth(svc)
		oldStatus := svc.Status
		if tcpStatus == "unreachable" {
			svc.Status = "critical"
		} else if svc.Status == "unreachable" || svc.Status == "critical" {
			svc.Status = "healthy"
		}
		svc.LastChecked = time.Now()
		svc.P95 = latency
		db.Save(&svc)
		updated = append(updated, gin.H{
			"id": svc.ID, "name": svc.Name, "port": svc.Port, "host": svc.Host,
			"old_status": oldStatus, "new_status": svc.Status, "latency": latency,
		})
		if rdb != nil {
			ctx, cancel := contextWithTimeout(2 * time.Second)
			rdb.HSet(ctx, "health:services", svc.ID, svc.Status)
			cancel()
		}
	}
	ok(c, gin.H{"checked": len(services), "updates": updated})
}

func handleServiceEvents(c *gin.Context) {
	events := []gin.H{
		{"time": time.Now().Add(-5 * time.Minute).Format(time.RFC3339), "service": "ai-service", "action": "health_change", "detail": "状态从 healthy 变为 warning，P95延迟 320ms"},
		{"time": time.Now().Add(-30 * time.Minute).Format(time.RFC3339), "service": "user-service", "action": "register", "detail": "新实例 user-service-3 注册"},
		{"time": time.Now().Add(-1 * time.Hour).Format(time.RFC3339), "service": "order-service", "action": "register", "detail": "实例 order-service-2 重启后重新注册"},
		{"time": time.Now().Add(-2 * time.Hour).Format(time.RFC3339), "service": "nats", "action": "health_change", "detail": "状态从 healthy 变为 warning"},
		{"time": time.Now().Add(-3 * time.Hour).Format(time.RFC3339), "service": "ai-service", "action": "register", "detail": "版本更新 v0.7.2 -> v0.8.0"},
		{"time": time.Now().Add(-5 * time.Hour).Format(time.RFC3339), "service": "translate-y", "action": "register", "detail": "新供应商 translate-y 注册，状态 testing"},
		{"time": time.Now().Add(-8 * time.Hour).Format(time.RFC3339), "service": "test-org", "action": "deregister", "detail": "租户 test-org 被冻结，相关服务实例注销"},
		{"time": time.Now().Add(-12 * time.Hour).Format(time.RFC3339), "service": "gateway", "action": "register", "detail": "Gateway 实例 gateway-2 上线，当前实例数 2"},
		{"time": time.Now().Add(-24 * time.Hour).Format(time.RFC3339), "service": "user-service", "action": "deregister", "detail": "旧实例 user-service-1 优雅下线"},
		{"time": time.Now().Add(-48 * time.Hour).Format(time.RFC3339), "service": "order-service", "action": "register", "detail": "Order Service 首次注册"},
	}
	ok(c, events)
}
