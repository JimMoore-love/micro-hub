package main

import "time"

// ==================== GORM 模型定义 ====================

type Service struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Port         int       `json:"port"`
	Host         string    `json:"host"`
	Node         string    `gorm:"column:node;type:varchar(100)" json:"node"`
	Status       string    `json:"status"`
	Version      string    `json:"version"`
	QPS          int       `json:"qps"`
	P95          int       `json:"p95"`
	ErrorRate    float64   `gorm:"column:error_rate" json:"error_rate"`
	Instances    int       `json:"instances"`
	Dependencies string    `json:"-"`
	ConsulID     string    `gorm:"column:consul_id" json:"consul_id"`
	RegisteredAt time.Time `gorm:"column:registered_at" json:"registered_at"`
	Source       string    `json:"source"`
	LastChecked  time.Time `gorm:"column:last_checked" json:"last_checked"`
	DescSource   string    `gorm:"column:desc_source;type:varchar(255)" json:"desc_source"`
	StartCmd     string    `gorm:"column:start_cmd;type:varchar(500)" json:"start_cmd"`
}

// Node 节点信息
type Node struct {
	ID           string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name         string    `gorm:"type:varchar(100)" json:"name"`
	Hostname     string    `gorm:"type:varchar(100)" json:"hostname"`
	IP           string    `gorm:"type:varchar(50)" json:"ip"`
	OS           string    `gorm:"type:varchar(50)" json:"os"`
	Arch         string    `gorm:"type:varchar(50)" json:"arch"`
	AgentVersion string    `gorm:"column:agent_version;type:varchar(50)" json:"agent_version"`
	Status       string    `gorm:"type:varchar(50)" json:"status"`
	CPUUsage     float64   `gorm:"column:cpu_usage" json:"cpu_usage"`
	MemUsage     float64   `gorm:"column:mem_usage" json:"mem_usage"`
	LastSeen     time.Time `gorm:"column:last_seen" json:"last_seen"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

type Tenant struct {
	ID          string    `gorm:"primaryKey;type:varchar(191)" json:"id"`
	Name        string    `gorm:"type:varchar(255)" json:"name"`
	Schema      string    `gorm:"type:varchar(255);column:schema" json:"schema"`
	Users       int       `json:"users"`
	Quota       int       `json:"quota"`
	Used        int       `json:"used"`
	Status      string    `gorm:"type:varchar(50)" json:"status"`
	Plan        string    `gorm:"type:varchar(50)" json:"plan"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	RedisPrefix string    `gorm:"type:varchar(255);column:redis_prefix" json:"redis_prefix"`
	DBTables    int       `gorm:"column:db_tables" json:"db_tables"`
	DBRecords   int       `gorm:"column:db_records" json:"db_records"`
	APIKeys     []APIKey  `gorm:"-" json:"api_keys"`
}

type APIKey struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	Key       string    `gorm:"type:varchar(255);uniqueIndex" json:"key"`
	TenantID  string    `gorm:"type:varchar(191);index" json:"-"`
	Status    string    `gorm:"type:varchar(50)" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type AIProvider struct {
	ID        string  `gorm:"primaryKey" json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Icon      string  `json:"icon"`
	Status    string  `json:"status"`
	Requests  int     `json:"requests"`
	Latency   int     `json:"latency"`
	CostPer1k float64 `gorm:"column:cost_per_1k" json:"cost_per_1k"`
	Models    string  `json:"-"`
	APIKey    string  `gorm:"column:api_key" json:"-"`
	Endpoint  string  `json:"endpoint"`
}

func (AIProvider) TableName() string { return "ai_providers" }

type RoutingRule struct {
	ID         string `gorm:"primaryKey" json:"id"`
	TenantID   string `gorm:"column:tenant_id" json:"tenant_id"`
	ProviderID string `gorm:"column:provider_id" json:"provider_id"`
	Priority   int    `json:"priority"`
	Condition  string `json:"condition"`
	Enabled    bool   `json:"enabled"`
}

type GatewayRoute struct {
	ID            string `gorm:"primaryKey" json:"id"`
	Path          string `json:"path"`
	Upstream      string `json:"upstream"`
	Methods       string `json:"-"`
	RateLimit     *int   `gorm:"column:rate_limit" json:"rate_limit"`
	TenantRouting bool   `gorm:"column:tenant_routing" json:"tenant_routing"`
	Status        string `json:"status"`
}

type ProofreadLog struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	Time       time.Time `json:"time"`
	TenantID   string    `gorm:"column:tenant_id" json:"tenant_id"`
	TextLength int       `gorm:"column:text_length" json:"text_length"`
	ErrorCount int       `gorm:"column:error_count" json:"error_count"`
	Latency    int       `json:"latency"`
	Provider   string    `json:"provider"`
	Status     string    `json:"status"`
}

type AlertRule struct {
	ID        string `gorm:"primaryKey" json:"id"`
	Name      string `json:"name"`
	Metric    string `json:"metric"`
	Condition string `json:"condition"`
	Threshold string `json:"threshold"`
	Duration  string `json:"duration"`
	Notify    string `json:"-"`
	Status    string `json:"status"`
}

type AlertEvent struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Time         time.Time `json:"time"`
	RuleName     string    `gorm:"column:rule_name" json:"rule_name"`
	Level        string    `json:"level"`
	TriggerValue string    `gorm:"column:trigger_value" json:"trigger_value"`
	Threshold    string    `json:"threshold"`
	Duration     string    `json:"duration"`
	Status       string    `json:"status"`
	Handler      string    `json:"handler"`
}

// User 用户模型（用于登录认证）
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"type:varchar(100);uniqueIndex" json:"username"`
	Password  string    `gorm:"type:varchar(255)" json:"-"`
	Role      string    `gorm:"type:varchar(50)" json:"role"`
	TenantID  string    `gorm:"column:tenant_id" json:"tenant_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}
