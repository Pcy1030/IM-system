package model

import (
	"time"

	"gorm.io/gorm"
)

// Friendship 好友关系
// Status: pending/accepted/blocked

type Friendship struct {
	ID        uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"not null;index;comment:用户ID"`
	FriendID  uint           `gorm:"not null;index;comment:好友ID"`
	Status    string         `gorm:"type:varchar(32);default:'pending';comment:关系状态"`
	CreatedAt time.Time      `gorm:"comment:创建时间"`
	UpdatedAt time.Time      `gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Friendship) TableName() string { return "friendship" }
