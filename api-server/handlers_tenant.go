package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func handleListTenants(c *gin.Context) {
	var tenants []Tenant
	db.Find(&tenants)
	for i := range tenants {
		var keys []APIKey
		db.Where("tenant_id = ?", tenants[i].ID).Find(&keys)
		tenants[i].APIKeys = keys
	}
	ok(c, tenants)
}

func handleGetTenant(c *gin.Context) {
	id := c.Param("id")
	var tenant Tenant
	if err := db.First(&tenant, "id = ?", id).Error; err != nil {
		fail(c, http.StatusNotFound, "租户不存在")
		return
	}
	var keys []APIKey
	db.Where("tenant_id = ?", id).Find(&keys)
	tenant.APIKeys = keys
	ok(c, tenant)
}

func handleCreateTenant(c *gin.Context) {
	var input struct {
		Name  string `json:"name"`
		Plan  string `json:"plan"`
		Quota int    `json:"quota"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if input.Plan == "" {
		input.Plan = "free"
	}
	if input.Quota == 0 {
		input.Quota = 10000
	}
	tenant := Tenant{
		ID: fmt.Sprintf("tenant-%03d", time.Now().UnixMilli()%10000), Name: input.Name,
		Schema: fmt.Sprintf("tenant_%s", strings.ToLower(input.Name)), Quota: input.Quota,
		Status: "active", Plan: input.Plan, CreatedAt: time.Now(),
		RedisPrefix: fmt.Sprintf("tenant:%s:", input.Name), DBTables: 5,
	}
	db.Create(&tenant)
	if rdb != nil {
		ctx, cancel := contextWithTimeout(3 * time.Second)
		defer cancel()
		rdb.Set(ctx, tenant.RedisPrefix+"usage", 0, 0)
	}
	ok(c, tenant)
}

func handleUpdateTenant(c *gin.Context) {
	id := c.Param("id")
	var tenant Tenant
	if err := db.First(&tenant, "id = ?", id).Error; err != nil {
		fail(c, http.StatusNotFound, "租户不存在")
		return
	}
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if v, ok := input["name"].(string); ok {
		tenant.Name = v
	}
	if v, ok := input["plan"].(string); ok {
		tenant.Plan = v
	}
	if v, ok := input["quota"].(float64); ok {
		tenant.Quota = int(v)
	}
	if v, ok := input["users"].(float64); ok {
		tenant.Users = int(v)
	}
	db.Save(&tenant)
	ok(c, tenant)
}

func handleFreezeTenant(c *gin.Context) {
	db.Model(&Tenant{}).Where("id = ?", c.Param("id")).Update("status", "frozen")
	okMsg(c, "租户已冻结")
}

func handleUnfreezeTenant(c *gin.Context) {
	db.Model(&Tenant{}).Where("id = ?", c.Param("id")).Update("status", "active")
	okMsg(c, "租户已解冻")
}

func handleCreateApiKey(c *gin.Context) {
	id := c.Param("id")
	key := APIKey{Key: fmt.Sprintf("mh-%s-sk-%s", id, randomString(12)), TenantID: id, Status: "active", CreatedAt: time.Now()}
	db.Create(&key)
	ok(c, gin.H{"key": key.Key})
}

func handleDeleteApiKey(c *gin.Context) {
	id := c.Param("id")
	key := c.Param("key")
	db.Where("tenant_id = ? AND `key` = ?", id, key).Delete(&APIKey{})
	okMsg(c, "API Key 已删除")
}
