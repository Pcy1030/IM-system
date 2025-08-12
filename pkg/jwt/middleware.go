package jwt

import (
	"strings"

	"im-system/pkg/logger"
	"im-system/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// ContextUserIDKey 用户ID在gin.Context中的键名
	ContextUserIDKey = "user_id"
	// ContextUsernameKey 用户名在gin.Context中的键名
	ContextUsernameKey = "username"
	// ContextClaimsKey JWT声明在gin.Context中的键名
	ContextClaimsKey = "jwt_claims"
)

// AuthMiddleware JWT认证中间件
// 从请求头中提取Authorization: Bearer <token>
// 验证token并将用户信息存入gin.Context
func (s *JWTService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少Authorization请求头")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Unauthorized(c, "Authorization格式错误，应为Bearer <token>")
			c.Abort()
			return
		}

		// 提取token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			response.Unauthorized(c, "token不能为空")
			c.Abort()
			return
		}

		// 验证token
		logger.Info("开始验证JWT token",
			zap.String("token_preview", tokenString[:20]+"..."),
			zap.String("secret_key_preview", string(s.secretKey[:10])+"..."),
		)

		claims, err := s.ValidateToken(tokenString)
		if err != nil {
			logger.Error("JWT验证失败",
				zap.Error(err),
				zap.String("token_preview", tokenString[:20]+"..."),
				zap.String("secret_key_preview", string(s.secretKey[:10])+"..."),
			)
			response.Unauthorized(c, "token无效或已过期")
			c.Abort()
			return
		}

		// 提取用户信息
		userID := claims.Subject
		username := ""
		if claims.Data != nil {
			if u, ok := claims.Data["username"].(string); ok {
				username = u
			}
		}

		// 将用户信息存入Context
		c.Set(ContextUserIDKey, userID)
		c.Set(ContextUsernameKey, username)
		c.Set(ContextClaimsKey, claims)

		// 记录访问日志
		logger.Info("用户访问接口",
			zap.String("user_id", userID),
			zap.String("username", username),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}

// GetUserID 从gin.Context中获取用户ID
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(ContextUserIDKey); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUsername 从gin.Context中获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(ContextUsernameKey); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}

// GetClaims 从gin.Context中获取JWT声明
func GetClaims(c *gin.Context) *CustomClaims {
	if claims, exists := c.Get(ContextClaimsKey); exists {
		if c, ok := claims.(*CustomClaims); ok {
			return c
		}
	}
	return nil
}

// RequireAuth 要求必须认证的中间件（可选，用于更严格的场景）
func (s *JWTService) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.AuthMiddleware()(c)
		if c.IsAborted() {
			return
		}

		// 额外检查：确保用户ID不为空
		userID := GetUserID(c)
		if userID == "" {
			response.Unauthorized(c, "用户信息无效")
			c.Abort()
			return
		}

		c.Next()
	}
}
