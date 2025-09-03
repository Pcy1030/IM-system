package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构体
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	JWT       JWTConfig       `yaml:"jwt"`
	Log       LogConfig       `yaml:"log"`
	Redis     RedisConfig     `yaml:"redis"`
	WebSocket WebSocketConfig `yaml:"websocket"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         string        `yaml:"port"`         // 服务器监听端口
	ReadTimeout  time.Duration `yaml:"readTimeout"`  // 读取超时时间
	WriteTimeout time.Duration `yaml:"writeTimeout"` // 写入超时时间
	IdleTimeout  time.Duration `yaml:"idleTimeout"`  // 空闲超时时间
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `yaml:"driver"`   // 数据库驱动类型
	Host     string `yaml:"host"`     // 数据库主机地址
	Port     int    `yaml:"port"`     // 数据库端口
	Username string `yaml:"username"` // 数据库用户名
	Password string `yaml:"password"` // 数据库密码
	Database string `yaml:"database"` // 数据库名称
	Charset  string `yaml:"charset"`  // 字符集
	MaxIdle  int    `yaml:"maxIdle"`  // 最大空闲连接数
	MaxOpen  int    `yaml:"maxOpen"`  // 最大打开连接数
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string        `yaml:"secret"`     // JWT密钥
	ExpireTime time.Duration `yaml:"expireTime"` // JWT过期时间
	Issuer     string        `yaml:"issuer"`     // JWT签发者
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`      // 日志级别
	Filename   string `yaml:"filename"`   // 日志文件名
	MaxSize    int    `yaml:"maxSize"`    // 单个日志文件最大大小(MB)
	MaxBackups int    `yaml:"maxBackups"` // 最大备份文件数
	MaxAge     int    `yaml:"maxAge"`     // 最大保存天数
	Compress   bool   `yaml:"compress"`   // 是否压缩
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`     // Redis主机地址
	Port     int    `yaml:"port"`     // Redis端口
	Password string `yaml:"password"` // Redis密码
	DB       int    `yaml:"db"`       // Redis数据库编号
}

// WebSocketConfig WebSocket 心跳配置
type WebSocketConfig struct {
	PingInterval time.Duration `yaml:"pingInterval"` // 发送ping的间隔
	ReadTimeout  time.Duration `yaml:"readTimeout"`  // 读超时时间（未收到任何数据则断开）
}

// LoadConfig 加载配置（混合方式：YAML文件 + 环境变量）
func LoadConfig() *Config {
	// 1. 首先从YAML文件加载默认配置
	config := loadFromYAML("config/config.yaml")

	// 2. 用环境变量覆盖配置（环境变量优先级更高）
	overrideWithEnvVars(config)

	return config
}

// loadFromYAML 从YAML文件加载配置
func loadFromYAML(filePath string) *Config {
	// 读取配置文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		// 如果文件不存在，返回默认配置
		return getDefaultConfig()
	}

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		// 如果解析失败，返回默认配置
		return getDefaultConfig()
	}

	return &config
}

// overrideWithEnvVars 用环境变量覆盖配置
func overrideWithEnvVars(config *Config) {
	// 服务器配置
	if port := getEnv("SERVER_PORT", ""); port != "" {
		config.Server.Port = port
	}
	if timeout := getEnvDuration("SERVER_READ_TIMEOUT", 0); timeout > 0 {
		config.Server.ReadTimeout = timeout
	}
	if timeout := getEnvDuration("SERVER_WRITE_TIMEOUT", 0); timeout > 0 {
		config.Server.WriteTimeout = timeout
	}
	if timeout := getEnvDuration("SERVER_IDLE_TIMEOUT", 0); timeout > 0 {
		config.Server.IdleTimeout = timeout
	}

	// 数据库配置
	if host := getEnv("DB_HOST", ""); host != "" {
		config.Database.Host = host
	}
	if port := getEnvInt("DB_PORT", 0); port > 0 {
		config.Database.Port = port
	}
	if username := getEnv("DB_USERNAME", ""); username != "" {
		config.Database.Username = username
	}
	if password := getEnv("DB_PASSWORD", ""); password != "" {
		config.Database.Password = password
	}
	if database := getEnv("DB_DATABASE", ""); database != "" {
		config.Database.Database = database
	}
	if charset := getEnv("DB_CHARSET", ""); charset != "" {
		config.Database.Charset = charset
	}
	if maxIdle := getEnvInt("DB_MAX_IDLE", 0); maxIdle > 0 {
		config.Database.MaxIdle = maxIdle
	}
	if maxOpen := getEnvInt("DB_MAX_OPEN", 0); maxOpen > 0 {
		config.Database.MaxOpen = maxOpen
	}

	// JWT配置
	if secret := getEnv("JWT_SECRET", ""); secret != "" {
		config.JWT.Secret = secret
	}
	if expireTime := getEnvDuration("JWT_EXPIRE_TIME", 0); expireTime > 0 {
		config.JWT.ExpireTime = expireTime
	}
	if issuer := getEnv("JWT_ISSUER", ""); issuer != "" {
		config.JWT.Issuer = issuer
	}

	// 日志配置
	if level := getEnv("LOG_LEVEL", ""); level != "" {
		config.Log.Level = level
	}
	if filename := getEnv("LOG_FILENAME", ""); filename != "" {
		config.Log.Filename = filename
	}
	if maxSize := getEnvInt("LOG_MAX_SIZE", 0); maxSize > 0 {
		config.Log.MaxSize = maxSize
	}
	if maxBackups := getEnvInt("LOG_MAX_BACKUPS", 0); maxBackups > 0 {
		config.Log.MaxBackups = maxBackups
	}
	if maxAge := getEnvInt("LOG_MAX_AGE", 0); maxAge > 0 {
		config.Log.MaxAge = maxAge
	}

	// Redis配置
	if host := getEnv("REDIS_HOST", ""); host != "" {
		config.Redis.Host = host
	}
	if port := getEnvInt("REDIS_PORT", 0); port > 0 {
		config.Redis.Port = port
	}
	if password := getEnv("REDIS_PASSWORD", ""); password != "" {
		config.Redis.Password = password
	}
	if db := getEnvInt("REDIS_DB", -1); db >= 0 {
		config.Redis.DB = db
	}

	// WebSocket配置
	if d := getEnvDuration("WS_PING_INTERVAL", 0); d > 0 {
		config.WebSocket.PingInterval = d
	}
	if d := getEnvDuration("WS_READ_TIMEOUT", 0); d > 0 {
		config.WebSocket.ReadTimeout = d
	}
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         "8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "im_user",
			Password: "Pcy010728.",
			Database: "im_system",
			Charset:  "utf8mb4",
			MaxIdle:  10,
			MaxOpen:  100,
		},
		JWT: JWTConfig{
			Secret:     "your-secret-key",
			ExpireTime: 24 * time.Hour,
			Issuer:     "im-system",
		},
		Log: LogConfig{
			Level:      "info",
			Filename:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		WebSocket: WebSocketConfig{
			PingInterval: 30 * time.Second,
			ReadTimeout:  90 * time.Second,
		},
	}
}

// 辅助函数：获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 辅助函数：获取整数环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// 辅助函数：获取布尔环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// 辅助函数：获取时间环境变量
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
