package db

import (
	"fmt"
	"time"

	"im-system/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	// 构建DSN连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)

	// 配置GORM
	gormConfig := &gorm.Config{
		// 日志配置
		Logger: logger.Default.LogMode(logger.Info), // 开发阶段显示SQL日志

		// 禁用默认事务（提高性能）
		SkipDefaultTransaction: true,

		// 准备语句（提高性能）
		PrepareStmt: true,

		// 命名策略
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 获取底层的sql.DB对象
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)  // 最大空闲连接数
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)  // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 保存全局数据库实例
	DB = db

	return db, nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return fmt.Errorf("获取数据库实例失败: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}

// HealthCheck 数据库健康检查
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	return sqlDB.Ping()
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(models ...interface{}) error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	return DB.AutoMigrate(models...)
}

// BeginTransaction 开始事务
func BeginTransaction() *gorm.DB {
	if DB == nil {
		return nil
	}
	return DB.Begin()
}

// CommitTransaction 提交事务
func CommitTransaction(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("事务为空")
	}
	return tx.Commit().Error
}

// RollbackTransaction 回滚事务
func RollbackTransaction(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("事务为空")
	}
	return tx.Rollback().Error
}
