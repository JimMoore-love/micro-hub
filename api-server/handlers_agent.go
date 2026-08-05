package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Agent 上报数据结构
type AgentReport struct {
	NodeName   string         `json:"node_name"`
	Hostname   string         `json:"hostname"`
	IP         string         `json:"ip"`
	OS         string         `json:"os"`
	Arch       string         `json:"arch"`
	AgentVer   string         `json:"agent_version"`
	Timestamp  time.Time      `json:"timestamp"`
	Services   []AgentService `json:"services"`
	SubnetScan []SubnetResult `json:"subnet_scan,omitempty"`
	CPUUsage   float64        `json:"cpu_usage"`
	MemUsage   float64        `json:"mem_usage"`
}

type AgentService struct {
	Port        int    `json:"port"`
	Name        string `json:"name"`
	ProcessName string `json:"process_name"`
	PID         int    `json:"pid"`
	Status      string `json:"status"`
	Latency     int    `json:"latency_ms"`
}

type SubnetResult struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Reachable bool  `json:"reachable"`
	Latency  int    `json:"latency_ms"`
}

// POST /api/v1/agents/report — 接收 agent 上报
func handleAgentReport(c *gin.Context) {
	var report AgentReport
	if err := c.ShouldBindJSON(&report); err != nil {
		fail(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 注册或更新节点
	nodeID := report.NodeName
	if nodeID == "" {
		nodeID = report.Hostname
	}

	var node Node
	if err := db.First(&node, "id = ?", nodeID).Error; err != nil {
		// 新节点
		node = Node{
			ID: nodeID, Name: report.NodeName, Hostname: report.Hostname,
			IP: report.IP, OS: report.OS, Arch: report.Arch,
			AgentVersion: report.AgentVer, Status: "online",
			CPUUsage: report.CPUUsage, MemUsage: report.MemUsage,
			LastSeen: time.Now(), CreatedAt: time.Now(),
		}
		db.Create(&node)
	} else {
		// 更新已有节点
		node.IP = report.IP
		node.OS = report.OS
		node.Arch = report.Arch
		node.AgentVersion = report.AgentVer
		node.Status = "online"
		node.CPUUsage = report.CPUUsage
		node.MemUsage = report.MemUsage
		node.LastSeen = time.Now()
		db.Save(&node)
	}

	// 更新该节点的服务列表
	// 先删除该节点的旧服务
	db.Where("node = ? AND source = ?", nodeID, "agent").Delete(&Service{})

	// 写入新服务
	newServices := []Service{}
	for _, svc := range report.Services {
		if svc.Status != "healthy" {
			continue
		}
		svcID := fmt.Sprintf("%s-%d", nodeID, svc.Port)
		// 检查是否已存在（非 agent 来源的）
		var existing Service
		if err := db.First(&existing, "id = ?", svcID).Error; err == nil {
			// 已存在，只更新状态
			existing.Status = svc.Status
			existing.LastChecked = time.Now()
			existing.Node = nodeID
			existing.DescSource = fmt.Sprintf("进程: %s (PID %d)", svc.ProcessName, svc.PID)
			if svc.Name != "" {
				existing.Name = svc.Name
			}
			db.Save(&existing)
			continue
		}

		name := svc.Name
		if name == "" {
			name = fmt.Sprintf("未知服务(%d)", svc.Port)
		}
		svcType := inferTypeFromName(name)
		newServices = append(newServices, Service{
			ID: svcID, Name: name, Type: svcType, Port: svc.Port,
			Host: report.IP, Node: nodeID, Status: svc.Status,
			Instances: 1, ConsulID: fmt.Sprintf("agent-%s", svcID),
			RegisteredAt: time.Now(), Source: "agent",
			LastChecked: time.Now(),
			DescSource: fmt.Sprintf("进程: %s (PID %d)", svc.ProcessName, svc.PID),
		})
	}
	if len(newServices) > 0 {
		db.Create(&newServices)
	}

	// 处理网段扫描结果
	subnetDiscovered := []Service{}
	for _, sr := range report.SubnetScan {
		if !sr.Reachable {
			continue
		}
		svcID := fmt.Sprintf("%s-%d", sr.IP, sr.Port)
		var existing Service
		if err := db.First(&existing, "id = ?", svcID).Error; err == nil {
			existing.Status = "healthy"
			existing.LastChecked = time.Now()
			db.Save(&existing)
			continue
		}
		name := inferServiceFromPort(sr.Port)
		subnetDiscovered = append(subnetDiscovered, Service{
			ID: svcID, Name: name, Type: inferTypeFromName(name), Port: sr.Port,
			Host: sr.IP, Node: report.IP, Status: "healthy",
			Instances: 1, ConsulID: fmt.Sprintf("subnet-%s", svcID),
			RegisteredAt: time.Now(), Source: "subnet_scan",
			LastChecked: time.Now(),
			DescSource: fmt.Sprintf("网段发现: %s:%d", sr.IP, sr.Port),
		})
	}
	if len(subnetDiscovered) > 0 {
		db.Create(&subnetDiscovered)
	}

	ok(c, gin.H{
		"node":              node.Name,
		"services_reported": len(report.Services),
		"subnet_discovered": len(subnetDiscovered),
		"new_registered":    len(newServices),
	})
}

// GET /api/v1/nodes — 获取所有节点
func handleListNodes(c *gin.Context) {
	var nodes []Node
	db.Order("last_seen desc").Find(&nodes)

	// 为每个节点附加服务计数
	result := []gin.H{}
	for _, n := range nodes {
		var svcCount int64
		db.Model(&Service{}).Where("node = ?", n.ID).Count(&svcCount)

		// 判断是否在线（30 秒内有心跳）
		online := time.Since(n.LastSeen) < 60*time.Second
		status := n.Status
		if !online {
			status = "offline"
		}

		result = append(result, gin.H{
			"id":            n.ID,
			"name":          n.Name,
			"hostname":      n.Hostname,
			"ip":            n.IP,
			"os":            n.OS,
			"arch":          n.Arch,
			"agent_version": n.AgentVersion,
			"status":        status,
			"cpu_usage":     n.CPUUsage,
			"mem_usage":     n.MemUsage,
			"last_seen":     n.LastSeen,
			"service_count": svcCount,
		})
	}
	ok(c, result)
}

// POST /api/v1/nodes/scan-subnet — 主动触发网段扫描（从 API Server 端发起）
func handleScanSubnet(c *gin.Context) {
	var input struct {
		Subnet string `json:"subnet"` // 如 192.168.1.0/24
		Ports  []int  `json:"ports"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Subnet == "" {
		fail(c, http.StatusBadRequest, "参数错误：需要 subnet")
		return
	}

	if len(input.Ports) == 0 {
		input.Ports = []int{80, 443, 3306, 6379, 8080, 8081, 22, 8500, 9090, 16686}
	}

	results := []gin.H{}
	ips, err := parseCIDR(input.Subnet)
	if err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("网段解析失败: %v", err))
		return
	}

	// 限制扫描范围（避免扫描太多 IP）
	if len(ips) > 256 {
		ips = ips[:256]
	}

	for _, ip := range ips {
		for _, port := range input.Ports {
			address := fmt.Sprintf("%s:%d", ip, port)
			start := time.Now()
			conn, err := net.DialTimeout("tcp", address, 800*time.Millisecond)
			latency := int(time.Since(start).Milliseconds())
			reachable := err == nil
			if conn != nil {
				conn.Close()
			}
			if reachable {
				// 自动注册发现的服务
				svcID := fmt.Sprintf("%s-%d", ip, port)
				var existing Service
				if err := db.First(&existing, "id = ?", svcID).Error; err != nil {
					name := inferServiceFromPort(port)
					svc := Service{
						ID: svcID, Name: name, Type: inferTypeFromName(name), Port: port,
						Host: ip, Node: ip, Status: "healthy",
						Instances: 1, ConsulID: fmt.Sprintf("subnet-%s", svcID),
						RegisteredAt: time.Now(), Source: "subnet_scan",
						LastChecked: time.Now(),
						DescSource: fmt.Sprintf("网段发现: %s:%d", ip, port),
					}
					db.Create(&svc)
				} else {
					existing.Status = "healthy"
					existing.LastChecked = time.Now()
					db.Save(&existing)
				}

				results = append(results, gin.H{
					"ip": ip, "port": port, "reachable": true,
					"latency": latency, "service": inferServiceFromPort(port),
				})
			}
		}
	}

	ok(c, gin.H{
		"subnet":       input.Subnet,
		"scanned_ips":  len(ips),
		"scanned_ports": len(input.Ports),
		"discovered":   results,
		"total":        len(results),
	})
}

// 辅助函数
func inferServiceFromPort(port int) string {
	known := map[int]string{
		80: "Nginx", 443: "HTTPS", 3306: "MySQL", 5432: "PostgreSQL",
		6379: "Redis", 8080: "API Gateway", 8500: "Consul", 4222: "NATS",
		9000: "MinIO", 9090: "Prometheus", 16686: "Jaeger",
		8081: "MicroHub API", 22: "SSH", 11434: "Ollama",
	}
	if name, ok := known[port]; ok {
		return name
	}
	return fmt.Sprintf("未知服务(%d)", port)
}

func inferTypeFromName(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "gateway") {
		return "gateway"
	}
	if strings.Contains(lower, "jaeger") || strings.Contains(lower, "prometheus") || strings.Contains(lower, "grafana") {
		return "observability"
	}
	if strings.Contains(lower, "mysql") || strings.Contains(lower, "redis") || strings.Contains(lower, "nginx") ||
		strings.Contains(lower, "consul") || strings.Contains(lower, "nats") || strings.Contains(lower, "minio") ||
		strings.Contains(lower, "ollama") || strings.Contains(lower, "ssh") {
		return "infra"
	}
	return "service"
}

func parseCIDR(cidr string) ([]string, error) {
	if !strings.Contains(cidr, "/") {
		return []string{cidr}, nil
	}
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips, nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
