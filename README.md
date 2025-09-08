# IM System - 即时通讯系统

一个基于 Go 语言开发的即时通讯系统，支持用户注册登录、私聊、实时消息推送等功能。
自己开发的第一个Golang项目，后续的功能还在持续开发中……

## 🚀 功能特性

- **用户管理**: 用户注册、登录、登出、在线状态管理
- **实时通讯**: 基于 WebSocket 的实时消息推送，支持心跳保活
- **消息系统**: 私聊消息、消息历史记录、未读消息管理、已读回执
- **在线状态**: 实时在线/离线状态，自动心跳检测（Redis 持久化在线状态）
- **Redis 缓存**: 私聊消息缓存（最近 N 条）、会话列表缓存、未读计数缓存
- **离线消息**: 离线消息入 Redis（多实例可用），上线自动推送并可查询/清理
- **安全认证**: JWT 令牌认证，密码加密存储
- **配置管理**: YAML 配置文件，支持环境变量覆盖
- **日志系统**: 完整的日志记录和错误追踪
- **数据库**: MySQL 数据库支持，自动迁移表结构

## 🛠️ 技术栈

- **后端**: Go 1.24.1
- **Web框架**: Gin
- **数据库**: MySQL
- **ORM**: GORM
- **WebSocket**: Gorilla WebSocket
- **认证**: JWT
- **日志**: Zap + Lumberjack
- **配置管理**: YAML
 - **缓存/队列**: Redis（go-redis v9）

## 📋 系统要求

- Go 1.24.1 或更高版本
- MySQL 8.0 或更高版本
 - Redis 6.0 或更高版本（本地或远程）
- Git

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/Pcy1030/IM-system.git
cd IM-system
```

### 2. 配置环境

复制环境配置文件：

```bash
cp config/env.example config/config.yaml
```

编辑 `config/config.yaml` 文件，配置数据库、Redis 与缓存信息：

```yaml
server:
  port: ":8080"
  readTimeout: "30s"
  writeTimeout: "30s"
  idleTimeout: "60s"

database:
  host: "localhost"
  port: 3306
  username: "your_username"
  password: "your_password"
  database: "im_system"
  charset: "utf8mb4"
  maxIdle: 10
  maxOpen: 100

jwt:
  secret: "your_jwt_secret_key"
  expireTime: "24h"
  issuer: "im-system"

log:
  level: "info"
  filename: "logs/app.log"
  maxSize: 100
  maxAge: 7
  maxBackups: 3
  compress: true

websocket:
  pingInterval: "30s"    # 服务器发送 ping 的间隔
  readTimeout: "90s"     # 读超时时间（未收到任何数据则断开）

redis:
  host: "127.0.0.1"
  port: 6379
  password: ""           # 如无密码留空
  db: 0

cache:
  enabled: true
  messageTTL: "1h"        # 消息/会话缓存 TTL
  maxCachedMessages: 30    # 每个会话缓存最近 N 条
  maxCachedConversations: 10 # 最近会话列表数量
```

### 环境变量配置（可选）

你也可以通过环境变量覆盖配置：

```bash
# 服务器配置
export SERVER_PORT=8080
export SERVER_READ_TIMEOUT=30s
export SERVER_WRITE_TIMEOUT=30s
export SERVER_IDLE_TIMEOUT=60s

# 数据库配置
export DB_HOST=localhost
export DB_PORT=3306
export DB_USERNAME=your_username
export DB_PASSWORD=your_password
export DB_DATABASE=im_system

# JWT配置
export JWT_SECRET=your_jwt_secret_key
export JWT_EXPIRE_TIME=24h
export JWT_ISSUER=im-system

# WebSocket配置
export WS_PING_INTERVAL=30s
export WS_READ_TIMEOUT=90s

# Redis 配置
export REDIS_HOST=127.0.0.1
export REDIS_PORT=6379
export REDIS_PASSWORD=
export REDIS_DB=0

# 缓存配置
export CACHE_ENABLED=true
export CACHE_MESSAGE_TTL=1h
export CACHE_MAX_CACHED_MESSAGES=30
export CACHE_MAX_CACHED_CONVERSATIONS=10
```

### 3. 创建数据库

在 MySQL 中创建数据库：

```sql
CREATE DATABASE im_system CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 4. 安装依赖

```bash
go mod tidy
```

### 5. 运行项目

```bash
# 开发模式
go run cmd/server/main.go

# 或者编译后运行
go build -o main.exe cmd/server/main.go
./main.exe
```

服务器将在 `http://localhost:8080` 启动。

## 📚 API 文档

### 基础信息

- **基础路径**: `/api/v1`
- **认证方式**: JWT Bearer Token
- **请求头**: `Authorization: Bearer <access_token>`

### 主要接口

#### 用户认证

- `POST /api/v1/users/register` - 用户注册
- `POST /api/v1/users/login` - 用户登录
- `POST /api/v1/users/logout` - 用户登出（置为离线状态）
- `GET /api/v1/users/profile` - 获取个人资料
- `GET /api/v1/users/test-auth` - 测试JWT认证

#### 消息系统

- `POST /api/v1/messages/send` - 发送消息
- `GET /api/v1/messages/unread` - 获取未读消息
- `GET /api/v1/messages/unread/count` - 获取未读消息数量
- `PUT /api/v1/messages/:message_id/read` - 标记消息为已读
- `DELETE /api/v1/messages/:message_id` - 删除消息
- `GET /api/v1/messages/conversations` - 获取最近对话

#### 私聊历史

- `GET /api/v1/conversations/:user_id/messages` - 获取与指定用户的私聊消息

#### WebSocket

- `WS /ws` - WebSocket 连接（需要 JWT 认证）
  - 支持查询参数：`?token=YOUR_JWT`
  - 支持子协议头：`Sec-WebSocket-Protocol: Bearer YOUR_JWT`
  - 自动心跳保活（30s ping，90s 超时）
  - 支持已读回执：`{"type":"ack_read","msg_id":123}`
  - 支持应用层心跳：`{"type":"heartbeat"}`

详细的 API 文档请参考 [api/http_api.md](api/http_api.md)

## 🏗️ 项目结构

```
IM-system/
├── api/                    # API 文档
├── cmd/                    # 应用程序入口
│   └── server/
│       └── main.go        # 主程序
├── config/                 # 配置文件
├── internal/               # 内部包
│   ├── handler/           # HTTP 处理器
│   ├── model/             # 数据模型
│   ├── repository/        # 数据访问层
│   └── service/           # 业务逻辑层
├── pkg/                    # 公共包
│   ├── db/                # 数据库连接
│   ├── jwt/               # JWT 认证
│   ├── logger/            # 日志系统
│   ├── password/          # 密码处理
│   ├── response/          # 响应处理
│   └── websocket/         # WebSocket 管理
├── web/                    # 前端静态文件
├── logs/                   # 日志文件
└── tools/                  # 工具脚本
```

## 🔧 开发

### 运行测试

```bash
go test ./...
```

### 代码格式化

```bash
go fmt ./...
```

### 代码检查

```bash
go vet ./...
```

## 📝 日志

日志文件位于 `logs/app.log`，支持日志轮转和级别控制。

## 🔧 WebSocket 使用说明

### 连接方式

1. **查询参数方式**（推荐）：
   ```
   ws://localhost:8080/ws?token=YOUR_JWT_TOKEN
   ```

2. **子协议头方式**：
   ```
   ws://localhost:8080/ws
   Headers: Sec-WebSocket-Protocol: Bearer YOUR_JWT_TOKEN
   ```

### 心跳机制

- **服务器心跳**：每30秒自动发送ping，客户端应回复pong
- **读超时**：90秒内未收到任何数据则断开连接
- **应用层心跳**：客户端可发送 `{"type":"heartbeat"}` 更新在线状态

### 消息格式

#### 接收消息
```json
{
  "type": "chat",
  "from": 123,
  "to": 456,
  "content": "Hello!",
  "msg_id": 789,
  "timestamp": 1640995200
}
```

#### 发送消息
```json
// 已读回执
{"type": "ack_read", "msg_id": 123}

// 应用层心跳
{"type": "heartbeat"}
```

### 在线状态管理

- **登录成功**：自动设置为 `online`
- **WebSocket连接**：设置为 `online`
- **WebSocket断开**：设置为 `offline`
- **登出接口**：设置为 `offline`
- **心跳超时**：自动断开并设置为 `offline`

## 🧪 测试

### Postman 测试 WebSocket

1. 创建 WebSocket 请求
2. URL: `ws://localhost:8080/ws?token=YOUR_JWT`
3. 连接成功后可以：
   - 接收实时消息推送
   - 发送已读回执：`{"type":"ack_read","msg_id":123}`
   - 发送心跳：`{"type":"heartbeat"}`

### 测试用户状态

```bash
# 查看用户在线状态
SELECT username, status, last_seen FROM user;

# 查看未读消息
SELECT id, sender_id, receiver_id, content, is_read FROM message WHERE is_read = 0;
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 Issue
- 发送邮件至：[3072230687@qq.com]

---

⭐ 如果这个项目对你有帮助，请给它一个星标！
