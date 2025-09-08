package redis

import (
	"context"
	"fmt"
	"time"

	"im-system/config"

	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
	ctx    = context.Background()
)

// InitRedis 初始化Redis连接
func InitRedis(cfg config.RedisConfig) error {
	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		// 连接池配置
		PoolSize:     10,              // 连接池大小
		MinIdleConns: 5,               // 最小空闲连接
		MaxRetries:   3,               // 最大重试次数
		DialTimeout:  5 * time.Second, // 连接超时
		ReadTimeout:  3 * time.Second, // 读超时
		WriteTimeout: 3 * time.Second, // 写超时
	})

	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis连接失败: %w", err)
	}

	return nil
}

// GetClient 获取Redis客户端
func GetClient() *redis.Client {
	return client
}

// GetContext 获取Redis上下文
func GetContext() context.Context {
	return ctx
}

// Close 关闭Redis连接
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// HealthCheck 检查Redis健康状态
func HealthCheck() error {
	if client == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis连接异常: %w", err)
	}

	return nil
}

// Set 设置键值对
func Set(key string, value interface{}, expiration time.Duration) error {
	return client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func Get(key string) (string, error) {
	return client.Get(ctx, key).Result()
}

// Del 删除键
func Del(keys ...string) error {
	return client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(keys ...string) (int64, error) {
	return client.Exists(ctx, keys...).Result()
}

// Incr 递增
func Incr(key string) (int64, error) {
	return client.Incr(ctx, key).Result()
}

// Decr 递减
func Decr(key string) (int64, error) {
	return client.Decr(ctx, key).Result()
}

// Expire 设置过期时间
func Expire(key string, expiration time.Duration) error {
	return client.Expire(ctx, key, expiration).Err()
}

// TTL 获取剩余过期时间
func TTL(key string) (time.Duration, error) {
	return client.TTL(ctx, key).Result()
}
