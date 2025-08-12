package handler

import (
	"im-system/internal/service"
	"im-system/pkg/jwt"
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
