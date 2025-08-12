package response

import (
	"net/http"

	"im-system/internal/model"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`            // 状态码：0表示成功，其他表示错误
	Message string      `json:"message"`         // 响应消息
	Data    interface{} `json:"data,omitempty"`  // 响应数据
	Error   string      `json:"error,omitempty"` // 错误详情（仅在开发环境显示）
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithDetails 带错误详情的错误响应
func ErrorWithDetails(c *gin.Context, code int, message string, err error) {
	response := Response{
		Code:    code,
		Message: message,
	}

	// 在开发环境下显示错误详情
	if gin.Mode() == gin.DebugMode && err != nil {
		response.Error = err.Error()
	}

	c.JSON(http.StatusOK, response)
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message string) {
	Error(c, 400, message)
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, 401, message)
}

// Forbidden 403错误
func Forbidden(c *gin.Context, message string) {
	Error(c, 403, message)
}

// NotFound 404错误
func NotFound(c *gin.Context, message string) {
	Error(c, 404, message)
}

// InternalError 500错误
func InternalError(c *gin.Context, message string) {
	Error(c, 500, message)
}

// UserInfo 用户信息（隐藏敏感字段）
type UserInfo struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Status    string `json:"status"`
	LastSeen  string `json:"last_seen"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FilterUserInfo 过滤用户信息，隐藏敏感字段
func FilterUserInfo(user *model.User) *UserInfo {
	if user == nil {
		return nil
	}

	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    user.Status,
		LastSeen:  user.LastSeen.Format("2006-01-02 15:04:05"),
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// LoginResponse 登录响应
type LoginResponse struct {
	User        *UserInfo `json:"user"`
	AccessToken string    `json:"access_token"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	User        *UserInfo `json:"user"`
	AccessToken string    `json:"access_token"`
}

// ProfileResponse 用户资料响应
type ProfileResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Message  string `json:"message,omitempty"`
}

// TokenInfoResponse Token信息响应
type TokenInfoResponse struct {
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	TokenInfo map[string]interface{} `json:"token_info"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	ID          uint   `json:"id"`
	SenderID    uint   `json:"sender_id"`
	ReceiverID  uint   `json:"receiver_id"`
	Content     string `json:"content"`
	MsgType     string `json:"msg_type"`
	Status      string `json:"status"`
	IsRead      bool   `json:"is_read"`
	SessionType int    `json:"session_type"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// FilterMessageInfo 过滤消息信息
func FilterMessageInfo(message *model.Message) *MessageResponse {
	if message == nil {
		return nil
	}

	return &MessageResponse{
		ID:          message.ID,
		SenderID:    message.SenderID,
		ReceiverID:  message.ReceiverID,
		Content:     message.Content,
		MsgType:     message.MsgType,
		Status:      message.Status,
		IsRead:      message.IsRead,
		SessionType: message.SessionType,
		CreatedAt:   message.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   message.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
