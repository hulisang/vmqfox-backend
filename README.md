# V免签 (VMQFox) —— 单用户 Go 语言免签支付网关

本项目是针对个人开发者的免签支付网关。现已完成**全面重构**，从原 ThinkPHP 8 版本彻底收敛为 **纯 Go 语言版本 (Go-only)**，移除了对 PHP、PHP-FPM、Composer 以及外部 Redis 的任何依赖。

> [!IMPORTANT]
> **关于数据兼容性**：本项目不迁移、不读取、亦不兼容旧版 PHP 服务端的数据结构。已运行的 Go-only 订单库升级到引入 `public_token` 的版本时，必须在新 API 对外启动前执行本文的公开令牌迁移命令。

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
| `VMQ_FRONTEND_URL` | *必填* | 买家可访问的最终公网 Origin，用于生成收银台链接。单服务同源部署时与对外访问地址一致，结尾不带 `/` |
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
| `VMQ_ALLOWED_ORIGIN` | 空 | 允许跨域访问 API 的精确 Origin 列表（逗号分隔）。留空表示**拒绝所有跨域请求**；同源部署无需配置，且不接受通配符 `*` |
| `VMQ_MONITOR_HEARTBEAT_TIMEOUT` | `3m` | 超过该时长未收到心跳即判定挂机端离线 |
| `VMQ_MONITOR_SIGN_TTL` | `5m` | 挂机端心跳/推送签名时间戳的双向允许偏移窗口，超窗按重放拒绝 |
| `VMQ_RATE_LOGIN_LIMIT` / `VMQ_RATE_LOGIN_WINDOW` | `10` / `1m` | 登录端点的每客户端固定窗口额度 |
| `VMQ_RATE_CREATE_LIMIT` / `VMQ_RATE_CREATE_WINDOW` | `30` / `1m` | 建单端点的每客户端固定窗口额度 |
| `VMQ_RATE_PUBLIC_READ_LIMIT` / `VMQ_RATE_PUBLIC_READ_WINDOW` | `60` / `1m` | 公开查询、状态和回跳端点的每 IP 固定窗口额度 |
| `VMQ_RATE_PUBLIC_TOKEN_LIMIT` / `VMQ_RATE_PUBLIC_TOKEN_WINDOW` | `60` / `1m` | 同一公开令牌的独立固定窗口额度 |
| `VMQ_RATE_QRCODE_LIMIT` / `VMQ_RATE_QRCODE_WINDOW` | `30` / `1m` | 二维码生成端点的每客户端固定窗口额度 |
| `VMQ_RATE_MONITOR_HEART_LIMIT` / `VMQ_RATE_MONITOR_HEART_WINDOW` | `120` / `1m` | 挂机端心跳的每客户端固定窗口额度 |
| `VMQ_RATE_MONITOR_PUSH_LIMIT` / `VMQ_RATE_MONITOR_PUSH_WINDOW` | `60` / `1m` | 挂机端推送的每客户端固定窗口额度 |
| `VMQ_TRUSTED_PROXY_CIDR` | 空 | 真实反向代理的 IP/CIDR 列表。留空时仅以 TCP 对端识别客户端，绝不信任自带的转发头；置于反向代理之后时必须填写实际代理网段，否则所有客户端会按代理地址共用限流桶 |
| `VMQ_NOTIFY_ALLOW_CIDR` | 空 | 出站 Webhook 的内网 IP/CIDR 精确白名单，留空表示禁止一切私网与回环目标 |
| `VMQ_NOTIFY_ALLOW_HTTP` | `false` | 出站 Webhook 默认只允许 `https`。设为 `true` 才接受明文 `http`，属于尚未关闭的安全门槛 |
| `VMQ_HTTP_PORT` | `8000` | Compose 对外暴露的宿主端口，映射到容器内 Go 服务的 8080 |

---

## 🚀 极速部署与运行

### 0. 前端资源构建（必做，Go 二进制会嵌入这份产物）

管理台与收银台前端源码位于 `web/`（React + Vite）。构建产物写入 `internal/http/static/out`，
由 `go:embed` 嵌入二进制，因此**必须先构建前端再编译 Go**，否则根路径没有页面可提供。

```bash
cd web
pnpm install --frozen-lockfile
pnpm build
cd ..
```

仓库不提交构建产物，`internal/http/static/out` 只保留 `.gitkeep` 占位。
使用 Docker 或 CI 时无需手动执行本步骤：镜像的 `web-builder` 阶段与发布工作流都会从源码重新构建并校验资源完整性。

### 1. 本地直接编译运行
确保您已配置 Go 1.26.5 或更高环境。

```bash
# 编译可执行程序（前提：已完成上一步的前端构建）
go build -o vmqfox-api ./cmd/vmqfox-api

# 启动 Web 服务 (请确保已通过命令行或 .env 注入上述环境变量)
./vmqfox-api
```

服务启动后，`/` 提供嵌入的前端页面，`/assets/*` 提供静态资源，`/api/*` 提供接口，三者同源。

### 2. 数据库结构初始化
项目运行时**不会**自动执行建表操作。
使用宿主机或容器客户端，在您的空数据库实例（如 `vmqgo`）中导入 `/database/schema.sql` 完成七张核心数据表的初始化：
```bash
# 使用客户端导入 schema 基线
mysql -h 127.0.0.1 -u vmqgo -p vmqgo < database/schema.sql
```

### 3. 公开订单令牌迁移（升级已有 Go-only 订单库时必做）

新版本以 64 位小写十六进制的 `publicToken` 作为匿名收银台凭据，并立即停用旧 `orderId` 公开链接。迁移命令会安全地添加 `orders.public_token`、分批回填随机令牌、创建唯一索引并设置非空约束；重复执行不会改写已有效的令牌。

> [!WARNING]
> 先在维护窗口运行迁移并确认成功，再同时发布后端与前端。旧订单号支付链接会失效，不能作为新版本的兼容入口。

```bash
# 已构建二进制：读取 .env 后执行可重复迁移
./vmqfox-api -migrate-public-tokens

# Docker Compose：保持 MySQL 已启动，但在新 API 对外启动前运行一次性迁移容器
# docker compose up -d mysql
# docker compose run --rm --no-deps vmqfox-api /usr/local/bin/vmqfox-api -migrate-public-tokens
```

迁移完成后核验 `orders.public_token` 均为非空、唯一的 64 位小写十六进制值，再启动或更新 API 与前端服务。

### 4. 一键初始化 / 重置管理员账户

> [!TIP]
> 系统支持交互式终端安全输入（输入密码时不回显且支持二次确认校验），也可配合命令行参数实现自动化配置。

#### 方式 A：一键交互式初始化（推荐）
在配置好 `.env` 且数据库已启动的情况下，直接运行：
```bash
# 本地直接运行
./vmqfox-api -init-admin

# 或在 Docker Compose 环境下执行
docker compose exec vmqfox-api ./vmqfox-api -init-admin
```
按照终端交互提示输入用户名与密码即可完成一键校验、Bcrypt 哈希加密并直接入库。

#### 方式 B：脚本/自动化非交互式初始化
适用于 CI/CD 自动化或 Docker 初始化脚本：
```bash
# 方式 1：通过参数直接传入
./vmqfox-api -init-admin -username admin -password 'YourSecurePassword123' -force

# 方式 2：通过标准输入管道传入密码
printf '%s\n' 'YourSecurePassword123' | ./vmqfox-api -init-admin -username admin -force
```

*(可选)* 若仍需纯离线生成 Bcrypt 哈希，可继续使用 `./vmqfox-api -hash-password`。

### 5. 使用 Docker Compose 一键启动
根目录下提供了直接可用的部署骨架：
1. 拷贝 `env.example` 到 `.env` 并填写其中的参数（数据库密码、Token 签名密钥、`VMQ_FRONTEND_URL` 等）。
2. 执行启动指令：
   ```bash
   docker compose up -d --build
   ```
3. Compose 只启动两个服务：`vmqfox-api`（内含嵌入式前端）与 `mysql`。
   镜像构建时会先安装锁定依赖构建 React 资源、校验 `index.html` 引用的每个文件都存在，再编译并嵌入 Go 二进制。
4. 访问 `http://<主机>:${VMQ_HTTP_PORT:-8000}` 即为唯一生产入口：管理台、收银台与 API 全部由该服务提供。

> [!NOTE]
> 若需在前面独立终止 TLS，可参考 `docker/nginx/default.conf`：它只做纯转发，不注入任何 CORS 头，
> 安全响应头与 CSP 由 Go 的 `SecurityHeaders` 中间件统一下发。反代必须透传 `X-Forwarded-Proto`，
> 服务据此决定是否下发 HSTS，同时需在 `VMQ_TRUSTED_PROXY_CIDR` 中登记该代理网段。

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
  * `POST /api/config/monitor` - 手动覆盖监控状态参数 `jkstate`（该值是监控端在线状态而非总开关，手动写入会在下次心跳或生命周期任务时被自动纠正，仅为兼容旧客户端保留）。

### 3. 公共支付网关与监控协议

建单与监控端请求需商户 v2 HMAC-SHA-256 验签；匿名收银台读取请求使用 `publicToken` 作为持有式凭据。
* **前端与静态资源**：
  * `GET /`、`GET /index.html` - 返回嵌入的前端页面（hash 路由），响应 `Cache-Control: no-cache`。
  * `GET /assets/*` - 带内容哈希的静态资源，响应 `Cache-Control: public, max-age=31536000, immutable`。
* **网关链路**：
  * `POST /api/order/create` - 传入商户单号、金额、类型，验签后自动匹配二维码，占用价格锁；响应中的 `publicToken` 是唯一的匿名收银台凭据。
  * `GET /api/order/get/:publicToken` - 获取前台收银台支付展示参数；只接受 64 位随机公开令牌，响应不包含内部订单号、回调地址或透传参数。
  * `GET /api/order/check/:publicToken` - 轮询订单状态；只返回状态和剩余时间，不下发回跳地址。
  * `GET /api/order/return-url/:publicToken` - 订单支付成功后才返回服务端生成并签名的回跳地址。
  * `GET /api/qrcode/generate` - 将收款支付链接极速生成为 PNG 图像输出 (`image/png`)。
  * 公开 `get`、`check`、`return-url` 路径不兼容旧 `orderId`；无效格式与不存在的令牌均返回一致的“订单不存在”业务响应。
* **安卓监控端交互**：
  * `ANY /api/monitor/heart` - 挂机 App 心跳同步，更新 `lastHeart` 时间。
  * `ANY /api/monitor/push` - 挂机 App 匹配通知栏到账信息推送至服务端，自动完成订单核销、释放价格锁并压入 Outbox 异步回调商户。

### 4. 安全响应头与跨域策略

所有响应由 `SecurityHeaders` 中间件统一附加 `X-Content-Type-Options`、`Referrer-Policy`、
`X-Frame-Options`、`Cross-Origin-Opener-Policy` 与 CSP。CSP 的 `script-src` 仅允许同源脚本与携带
本次请求 nonce 的内联脚本，因此前端入口不含无 nonce 的内联 `<script>`；`isHtml=1` 的支付跳转页
会自动注入该 nonce。只有请求确实经由 HTTPS 到达时才下发 HSTS，避免强制升级中断仍走 HTTP 的挂机端。

跨域遵循默认拒绝：`VMQ_ALLOWED_ORIGIN` 留空时不下发任何 `Access-Control-Allow-Origin`，
配置值中的 `*` 会被忽略，命中白名单时才回显请求 Origin 并附加 `Vary: Origin`。

### 5. 签名协议 v2（HMAC-SHA-256）

通讯密钥作为 HMAC 密钥参与运算，不再拼接进签名明文；结果取 64 位小写 hex，服务端以常量时间比较。
v1 的 MD5 签名已停止受理，仅保留用于识别未升级的旧 SDK 并给出明确报错。

| 场景 | 待签 canonical 串 |
| :--- | :--- |
| 建单 | `payId=<商户单号>&param=<param>&type=<1\|2>&price=<两位小数>&notifyUrl=<notifyUrl>&returnUrl=<returnUrl>` |
| 回调 | `payId=<商户单号>&param=<param>&type=<1\|2>&price=<两位小数>&reallyPrice=<两位小数>` |
| 心跳 | `t=<毫秒时间戳>` |
| 推送 | `type=<1\|2>&price=<两位小数>&t=<毫秒时间戳>` |

约束：

* 金额一律定标为两位小数，且签名串与请求参数必须使用同一份文本。
* `notifyUrl` / `returnUrl` 未传时以空串参与签名，且只接受 `http(s)` 协议。
* 心跳与推送的 `t` 为毫秒 epoch，偏移超出 `VMQ_MONITOR_SIGN_TTL` 即按重放拒绝。
* 三端（Go 服务端、PHP 商户插件 `vmqfox_plugin.php`、安卓挂机端 `MonitorSign.java`）共用同一组黄金向量，
  固化在 `internal/domain/payment/payment_test.go`；前端联调页 `testOrder` 会在界面上自检浏览器端实现是否与该组向量一致。
* 独立复算方式：

```bash
printf '%s' 't=1773500000000' | openssl dgst -sha256 -hmac 'testkey123456' -hex
```

---

## 📝 开发者备注

修改代码后建议按以下顺序自检；本仓库没有额外的集成脚本，质量门禁与发布工作流保持一致：

```bash
# 前端类型检查与构建（同时刷新嵌入资源）
cd web && pnpm build && cd ..

# 后端静态检查与测试
go vet ./...
go test ./...

# 完整镜像构建（含前端构建与资源完整性校验）
docker compose build vmqfox-api
```
