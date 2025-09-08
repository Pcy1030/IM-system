package handler

import (
	"fmt"
	"im-system/internal/service"
	"im-system/pkg/jwt"
	"im-system/pkg/redis"
	"im-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

// Register 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	type req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email"`
		Password string `json:"password" binding:"required"`
	}
	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, token, err := h.service.Register(r.Username, r.Email, r.Password)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "注册成功", &response.RegisterResponse{
		User:        response.FilterUserInfo(user),
		AccessToken: token,
	})
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	type req struct {
		UsernameOrEmail string `json:"usernameOrEmail" binding:"required"`
		Password        string `json:"password" binding:"required"`
	}
	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, token, err := h.service.Login(r.UsernameOrEmail, r.Password)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "登录成功", &response.LoginResponse{
		User:        response.FilterUserInfo(user),
		AccessToken: token,
	})
}

// GetProfile 获取用户资料（需要JWT认证）
func (h *UserHandler) GetProfile(c *gin.Context) {
	// 从JWT中间件设置的Context中获取用户信息
	userID := jwt.GetUserID(c)
	username := jwt.GetUsername(c)

	response.SuccessWithMessage(c, "获取用户资料成功", &response.ProfileResponse{
		UserID:   userID,
		Username: username,
		Message:  "这是受保护的接口，只有携带有效JWT token才能访问",
	})
}

// TestAuth 测试JWT认证的接口
func (h *UserHandler) TestAuth(c *gin.Context) {
	userID := jwt.GetUserID(c)
	username := jwt.GetUsername(c)
	claims := jwt.GetClaims(c)

	// 转换时间戳为可读格式
	var issuedAtStr, expiresAtStr string
	if claims.IssuedAt != nil {
		issuedAtStr = claims.IssuedAt.Time.Format("2006-01-02 15:04:05")
	}
	if claims.ExpiresAt != nil {
		expiresAtStr = claims.ExpiresAt.Time.Format("2006-01-02 15:04:05")
	}

	tokenInfo := map[string]interface{}{
		"issuer":               claims.Issuer,
		"issued_at":            issuedAtStr,
		"expires_at":           expiresAtStr,
		"issued_at_timestamp":  claims.IssuedAt,
		"expires_at_timestamp": claims.ExpiresAt,
	}

	response.SuccessWithMessage(c, "JWT认证测试成功", &response.TokenInfoResponse{
		UserID:    userID,
		Username:  username,
		TokenInfo: tokenInfo,
	})
}

// Logout 用户登出（需要JWT认证）：仅更新在线状态为offline
func (h *UserHandler) Logout(c *gin.Context) {
	userIDStr := jwt.GetUserID(c)
	if userIDStr == "" {
		response.Unauthorized(c, "用户未认证")
		return
	}
	// 将字符串ID转换为uint
	var uid uint
	if _, err := fmt.Sscanf(userIDStr, "%d", &uid); err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.service.Logout(uid); err != nil {
		response.InternalError(c, "登出失败")
		return
	}
	response.SuccessWithMessage(c, "已离线", nil)
}

// GetOnlineUsers 获取在线用户列表（需要JWT认证）
func (h *UserHandler) GetOnlineUsers(c *gin.Context) {
	// 获取在线用户详细信息
	presences, err := redis.GetOnlineUsersWithDetails()
	if err != nil {
		response.InternalError(c, "获取在线用户失败")
		return
	}

	// 转换为响应格式
	var onlineUsers []gin.H
	for _, presence := range presences {
		onlineUsers = append(onlineUsers, gin.H{
			"user_id":   presence.UserID,
			"username":  presence.Username,
			"status":    presence.Status,
			"last_seen": presence.LastSeen.Format("2006-01-02 15:04:05"),
			"connected": presence.Connected,
		})
	}

	response.SuccessWithMessage(c, "获取在线用户成功", gin.H{
		"online_count": len(onlineUsers),
		"users":        onlineUsers,
	})
}

// CheckUserOnline 检查指定用户是否在线（需要JWT认证）
func (h *UserHandler) CheckUserOnline(c *gin.Context) {
	// 获取要检查的用户ID
	userIDStr := c.Param("user_id")
	if userIDStr == "" {
		response.BadRequest(c, "user_id is required")
		return
	}

	var userID uint
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		response.BadRequest(c, "invalid user_id")
		return
	}

	// 检查是否在线
	online, err := redis.IsUserOnline(userID)
	if err != nil {
		response.InternalError(c, "检查用户在线状态失败")
		return
	}

	// 如果在线，获取详细信息
	var presence *redis.PresenceData
	if online {
		presence, err = redis.GetUserPresence(userID)
		if err != nil {
			response.InternalError(c, "获取用户在线信息失败")
			return
		}
	}

	result := gin.H{
		"user_id": userID,
		"online":  online,
	}

	if presence != nil {
		result["username"] = presence.Username
		result["last_seen"] = presence.LastSeen.Format("2006-01-02 15:04:05")
		result["connected"] = presence.Connected
	}

	response.SuccessWithMessage(c, "检查用户在线状态成功", result)
}
