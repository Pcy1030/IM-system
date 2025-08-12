# IM System HTTP API 规范（v1）

- 基础路径: `/api/v1`
- 鉴权: 绝大多数接口需 JWT Bearer 令牌
  - Header: `Authorization: Bearer <access_token>`
- 响应通用格式:
```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```
- 错误码约定: `code != 0` 表示失败；`401` 未认证，`403` 无权限，`400` 参数错误，`500` 服务器错误
- 分页参数: `page` 从1开始，`pageSize` 默认20，最大100

---

## 1. 鉴权 Auth

### 1.1 注册
- POST `/api/v1/users/register`
- Body
```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "P@ssw0rd!"
}
```
- Response
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user": {
      "id": 1,
      "username": "alice",
      "nickname": "",
      "avatar": "",
      "status": "offline",
      "createdAt": "2025-08-11T08:00:00Z"
    },
    "accessToken": "<jwt>",
    "refreshToken": "<jwt_refresh>"
  }
}
```

### 1.2 登录
- POST `/api/v1/users/login`
- Body
```json
{
  "usernameOrEmail": "alice",
  "password": "P@ssw0rd!"
}
```
- Response 同注册，返回 `accessToken`、`refreshToken`

### 1.3 刷新令牌（可选）
- POST `/api/v1/auth/refresh`
- Body
```json
{ "refreshToken": "<jwt_refresh>" }
```
- Response: 新的 `accessToken`

### 1.4 登出（可选）
- POST `/api/v1/users/logout`
- Header 需携带 access token；服务端可作黑名单或前端删除本地令牌

---

## 2. 用户 Users

### 2.1 获取个人资料
- GET `/api/v1/users/profile`
- Header: `Authorization`
- Response
```json
{
  "code": 0,
  "data": { "id":1, "username":"alice", "nickname":"", "avatar":"", "status":"online" }
}
```

### 2.2 更新个人资料
- PUT `/api/v1/users/profile`
- Body（任意可选字段）
```json
{ "nickname": "Alice", "avatar": "https://.../a.png" }
```

### 2.3 用户列表（搜索/分页）
- GET `/api/v1/users/list?keyword=&page=1&pageSize=20`
- Response: `users: []`, `total`

---

## 3. 消息 Messages

### 3.1 发送消息
- POST `/api/v1/messages/send`
- Body（单聊与群聊二选一）
```json
{
  "toUserId": 2,
  "groupId": 0,
  "content": "hello",
  "msgType": "text"
}
```
- Response: 返回消息对象（含服务器生成的 id、时间等）

### 3.2 历史消息
- GET `/api/v1/messages/history?withUserId=2&groupId=0&beforeId=&limit=20`
- 说明: `withUserId` 用于单聊，`groupId` 用于群聊，二者选其一；支持 `beforeId` 上拉分页

### 3.3 未读数
- GET `/api/v1/messages/unread?withUserId=2`
- Response: `{ count: 3 }`

### 3.4 标记已读（可选）
- POST `/api/v1/messages/read`
- Body
```json
{ "withUserId": 2, "messageIds": [101,102,103] }
```

---

## 4. 好友关系 Friendships（模型已定义，接口待实现）

### 4.1 发送好友请求
- POST `/api/v1/friends/request`
- Body: `{ "toUserId": 2 }`

### 4.2 接受好友请求
- POST `/api/v1/friends/accept`
- Body: `{ "requestId": 10 }`

### 4.3 好友列表
- GET `/api/v1/friends/list`

---

## 5. WebSocket 实时通道
- 路径: `GET /api/v1/ws`
- 鉴权: 建议使用 `Sec-WebSocket-Protocol: Bearer,<token>` 或 URL 参数 `?token=...`
- 事件格式:
```json
{
  "type": "message|read|typing|presence",
  "payload": { /* 不同事件的负载 */ }
}
```
- 典型事件:
  - `message` 下发新消息
  - `read` 已读回执
  - `typing` 正在输入
  - `presence` 上下线状态

- 客户端发送示例（发送消息）:
```json
{
  "type": "message",
  "payload": { "toUserId":2, "content":"hi", "msgType":"text" }
}
```

---

## 6. 附件上传（功能待实现）
- POST `/api/v1/attachments/upload`
- multipart/form-data: `file`
- Response: `{ "url": "https://..." }`

---

## 7. 其他

### 7.1 健康检查
- GET `/health`
- Response: `{ "status":"ok" }`（当前已实现，并包含 DB 健康）

### 7.2 安全约定
- 密码存储使用 `bcrypt` 哈希
- JWT: `issuer=im-system`，`exp` 由配置控制

### 7.3 速率限制（功能待实现）
- 响应头可增加 `X-RateLimit-*`（功能待实现）

---

## 8. 开发状态
1) 用户注册/登录（bcrypt + JWT）✅ 已完成
2) WebSocket 连接与心跳 ✅ 已完成
3) 发送/接收消息（HTTP + WS 双通道）✅ 已完成
4) 未读、已读、历史分页 ✅ 已完成
5) 好友/群组等社交要素（按需）🔄 模型已定义，接口待实现
