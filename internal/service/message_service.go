package service

import (
	"errors"
	"strconv"

	"encoding/json"
	"im-system/internal/model"
	"im-system/internal/repository"
	"im-system/pkg/redis"
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

	// 添加到缓存
	_ = redis.AddMessageToCache(senderID, uint(receiverID), message)

	// 增加接收者未读消息计数
	_ = redis.IncrementUnreadCount(uint(receiverID))

	// 更新对话缓存
	receiver, _ := s.userRepo.GetByID(uint(receiverID))
	if receiver != nil {
		// 获取Redis中的未读消息数
		unreadCount, _ := redis.GetUnreadCount(uint(receiverID))
		_ = redis.UpdateConversationCache(senderID, uint(receiverID), receiver.Username, content, unreadCount)
		_ = redis.UpdateConversationCache(uint(receiverID), senderID, "", content, 0) // 发送者不需要未读数
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

	var messages []*model.Message

	// 如果是第一页且请求数量在缓存范围内，尝试从缓存获取
	if page == 1 && pageSize <= redis.MaxCachedMessages {
		cachedMessages, cacheErr := redis.GetCachedPrivateMessages(userID, uint(otherUserID))
		if cacheErr == nil && len(cachedMessages) > 0 {
			// 缓存命中，直接返回缓存数据
			if len(cachedMessages) >= pageSize {
				messages = cachedMessages[:pageSize]
			} else {
				messages = cachedMessages
			}
		} else {
			// 缓存未命中，从数据库获取并缓存
			messages, err = s.messageRepo.GetPrivateMessages(userID, uint(otherUserID), pageSize, offset)
			if err != nil {
				return nil, err
			}
			// 异步缓存消息
			go func() {
				_ = redis.CachePrivateMessages(userID, uint(otherUserID), messages)
			}()
		}
	} else {
		// 超出缓存范围或非第一页，直接从数据库获取
		messages, err = s.messageRepo.GetPrivateMessages(userID, uint(otherUserID), pageSize, offset)
		if err != nil {
			return nil, err
		}
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

	// 检查消息是否已经标记为已读
	if message.IsRead {
		return nil // 已经标记为已读，无需重复操作
	}

	// 标记数据库中的消息为已读
	err = s.messageRepo.MarkAsRead(uint(messageID))
	if err != nil {
		return err
	}

	// 减少Redis中的未读消息计数
	_ = redis.DecrementUnreadCount(userID)

	return nil
}

// GetUnreadCount 获取未读消息数量（优先从Redis获取）
func (s *MessageService) GetUnreadCount(userID uint) (int64, error) {
	// 优先从Redis获取
	count, err := redis.GetUnreadCount(userID)
	if err == nil {
		return count, nil
	}

	// Redis获取失败，从数据库获取并同步到Redis
	dbCount, err := s.messageRepo.GetUnreadCount(userID)
	if err != nil {
		return 0, err
	}

	// 同步到Redis
	_ = redis.SetUnreadCount(userID, dbCount)

	return dbCount, nil
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

// GetConversationList 获取对话列表（带缓存）
func (s *MessageService) GetConversationList(userID uint, limit int) ([]redis.CachedConversation, error) {
	if limit <= 0 || limit > redis.MaxCachedConversations {
		limit = redis.MaxCachedConversations
	}

	// 尝试从缓存获取
	cachedConversations, err := redis.GetCachedConversations(userID)
	if err == nil && len(cachedConversations) > 0 {
		// 缓存命中，返回缓存数据
		if len(cachedConversations) > limit {
			return cachedConversations[:limit], nil
		}
		return cachedConversations, nil
	}

	// 缓存未命中，从数据库获取并构建对话列表
	messages, err := s.messageRepo.GetRecentConversations(userID, limit*2) // 获取更多消息用于构建对话
	if err != nil {
		return nil, err
	}

	// 构建对话列表
	conversationMap := make(map[uint]*redis.CachedConversation)
	for _, msg := range messages {
		var otherUserID uint
		if msg.SenderID == userID {
			otherUserID = msg.ReceiverID
		} else {
			otherUserID = msg.SenderID
		}

		if conv, exists := conversationMap[otherUserID]; exists {
			// 更新现有对话
			if msg.CreatedAt.After(conv.LastTime) {
				conv.LastMessage = msg.Content
				conv.LastTime = msg.CreatedAt
			}
		} else {
			// 创建新对话
			otherUser, _ := s.userRepo.GetByID(otherUserID)
			username := ""
			if otherUser != nil {
				username = otherUser.Username
			}

			conversationMap[otherUserID] = &redis.CachedConversation{
				UserID:      otherUserID,
				Username:    username,
				LastMessage: msg.Content,
				LastTime:    msg.CreatedAt,
				UnreadCount: 0, // 稍后统一设置
			}
		}
	}

	// 转换为切片并按时间排序
	var conversations []redis.CachedConversation
	for _, conv := range conversationMap {
		conversations = append(conversations, *conv)
	}

	// 按最后消息时间排序
	for i := 0; i < len(conversations)-1; i++ {
		for j := i + 1; j < len(conversations); j++ {
			if conversations[i].LastTime.Before(conversations[j].LastTime) {
				conversations[i], conversations[j] = conversations[j], conversations[i]
			}
		}
	}

	// 限制数量
	if len(conversations) > limit {
		conversations = conversations[:limit]
	}

	// 统一设置未读计数（从Redis获取）
	for i := range conversations {
		unreadCount, _ := redis.GetUnreadCount(userID)
		conversations[i].UnreadCount = unreadCount
	}

	// 异步缓存对话列表
	go func() {
		_ = redis.CacheConversations(userID, conversations)
	}()

	return conversations, nil
}

// MarkConversationAsRead 标记整个对话为已读（批量操作）
func (s *MessageService) MarkConversationAsRead(userID, otherUserID uint) error {
	// 标记数据库中的消息为已读
	err := s.messageRepo.MarkConversationAsRead(userID, otherUserID)
	if err != nil {
		return err
	}

	// 获取该对话的未读消息数量
	unreadCount, err := s.messageRepo.GetUnreadCount(userID)
	if err != nil {
		return err
	}

	// 更新Redis中的未读计数
	_ = redis.SetUnreadCount(userID, unreadCount)

	return nil
}

// MarkAllAsRead 标记所有消息为已读
func (s *MessageService) MarkAllAsRead(userID uint) error {
	// 获取当前未读消息
	unreadMessages, err := s.messageRepo.GetUnreadMessages(userID)
	if err != nil {
		return err
	}

	// 批量标记为已读
	for _, msg := range unreadMessages {
		_ = s.messageRepo.MarkAsRead(msg.ID)
	}

	// 重置Redis未读计数
	_ = redis.ResetUnreadCount(userID)

	return nil
}
