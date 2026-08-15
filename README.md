# V免签 (VMQFox) —— 单用户 Go 语言免签支付网关

本项目是针对个人开发者的免签支付网关。现已完成**全面重构**，从原 ThinkPHP 8 版本彻底收敛为 **纯 Go 语言版本 (Go-only)**，移除了对 PHP、PHP-FPM、Composer 以及外部 Redis 的任何依赖。

> [!IMPORTANT]
> **关于数据兼容性**：本项目仅支持全新空 MySQL/MariaDB 数据库实例初始化，不迁移、不读取、亦不兼容任何旧版 PHP 服务端的数据结构。

---

## 🌟 项目特色与优势

* **极速与低消耗**：纯 Go 编译为单可执行二进制文件，启动时间在毫秒级，运行时内存占用极低（通常低于 15MB）。
* **安全加固与配置鉴权**：彻底修复原 PHP 版本敏感配置接口未鉴权导致的通讯密钥与密码泄露重大漏洞。所有管理与配置接口（`/api/config/*`）强制挂载无状态 JWT 鉴权中间件，密码采用 Bcrypt 单向哈希加密存储并全局强制输出脱敏，杜绝明文泄露风险。
* **单用户无状态设计**：面向个人开发者，采用单管理员模型，抛弃了复杂的多租户和 RBAC 表；使用无状态 JWT 方案作为管理后台的鉴权基石。
* **高可靠金额占锁（Price Lock）**：同一时间同类额度分配独立锁，并在 Pending 订单物理删除、超时关闭、或支付成功时自动级联释放，避免金额占死与重复占款。
* **事务型消息通知（Outbox Pattern）**：保障订单支付与异步通知投递的强一致性。采用后台定时 Worker 驱动重试机制（指数退避），且首选新版 POST 规范化请求发送，支持历史 GET（Query 传参）形式的回退投递。
* **双模式健康检查**：支持 `/health` 与 `/ready` 接口，分别反馈进程活跃度及后端数据库连接的健康情况，适合各种容器编排和反向代理监控。

---

## ⚙️ 核心环境变量配置

系统通过读取环境变量进行配置，支持读取根目录 `.env` 文件。

| 环境变量名 | 默认值 | 描述 |
| :--- | :--- | :--- |
| `VMQ_SERVER_HOST` | `0.0.0.0` | API 服务监听网卡 IP |
| `VMQ_SERVER_PORT` | `8080` | API 服务监听端口 |
| `VMQ_SERVER_MODE` | `release` | Gin 框架运行模式 (`debug` / `release`) |
| `VMQ_RUNTIME_MODE` | `writer` | 运行模式。`writer` 模式下会随服务启动后台定时轮询任务（过期处理、Outbox 推送） |
| `VMQ_FRONTEND_URL` | `http://localhost:3006` | 前端管理台地址（用于跨域与支付跳转） |
| `VMQ_TOKEN_SECRET` | *必填* | 至少 32 位的 JWT 签名密钥（离线生成随机值） |
| `VMQ_TOKEN_ISSUER` | `vmqfox` | JWT Token 签发者名 |
| `VMQ_TOKEN_TTL` | `24h` | JWT 凭据有效时长 |
| `VMQ_DB_HOST` | `127.0.0.1` | MySQL/MariaDB 主机地址 |
| `VMQ_DB_PORT` | `3306` | MySQL/MariaDB 端口 |
| `VMQ_DB_USER` | `vmqgo` | 数据库用户名 |
| `VMQ_DB_PASSWORD` | *必填* | 数据库密码 |
| `VMQ_DB_NAME` | `vmqgo` | 数据库名 |
| `VMQ_DB_CHARSET` | `utf8mb4` | 字符集 |
| `VMQ_LIFECYCLE_POLL_INTERVAL`| `10s` | 订单超时关闭扫描周期 |
| `VMQ_NOTIFICATION_POLL_INTERVAL`| `2s` | Outbox 推送通知扫描周期 |

---

## 🚀 极速部署与运行

### 1. 本地直接编译运行
确保您已配置 Go 1.26.5 或更高环境。

```bash
# 编译可执行程序
go build -o vmqfox-api ./cmd/vmqfox-api

# 启动 Web 服务 (请确保已通过命令行或 .env 注入上述环境变量)
./vmqfox-api
```

### 2. 数据库结构初始化
项目运行时**不会**自动执行建表操作。
使用宿主机或容器客户端，在您的空数据库实例（如 `vmqgo`）中导入 `/database/schema.sql` 完成七张核心数据表的初始化：
```bash
# 使用客户端导入 schema 基线
mysql -h 127.0.0.1 -u vmqgo -p vmqgo < database/schema.sql
```

### 3. 生成管理员密码并写入

> [!WARNING]
> 为了安全，系统没有内置默认的管理账户和明文密码。您必须离线生成 Bcrypt 哈希并手动插入。

合并后的 `vmqfox-api` 二进制文件内置了 `-hash-password` 参数来离线生成密码哈希。
```bash
# 1. 离线生成 bcrypt 密码哈希 (如设置密码为: admin123)
printf '%s\n' 'admin123' | ./vmqfox-api -hash-password

# 输出样例
# $2a$10$G3QwZVKGrAHttWlZAenrueID8jEFyRpcwhdiGCrq3tCUd8xTqBrnK

# 2. 将此哈希直接用 SQL 写入 admin_credentials 表中以激活管理员账户
# 请将 <username> 与 <bcrypt-hash> 替换为您设定的用户名与刚刚输出的哈希值。
INSERT INTO admin_credentials 
  (id, username, password_hash, enabled, created_at, updated_at) 
VALUES 
  (1, 'admin', '$2a$10$G3QwZVKGrAHttWlZAenrueID8jEFyRpcwhdiGCrq3tCUd8xTqBrnK', 1, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6));
```

### 4. 使用 Docker Compose 一键启动
根目录下提供了直接可用的部署骨架：
1. 拷贝 `env.example` 到 `.env` 并填写其中的参数（主要是数据库密码、Token 签名秘钥等）。
2. 执行启动指令：
   ```bash
   docker compose up -d --build
   ```
3. 容器会自动完成 Nginx 网关、Go API、前端面板以及 MySQL 容器的装配，访问前端面板端口即可进行二维码管理、监控端绑定和系统设置。

---

## 🔗 HTTP 接口契约清单

### 1. 健康与就绪检查 (不需鉴权)
* `GET /health`：进程存活检查，返回 `healthy`。
* `GET /ready`：数据库健康度状态，数据库畅通返回 `ready`。

### 2. 在线管理 API (均需 Authorization: Bearer <JWT>)
* **登录认证**：
  * `POST /api/auth/login` - 用户名/密码登录并换取 Token。
  * `POST /api/auth/logout` - 清除本地缓存 Token 退出。
* **管理员与菜单**：
  * `GET /api/user/info` - 返回单管理员用户名及仪表盘权限集。
  * `GET /api/user/list` - 返回包含该管理员的单用户列表。
  * `GET /api/menu` - 静态 Layui 后台菜单结构定义。
* **订单管理**：
  * `GET /api/order/list` - 分页查询历史订单.
  * `GET /api/order/detail/:id` - 订单详情.
  * `POST /api/order/close/:id` - 管理员手动关闭未付款 Pending 订单（自动释锁）。
  * `DELETE /api/order/:id` - 物理删除特定订单（Pending 状态将级联释锁）。
  * `POST /api/order/expired` - 批量手动触发关闭已超时的 Pending 订单。
  * `DELETE /api/order/expired` - **删除过期订单**：物理批量删除所有超时未付/已关闭的订单（Pending 级联释锁）。
  * `DELETE /api/order/last` - 物理清理历史归档旧订单（24小时前）。
  * `POST /api/order/reissue/:id` - **人工补单**：事务内强制改变订单为已支付、释放金额锁并向 Outbox 压入重签通知。
* **系统配置**：
  * `GET /api/config/settings` - 获取系统收款配置、有效期、区分额度方式等。
  * `POST /api/config/settings` (及别名 `POST /api/config/save`) - 保存/更新系统全局配置与管理员密码。
  * `GET /api/config/monitor` - 读取当前系统配置的监控心跳状态。
  * `POST /api/config/monitor` - 设置/更改监控在线指示参数。

### 3. 公共支付网关与监控协议 (需商户 MD5 验签)
* **网关链路**：
  * `POST /api/order/create` (及兼容 `ANY /createOrder`) - 传入商户单号、金额、类型，验签后自动匹配二维码，占用价格锁。
  * `GET /api/order/get/:id` (及兼容 `ANY /getOrder`) - 获取前台收银台支付展示参数（包含reallyPrice、二维码路径、超时秒数）。
  * `GET /api/order/check/:id` (及兼容 `ANY /checkOrder`) - 异步轮询订单状态。
  * `GET /api/order/return-url/:id` - 用户付款成功后的重定向回跳签名校验。
  * `GET /api/qrcode/generate` (及兼容 `ANY /enQrcode`) - 将收款支付链接极速生成为 PNG 图像输出 (`image/png`)。
* **安卓监控端交互**：
  * `POST /api/monitor/heart` (及兼容 `ANY /appHeart`) - 挂机 App 心跳同步，更新 `lastHeart` 时间。
  * `POST /api/monitor/push` (及兼容 `ANY /appPush`) - 挂机 App 匹配通知栏到账信息推送至服务端，自动完成订单核销、释放价格锁并压入 Outbox 异步回调商户。

---

## 📝 开发者备注
* 本项目纯 Go 逻辑的单元验证及接口覆盖已通过 `scratch/test_api.sh` 集成回归，若对代码做出了二次开发修改，请随时运行该脚本完成对本地服务的回归校验。
