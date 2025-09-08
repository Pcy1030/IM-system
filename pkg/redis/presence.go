package redis

import (
	"encoding/json"
	"fmt"
	"time"
)

// PresenceData 在线状态数据
type PresenceData struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Status    string    `json:"status"` // online/offline
	LastSeen  time.Time `json:"last_seen"`
	Connected bool      `json:"connected"` // 是否有活跃连接
}

// 在线状态相关常量
const (
	PresenceKeyPrefix = "im:presence:user:" // 用户在线状态key前缀
	OnlineUsersKey    = "im:online:users"   // 在线用户集合key
	PresenceTTL       = 2 * time.Minute     // 在线状态TTL（2倍心跳周期）
)

// SetUserPresence 设置用户在线状态
func SetUserPresence(userID uint, username string, status string) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", PresenceKeyPrefix, userID)

	presence := PresenceData{
		UserID:    userID,
		Username:  username,
		Status:    status,
		LastSeen:  time.Now(),
		Connected: status == "online",
	}

	data, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("序列化在线状态失败: %w", err)
	}

	// 设置用户状态，带TTL
	err = Set(key, data, PresenceTTL)
	if err != nil {
		return fmt.Errorf("设置用户在线状态失败: %w", err)
	}

	// 更新在线用户集合
	if status == "online" {
		err = client.SAdd(ctx, OnlineUsersKey, userID).Err()
	} else {
		err = client.SRem(ctx, OnlineUsersKey, userID).Err()
	}

	if err != nil {
		return fmt.Errorf("更新在线用户集合失败: %w", err)
	}

	return nil
}

// GetUserPresence 获取用户在线状态
func GetUserPresence(userID uint) (*PresenceData, error) {
	key := fmt.Sprintf("%s%d", PresenceKeyPrefix, userID)

	data, err := Get(key)
	if err != nil {
		return nil, fmt.Errorf("获取用户在线状态失败: %w", err)
	}

	var presence PresenceData
	err = json.Unmarshal([]byte(data), &presence)
	if err != nil {
		return nil, fmt.Errorf("反序列化在线状态失败: %w", err)
	}

	return &presence, nil
}

// IsUserOnline 检查用户是否在线
func IsUserOnline(userID uint) (bool, error) {
	key := fmt.Sprintf("%s%d", PresenceKeyPrefix, userID)

	exists, err := Exists(key)
	if err != nil {
		return false, fmt.Errorf("检查用户在线状态失败: %w", err)
	}

	return exists > 0, nil
}

// GetOnlineUsers 获取所有在线用户ID列表
func GetOnlineUsers() ([]uint, error) {
	members, err := client.SMembers(ctx, OnlineUsersKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取在线用户列表失败: %w", err)
	}

	var userIDs []uint
	for _, member := range members {
		var userID uint
		if _, err := fmt.Sscanf(member, "%d", &userID); err == nil {
			userIDs = append(userIDs, userID)
		}
	}

	return userIDs, nil
}

// GetOnlineUsersWithDetails 获取在线用户详细信息
func GetOnlineUsersWithDetails() ([]PresenceData, error) {
	if client == nil {
		return nil, fmt.Errorf("redis客户端未初始化")
	}

	userIDs, err := GetOnlineUsers()
	if err != nil {
		return nil, err
	}

	var presences []PresenceData
	for _, userID := range userIDs {
		presence, err := GetUserPresence(userID)
		if err != nil {
			// 如果获取失败，可能是TTL过期，从集合中移除
			client.SRem(ctx, OnlineUsersKey, userID)
			continue
		}
		presences = append(presences, *presence)
	}

	return presences, nil
}

// RefreshUserPresence 刷新用户在线状态（延长TTL）
func RefreshUserPresence(userID uint) error {
	key := fmt.Sprintf("%s%d", PresenceKeyPrefix, userID)

	// 检查key是否存在
	exists, err := Exists(key)
	if err != nil {
		return fmt.Errorf("检查用户状态失败: %w", err)
	}

	if exists == 0 {
		return fmt.Errorf("用户不在线")
	}

	// 延长TTL
	err = Expire(key, PresenceTTL)
	if err != nil {
		return fmt.Errorf("刷新用户在线状态失败: %w", err)
	}

	return nil
}

// RemoveUserPresence 移除用户在线状态
func RemoveUserPresence(userID uint) error {
	key := fmt.Sprintf("%s%d", PresenceKeyPrefix, userID)

	// 删除用户状态
	err := Del(key)
	if err != nil {
		return fmt.Errorf("删除用户在线状态失败: %w", err)
	}

	// 从在线用户集合中移除
	err = client.SRem(ctx, OnlineUsersKey, userID).Err()
	if err != nil {
		return fmt.Errorf("从在线用户集合移除失败: %w", err)
	}

	return nil
}

// CleanExpiredPresence 清理过期的在线状态（定期任务）
func CleanExpiredPresence() error {
	// 获取所有在线用户
	userIDs, err := GetOnlineUsers()
	if err != nil {
		return err
	}

	// 检查每个用户的状态是否过期
	for _, userID := range userIDs {
		key := fmt.Sprintf("%s%d", PresenceKeyPrefix, userID)
		ttl, err := TTL(key)
		if err != nil {
			continue
		}

		// 如果TTL为-2（key不存在）或-1（无过期时间），从集合中移除
		if ttl == -2 || ttl == -1 {
			client.SRem(ctx, OnlineUsersKey, userID)
		}
	}

	return nil
}
