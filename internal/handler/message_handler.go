package handler

import (
	"strconv"

	"im-system/internal/service"
	"im-system/pkg/jwt"
	"im-system/pkg/response"

	"github.com/gin-gonic/gin"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	service *service.MessageService
}

// NewMessageHandler 创建MessageHandler实例
func NewMessageHandler(s *service.MessageService) *MessageHandler {
	return &MessageHandler{service: s}
}

// SendMessage 发送消息
func (h *MessageHandler) SendMessage(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 绑定请求参数
	type req struct {
		ReceiverID string `json:"receiver_id" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}
	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 发送消息
	message, err := h.service.SendMessage(uint(userID), r.ReceiverID, r.Content)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "消息发送成功", message)
}

// GetPrivateMessages 获取私聊消息历史
func (h *MessageHandler) GetPrivateMessages(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取对方用户ID
	otherUserID := c.Param("user_id")
	if otherUserID == "" {
		response.BadRequest(c, "user_id is required")
		return
	}

	// 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取消息历史
	messages, err := h.service.GetPrivateMessages(uint(userID), otherUserID, page, pageSize)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "获取消息历史成功", messages)
}

// GetUnreadMessages 获取未读消息
func (h *MessageHandler) GetUnreadMessages(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取未读消息
	messages, err := h.service.GetUnreadMessages(uint(userID))
	if err != nil {
		response.InternalError(c, "获取未读消息失败")
		return
	}

	response.SuccessWithMessage(c, "获取未读消息成功", messages)
}

// MarkAsRead 标记消息为已读
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取消息ID
	messageID := c.Param("message_id")
	if messageID == "" {
		response.BadRequest(c, "message_id is required")
		return
	}

	// 标记为已读
	err = h.service.MarkAsRead(messageID, uint(userID))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "消息已标记为已读", nil)
}

// GetUnreadCount 获取未读消息数量
func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取未读数量
	count, err := h.service.GetUnreadCount(uint(userID))
	if err != nil {
		response.InternalError(c, "获取未读消息数量失败")
		return
	}

	response.SuccessWithMessage(c, "获取未读消息数量成功", gin.H{
		"unread_count": count,
	})
}

// DeleteMessage 删除消息
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取消息ID
	messageID := c.Param("message_id")
	if messageID == "" {
		response.BadRequest(c, "message_id is required")
		return
	}

	// 删除消息
	err = h.service.DeleteMessage(messageID, uint(userID))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "消息删除成功", nil)
}

// GetRecentConversations 获取最近对话
func (h *MessageHandler) GetRecentConversations(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取限制参数
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 20
	}

	// 获取最近对话
	conversations, err := h.service.GetRecentConversations(uint(userID), limit)
	if err != nil {
		response.InternalError(c, "获取最近对话失败")
		return
	}

	response.SuccessWithMessage(c, "获取最近对话成功", conversations)
}
