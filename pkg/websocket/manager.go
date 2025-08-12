package websocket

import (
	"sync"

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
// 支持并发安全、离线消息缓存

type Manager struct {
	clients     map[uint]*Client  // 在线用户
	offlineMsgs map[uint][][]byte // 离线消息缓存
	lock        sync.RWMutex
}

var manager = &Manager{
	clients:     make(map[uint]*Client),
	offlineMsgs: make(map[uint][][]byte),
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

	// 有离线消息则推送
	if msgs, ok := m.offlineMsgs[userID]; ok && len(msgs) > 0 {
		for _, msg := range msgs {
			client.Send <- msg
		}
		delete(m.offlineMsgs, userID)
	}
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
// 若用户不在线则缓存为离线消息
func (m *Manager) SendToUser(userID uint, msg []byte) {
	m.lock.RLock()
	client, ok := m.clients[userID]
	m.lock.RUnlock()
	if ok {
		// 在线，直接推送
		client.Send <- msg
	} else {
		// 不在线，缓存离线消息
		m.lock.Lock()
		m.offlineMsgs[userID] = append(m.offlineMsgs[userID], msg)
		m.lock.Unlock()
	}
}

// IsOnline 判断用户是否在线
func (m *Manager) IsOnline(userID uint) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.clients[userID]
	return ok
}
