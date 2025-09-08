package redis

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// OfflineMessage 离线消息结构
type OfflineMessage struct {
	ID         uint      `json:"id"`
	SenderID   uint      `json:"sender_id"`
	ReceiverID uint      `json:"receiver_id"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
}

// 离线消息相关常量
const (
	OfflineMessagesKeyPrefix = "im:offline:"      // 离线消息key前缀
	OfflineMessagesTTL       = 7 * 24 * time.Hour // 7天过期
)

// AddOfflineMessage 添加离线消息
func AddOfflineMessage(receiverID uint, message *OfflineMessage) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", OfflineMessagesKeyPrefix, receiverID)

	// 将消息序列化为JSON
	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化离线消息失败: %w", err)
	}

	// 使用LPUSH添加到列表头部（最新的消息在前面）
	err = client.LPush(ctx, key, messageData).Err()
	if err != nil {
		return fmt.Errorf("添加离线消息失败: %w", err)
	}

	// 设置TTL
	err = client.Expire(ctx, key, OfflineMessagesTTL).Err()
	if err != nil {
		return fmt.Errorf("设置离线消息TTL失败: %w", err)
	}

	// 限制离线消息数量（最多保存100条）
	err = client.LTrim(ctx, key, 0, 99).Err()
	if err != nil {
		return fmt.Errorf("限制离线消息数量失败: %w", err)
	}

	return nil
}

// GetOfflineMessages 获取用户的离线消息
func GetOfflineMessages(receiverID uint, limit int) ([]*OfflineMessage, error) {
	if client == nil {
		return nil, fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", OfflineMessagesKeyPrefix, receiverID)

	// 从列表头部获取指定数量的消息
	results, err := client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("获取离线消息失败: %w", err)
	}

	var messages []*OfflineMessage
	for _, result := range results {
		var message OfflineMessage
		err := json.Unmarshal([]byte(result), &message)
		if err != nil {
			continue // 跳过无法解析的消息
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

// ClearOfflineMessages 清空用户的离线消息
func ClearOfflineMessages(receiverID uint) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", OfflineMessagesKeyPrefix, receiverID)

	// 删除离线消息列表
	err := client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("清空离线消息失败: %w", err)
	}

	return nil
}

// GetOfflineMessageCount 获取用户离线消息数量
func GetOfflineMessageCount(receiverID uint) (int64, error) {
	if client == nil {
		return 0, fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", OfflineMessagesKeyPrefix, receiverID)

	// 获取列表长度
	count, err := client.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("获取离线消息数量失败: %w", err)
	}

	return count, nil
}

// BatchAddOfflineMessages 批量添加离线消息
func BatchAddOfflineMessages(receiverID uint, messages []*OfflineMessage) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	if len(messages) == 0 {
		return nil
	}

	key := fmt.Sprintf("%s%d", OfflineMessagesKeyPrefix, receiverID)

	// 使用Pipeline批量操作
	pipe := client.Pipeline()

	// 将消息序列化并添加到列表
	for _, message := range messages {
		messageData, err := json.Marshal(message)
		if err != nil {
			continue // 跳过无法序列化的消息
		}
		pipe.LPush(ctx, key, messageData)
	}

	// 设置TTL和限制数量
	pipe.Expire(ctx, key, OfflineMessagesTTL)
	pipe.LTrim(ctx, key, 0, 99)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("批量添加离线消息失败: %w", err)
	}

	return nil
}

// RemoveOfflineMessage 移除指定的离线消息
func RemoveOfflineMessage(receiverID uint, messageID uint) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", OfflineMessagesKeyPrefix, receiverID)

	// 获取所有离线消息
	messages, err := GetOfflineMessages(receiverID, 100)
	if err != nil {
		return err
	}

	// 找到要删除的消息并重新构建列表
	var newMessages []*OfflineMessage
	for _, msg := range messages {
		if msg.ID != messageID {
			newMessages = append(newMessages, msg)
		}
	}

	// 清空原列表
	err = client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("清空离线消息列表失败: %w", err)
	}

	// 重新添加剩余消息
	if len(newMessages) > 0 {
		err = BatchAddOfflineMessages(receiverID, newMessages)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAllOfflineMessageKeys 获取所有离线消息key（用于管理后台）
func GetAllOfflineMessageKeys() ([]string, error) {
	if client == nil {
		return nil, fmt.Errorf("redis客户端未初始化")
	}

	// 使用 SCAN 非阻塞地遍历所有离线消息 key
	var keys []string
	var cursor uint64
	pattern := fmt.Sprintf("%s*", OfflineMessagesKeyPrefix)
	for {
		ks, c, err := client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("获取离线消息key失败: %w", err)
		}
		keys = append(keys, ks...)
		cursor = c
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// GetOfflineMessageStats 获取离线消息统计信息
func GetOfflineMessageStats() (map[uint]int64, error) {
	if client == nil {
		return nil, fmt.Errorf("redis客户端未初始化")
	}

	// 获取所有离线消息key
	keys, err := GetAllOfflineMessageKeys()
	if err != nil {
		return nil, err
	}

	stats := make(map[uint]int64)

	// 批量获取每个用户的离线消息数量
	if len(keys) > 0 {
		pipe := client.Pipeline()
		cmds := make(map[string]*redis.IntCmd)

		for _, key := range keys {
			cmds[key] = pipe.LLen(ctx, key)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("批量获取离线消息统计失败: %w", err)
		}

		// 解析结果
		for key, cmd := range cmds {
			count, err := cmd.Result()
			if err == nil && count > 0 {
				// 从key中提取userID
				userIDStr := key[len(OfflineMessagesKeyPrefix):]
				if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
					stats[uint(userID)] = count
				}
			}
		}
	}

	return stats, nil
}
