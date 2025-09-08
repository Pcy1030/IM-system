package repository

import (
	"errors"
	"time"

	"im-system/internal/model"

	"gorm.io/gorm"
)

// MessageRepository 消息数据仓储
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 创建MessageRepository实例
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create 创建消息
func (r *MessageRepository) Create(message *model.Message) error {
	return r.db.Create(message).Error
}

// GetByID 根据ID获取消息
func (r *MessageRepository) GetByID(id uint) (*model.Message, error) {
	var message model.Message
	err := r.db.First(&message, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("message not found")
		}
		return nil, err
	}
	return &message, nil
}

// GetPrivateMessages 获取两个用户之间的私聊消息
func (r *MessageRepository) GetPrivateMessages(senderID, receiverID uint, limit, offset int) ([]*model.Message, error) {
	var messages []*model.Message

	// 查询发送者和接收者之间的消息（双向）
	err := r.db.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		senderID, receiverID, receiverID, senderID,
	).
		Where("group_id IS NULL"). // 确保是私聊消息
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	return messages, err
}

// GetUnreadMessages 获取用户未读消息
func (r *MessageRepository) GetUnreadMessages(userID uint) ([]*model.Message, error) {
	var messages []*model.Message

	err := r.db.Where("receiver_id = ? AND is_read = ?", userID, false).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, err
}

// MarkAsRead 标记消息为已读
func (r *MessageRepository) MarkAsRead(messageID uint) error {
	return r.db.Model(&model.Message{}).
		Where("id = ?", messageID).
		Update("is_read", true).Error
}

// MarkConversationAsRead 标记整个对话为已读
func (r *MessageRepository) MarkConversationAsRead(userID, otherUserID uint) error {
	return r.db.Model(&model.Message{}).
		Where("receiver_id = ? AND sender_id = ? AND is_read = ?", userID, otherUserID, false).
		Update("is_read", true).Error
}

// GetUnreadCount 获取用户未读消息数量
func (r *MessageRepository) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Message{}).
		Where("receiver_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// GetConversationUnreadCount 获取与特定用户的未读消息数量
func (r *MessageRepository) GetConversationUnreadCount(userID, otherUserID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Message{}).
		Where("receiver_id = ? AND sender_id = ? AND is_read = ?", userID, otherUserID, false).
		Count(&count).Error
	return count, err
}

// DeleteMessage 删除消息（软删除）
func (r *MessageRepository) DeleteMessage(messageID, userID uint) error {
	// 只能删除自己发送的消息
	return r.db.Model(&model.Message{}).
		Where("id = ? AND sender_id = ?", messageID, userID).
		Update("deleted_at", time.Now()).Error
}

// GetRecentConversations 获取用户最近的对话列表
func (r *MessageRepository) GetRecentConversations(userID uint, limit int) ([]*model.Message, error) {
	var messages []*model.Message

	// 简化版本：获取用户发送或接收的最新消息
	err := r.db.Where("(sender_id = ? OR receiver_id = ?) AND group_id IS NULL", userID, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error

	return messages, err
}
