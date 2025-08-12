package websocket

import (
	"encoding/json"
	"im-system/config"
	"im-system/internal/repository"
	dbPkg "im-system/pkg/db"
	"im-system/pkg/jwt"
	"im-system/pkg/response"
	"net/http"
	"strconv"
	"strings"

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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &Client{
		UserID: uint(userID),
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}
	GetManager().AddClient(uint(userID), client)
	defer GetManager().RemoveClient(uint(userID))

	// 启动写协程
	go func() {
		for msg := range client.Send {
			_ = conn.WriteMessage(websocket.TextMessage, msg)
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

	// 读协程（可扩展为接收心跳/客户端消息）
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// 可处理心跳或客户端主动发消息
	}
}
