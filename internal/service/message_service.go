package service

import (
	"errors"
	"strconv"

	"encoding/json"
	"im-system/internal/model"
	"im-system/internal/repository"
	"im-system/pkg/websocket"
)

// MessageService 消息服务
type MessageService struct {
	messageRepo *repository.MessageRepository
	userRepo    *repository.UserRepository
}

// NewMessageService 创建MessageService实例
func NewMessageService(messageRepo *repository.MessageRepository, userRepo *repository.UserRepository) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		userRepo:    userRepo,
	}
}

// SendMessage 发送私聊消息
func (s *MessageService) SendMessage(senderID uint, receiverIDStr, content string) (*model.Message, error) {
	// 验证接收者ID
	receiverID, err := strconv.ParseUint(receiverIDStr, 10, 32)
	if err != nil {
		return nil, errors.New("invalid receiver ID")
	}

	// 检查接收者是否存在
	_, err = s.userRepo.GetByID(uint(receiverID))
	if err != nil {
		return nil, errors.New("receiver not found")
	}

	// 不能给自己发消息
	if senderID == uint(receiverID) {
		return nil, errors.New("cannot send message to yourself")
	}

	// 创建消息
	message := &model.Message{
		SenderID:    senderID,
		ReceiverID:  uint(receiverID),
		Content:     content,
		MsgType:     "text", // 默认文本消息
		IsRead:      false,
		SessionType: 1,      // 单聊
		Status:      "sent", // 已发送
	}

	// 保存消息
	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}

	// WebSocket推送
	msgData := map[string]interface{}{
		"type":      "chat",
		"from":      senderID,
		"to":        uint(receiverID),
		"content":   content,
		"msg_id":    message.ID,
		"timestamp": message.CreatedAt.Unix(),
	}
	msgBytes, _ := json.Marshal(msgData)
	websocket.GetManager().SendToUser(uint(receiverID), msgBytes)

	return message, nil
}

// GetPrivateMessages 获取私聊消息历史
func (s *MessageService) GetPrivateMessages(userID uint, otherUserIDStr string, page, pageSize int) ([]*model.Message, error) {
	// 验证对方用户ID
	otherUserID, err := strconv.ParseUint(otherUserIDStr, 10, 32)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// 检查对方用户是否存在
	_, err = s.userRepo.GetByID(uint(otherUserID))
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 计算分页参数
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20 // 默认每页20条
	}

	// 获取消息
	messages, err := s.messageRepo.GetPrivateMessages(userID, uint(otherUserID), pageSize, offset)
	if err != nil {
		return nil, err
	}

	// 标记消息为已读
	go s.messageRepo.MarkConversationAsRead(userID, uint(otherUserID))

	return messages, nil
}

// GetUnreadMessages 获取未读消息
func (s *MessageService) GetUnreadMessages(userID uint) ([]*model.Message, error) {
	return s.messageRepo.GetUnreadMessages(userID)
}

// MarkAsRead 标记消息为已读
func (s *MessageService) MarkAsRead(messageIDStr string, userID uint) error {
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		return errors.New("invalid message ID")
	}

	// 检查消息是否存在且属于当前用户
	message, err := s.messageRepo.GetByID(uint(messageID))
	if err != nil {
		return errors.New("message not found")
	}

	// 只能标记发给自己的消息为已读
	if message.ReceiverID != userID {
		return errors.New("permission denied")
	}

	return s.messageRepo.MarkAsRead(uint(messageID))
}

// GetUnreadCount 获取未读消息数量
func (s *MessageService) GetUnreadCount(userID uint) (int64, error) {
	return s.messageRepo.GetUnreadCount(userID)
}

// DeleteMessage 删除消息
func (s *MessageService) DeleteMessage(messageIDStr string, userID uint) error {
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		return errors.New("invalid message ID")
	}

	// 检查消息是否存在且属于当前用户
	message, err := s.messageRepo.GetByID(uint(messageID))
	if err != nil {
		return errors.New("message not found")
	}

	// 只能删除自己发送的消息
	if message.SenderID != userID {
		return errors.New("permission denied")
	}

	return s.messageRepo.DeleteMessage(uint(messageID), userID)
}

// GetRecentConversations 获取最近对话
func (s *MessageService) GetRecentConversations(userID uint, limit int) ([]*model.Message, error) {
	if limit <= 0 || limit > 50 {
		limit = 20 // 默认20条
	}

	return s.messageRepo.GetRecentConversations(userID, limit)
}
