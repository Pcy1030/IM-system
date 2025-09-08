package redis

import (
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 未读消息计数相关常量
const (
	UnreadCountKeyPrefix = "im:unread:" // 未读消息计数key前缀
)

// IncrementUnreadCount 增加用户未读消息计数
func IncrementUnreadCount(userID uint) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)

	// 使用Redis INCR命令原子性增加计数
	err := client.Incr(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("增加未读消息计数失败: %w", err)
	}

	// 设置TTL，避免计数无限增长（24小时过期）
	err = client.Expire(ctx, key, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("设置未读消息计数TTL失败: %w", err)
	}

	return nil
}

// DecrementUnreadCount 减少用户未读消息计数
func DecrementUnreadCount(userID uint) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)

	// 使用Redis DECR命令原子性减少计数
	err := client.Decr(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("减少未读消息计数失败: %w", err)
	}

	// 如果计数为0或负数，删除key
	count, err := client.Get(ctx, key).Int64()
	if err == nil && count <= 0 {
		client.Del(ctx, key)
	}

	return nil
}

// GetUnreadCount 获取用户未读消息计数
func GetUnreadCount(userID uint) (int64, error) {
	if client == nil {
		return 0, fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)

	// 从Redis获取计数
	result, err := client.Get(ctx, key).Result()
	if err != nil {
		// 如果key不存在，返回-1表示需要从数据库获取
		if err.Error() == "redis: nil" {
			return -1, nil
		}
		return 0, fmt.Errorf("获取未读消息计数失败: %w", err)
	}

	// 转换为int64
	count, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析未读消息计数失败: %w", err)
	}

	return count, nil
}

// SetUnreadCount 设置用户未读消息计数（用于初始化或重置）
func SetUnreadCount(userID uint, count int64) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)

	// 设置计数
	err := client.Set(ctx, key, count, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("设置未读消息计数失败: %w", err)
	}

	return nil
}

// ResetUnreadCount 重置用户未读消息计数为0
func ResetUnreadCount(userID uint) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)

	// 删除key，相当于重置为0
	err := client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("重置未读消息计数失败: %w", err)
	}

	return nil
}

// BatchIncrementUnreadCount 批量增加用户未读消息计数
func BatchIncrementUnreadCount(userIDs []uint, count int64) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	// 使用Pipeline批量操作
	pipe := client.Pipeline()

	for _, userID := range userIDs {
		key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)
		pipe.IncrBy(ctx, key, count)
		pipe.Expire(ctx, key, 24*time.Hour)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("批量增加未读消息计数失败: %w", err)
	}

	return nil
}

// BatchDecrementUnreadCount 批量减少用户未读消息计数
func BatchDecrementUnreadCount(userIDs []uint, count int64) error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	// 使用Pipeline批量操作
	pipe := client.Pipeline()

	for _, userID := range userIDs {
		key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)
		pipe.DecrBy(ctx, key, count)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("批量减少未读消息计数失败: %w", err)
	}

	// 检查并删除计数为0或负数的key
	for _, userID := range userIDs {
		key := fmt.Sprintf("%s%d", UnreadCountKeyPrefix, userID)
		count, err := client.Get(ctx, key).Int64()
		if err == nil && count <= 0 {
			client.Del(ctx, key)
		}
	}

	return nil
}

// GetAllUnreadCounts 获取所有用户的未读消息计数（用于管理后台）
func GetAllUnreadCounts() (map[uint]int64, error) {
	if client == nil {
		return nil, fmt.Errorf("redis客户端未初始化")
	}

	// 使用 SCAN 非阻塞地遍历所有未读计数 key
	var keys []string
	var cursor uint64
	pattern := fmt.Sprintf("%s*", UnreadCountKeyPrefix)
	for {
		ks, c, err := client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("获取未读计数key失败: %w", err)
		}
		keys = append(keys, ks...)
		cursor = c
		if cursor == 0 {
			break
		}
	}

	result := make(map[uint]int64)

	// 批量获取所有计数
	if len(keys) > 0 {
		pipe := client.Pipeline()
		cmds := make(map[string]*redis.StringCmd)

		for _, key := range keys {
			cmds[key] = pipe.Get(ctx, key)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("批量获取未读计数失败: %w", err)
		}

		// 解析结果
		for key, cmd := range cmds {
			val, err := cmd.Result()
			if err == nil {
				count, err := strconv.ParseInt(val, 10, 64)
				if err == nil {
					// 从key中提取userID
					userIDStr := key[len(UnreadCountKeyPrefix):]
					if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
						result[uint(userID)] = count
					}
				}
			}
		}
	}

	return result, nil
}
