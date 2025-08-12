package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"im-system/config"
	"im-system/internal/handler"
	"im-system/internal/model"
	"im-system/internal/repository"
	"im-system/internal/service"
	dbPkg "im-system/pkg/db"
	"im-system/pkg/jwt"
	"im-system/pkg/logger"
	"im-system/pkg/response"
	"im-system/pkg/websocket"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	cfg := config.LoadConfig()

	// 2. 初始化日志系统
	log := logger.InitLogger(cfg.Log)
	defer log.Sync()

	log.Info("=== IM系统启动 ===")
	log.Info("服务器配置信息",
		zap.String("port", cfg.Server.Port),
		zap.String("database_host", cfg.Database.Host),
		zap.Int("database_port", cfg.Database.Port),
		zap.String("database_name", cfg.Database.Database),
		zap.String("database_user", cfg.Database.Username),
		zap.Duration("jwt_expire_time", cfg.JWT.ExpireTime),
		zap.String("log_level", cfg.Log.Level),
	)

	// 3. 初始化数据库连接
	if _, err := dbPkg.InitDB(cfg.Database); err != nil {
		log.Fatal("数据库连接失败", zap.Error(err))
	}
	defer func() {
		if err := dbPkg.CloseDB(); err != nil {
			log.Error("关闭数据库连接失败", zap.Error(err))
		}
	}()
	log.Info("数据库连接成功")

	// 3.1 自动迁移表结构
	if err := dbPkg.AutoMigrate(&model.User{}, &model.Message{}, &model.Friendship{}); err != nil {
		log.Fatal("自动迁移失败", zap.Error(err))
	}
	log.Info("自动迁移完成")

	// 3.2 初始化业务服务
	jwtSvc := jwt.NewJWTService(cfg.JWT)
	userRepo := repository.NewUserRepository()
	messageRepo := repository.NewMessageRepository(dbPkg.GetDB())
	userSvc := service.NewUserService(userRepo, jwtSvc)
	messageSvc := service.NewMessageService(messageRepo, userRepo)
	userHandler := handler.NewUserHandler(userSvc)
	messageHandler := handler.NewMessageHandler(messageSvc)

	// 4. 设置Gin模式
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 5. 创建Gin路由
	router := gin.New()

	// 注入jwt_config到Gin context，供WebSocket使用
	router.Use(func(c *gin.Context) {
		c.Set("jwt_config", cfg.JWT)
		c.Next()
	})

	// 使用中间件
	router.Use(logger.LoggerMiddleware())      // 自定义日志中间件
	router.Use(logger.ErrorLoggerMiddleware()) // 错误日志中间件

	// 6. 设置基础路由
	setupBasicRoutes(router)

	// 6.1 绑定用户路由
	v1 := router.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			// 公开接口（无需认证）
			users.POST("/register", userHandler.Register)
			users.POST("/login", userHandler.Login)

			// 需要认证的接口
			authUsers := users.Group("")
			authUsers.Use(jwtSvc.AuthMiddleware()) // 应用JWT中间件
			{
				authUsers.GET("/profile", userHandler.GetProfile)
				authUsers.GET("/test-auth", userHandler.TestAuth)
			}
		}

		// 消息路由（需要认证）
		messages := v1.Group("/messages")
		messages.Use(jwtSvc.AuthMiddleware())
		{
			messages.POST("/send", messageHandler.SendMessage)                    // 发送消息
			messages.GET("/conversations", messageHandler.GetRecentConversations) // 获取最近对话
			messages.GET("/unread", messageHandler.GetUnreadMessages)             // 获取未读消息
			messages.GET("/unread/count", messageHandler.GetUnreadCount)          // 获取未读消息数量
			messages.PUT("/:message_id/read", messageHandler.MarkAsRead)          // 标记消息为已读
			messages.DELETE("/:message_id", messageHandler.DeleteMessage)         // 删除消息
		}

		// 私聊消息历史（需要认证）
		conversations := v1.Group("/conversations")
		conversations.Use(jwtSvc.AuthMiddleware())
		{
			conversations.GET("/:user_id/messages", messageHandler.GetPrivateMessages) // 获取与指定用户的私聊消息
		}
	}

	// WebSocket路由
	router.GET("/ws", websocket.WsHandler)

	// 7. 创建HTTP服务器
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 8. 启动HTTP服务器
	go func() {
		log.Info("HTTP服务器启动", zap.String("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 9. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在关闭服务器...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP服务器关闭失败", zap.Error(err))
	}

	log.Info("服务器已安全关闭")
}

// setupBasicRoutes 设置基础路由
func setupBasicRoutes(router *gin.Engine) {
	// 健康检查
	// 完整url为：http://localhost:8080/health
	router.GET("/health", func(c *gin.Context) {
		status := "ok"
		if err := dbPkg.HealthCheck(); err != nil {
			status = "db-down"
		}
		response.Success(c, gin.H{
			"status":  status,
			"message": "IM系统运行状态",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 根路径
	// 完整url为：http://localhost:8080/
	router.GET("/", func(c *gin.Context) {
		response.Success(c, gin.H{
			"message": "欢迎使用IM系统",
			"version": "1.1.0",
			"status":  "系统运行正常，核心功能已实现",
		})
	})

	// 配置信息路由（系统状态监控）
	// 完整url为：http://localhost:8080/config
	router.GET("/config", func(c *gin.Context) {
		cfg := config.LoadConfig()
		response.Success(c, gin.H{
			"server": gin.H{
				"port": cfg.Server.Port,
			},
			"database": gin.H{
				"host":     cfg.Database.Host,
				"port":     cfg.Database.Port,
				"database": cfg.Database.Database,
				"driver":   cfg.Database.Driver,
				"username": cfg.Database.Username,
			},
			"jwt": gin.H{
				"expireTime": cfg.JWT.ExpireTime.String(),
				"issuer":     cfg.JWT.Issuer,
			},
			"log": gin.H{
				"level":    cfg.Log.Level,
				"filename": cfg.Log.Filename,
			},
		})
	})

	// API版本组（核心功能已实现）
	v1 := router.Group("/api/v1")
	{
		// 完整url为：http://localhost:8080/api/v1/status
		v1.GET("/status", func(c *gin.Context) {
			response.Success(c, gin.H{
				"message": "API v1 路由已配置，核心功能已实现",
				"modules": []string{
					"用户管理 - 已完成",
					"消息系统 - 已完成",
					"WebSocket - 已完成",
					"JWT认证 - 已完成",
					"数据库连接 - 已完成",
					"日志系统 - 已完成",
				},
			})
		})
	}
}
