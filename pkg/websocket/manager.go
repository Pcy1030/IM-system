package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"im-system/pkg/redis"

	"github.com/gorilla/websocket"
)

// Client 代表一个WebSocket连接的用户
// UserID: 用户ID
// Conn: WebSocket连接
// Send: 发送消息的通道

type Client struct {
	UserID uint
	Conn   *websocket.Conn
	Send   chan []byte
}

// Manager 管理所有在线用户的WebSocket连接
// 支持并发安全、Redis离线消息存储

type Manager struct {
	clients map[uint]*Client // 在线用户
	lock    sync.RWMutex
}

var manager = &Manager{
	clients: make(map[uint]*Client),
}

// GetManager 获取全局WebSocket管理器
func GetManager() *Manager {
	return manager
}

// AddClient 添加新连接
func (m *Manager) AddClient(userID uint, client *Client) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.clients[userID] = client

	// 推送Redis中的离线消息
	go m.pushOfflineMessages(userID, client)
}

// RemoveClient 移除连接
func (m *Manager) RemoveClient(userID uint) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok := m.clients[userID]; ok {
		close(c.Send)
		delete(m.clients, userID)
	}
}

// SendToUser 推送消息给指定用户
// 若用户不在线则存储到Redis离线消息
func (m *Manager) SendToUser(userID uint, msg []byte) {
	m.lock.RLock()
	client, ok := m.clients[userID]
	m.lock.RUnlock()
	if ok {
		// 在线，直接推送
		select {
		case client.Send <- msg:
		default:
			// 发送失败，可能连接已断开
		}
	} else {
		// 不在线，存储到Redis离线消息
		go m.storeOfflineMessage(userID, msg)
	}
}

// IsOnline 判断用户是否在线
func (m *Manager) IsOnline(userID uint) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.clients[userID]
	return ok
}

// pushOfflineMessages 推送离线消息给用户
func (m *Manager) pushOfflineMessages(userID uint, client *Client) {
	// 从Redis获取离线消息
	offlineMessages, err := redis.GetOfflineMessages(userID, 50) // 最多推送50条
	if err != nil {
		return
	}

	// 推送离线消息
	for _, msg := range offlineMessages {
		msgData, err := json.Marshal(map[string]interface{}{
			"type":       "offline_message",
			"id":         msg.ID,
			"sender_id":  msg.SenderID,
			"content":    msg.Content,
			"created_at": msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
		if err != nil {
			continue
		}

		select {
		case client.Send <- msgData:
		case <-time.After(5 * time.Second):
			// 发送超时，停止推送
			return
		}
	}

	// 推送完成后清空离线消息
	_ = redis.ClearOfflineMessages(userID)
}

// storeOfflineMessage 存储离线消息到Redis
func (m *Manager) storeOfflineMessage(userID uint, msgData []byte) {
	// 解析消息数据
	var msg map[string]interface{}
	err := json.Unmarshal(msgData, &msg)
	if err != nil {
		return
	}

	// 构建离线消息对象
	offlineMsg := &redis.OfflineMessage{
		SenderID:   uint(msg["from"].(float64)),
		ReceiverID: userID,
		Content:    msg["content"].(string),
		Type:       msg["type"].(string),
		CreatedAt:  time.Now(),
	}

	// 存储到Redis
	_ = redis.AddOfflineMessage(userID, offlineMsg)
}
