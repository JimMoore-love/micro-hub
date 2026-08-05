package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 已知端口映射表 — 仅作为"预期服务名"参考，实际以进程查询为准
var knownPorts = map[int]struct {
	Name     string
	Type     string
	Version  string
	Source   string
	StartCmd string
}{
	80:    {"Nginx", "infra", "1.x", "ServBay", "D:\\ServBay\\bin\\nginx-server.cmd"},
	443:   {"HTTPS", "infra", "1.x", "ServBay", "D:\\ServBay\\bin\\nginx-server.cmd"},
	3306:  {"MySQL", "infra", "8.x", "ServBay", "D:\\ServBay\\bin\\mysql-server.cmd"},
	5432:  {"PostgreSQL", "infra", "16.x", "ServBay/Podman", "D:\\ServBay\\bin\\pg-server.cmd"},
	6379:  {"Redis", "infra", "7.x", "ServBay", "D:\\ServBay\\bin\\redis-server.cmd"},
	8080:  {"API Gateway", "gateway", "v1.x", "手动启动", "cd api-server && ./microhub-api.exe"},
	8500:  {"Consul", "infra", "1.x", "Podman", "podman run consul:1.19"},
	4222:  {"NATS", "infra", "2.x", "Podman", "podman run nats:2.10"},
	9000:  {"MinIO", "infra", "RELEASE.2024", "Podman", "podman run minio/minio"},
	9001:  {"MinIO Console", "infra", "RELEASE.2024", "Podman", "minio/minio (console)"},
	16686: {"Jaeger", "observability", "1.x", "Podman", "podman run jaegertracing/all-in-one"},
	9090:  {"Prometheus", "observability", "2.x", "otel-lgtm/Podman", "podman run grafana/otel-lgtm"},
	9100:  {"Node Exporter", "observability", "1.x", "手动安装", "node_exporter"},
	3000:  {"Grafana", "observability", "1.x", "otel-lgtm/ServBay", "podman run grafana/otel-lgtm"},
	3200:  {"Frontend Dev", "service", "v1.x", "Vite", "node node_modules/vite/bin/vite.js"},
	8081:  {"MicroHub API", "service", "v1.x", "手动启动", "cd api-server && ./microhub-api.exe"},
	8082:  {"Order Service", "service", "v0.x", "未部署", ""},
	8083:  {"AI Service", "service", "v0.x", "未部署", ""},
	27017: {"MongoDB", "infra", "7.x", "手动安装/Podman", "mongod"},
	2379:  {"etcd", "infra", "3.x", "手动安装/Podman", "etcd"},
	5601:  {"Kibana", "observability", "7.x", "手动安装/Podman", "kibana"},
	9200:  {"Elasticsearch", "infra", "8.x", "手动安装/Podman", "elasticsearch"},
	15672: {"RabbitMQ Management", "infra", "3.x", "手动安装/Podman", "rabbitmq-server"},
	5672:  {"RabbitMQ", "infra", "3.x", "手动安装/Podman", "rabbitmq-server"},
	7000:  {"FRP Server", "infra", "v0.x", "阿里云服务器", "frps (远程)"},
	7001:  {"FRP Dashboard", "infra", "v0.x", "阿里云服务器", "frps dashboard (远程)"},
	7400:  {"FRP Client Admin", "infra", "v0.x", "本地 frpc", "F:\\FrpSpace\\run_frpc.bat"},
	7500:  {"FRP Client Port", "infra", "v0.x", "本地 frpc", "F:\\FrpSpace\\run_frpc.bat"},
	8890:  {"FRP Nginx Tunnel", "infra", "v0.x", "本地 nginx + frpc", "D:\\nginx-1.30.2 + frpc"},
	11434: {"Ollama", "infra", "v0.x", "本地安装", "ollama serve"},
	9073:  {"PHP-FPM", "infra", "8.x", "ServBay", "D:\\ServBay\\bin\\php-fpm.cmd"},
	9054:  {"PHP-FPM Alt", "infra", "8.x", "ServBay", "D:\\ServBay\\bin\\php-fpm.cmd"},
	11211: {"Memcached", "infra", "1.x", "ServBay", "D:\\ServBay\\bin\\memcached.cmd"},
}

// PortProcess 端口占用的真实进程信息
type PortProcess struct {
	PID          int    `json:"pid"`
	ProcessName  string `json:"process_name"`
	ExecPath     string `json:"exec_path"`
	IsSystemProc bool   `json:"is_system_proc"`
}

// 知名 Windows 系统服务端口映射
var windowsServiceHints = map[int]string{
	8080: "IP Helper (iphlpsvc) - WinHTTP 代理/WPAD",
	135:  "RPC (RpcSs) - 远程过程调用",
	139:  "NetBIOS (LanmanServer) - SMB over NetBIOS",
	445:  "SMB (LanmanServer) - Windows 文件共享",
	5040: "Windows Subsystem for Linux",
	7890: "Clash/代理软件常用端口",
}

// getPortProcess 查询占用指定端口的真实进程信息
func getPortProcess(port int) *PortProcess {
	if runtime.GOOS != "windows" {
		return getPortProcessLinux(port)
	}
	return getPortProcessWindows(port)
}

// Windows: netstat -ano + wmic
func getPortProcessWindows(port int) *PortProcess {
	// 1. netstat -ano 找到 PID
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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
		pid, err := strconv.Atoi(fields[4])
		if err != nil {
			continue
		}

		// 2. tasklist 获取进程名（直接执行，不通过 cmd /c）
		processName := "unknown"
		taskOut, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
		if err == nil {
			// 输出格式: "svchost.exe","5648","Services","0","7,068 K"
			taskLine := strings.TrimSpace(string(taskOut))
			if strings.HasPrefix(taskLine, "\"") {
				endIdx := strings.Index(taskLine[1:], "\"")
				if endIdx > 0 {
					processName = taskLine[1 : 1+endIdx]
				}
			}
		}

		// 进程路径暂时留空（wmic 被安全策略禁用，如需路径可用 PowerShell Get-Process）
		execPath := ""

		return &PortProcess{
			PID: pid, ProcessName: processName, ExecPath: execPath,
			IsSystemProc: isWindowsSystemProcess(processName, execPath),
		}
	}
	return nil
}

// Linux: lsof -i :PORT
func getPortProcessLinux(port int) *PortProcess {
	out, err := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-F", "pc").Output()
	if err != nil {
		return nil
	}
	var pid int
	var processName string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "p") {
			pid, _ = strconv.Atoi(line[1:])
		} else if strings.HasPrefix(line, "c") {
			processName = line[1:]
		}
	}
	if pid == 0 {
		return nil
	}
	execPath, _ := readSymlink(fmt.Sprintf("/proc/%d/exe", pid))
	return &PortProcess{
		PID: pid, ProcessName: processName, ExecPath: execPath,
		IsSystemProc: processName == "systemd",
	}
}

func isWindowsSystemProcess(name, path string) bool {
	systemProcs := map[string]bool{
		"svchost.exe": true, "system": true, "smss.exe": true,
		"csrss.exe": true, "wininit.exe": true, "services.exe": true,
		"lsass.exe": true, "spoolsv.exe": true, "winlogon.exe": true,
	}
	if systemProcs[strings.ToLower(name)] {
		return true
	}
	lowerPath := strings.ToLower(path)
	return strings.Contains(lowerPath, "c:\\windows\\system32") || strings.Contains(lowerPath, "c:\\windows\\syswow64")
}

func scanPorts(ports []int) []gin.H {
	results := []gin.H{}
	for _, port := range ports {
		address := fmt.Sprintf("127.0.0.1:%d", port)
		start := time.Now()
		conn, err := net.DialTimeout("tcp", address, 1500*time.Millisecond)
		latency := int(time.Since(start).Milliseconds())
		reachable := err == nil
		if conn != nil {
			conn.Close()
		}

		info, known := knownPorts[port]
		expectedName := fmt.Sprintf("未知服务(%d)", port)
		svcType := "custom"
		version := ""
		expectedSource := ""
		expectedStartCmd := ""
		if known {
			expectedName = info.Name
			svcType = info.Type
			version = info.Version
			expectedSource = info.Source
			expectedStartCmd = info.StartCmd
		}

		status := "unreachable"
		if reachable {
			status = "healthy"
		}

		var procInfo *PortProcess
		actualName := expectedName
		actualSource := expectedSource
		actualStartCmd := expectedStartCmd
		matchType := "unknown"

		if reachable {
			procInfo = getPortProcess(port)
			if procInfo != nil {
				inferred := inferServiceFromProcess(procInfo.ProcessName, procInfo.ExecPath)
				if inferred != "" {
					actualName = inferred
				}
				if procInfo.IsSystemProc {
					actualSource = fmt.Sprintf("Windows 系统服务 (%s)", procInfo.ProcessName)
					if hint, ok := windowsServiceHints[port]; ok {
						actualStartCmd = fmt.Sprintf("系统服务: %s (PID %d)", hint, procInfo.PID)
					} else {
						actualStartCmd = fmt.Sprintf("系统进程: %s (PID %d, %s)", procInfo.ProcessName, procInfo.PID, procInfo.ExecPath)
					}
					if known {
						matchType = "conflict"
					}
				} else {
					// 非系统进程，用进程信息填充来源
					actualSource = fmt.Sprintf("进程: %s (PID %d)", procInfo.ProcessName, procInfo.PID)
					if procInfo.ExecPath != "" {
						actualStartCmd = procInfo.ExecPath
					} else {
						actualStartCmd = fmt.Sprintf("进程: %s (PID %d)", procInfo.ProcessName, procInfo.PID)
					}
					if known {
						if strings.Contains(strings.ToLower(procInfo.ProcessName), strings.ToLower(expectedName)) {
							matchType = "matched"
						} else {
							matchType = "conflict"
						}
					}
				}
			}
		}

		result := gin.H{
			"port": port, "name": actualName, "expected": expectedName,
			"type": svcType, "version": version, "status": status,
			"latency": latency, "address": address, "known": known,
			"source": actualSource, "start_cmd": actualStartCmd, "match_type": matchType,
		}
		if procInfo != nil {
			result["process"] = procInfo
		}
		results = append(results, result)
	}
	return results
}

// inferServiceFromProcess 根据进程名和路径推断服务名
func inferServiceFromProcess(procName, execPath string) string {
	name := strings.ToLower(procName)
	path := strings.ToLower(execPath)

	if strings.Contains(name, "nginx") {
		return "Nginx"
	}
	if strings.Contains(name, "mysql") {
		return "MySQL"
	}
	if strings.Contains(name, "redis") {
		return "Redis"
	}
	if strings.Contains(name, "node") || strings.Contains(name, "node.exe") {
		if strings.Contains(path, "vite") {
			return "Frontend Dev (Vite)"
		}
		return "Node.js 服务"
	}
	if strings.Contains(name, "microhub") {
		return "MicroHub API"
	}
	if strings.Contains(name, "ollama") {
		return "Ollama"
	}
	if strings.Contains(name, "php") {
		return "PHP-FPM"
	}
	if strings.Contains(name, "memcached") {
		return "Memcached"
	}
	if strings.Contains(name, "frpc") {
		return "FRP Client"
	}
	if strings.Contains(name, "frps") {
		return "FRP Server"
	}
	if strings.Contains(name, "consul") {
		return "Consul"
	}
	if strings.Contains(name, "nats") {
		return "NATS"
	}
	if strings.Contains(name, "minio") {
		return "MinIO"
	}
	if strings.Contains(name, "jaeger") {
		return "Jaeger"
	}
	if strings.Contains(name, "prometheus") {
		return "Prometheus"
	}
	if strings.Contains(name, "grafana") || strings.Contains(path, "lgtm") {
		return "Grafana"
	}
	if strings.Contains(name, "mongo") {
		return "MongoDB"
	}
	if strings.Contains(name, "postgres") {
		return "PostgreSQL"
	}
	if strings.Contains(name, "rabbitmq") || strings.Contains(name, "beam") {
		return "RabbitMQ"
	}
	if strings.Contains(name, "etcd") {
		return "etcd"
	}
	if strings.Contains(name, "kibana") {
		return "Kibana"
	}
	if strings.Contains(name, "svchost") {
		return "Windows 系统服务"
	}
	return ""
}

// MySQL 协议握手检查
func checkMySQLProtocol(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	buf := make([]byte, 128)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buf)
	if err != nil || n < 5 {
		return false
	}
	return buf[4] == 10 || buf[4] == 9
}

// TCP + HTTP 健康检查
func checkServiceHealth(svc Service) (tcpStatus, httpStatus string, tcpLatency int) {
	address := fmt.Sprintf("%s:%d", svc.Host, svc.Port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	tcpLatency = int(time.Since(start).Milliseconds())
	if err != nil {
		return "unreachable", "unreachable", -1
	}
	conn.Close()
	tcpStatus = "healthy"

	httpStatus = "unreachable"
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/health", address))
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			httpStatus = "healthy"
		} else {
			httpStatus = "warning"
		}
	}
	return
}
