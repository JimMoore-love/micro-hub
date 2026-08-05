package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ==================== 配置 ====================

type Config struct {
	ServerURL    string        // API Server 地址
	NodeName     string        // 节点名称
	ScanInterval time.Duration // 上报间隔
	ScanSubnet   string        // 要扫描的网段（如 192.168.1.0/24），为空则不扫描
	ScanPorts    []int         // 要扫描的端口列表
	Token        string        // JWT token
}

// ==================== 上报数据结构 ====================

type Report struct {
	NodeName    string         `json:"node_name"`
	Hostname    string         `json:"hostname"`
	IP          string         `json:"ip"`
	OS          string         `json:"os"`
	Arch        string         `json:"arch"`
	AgentVer    string         `json:"agent_version"`
	Timestamp   time.Time      `json:"timestamp"`
	Services    []ServiceInfo  `json:"services"`
	SubnetScan  []SubnetResult `json:"subnet_scan,omitempty"`
	CPUUsage    float64        `json:"cpu_usage"`
	MemUsage    float64        `json:"mem_usage"`
}

type ServiceInfo struct {
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

// ==================== 主函数 ====================

func main() {
	cfg := parseFlags()
	log.Printf("[Agent] 启动: node=%s server=%s interval=%v subnet=%s", cfg.NodeName, cfg.ServerURL, cfg.ScanInterval, cfg.ScanSubnet)

	for {
		report := collectReport(cfg)
		sendReport(cfg, report)
		time.Sleep(cfg.ScanInterval)
	}
}

func parseFlags() *Config {
	serverURL := flag.String("server", "http://localhost:8081", "API Server 地址")
	nodeName := flag.String("name", "", "节点名称（默认用 hostname）")
	interval := flag.Int("interval", 30, "上报间隔（秒）")
	subnet := flag.String("subnet", "", "扫描网段（如 192.168.1.0/24），为空不扫描")
	token := flag.String("token", "", "JWT 认证 token")
	ports := flag.String("ports", "80,443,3306,5432,6379,8080,8081,8082,8083,8500,4222,9000,9090,16686,11434,3200,8890", "扫描端口列表（逗号分隔）")
	flag.Parse()

	if *nodeName == "" {
		h, _ := os.Hostname()
		*nodeName = h
	}

	var portList []int
	for _, p := range strings.Split(*ports, ",") {
		p = strings.TrimSpace(p)
		if n, err := strconv.Atoi(p); err == nil {
			portList = append(portList, n)
		}
	}

	return &Config{
		ServerURL:    strings.TrimRight(*serverURL, "/"),
		NodeName:     *nodeName,
		ScanInterval: time.Duration(*interval) * time.Second,
		ScanSubnet:   *subnet,
		ScanPorts:    portList,
		Token:        *token,
	}
}

// ==================== 采集 ====================

func collectReport(cfg *Config) *Report {
	hostname, _ := os.Hostname()
	ip := getLocalIP()

	report := &Report{
		NodeName:  cfg.NodeName,
		Hostname:  hostname,
		IP:        ip,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		AgentVer:  "v1.0.0",
		Timestamp: time.Now(),
	}

	// 扫描本机端口
	report.Services = scanLocalPorts(cfg.ScanPorts)

	// 网段扫描
	if cfg.ScanSubnet != "" {
		report.SubnetScan = scanSubnet(cfg.ScanSubnet, cfg.ScanPorts)
	}

	// 系统指标（简化版）
	report.CPUUsage = getCPUUsage()
	report.MemUsage = getMemUsage()

	return report
}

// 扫描本机端口 + 进程信息
func scanLocalPorts(ports []int) []ServiceInfo {
	results := []ServiceInfo{}
	for _, port := range ports {
		address := fmt.Sprintf("127.0.0.1:%d", port)
		start := time.Now()
		conn, err := net.DialTimeout("tcp", address, 1500*time.Millisecond)
		latency := int(time.Since(start).Milliseconds())

		svc := ServiceInfo{Port: port, Latency: latency}
		if err != nil {
			svc.Status = "unreachable"
		} else {
			conn.Close()
			svc.Status = "healthy"
			svc.Name, svc.PID, svc.ProcessName = getPortProcessInfo(port)
		}
		results = append(results, svc)
	}
	return results
}

// 网段扫描：扫描 CIDR 网段内所有 IP 的指定端口
func scanSubnet(cidr string, ports []int) []SubnetResult {
	results := []SubnetResult{}
	ips, err := parseCIDR(cidr)
	if err != nil {
		log.Printf("[Agent] 网段解析失败 %s: %v", cidr, err)
		return results
	}

	log.Printf("[Agent] 扫描网段 %s (%d 个IP, %d 个端口)", cidr, len(ips), len(ports))
	for _, ip := range ips {
		for _, port := range ports {
			address := fmt.Sprintf("%s:%d", ip, port)
			start := time.Now()
			conn, err := net.DialTimeout("tcp", address, 800*time.Millisecond)
			latency := int(time.Since(start).Milliseconds())
			reachable := err == nil
			if conn != nil {
				conn.Close()
			}
			if reachable {
				results = append(results, SubnetResult{
					IP: ip, Port: port, Reachable: true, Latency: latency,
				})
				log.Printf("[Agent] 发现 %s:%d 可达 (%dms)", ip, port, latency)
			}
		}
	}
	log.Printf("[Agent] 网段扫描完成: %d 个可达端口", len(results))
	return results
}

// 解析 CIDR 获取 IP 列表
func parseCIDR(cidr string) ([]string, error) {
	// 支持 192.168.1.0/24 格式
	if !strings.Contains(cidr, "/") {
		// 单个 IP
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
	// 去掉网络地址和广播地址（/24 以下）
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

// 获取本机 IP
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// 获取占用端口的进程信息
func getPortProcessInfo(port int) (name string, pid int, processName string) {
	if runtime.GOOS == "windows" {
		return getPortProcessWindows(port)
	}
	return getPortProcessLinux(port)
}

func getPortProcessWindows(port int) (string, int, string) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return "", 0, ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.Contains(fields[3], "LISTENING") {
			continue
		}
		addrParts := strings.Split(fields[1], ":")
		if len(addrParts) < 2 {
			continue
		}
		listenPort, err := strconv.Atoi(addrParts[len(addrParts)-1])
		if err != nil || listenPort != port {
			continue
		}
		pid, _ := strconv.Atoi(fields[4])

		procName := "unknown"
		taskOut, _ := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
		taskLine := strings.TrimSpace(string(taskOut))
		if strings.HasPrefix(taskLine, "\"") {
			endIdx := strings.Index(taskLine[1:], "\"")
			if endIdx > 0 {
				procName = taskLine[1 : 1+endIdx]
			}
		}
		return inferServiceName(procName), pid, procName
	}
	return "", 0, ""
}

func getPortProcessLinux(port int) (string, int, string) {
	out, err := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-F", "pc").Output()
	if err != nil {
		return "", 0, ""
	}
	var pid int
	var procName string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "p") {
			pid, _ = strconv.Atoi(line[1:])
		} else if strings.HasPrefix(line, "c") {
			procName = line[1:]
		}
	}
	if pid == 0 {
		return "", 0, ""
	}
	return inferServiceName(procName), pid, procName
}

func inferServiceName(procName string) string {
	name := strings.ToLower(procName)
	known := map[string]string{
		"nginx": "Nginx", "mysql": "MySQL", "mysqld": "MySQL",
		"redis": "Redis", "redis-server": "Redis",
		"node": "Node.js", "node.exe": "Node.js",
		"microhub": "MicroHub API", "ollama": "Ollama",
		"php": "PHP-FPM", "php-cgi": "PHP-FPM",
		"memcached": "Memcached", "frpc": "FRP Client",
		"frps": "FRP Server", "consul": "Consul", "nats": "NATS",
		"minio": "MinIO", "jaeger": "Jaeger", "prometheus": "Prometheus",
		"grafana": "Grafana", "mongo": "MongoDB", "mongod": "MongoDB",
		"postgres": "PostgreSQL", "etcd": "etcd", "svchost": "Windows 系统服务",
	}
	for key, val := range known {
		if strings.Contains(name, key) {
			return val
		}
	}
	return ""
}

// 简化版 CPU/内存使用率
func getCPUUsage() float64 {
	// 生产环境可用 gopsutil 获取精确值
	return 0
}

func getMemUsage() float64 {
	return 0
}

// ==================== 上报 ====================

func sendReport(cfg *Config, report *Report) {
	data, _ := json.Marshal(report)
	url := cfg.ServerURL + "/api/v1/agents/report"

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Agent] 上报失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Printf("[Agent] 上报成功: %d 个服务, %d 个网段结果", len(report.Services), len(report.SubnetScan))
	} else {
		log.Printf("[Agent] 上报失败: HTTP %d", resp.StatusCode)
	}
}
