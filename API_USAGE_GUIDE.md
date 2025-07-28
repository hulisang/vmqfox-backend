# 📚 VMQFox Go版 API 使用指南

## 🎯 概述

VMQFox Go版本提供了现代化、统一的RESTful API接口，支持支付页面和管理后台的所有功能需求。

## 🚀 核心特性

- **统一API设计**: 支付页面和管理后台使用相同的基础API路径
- **智能认证**: 条件认证中间件自动判断访问类型
- **RESTful标准**: 完全符合REST API设计原则
- **类型安全**: 完整的请求/响应类型定义
- **高性能**: Go语言原生性能优势
- **多用户支持**: 完整的用户注册、权限管理系统

## 📋 API 路径结构

### 基础路径
```
Base URL: http://localhost:8000
API Version: v2
```

### 路径规范
```
/health                     # 健康检查
/api/public/order/*         # 公开API（第三方商户创建订单）
/api/public/orders/*        # 公开API（支付页面访问）
/api/v2/auth/*              # 用户认证（登录、注册、刷新token）
/api/v2/orders/*            # 订单管理（统一接口，支持认证和公开访问）
/api/v2/users/*             # 用户管理（多用户系统）
/api/v2/qrcodes/*           # 收款码管理
/api/v2/qrcode/*            # 二维码生成
/api/v2/settings/*          # 系统设置
/api/v2/system/*            # 系统信息
/api/v2/dashboard           # 数据看板
/api/v2/menu                # 菜单接口
/api/v2/me                  # 当前用户信息
/api/v2/monitor/*           # 监控端API
```

## 🔐 认证机制

### JWT认证（管理后台）
```http
Authorization: Bearer <jwt_token>
```

### 签名验证（第三方商户）
```http
# MD5签名计算
signStr = "payId=" + payId + "&param=" + param + "&type=" + type + "&price=" + price + "&key=" + secretKey
sign = md5(signStr)

# 请求示例
POST /api/public/order
Content-Type: application/json
{
  "payId": "ORDER_123",
  "param": "custom_param",
  "type": 1,
  "price": 0.01,
  "sign": "calculated_md5_signature"
}
```

### 公开访问（支付页面）
```http
# 方式1: 查询参数
GET /api/v2/orders/ABC123?public=true

# 方式2: 使用专用公开路径
GET /api/public/orders/ABC123
```

## 🔓 公开API（第三方商户）

### 1. 创建订单
```http
POST /api/public/order
Content-Type: application/json

{
  "payId": "MERCHANT_ORDER_123",
  "param": "custom_parameter",
  "type": 1,
  "price": 0.01,
  "sign": "calculated_md5_signature",
  "notifyUrl": "http://merchant.com/notify",
  "returnUrl": "http://merchant.com/return",
  "isHtml": 0
}
```

**签名计算**:
```javascript
// 签名字符串
const signStr = `payId=${payId}&param=${param}&type=${type}&price=${price}&key=${secretKey}`;
// MD5签名
const sign = md5(signStr);
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "payId": "MERCHANT_ORDER_123",
    "orderId": "20250722143052123456",
    "payType": 1,
    "price": 0.01,
    "reallyPrice": 0.01,
    "payUrl": "wxp://...",
    "isAuto": 1,
    "redirectUrl": "http://localhost:3000/#/payment/20250722143052123456"
  }
}
```

### 2. 获取订单详情
```http
GET /api/public/order/{order_id}
```

### 3. 检查订单状态
```http
GET /api/public/order/{order_id}/status
```

**状态响应**:
```json
{
  "code": 200,
  "msg": "订单未支付",
  "data": {
    "state": 0,
    "remainingSeconds": 285,
    "return_url": "http://merchant.com/return",
    "param": "custom_parameter"
  }
}
```

## 🎯 支付页面API（公开访问）

### 1. 获取支付订单详情
```http
GET /api/public/orders/{order_id}
```

### 2. 检查支付状态
```http
GET /api/public/orders/{order_id}/status
```

### 3. 生成回调链接
```http
GET /api/public/orders/{order_id}/return-url
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "Success", 
  "data": {
    "return_url": "http://merchant.com/return?payId=ORDER_123&param=custom&sign=abc123"
  }
}
```

## 📦 订单API（统一接口）

### 1. 获取订单详情
```http
# 支付页面访问（无需认证）
GET /api/v2/orders/{order_id}?public=true

# 管理后台访问（需要认证）
GET /api/v2/orders/{order_id}
Authorization: Bearer <jwt_token>
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "order_id": "ABC123",
    "type": 1,
    "price": 0.01,
    "really_price": 0.01,
    "state": 1,
    "pay_url": "wxp://...",
    "is_auto": 1,
    "create_date": 1642780800,
    "subject": "测试订单",
    "body": "订单描述"
  }
}
```

### 2. 检查订单状态
```http
# 支付页面访问
GET /api/v2/orders/{order_id}/status?public=true

# 管理后台访问
GET /api/v2/orders/{order_id}/status
Authorization: Bearer <jwt_token>
```

### 3. 订单列表（仅管理后台）
```http
GET /api/v2/orders?page=1&limit=10&status=1
Authorization: Bearer <jwt_token>
```

### 4. 创建订单（仅管理后台）
```http
POST /api/v2/orders
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "type": 1,
  "price": 0.01,
  "subject": "测试订单",
  "body": "订单描述",
  "notify_url": "http://example.com/notify",
  "return_url": "http://example.com/return"
}
```

### 5. 更新订单
```http
PUT /api/v2/orders/{order_id}
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "subject": "更新的订单标题",
  "body": "更新的订单描述"
}
```

### 6. 删除订单
```http
DELETE /api/v2/orders/{order_id}
Authorization: Bearer <jwt_token>
```

### 7. 关闭订单
```http
PUT /api/v2/orders/{order_id}/close
Authorization: Bearer <jwt_token>
```

### 8. 批量关闭过期订单
```http
POST /api/v2/orders/close-expired
Authorization: Bearer <jwt_token>
```

### 9. 批量删除过期订单
```http
POST /api/v2/orders/delete-expired
Authorization: Bearer <jwt_token>
```

### 10. 生成回调链接
```http
GET /api/v2/orders/{order_id}/return-url
Authorization: Bearer <jwt_token>
```

## 🔐 用户认证API

### 1. 用户登录
```http
POST /api/v2/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "123456"
}
```

**响应**:
```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "super_admin"
    }
  }
}
```

### 2. 用户注册（多用户功能）
```http
POST /api/v2/auth/register
Content-Type: application/json

{
  "username": "newuser",
  "password": "123456",
  "email": "newuser@example.com"
}
```

### 3. 获取当前用户信息
```http
GET /api/v2/me
Authorization: Bearer <jwt_token>
```

### 4. 刷新Token
```http
POST /api/v2/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### 5. 用户登出
```http
POST /api/v2/logout
Authorization: Bearer <jwt_token>
```

## 👥 用户管理API（多用户系统）

### 1. 获取用户列表
```http
GET /api/v2/users?page=1&limit=10
Authorization: Bearer <jwt_token>
```

### 2. 创建用户
```http
POST /api/v2/users
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "username": "newuser",
  "password": "123456",
  "email": "newuser@example.com",
  "role": "user"
}
```

### 3. 获取用户详情
```http
GET /api/v2/users/{user_id}
Authorization: Bearer <jwt_token>
```

### 4. 更新用户
```http
PUT /api/v2/users/{user_id}
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "username": "updateduser",
  "email": "updated@example.com",
  "role": "admin"
}
```

### 5. 删除用户
```http
DELETE /api/v2/users/{user_id}
Authorization: Bearer <jwt_token>
```

### 6. 重置用户密码
```http
PATCH /api/v2/users/{user_id}/password
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "new_password": "newpassword123"
}
```

## 💳 收款码API

### 1. 获取收款码列表
```http
GET /api/v2/qrcodes?type=1&page=1&limit=10
Authorization: Bearer <jwt_token>
```

### 2. 添加收款码
```http
POST /api/v2/qrcodes
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "type": 1,
  "price": 0.01,
  "pay_url": "wxp://..."
}
```

### 3. 删除收款码
```http
DELETE /api/v2/qrcodes/{qrcode_id}
Authorization: Bearer <jwt_token>
```

### 4. 更新收款码状态
```http
PUT /api/v2/qrcodes/{qrcode_id}/status
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "status": 1
}
```

### 5. 解析收款码
```http
POST /api/v2/qrcodes/parse
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "qr_data": "wxp://f2f0..."
}
```

### 6. 生成二维码图片
```http
GET /api/v2/qrcode/generate?url=wxp://...&size=200
```

## ⚙️ 系统设置API

### 1. 获取系统配置
```http
GET /api/v2/settings
Authorization: Bearer <jwt_token>
```

### 2. 保存系统配置
```http
POST /api/v2/settings
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "user": "admin",
  "pass": "123456",
  "notify_url": "http://example.com/notify",
  "return_url": "http://example.com/return",
  "key": "your_secret_key"
}
```

### 3. 获取监控配置
```http
GET /api/v2/settings/monitor
Authorization: Bearer <jwt_token>
```

### 4. 更新监控配置
```http
PUT /api/v2/settings/monitor
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "monitor_timeout": 300,
  "check_interval": 60
}
```

## 📊 系统信息API

### 1. 系统状态
```http
GET /api/v2/system/status
Authorization: Bearer <jwt_token>
```

### 2. 系统信息
```http
GET /api/v2/system/info
Authorization: Bearer <jwt_token>
```

### 3. 检查更新
```http
GET /api/v2/system/update
Authorization: Bearer <jwt_token>
```

### 4. 获取IP信息
```http
GET /api/v2/system/ip
Authorization: Bearer <jwt_token>
```

### 5. 全局系统状态
```http
GET /api/v2/system/global-status
Authorization: Bearer <jwt_token>
```

### 6. 健康检查
```http
GET /health
```

## 📋 其他API

### 1. 数据看板
```http
GET /api/v2/dashboard
Authorization: Bearer <jwt_token>
```

### 2. 菜单接口
```http
GET /api/v2/menu
Authorization: Bearer <jwt_token>
```

## 📱 监控端API

### 1. 心跳检测
```http
# GET方式
GET /api/v2/monitor/heart

# POST方式
POST /api/v2/monitor/heart
Content-Type: application/json

{
  "device_id": "device_123",
  "status": "online"
}
```

### 2. 监控推送
```http
POST /api/v2/monitor/push
Content-Type: application/json

{
  "order_id": "ABC123",
  "amount": 0.01,
  "status": "paid"
}
```

## 🔧 错误处理

### 标准错误响应
```json
{
  "code": 400,
  "msg": "Bad Request",
  "data": null
}
```

### 常见错误码
- `200`: 成功
- `400`: 请求参数错误
- `401`: 未认证
- `403`: 权限不足
- `404`: 资源不存在
- `500`: 服务器内部错误

## 🧪 测试示例

### 使用curl测试
```bash
# 健康检查
curl http://localhost:8000/health

# 用户登录
curl -X POST http://localhost:8000/api/v2/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

# 用户注册
curl -X POST http://localhost:8000/api/v2/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","password":"123456","email":"newuser@example.com"}'

# 公开访问支付订单
curl "http://localhost:8000/api/public/orders/ABC123"

# 认证访问订单
curl -H "Authorization: Bearer <token>" \
  http://localhost:8000/api/v2/orders/ABC123

# 获取用户列表
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8000/api/v2/users?page=1&limit=10"

# 创建收款码
curl -X POST http://localhost:8000/api/v2/qrcodes \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"type":1,"price":0.01,"pay_url":"wxp://..."}'

# 获取数据看板
curl -H "Authorization: Bearer <token>" \
  http://localhost:8000/api/v2/dashboard
```

## 🎯 最佳实践

1. **使用统一路径**: 优先使用 `/api/v2/orders` 而不是 `/api/public/orders`
2. **正确认证**: 支付页面使用公开路径或 `?public=true`，管理后台使用JWT
3. **错误处理**: 始终检查响应的 `code` 字段
4. **分页查询**: 使用 `page` 和 `limit` 参数
5. **类型安全**: 使用TypeScript类型定义
6. **多用户支持**: 利用用户注册和权限管理功能

## 🔄 向后兼容

为了平滑迁移，我们保持了以下兼容性：
- `/api/public/order/*` 路径仍然可用（第三方商户）
- `/api/public/orders/*` 路径用于支付页面
- 响应格式保持一致
- 错误码标准化

## 📈 性能特性

- **多用户隔离**: 每个用户的数据完全隔离
- **智能认证**: 自动判断访问类型，无需重复配置
- **高并发**: Go语言天然支持高并发处理
- **低延迟**: API响应时间通常在10ms以内
- **轻量部署**: 单一二进制文件，Docker镜像小于20MB

---

**更新时间**: 2025-07-28  
**API版本**: v2  
**文档版本**: 2.0
**服务端口**: 8000
