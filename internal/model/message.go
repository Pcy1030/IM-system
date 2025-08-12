package model

import (
	"time"

	"gorm.io/gorm"
)

// Message 消息模型
// SessionType: 1-单聊 2-群聊
// Status: sent/delivered/read

type Message struct {
	ID          uint           `gorm:"primaryKey"`
	SessionType int            `gorm:"type:int;not null;default:1;comment:会话类型(1单聊,2群聊)"`
	SenderID    uint           `gorm:"not null;index;comment:发送者ID"`
	ReceiverID  uint           `gorm:"index;comment:接收者ID(单聊)"`
	GroupID     *uint          `gorm:"index;comment:群ID(群聊)"`
	Content     string         `gorm:"type:text;not null;comment:消息内容"`
	MsgType     string         `gorm:"type:varchar(32);default:'text';comment:消息类型"`
	Status      string         `gorm:"type:varchar(32);default:'sent';comment:消息状态"`
	IsRead      bool           `gorm:"default:false;comment:是否已读"`
	CreatedAt   time.Time      `gorm:"comment:创建时间"`
	UpdatedAt   time.Time      `gorm:"comment:更新时间"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Message) TableName() string { return "message" }
