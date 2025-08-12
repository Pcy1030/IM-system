package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
// 索引与唯一约束：用户名唯一、邮箱唯一
// 说明：密码仅存储哈希（PasswordHash），不存储明文
// 状态可用于标记用户在线/离线/禁用等
// LastSeen 用于最近在线时间
// gorm.Model 包含 ID、CreatedAt、UpdatedAt、DeletedAt

type User struct {
	ID           uint           `gorm:"primaryKey"`
	Username     string         `gorm:"type:varchar(64);not null;uniqueIndex;comment:用户名"`
	Email        string         `gorm:"type:varchar(128);uniqueIndex;comment:邮箱"`
	PasswordHash string         `gorm:"type:varchar(255);not null;comment:密码哈希"`
	Nickname     string         `gorm:"type:varchar(64);comment:昵称"`
	Avatar       string         `gorm:"type:varchar(255);comment:头像URL"`
	Status       string         `gorm:"type:varchar(32);default:'offline';comment:状态"`
	LastSeen     time.Time      `gorm:"comment:最近在线时间"`
	CreatedAt    time.Time      `gorm:"comment:创建时间"`
	UpdatedAt    time.Time      `gorm:"comment:更新时间"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名（因全局配置使用单数表名，这里与结构体名一致为 user）
func (User) TableName() string { return "user" }
