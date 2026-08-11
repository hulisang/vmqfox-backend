# 单用户 Go 后端重构计划

> 目标：将 `vmqfox-backend` 收敛为单用户、Go-only、可全新部署的后端。
>
> 数据策略：仅支持空 MySQL 实例初始化，不迁移、不读取、不兼容旧 PHP 数据。

## 1. 目标与边界

### 1.1 目标

- 以 Go 服务作为唯一后端运行时，不再依赖 PHP、PHP-FPM、ThinkPHP、Composer 或 Redis Session。
- 保持单管理员模型，不引入多用户、角色、租户或数据隔离机制。
- 保留前端、Android 监控端和商户调用仍需要的外部协议。
- 以 `database/schema.sql` 作为唯一数据库结构来源，在空 MySQL 实例上完成初始化。
- 交付可直接通过 Docker Compose 启动的前端、Nginx、Go API 和 MySQL 组合。

### 1.2 不在本次重构范围内

- 旧 PHP 四表数据迁移、导入、双写或兼容读取。
- PHP 与 Go 双运行、流量切换、shadow 切流或旧服务回滚流程。
- 多管理员、角色权限、租户隔离和用户管理功能。
- 无实际调用方的 PHP 后台页面协议。
- 为不确定的未来需求预留抽象层或通用插件系统。

### 1.3 外部协议与旧数据的边界

旧数据不兼容不等于外部调用协议全部废弃。以下协议仍需按真实调用方决定是否保留：

- 前端使用的 `/api/*` 管理接口和支付查询接口。
- Android 使用的监控心跳、到账推送路径及签名规则。
- 商户创建订单、查询订单、检查支付状态和接收通知的契约。
- 二维码 PNG、支付跳转 HTML 等非 JSON 响应。

PHP 源码只作为这些外部行为的参考，不属于最终运行产物。

## 2. 工程约束

### 2.1 模块职责

保持当前依赖方向：

`HTTP handler → usecase → port → adapter/domain`

- `internal/http/handler` 只负责请求规范化、鉴权上下文、协议映射和响应输出。
- `internal/usecase` 负责业务流程、事务边界和状态转换。
- `internal/port` 定义业务需要的持久化和外部能力。
- `internal/adapter` 实现 MySQL、HTTP 通知、二维码和系统能力。
- `internal/domain` 只表达订单、支付、二维码、设置和管理员凭据等核心模型。
- `internal/app` 只负责依赖组装、后台任务和服务生命周期。

禁止为赶进度把订单、补单、通知或数据库操作直接写入 handler。

### 2.2 数据与并发

- 金额始终以整数分参与计算和持久化，文本金额只用于协议输出。
- 订单状态、价格锁、支付事件和通知 outbox 必须在明确事务中更新。
- 重复支付推送必须通过 `payment_events` 幂等处理。
- 网络通知不得在数据库锁或长事务内执行。
- 通知失败不得回滚已确认的付款事实。
- 正式部署默认只运行一个 `writer` 实例；多 writer 扩展不在本计划范围内。

### 2.3 数据库初始化

- 应用运行时不自动建表、不自动改表、不创建默认管理员。
- `database/schema.sql` 是唯一 Schema 基线。
- 管理员密码使用 `vmqfox-api` 的 `-hash-password` 命令行参数生成管理员 bcrypt 哈希后写入。
- 仓库、镜像和示例配置不得包含默认密码、生产密钥或收款配置。

## 3. 当前实现基线

### 3.1 已完成的主要能力

- Go 入口与应用生命周期：`cmd/vmqfox-api`、优雅停机、数据库关闭。
- 分层与依赖组装：domain、port、usecase、adapter、handler、middleware、job。
- 认证：单管理员凭据、bcrypt 密码、Token 签发与校验。
- 设置：系统设置读取与更新、监控状态读取。
- 订单：创建、查询、列表、统计、详情、检查、返回 URL、关闭。
- 支付监控：心跳、到账匹配、支付事件幂等、价格锁释放。
- 通知：outbox、异步投递、失败重试、新版 POST 与历史 GET 回退。
- 二维码：生成、解析、列表、新增、删除和状态更新。
- 后台任务：订单生命周期处理和通知投递。
- 运行状态：`/health` 表示进程存活，`/ready` 检查数据库连接。
- 部署骨架：Go Dockerfile、Compose、Nginx 反向代理、环境示例和发布工作流。

### 3.2 当前未收口能力

`internal/http/router.go` 仍注册了多组 501 占位路由，主要包括：

- `/api/user/info`
- `/api/user/list`
- `/api/menu`
- `DELETE /api/order/:id`
- `POST /api/order/expired`
- `DELETE /api/order/last`
- `POST /api/order/reissue/:id`
- `POST /api/config/save`
- `POST /api/config/monitor`
- `/api/admin/index/*` 历史别名
- `/login`、`/getMenu`、`/closeOrder`、`/getState`、`/closeEndOrder`、`/getMain` 等历史管理路径
- 多数 `/admin/*` PHP 后台路径

这些路径必须在阶段 0 被明确归类，最终不能以“已注册但返回 501”作为完成状态。

### 3.3 当前版本状态

当前工作位于 `refactor/go-single-user` 分支。Go 源码、Schema 及部分部署配置尚未形成稳定的重构基线提交。后续实现前应先固定当前代码状态，使每个阶段的变更可独立审查，避免将业务补齐、部署改造和 PHP 清理混在同一批变更中。

## 4. 目标接口范围

### 4.1 必须保留并完成

#### 基础状态

- `GET /health`
- `GET /ready`
- `GET /`

#### 认证与单管理员视图

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/user/info`
- `GET /api/user/list`
- `GET /api/menu`

`user/info` 和 `user/list` 只返回唯一管理员视图；不得为兼容这两个接口引入用户表、角色表或权限系统。菜单使用静态定义，不建立菜单持久化模型。

#### 订单

- `/api/order/create`
- `/api/order/get/:id`
- `/api/order/check/:id`
- `/api/order/return-url/:id`
- `/api/order/list`
- `/api/order/detail/:id`
- `/api/order/close/:id`
- `DELETE /api/order/:id`
- `POST /api/order/expired`
- `DELETE /api/order/last`
- `POST /api/order/reissue/:id`

#### 二维码

- `/api/qrcode/generate`
- `/api/qrcode/parse`
- `/api/qrcode/list`
- `/api/qrcode/wechat`
- `/api/qrcode/alipay`
- `/api/qrcode/add`
- `/api/qrcode/bind/:id`
- 二维码删除接口

#### 配置

- `GET /api/config/get`
- `GET /api/config/status`
- `GET /api/config/settings`
- `POST /api/config/settings`
- `GET /api/config/monitor`
- `POST /api/config/monitor`

`POST /api/config/save` 只有在确认存在调用方时才作为设置更新别名保留，否则删除。

#### 监控

- `/api/monitor/heart`
- `/api/monitor/push`

这两个路径由当前 Android 调用，属于正式接口。

### 4.2 待确认的兼容接口

以下路径没有当前仓库内调用方，阶段 0 应根据公开商户文档和实际部署约定决定保留或删除：

- `/api/v2/monitor/heart`
- `/api/v2/monitor/push`
- `/createOrder`、`/getOrder`、`/checkOrder`
- `/appHeart`、`/appPush`
- `/enQrcode`
- `/index/index/getReturn`

需要保留时只实现 Go 协议适配，不复制 PHP 内部实现；没有真实调用约束时按 YAGNI 删除。

### 4.3 默认删除

阶段 0 未发现真实调用方时，删除以下路由及其 PHP 页面语义：

- PHP Session 登录和菜单路径 `/login`、`/getMenu`。
- PHP 中缺少有效实现的 `/closeOrder`、`/getState`。
- 已由生命周期 worker 取代的 `/closeEndOrder`。
- `/getMain` 及 `/admin/getMain`。
- `/api/admin/index/*` 历史后台别名。
- `/admin/*` 中的首页、更新检查、IP 查询和页面型管理接口。
- 其他仅服务于 ThinkPHP 后台页面、且当前前端未调用的路径。

删除前只核对现有前端、Android 和公开商户文档；不因旧 PHP 源码中存在路由就默认保留。

## 5. 分阶段实施计划

## 阶段 0：范围冻结与基线固化

### 任务

- [x] 对照当前前端 API 封装、Android 请求代码和商户接口，形成最终支持路径清单。
- [x] 将 `router.go` 中每条 501 路由标记为“实现”或“删除”。
- [x] 固化每条保留接口的 HTTP 方法、鉴权要求、输入来源、签名和响应格式。
- [x] 确认 PNG、HTML、Layui 和 JSON envelope 等特殊响应是否仍有调用方。
- [x] 固定当前 Go 重构基线，后续按阶段拆分变更。

### 检查点

- 保留接口没有未确定的认证策略。
- 公开订单接口和管理接口边界明确。
- 不再以 PHP Session 作为任何正式接口的认证方式。
- 旧数据兼容、数据库迁移和 PHP/Go 双运行明确排除。

### 退出条件

- 所有已注册路由都有明确去向。
- 核心接口契约无待定项。
- 待删除路径不会被仓库内调用方使用。

## 阶段 1：管理 API 最小闭环

### 任务

- [x] 实现单管理员信息和列表查询，不新增用户领域模型。
- [x] 实现静态菜单响应，不引入菜单仓储。
- [x] 在订单 usecase 中补齐按 ID 删除、过期处理和最后订单清理。
- [x] 将补单定义为人工确认到账业务：校验可操作状态，在事务内更新付款状态和支付时间、释放价格锁并写入通知 outbox；handler 不直接调用商户 URL。
- [x] 复用设置服务实现监控配置更新。
- [x] 根据阶段 0 结论实现或删除 `/api/config/save`。
- [x] 为所有破坏性管理操作统一配置 Token 鉴权和写模式保护。

### 检查点

- 删除订单时同步处理关联价格锁，且事务失败不会留下半完成状态。
- 过期处理复用生命周期用例，不产生第二套状态转换规则。
- 补单的状态更新、价格锁释放和通知入队保持原子性，通知失败不回退已确认的付款状态。
- 单管理员接口只暴露必要字段，不返回密码哈希、密钥或敏感设置。

### 退出条件

- 正式支持的管理 `/api/*` 路径不再返回 501。
- handler 中不存在数据库操作或外部通知调用。
- 订单管理操作与后台 worker 使用同一业务规则。

## 阶段 2：核心支付链路与协议收口

### 任务

- [x] 核对“创建订单 → 分配价格锁 → 选择二维码 → 查询支付页”的完整流程。
- [x] 核对“监控推送 → 支付事件幂等 → 订单状态更新 → 释放价格锁 → 写入 outbox”的事务流程。
- [x] 核对“通知投递 → 新版 POST → 历史 GET 回退 → 重试/完成”的异步流程。
- [x] 对齐新旧订单创建签名、返回 URL 签名和通知签名。
- [x] 对齐 query、form、JSON 中数字字符串和空值的规范化行为。
- [x] 保持二维码 PNG 与 `isHtml=1` 跳转页面的正确 Content-Type。
- [x] 删除阶段 0 已确认废弃的历史路由，不保留长期 501。

### 检查点

- 同一支付事件重复推送不会重复更新订单或重复创建通知任务。
- 订单关闭、过期、删除和支付成功都会按规则处理价格锁。
- 通知失败保留已支付状态，并可由 outbox 继续重试。
- API、Legacy、Layui 和 Raw 响应只在确定需要的路径存在。

### 退出条件

- 前端支付页、Android 监控和商户调用均落到真实业务实现。
- 核心链路不存在依赖 PHP 运行时的行为。
- 正式路由清单中不存在 501 handler。

## 阶段 3：全新数据库初始化

### 任务

- [x] 将 `database/schema.sql` 固定为唯一数据库初始化文件。
- [x] 校验 `settings`、`admin_credentials`、`qrcodes`、`orders`、`price_locks`、`payment_events`、`notification_outbox` 七张表。
- [x] 校验唯一约束、事务所需索引、字符集、时间精度和金额字段。
- [x] 明确空库初始化后必须设置的管理员、通讯密钥、回调地址和收款配置。
- [x] 提供 `-hash-password` 命令行参数以生成管理员 bcrypt 哈希，并提供最小初始化步骤。
- [x] 确认缺少管理员或业务密钥时接口返回明确错误，不写入无效订单。

### 检查点

- 不引用 `vmq.sql`，不读取旧 PHP 表，不包含迁移分支。
- Schema 不写入默认管理员密码或生产密钥。
- Compose 只在空 MySQL 数据卷首次初始化时执行 Schema。
- `/ready` 只表达数据库可连接；业务配置错误由对应 API 明确报告。

### 退出条件

- 空 MySQL 实例能够一次完成建库初始化。
- 创建唯一管理员后可登录。
- 配置必要设置后可完成订单创建、监控推送和通知投递。

## 阶段 4：部署与发布定稿

### 任务

- [x] 收敛 `Dockerfile`，运行镜像只包含 Go 服务及必要系统依赖。
- [x] 收敛 `docker-compose.yml`，默认以 `VMQ_RUNTIME_MODE=writer` 启动单个 API 实例。
- [x] 收敛 Nginx 配置，将正式 API、监控和兼容路径转发到 Go HTTP 服务。
- [x] 校验 `/health` 与 `/ready` 在容器和网关中的使用方式。
- [x] 收敛 `entrypoint.sh` 和 `env.example`，移除 PHP、Session、Redis 等失效变量。
- [x] 收敛 GitHub Actions，只发布 Go 二进制、Schema、示例配置和多架构镜像。
- [x] 明确通知 worker 与生命周期 worker 在 writer 模式下随应用启动。

### 检查点

- 新环境只需环境变量和空 MySQL 数据卷即可启动。
- Nginx 不包含 PHP-FPM upstream、ThinkPHP rewrite 或 PHP 静态目录规则。
- 运行镜像使用非 root 用户，健康检查可执行。
- 发布产物中的二进制、Schema 和配置来自同一版本。

### 退出条件

- Docker Compose 可启动前端、Nginx、Go API 和 MySQL。
- `/health`、`/ready`、登录和核心业务接口均可从网关访问。
- 正式镜像不包含 PHP 应用、Composer 或旧 SQL。

## 阶段 5：PHP 清理与文档收口

### 任务

- [x] 删除 `app/`、根目录 PHP `config/`、`route/`、PHP `public/` 和 `vendor/`。
- [x] 删除 Composer 文件、ThinkPHP 入口、旧构建脚本和 `vmq.sql`。
- [x] 删除 PHP-FPM、旧 Nginx rewrite、Travis PHP 构建及失效部署文件。
- [x] 保留仍承担外部协议适配职责的 Go 兼容代码；不因删除 PHP 源码而破坏已确认的调用契约。
- [ ] 更新 README，写明系统边界、环境变量、空库初始化、管理员创建、启动方式、健康检查和正式接口。
- [ ] 更新 CI 文件路径和发布清单，确认仓库只包含有效运行方式。

### 检查点

- 删除清单不包含 Go 构建、运行或协议兼容实际需要的文件。
- 文档不再指导安装 PHP、Composer、PHP-FPM 或旧数据库。
- 仓库搜索不到被正式代码引用的已删除 PHP 路径。

### 退出条件

- Go 应用是仓库内唯一后端实现和运行入口。
- CI 只构建 Go 二进制与镜像。
- README 与实际 Compose、环境变量和 Schema 一致。

## 6. 优先级

### P0：资金与状态正确性

- 整数分金额。
- 价格锁获取与释放。
- 支付事件幂等。
- 订单状态事务。
- 通知 outbox 与重试。
- 补单状态转换与通知入队原子性。

### P1：可用性与部署

- 前端管理接口完整。
- Token 鉴权边界。
- Android 与商户协议。
- 空库初始化。
- 管理员安全创建。
- Docker、Nginx 和健康检查。

### P2：仓库收口

- 删除废弃历史路由。
- 删除 PHP 运行时和旧 SQL。
- 更新 README 与 CI。

## 7. 阶段验收清单

### 接口

- [ ] 正式支持的路由均有真实 handler，无 501。
- [ ] 不支持的路由已从注册表和文档删除。
- [ ] 管理接口要求有效 Token。
- [ ] 商户订单和监控接口按约定验签。
- [ ] PNG、HTML 与 JSON 响应类型正确。

### 数据与业务

- [ ] 空库 Schema 初始化成功。
- [ ] 唯一管理员创建与登录成功。
- [ ] 重复商户订单号被唯一约束阻止。
- [ ] 并发订单不会获得冲突价格。
- [ ] 重复支付推送不会重复处理。
- [ ] 关闭、过期、删除和支付成功不会遗留价格锁。
- [ ] 通知失败可重试且不回退付款状态。

### 部署

- [ ] Compose 不依赖 PHP、PHP-FPM 或旧数据库文件。
- [ ] API 以 writer 模式启动后台任务。
- [ ] `/health` 和 `/ready` 反映正确状态。
- [ ] Nginx 只代理正式支持的后端路径。
- [ ] 发布产物不包含 PHP 运行时。

### 仓库

- [ ] Go 源码、Schema、部署配置和文档形成一致基线。
- [ ] README 只描述全新 Go 部署。
- [ ] PHP 目录、Composer 产物和 `vmq.sql` 已删除。
- [ ] CI 只包含有效的 Go 构建和镜像发布流程。

## 8. 完成定义

满足以下条件时，单用户 Go 后端重构完成：

1. 正式接口清单中不存在 501 或 PHP Session 依赖。
2. 订单、监控、通知、二维码、设置和单管理员认证形成完整 Go 业务闭环。
3. 空 MySQL 环境可通过 `database/schema.sql` 初始化，不包含任何旧数据兼容逻辑。
4. 前端、Android 和商户核心调用均由 Go 服务直接处理。
5. Docker Compose 和发布流水线能够构建、启动并发布纯 Go 后端。
6. PHP 运行时、旧 SQL 和失效部署说明已从最终仓库移除。