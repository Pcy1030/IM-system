package handler

import (
	"strconv"

	"im-system/internal/service"
	"im-system/pkg/jwt"
	"im-system/pkg/redis"
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

// GetConversationList 获取对话列表（带缓存）
func (h *MessageHandler) GetConversationList(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取限制参数
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	// 获取对话列表
	conversations, err := h.service.GetConversationList(uint(userID), limit)
	if err != nil {
		response.InternalError(c, "获取对话列表失败")
		return
	}

	// 转换为响应格式
	var conversationList []gin.H
	for _, conv := range conversations {
		conversationList = append(conversationList, gin.H{
			"user_id":      conv.UserID,
			"username":     conv.Username,
			"last_message": conv.LastMessage,
			"last_time":    conv.LastTime.Format("2006-01-02 15:04:05"),
			"unread_count": conv.UnreadCount,
		})
	}

	response.SuccessWithMessage(c, "获取对话列表成功", gin.H{
		"conversations": conversationList,
		"total":         len(conversationList),
	})
}

// MarkConversationAsRead 标记整个对话为已读
func (h *MessageHandler) MarkConversationAsRead(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取对方用户ID
	otherUserIDStr := c.Param("user_id")
	otherUserID, err := strconv.ParseUint(otherUserIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user_id parameter")
		return
	}

	// 标记对话为已读
	err = h.service.MarkConversationAsRead(uint(userID), uint(otherUserID))
	if err != nil {
		response.InternalError(c, "标记对话为已读失败")
		return
	}

	response.SuccessWithMessage(c, "标记对话为已读成功", nil)
}

// MarkAllAsRead 标记所有消息为已读
func (h *MessageHandler) MarkAllAsRead(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 标记所有消息为已读
	err = h.service.MarkAllAsRead(uint(userID))
	if err != nil {
		response.InternalError(c, "标记所有消息为已读失败")
		return
	}

	response.SuccessWithMessage(c, "标记所有消息为已读成功", nil)
}

// GetOfflineMessages 获取离线消息
func (h *MessageHandler) GetOfflineMessages(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取离线消息
	offlineMessages, err := redis.GetOfflineMessages(uint(userID), 50)
	if err != nil {
		response.InternalError(c, "获取离线消息失败")
		return
	}

	// 转换为API格式
	var messageList []gin.H
	for _, msg := range offlineMessages {
		messageList = append(messageList, gin.H{
			"id":         msg.ID,
			"sender_id":  msg.SenderID,
			"content":    msg.Content,
			"type":       msg.Type,
			"created_at": msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	response.SuccessWithMessage(c, "获取离线消息成功", gin.H{
		"messages": messageList,
		"total":    len(messageList),
	})
}

// ClearOfflineMessages 清空离线消息
func (h *MessageHandler) ClearOfflineMessages(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 清空离线消息
	err = redis.ClearOfflineMessages(uint(userID))
	if err != nil {
		response.InternalError(c, "清空离线消息失败")
		return
	}

	response.SuccessWithMessage(c, "清空离线消息成功", nil)
}

// GetOfflineMessageCount 获取离线消息数量
func (h *MessageHandler) GetOfflineMessageCount(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := jwt.GetUserID(c)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	// 获取离线消息数量
	count, err := redis.GetOfflineMessageCount(uint(userID))
	if err != nil {
		response.InternalError(c, "获取离线消息数量失败")
		return
	}

	response.SuccessWithMessage(c, "获取离线消息数量成功", gin.H{
		"count": count,
	})
}
