package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"im-system/internal/model"
)

// 消息缓存相关常量
const (
	PrivateMessagesKeyPrefix = "im:chat:"          // 私聊消息缓存key前缀
	ConversationsKeyPrefix   = "im:conversations:" // 对话列表缓存key前缀
)

// 缓存配置（从配置文件获取）
var (
	MessageCacheTTL        = 1 * time.Hour // 消息缓存TTL
	MaxCachedMessages      = 30            // 最大缓存消息数
	MaxCachedConversations = 10            // 最大缓存对话数
)

// SetCacheConfig 设置缓存配置
func SetCacheConfig(messageTTL time.Duration, maxMessages, maxConversations int) {
	MessageCacheTTL = messageTTL
	MaxCachedMessages = maxMessages
	MaxCachedConversations = maxConversations
}

// CachedMessage 缓存的消息结构
type CachedMessage struct {
	ID         uint      `json:"id"`
	SenderID   uint      `json:"sender_id"`
	ReceiverID uint      `json:"receiver_id"`
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CachedConversation 缓存的对话结构
type CachedConversation struct {
	UserID      uint      `json:"user_id"`
	Username    string    `json:"username"`
	LastMessage string    `json:"last_message"`
	LastTime    time.Time `json:"last_time"`
	UnreadCount int64     `json:"unread_count"`
}

// CachePrivateMessages 缓存私聊消息
func CachePrivateMessages(userID1, userID2 uint, messages []*model.Message) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	// 确保userID1 < userID2，保证key的一致性
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	key := fmt.Sprintf("%s%d:%d", PrivateMessagesKeyPrefix, userID1, userID2)

	// 转换为缓存格式
	var cachedMessages []CachedMessage
	for _, msg := range messages {
		cachedMessages = append(cachedMessages, CachedMessage{
			ID:         msg.ID,
			SenderID:   msg.SenderID,
			ReceiverID: msg.ReceiverID,
			Content:    msg.Content,
			IsRead:     msg.IsRead,
			CreatedAt:  msg.CreatedAt,
			UpdatedAt:  msg.UpdatedAt,
		})
	}

	// 序列化并存储
	data, err := json.Marshal(cachedMessages)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	err = Set(key, data, MessageCacheTTL)
	if err != nil {
		return fmt.Errorf("缓存私聊消息失败: %w", err)
	}

	return nil
}

// GetCachedPrivateMessages 获取缓存的私聊消息
func GetCachedPrivateMessages(userID1, userID2 uint) ([]*model.Message, error) {
	if client == nil {
		return nil, fmt.Errorf("redis客户端未初始化")
	}

	// 确保userID1 < userID2，保证key的一致性
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	key := fmt.Sprintf("%s%d:%d", PrivateMessagesKeyPrefix, userID1, userID2)

	// 从Redis获取数据
	data, err := Get(key)
	if err != nil {
		return nil, err
	}

	// 反序列化
	var cachedMessages []CachedMessage
	err = json.Unmarshal([]byte(data), &cachedMessages)
	if err != nil {
		return nil, fmt.Errorf("反序列化消息失败: %w", err)
	}

	// 转换为模型格式
	var messages []*model.Message
	for _, cached := range cachedMessages {
		messages = append(messages, &model.Message{
			ID:         cached.ID,
			SenderID:   cached.SenderID,
			ReceiverID: cached.ReceiverID,
			Content:    cached.Content,
			IsRead:     cached.IsRead,
			CreatedAt:  cached.CreatedAt,
			UpdatedAt:  cached.UpdatedAt,
		})
	}

	return messages, nil
}

// AddMessageToCache 添加新消息到缓存
func AddMessageToCache(userID1, userID2 uint, message *model.Message) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	// 确保userID1 < userID2
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	// 获取现有缓存
	existingMessages, err := GetCachedPrivateMessages(userID1, userID2)
	if err != nil {
		// 如果缓存不存在，创建新的
		existingMessages = []*model.Message{}
	}

	// 添加新消息到开头
	existingMessages = append([]*model.Message{message}, existingMessages...)

	// 限制缓存数量
	if len(existingMessages) > MaxCachedMessages {
		existingMessages = existingMessages[:MaxCachedMessages]
	}

	// 重新缓存
	return CachePrivateMessages(userID1, userID2, existingMessages)
}

// CacheConversations 缓存对话列表
func CacheConversations(userID uint, conversations []CachedConversation) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", ConversationsKeyPrefix, userID)

	// 限制缓存数量
	if len(conversations) > MaxCachedConversations {
		conversations = conversations[:MaxCachedConversations]
	}

	// 序列化并存储
	data, err := json.Marshal(conversations)
	if err != nil {
		return fmt.Errorf("序列化对话列表失败: %w", err)
	}

	err = Set(key, data, MessageCacheTTL)
	if err != nil {
		return fmt.Errorf("缓存对话列表失败: %w", err)
	}

	return nil
}

// GetCachedConversations 获取缓存的对话列表
func GetCachedConversations(userID uint) ([]CachedConversation, error) {
	if client == nil {
		return nil, fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", ConversationsKeyPrefix, userID)

	// 从Redis获取数据
	data, err := Get(key)
	if err != nil {
		return nil, err
	}

	// 反序列化
	var conversations []CachedConversation
	err = json.Unmarshal([]byte(data), &conversations)
	if err != nil {
		return nil, fmt.Errorf("反序列化对话列表失败: %w", err)
	}

	return conversations, nil
}

// UpdateConversationCache 更新对话缓存（当有新消息时）
func UpdateConversationCache(userID, otherUserID uint, username, lastMessage string, unreadCount int64) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	// 获取现有缓存
	conversations, err := GetCachedConversations(userID)
	if err != nil {
		// 如果缓存不存在，创建新的
		conversations = []CachedConversation{}
	}

	// 查找是否已存在该对话
	found := false
	for i, conv := range conversations {
		if conv.UserID == otherUserID {
			// 更新现有对话
			conversations[i].LastMessage = lastMessage
			conversations[i].LastTime = time.Now()
			conversations[i].UnreadCount = unreadCount
			found = true
			break
		}
	}

	if !found {
		// 添加新对话到开头
		newConv := CachedConversation{
			UserID:      otherUserID,
			Username:    username,
			LastMessage: lastMessage,
			LastTime:    time.Now(),
			UnreadCount: unreadCount,
		}
		conversations = append([]CachedConversation{newConv}, conversations...)
	}

	// 按最后消息时间排序
	for i := 0; i < len(conversations)-1; i++ {
		for j := i + 1; j < len(conversations); j++ {
			if conversations[i].LastTime.Before(conversations[j].LastTime) {
				conversations[i], conversations[j] = conversations[j], conversations[i]
			}
		}
	}

	// 重新缓存
	return CacheConversations(userID, conversations)
}

// ClearMessageCache 清除消息缓存
func ClearMessageCache(userID1, userID2 uint) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	// 确保userID1 < userID2
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	key := fmt.Sprintf("%s%d:%d", PrivateMessagesKeyPrefix, userID1, userID2)
	return Del(key)
}

// ClearConversationCache 清除对话缓存
func ClearConversationCache(userID uint) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", ConversationsKeyPrefix, userID)
	return Del(key)
}
