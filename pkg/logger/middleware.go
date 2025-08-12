package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 记录请求信息
		Info("HTTP请求",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.String("ip", param.ClientIP),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("user_agent", param.Request.UserAgent()),
			zap.String("error", param.ErrorMessage),
		)
		return ""
	})
}

// ErrorLoggerMiddleware 错误日志中间件
func ErrorLoggerMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			Error("HTTP请求发生panic",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
				zap.String("error", err),
			)
		}
		c.AbortWithStatus(500)
	})
}

// RequestLogger 请求日志记录器
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		
		// 处理请求
		c.Next()
		
		// 结束时间
		end := time.Now()
		latency := end.Sub(start)
		
		// 获取状态码
		status := c.Writer.Status()
		
		// 记录请求日志
		logger := WithFields(map[string]interface{}{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip":         c.ClientIP(),
			"status":     status,
			"latency":    latency.String(),
			"user_agent": c.Request.UserAgent(),
		})
		
		// 根据状态码选择日志级别
		switch {
		case status >= 500:
			logger.Error("HTTP请求错误")
		case status >= 400:
			logger.Warn("HTTP请求警告")
		default:
			logger.Info("HTTP请求成功")
		}
	}
}
