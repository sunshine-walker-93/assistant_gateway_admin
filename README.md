# Assistant Gateway Admin Service

网关管理服务，提供配置管理的 REST API，实现控制流和数据流的分离。

## 功能

- ✅ **后端服务管理**: 创建、查询、更新、删除后端服务配置
- ✅ **路由管理**: 创建、查询、更新、删除路由配置
- ✅ **配置历史**: 查询配置变更历史记录
- ✅ **软删除**: 使用 `enabled` 字段实现软删除
- ✅ **自动审计**: 所有配置变更自动记录到历史表

## 架构

本服务与网关数据转发服务（`assistant_gateway`）分离：
- **管理服务**（本服务）: 处理配置管理 API（控制流）
- **数据转发服务**: 处理 HTTP → gRPC 转发（数据流）

## 快速开始

### 环境变量

- `ADMIN_DB_DSN`: 数据库连接字符串（必需）
  - 格式: `user:password@tcp(host:port)/assistant_gateway_db?parseTime=true`
- `ADMIN_HTTP_LISTEN`: HTTP 监听地址（默认: `:8081`）

### 本地运行

```bash
# 设置环境变量
export ADMIN_DB_DSN="assistant:password@tcp(127.0.0.1:3306)/assistant_gateway_db?parseTime=true"

# 运行
go run ./cmd/admin
```

### 构建

```bash
make go-build
```

## API 文档

### 后端服务管理

#### 列出所有后端
```bash
GET /api/v1/backends?enabled=true
```

#### 获取单个后端
```bash
GET /api/v1/backends/{name}
```

#### 创建后端
```bash
POST /api/v1/backends
Content-Type: application/json

{
  "name": "account",
  "addr": "127.0.0.1:50051",
  "description": "Account service",
  "enabled": true
}
```

#### 更新后端
```bash
PUT /api/v1/backends/{name}
Content-Type: application/json

{
  "addr": "127.0.0.1:50052",
  "description": "Updated description",
  "enabled": true
}
```

#### 删除后端（软删除）
```bash
DELETE /api/v1/backends/{name}
```

### 路由管理

#### 列出所有路由
```bash
GET /api/v1/routes?enabled=true
```

#### 获取单个路由
```bash
GET /api/v1/routes/{id}
```

#### 创建路由
```bash
POST /api/v1/routes
Content-Type: application/json

{
  "http_method": "POST",
  "http_pattern": "/v1/user/login",
  "backend_name": "account",
  "backend_service": "user.v1.UserService",
  "backend_method": "Login",
  "timeout_ms": 5000,
  "description": "User login route",
  "enabled": true
}
```

#### 更新路由
```bash
PUT /api/v1/routes/{id}
Content-Type: application/json

{
  "http_method": "POST",
  "http_pattern": "/v1/user/login",
  "backend_name": "account",
  "backend_service": "user.v1.UserService",
  "backend_method": "Login",
  "timeout_ms": 3000,
  "enabled": true
}
```

#### 删除路由（软删除）
```bash
DELETE /api/v1/routes/{id}
```

### 配置历史

#### 查询配置变更历史
```bash
GET /api/v1/history?config_type=backend&config_id=1&limit=10&offset=0
```

查询参数：
- `config_type`: 配置类型（`backend` 或 `route`）
- `config_id`: 配置 ID
- `limit`: 每页数量（默认 50，最大 100）
- `offset`: 偏移量（默认 0）

### 健康检查

```bash
GET /health
```

## Docker 部署

### 构建镜像

```bash
make build
```

### 使用 docker-compose

```bash
make run
```

## 数据库

本服务使用与网关数据转发服务相同的数据库（`assistant_gateway_db`），表结构请参考 `assistant_gateway/db/schema.sql`。

## 配置变更流程

1. 通过管理 API 修改配置（后端或路由）
2. 配置变更自动记录到 `config_history` 表
3. 网关数据转发服务通过轮询检测到配置变更
4. 网关自动重新加载配置并更新路由

## 项目结构

```
.
├── cmd/
│   └── admin/          # 服务入口
├── internal/
│   ├── config/         # 配置存储层
│   ├── handler/        # API handlers
│   └── middleware/     # 中间件
├── Dockerfile
├── Makefile
└── README.md
```

## License

MIT

