package websocket

import (
	"encoding/json"
	"im-system/config"
	"im-system/internal/repository"
	dbPkg "im-system/pkg/db"
	"im-system/pkg/jwt"
	"im-system/pkg/redis"
	"im-system/pkg/response"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// WsHandler Gin路由处理函数
func WsHandler(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = strings.TrimPrefix(c.GetHeader("Sec-WebSocket-Protocol"), "Bearer ")
	}
	if token == "" {
		response.Unauthorized(c, "缺少token")
		return
	}

	jwtCfg := c.MustGet("jwt_config").(config.JWTConfig) // 需在main.go注入
	jwtSvc := jwt.NewJWTService(jwtCfg)
	claims, err := jwtSvc.ValidateToken(token)
	if err != nil {
		response.Unauthorized(c, "token无效或已过期")
		return
	}
	userID, _ := strconv.ParseUint(claims.Subject, 10, 32)
	if userID == 0 {
		response.Unauthorized(c, "token无效")
		return
	}

	// 回显子协议，避免客户端提示 "Server sent no subprotocol"
	respHeader := http.Header{}
	if protocol := c.GetHeader("Sec-WebSocket-Protocol"); protocol != "" {
		respHeader.Set("Sec-WebSocket-Protocol", protocol)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, respHeader)
	if err != nil {
		return
	}

	client := &Client{
		UserID: uint(userID),
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}
	GetManager().AddClient(uint(userID), client)

	// WebSocket连接建立后，设置用户状态为 online
	// 1. 更新数据库状态
	if db := dbPkg.GetDB(); db != nil {
		userRepo := repository.NewUserRepository()
		_ = userRepo.UpdateStatus(uint(userID), "online")
	}

	// 2. 更新Redis在线状态
	username := claims.Data["username"].(string)
	_ = redis.SetUserPresence(uint(userID), username, "online")

	defer func() {
		GetManager().RemoveClient(uint(userID))

		// 连接关闭后，设置用户状态为 offline
		// 1. 更新数据库状态
		if db := dbPkg.GetDB(); db != nil {
			userRepo := repository.NewUserRepository()
			_ = userRepo.UpdateStatus(uint(userID), "offline")
		}

		// 2. 更新Redis在线状态
		_ = redis.SetUserPresence(uint(userID), username, "offline")
	}()

	// 从上下文读取心跳配置
	wsCfg := c.MustGet("ws_config").(config.WebSocketConfig)

	// 启动写协程 + 定时发送ping心跳
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(wsCfg.PingInterval)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					return
				}
				_ = conn.WriteMessage(websocket.TextMessage, msg)
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					close(done)
					return
				}
			}
		}
	}()

	// 用户上线后，自动推送数据库中的未读消息
	if db := dbPkg.GetDB(); db != nil {
		msgRepo := repository.NewMessageRepository(db)
		if unreadMessages, err := msgRepo.GetUnreadMessages(uint(userID)); err == nil {
			for _, m := range unreadMessages {
				payload := map[string]interface{}{
					"type":      "chat",
					"from":      m.SenderID,
					"to":        m.ReceiverID,
					"content":   m.Content,
					"msg_id":    m.ID,
					"timestamp": m.CreatedAt.Unix(),
				}
				if b, e := json.Marshal(payload); e == nil {
					client.Send <- b
				}
			}
		}
	}

	// 读协程（接收心跳/客户端消息）。若超时未收到任何读事件则断开
	_ = conn.SetReadDeadline(time.Now().Add(wsCfg.ReadTimeout))
	conn.SetPongHandler(func(appData string) error {
		return conn.SetReadDeadline(time.Now().Add(wsCfg.ReadTimeout))
	})
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(wsCfg.ReadTimeout))
		var msg map[string]interface{}
		if err := json.Unmarshal(payload, &msg); err == nil {
			if t, ok := msg["type"].(string); ok {
				switch t {
				case "ack_read":
					var msgID uint64
					switch v := msg["msg_id"].(type) {
					case float64:
						msgID = uint64(v)
					case string:
						if id, e := strconv.ParseUint(v, 10, 64); e == nil {
							msgID = id
						}
					}
					if msgID > 0 {
						if db := dbPkg.GetDB(); db != nil {
							repo := repository.NewMessageRepository(db)
							if m, e := repo.GetByID(uint(msgID)); e == nil {
								if m.ReceiverID == uint(userID) {
									_ = repo.MarkAsRead(uint(msgID))
								}
							}
						}
					}
				case "heartbeat":
					// 刷新用户在线状态（延长TTL）
					_ = redis.RefreshUserPresence(uint(userID))
					if db := dbPkg.GetDB(); db != nil {
						userRepo := repository.NewUserRepository()
						_ = userRepo.UpdateStatus(uint(userID), "online")
					}
				}
			}
		}
	}
	select {
	case <-done:
	default:
		close(done)
	}
}
