package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// 登录接口 — 验证用户名密码，返回 JWT token
func handleLogin(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 查找用户
	var user User
	if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
		fail(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 验证密码（种子数据中 admin/admin123 的 bcrypt hash）
	if !checkPassword(user.Password, input.Password) {
		fail(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 生成 JWT token
	token, err := generateToken(user.Username, user.Role, user.TenantID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成令牌失败")
		return
	}

	ok(c, gin.H{
		"token":      token,
		"username":   user.Username,
		"role":       user.Role,
		"tenant_id":  user.TenantID,
		"expires_in": 86400,
	})
}

// 获取当前用户信息
func handleProfile(c *gin.Context) {
	username := c.GetString("username")
	role := c.GetString("role")
	tenantID := c.GetString("tenant_id")
	ok(c, gin.H{
		"username":  username,
		"role":      role,
		"tenant_id": tenantID,
	})
}

// checkPassword 使用 bcrypt 验证密码
func checkPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

// hashPassword 用 bcrypt 加密密码
func hashPassword(plain string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(h)
}
