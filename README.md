# IM System - 即时通讯系统

一个基于 Go 语言开发的即时通讯系统，支持用户注册登录、私聊、实时消息推送等功能。
自己开发的第一个Golang项目，后续的功能还在持续开发中……

## 🚀 功能特性

- **用户管理**: 用户注册、登录
- **实时通讯**: 基于 WebSocket 的实时消息推送
- **好友系统**: 好友添加、删除、好友列表管理等（暂未实现）
- **消息系统**: 私聊消息、消息历史记录
- **安全认证**: JWT 令牌认证，密码加密存储
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

## 📋 系统要求

- Go 1.24.1 或更高版本
- MySQL 8.0 或更高版本
- Git

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/你的用户名/IM-system.git
cd IM-system
```

### 2. 配置环境

复制环境配置文件：

```bash
cp config/env.example config/config.yaml
```

编辑 `config/config.yaml` 文件，配置数据库连接信息：

```yaml
server:
  port: ":8080"

database:
  host: "localhost"
  port: 3306
  username: "your_username"
  password: "your_password"
  database: "im_system"

jwt:
  secret: "your_jwt_secret_key"
  expire_time: "24h"

log:
  level: "info"
  filename: "logs/app.log"
  max_size: 100
  max_age: 30
  max_backups: 10
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
- `GET /api/v1/users/profile` - 获取个人资料
- `PUT /api/v1/users/profile` - 更新个人资料

#### 好友管理

- `POST /api/v1/friendships/request` - 发送好友请求
- `GET /api/v1/friendships/list` - 获取好友列表
- `PUT /api/v1/friendships/accept/:id` - 接受好友请求
- `DELETE /api/v1/friendships/:id` - 删除好友

#### 消息系统

- `GET /api/v1/messages/history/:friend_id` - 获取聊天历史
- `POST /api/v1/messages/send` - 发送消息

#### WebSocket

- `WS /ws` - WebSocket 连接（需要 JWT 认证）

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
