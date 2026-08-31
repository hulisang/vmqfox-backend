> 本页为 [项目 Wiki](https://github.com/hulisang/vmqfox-backend/wiki) 的离线副本（同步于 2026-09-01），在线版以 Wiki 为准。

# Docker Compose 一键部署（推荐）

> 适用版本：V3.0（Go-only 单服务）。compose 只启动 **两个服务**：`vmqfox-api`（内嵌 React 前端）与 `mysql`，不再有独立前端容器、Nginx 容器与 Redis。

## 前置要求

- Docker Engine 20.10+ / Docker Compose v2
- 建议 ≥ 1GB 可用内存（Go 服务常驻内存通常低于 15MB）

## 步骤

### 1. 获取代码并准备配置

```bash
git clone https://github.com/hulisang/vmqfox-backend.git
cd vmqfox-backend
cp env.production.example .env
```

逐项替换 `.env` 中所有 `<>` 占位符，**替换完必须复查**：

```bash
grep -n '<' .env   # 不应命中任何未替换的占位符
```

> ⚠️ 真实事故：`VMQ_FRONTEND_URL` 残留 `https://<pay.your-domain.com>` 占位符时，
> 建单返回的收银台跳转链接是含字面 `<>` 的非法 URL，浏览器会静默拒绝跳转，
> 买家永远卡在「正在跳转到支付页面...」中间页。该值必须是**买家浏览器可达**的地址，结尾不带 `/`。

关键项速查（完整注释见仓库内 `env.production.example`）：

| 变量 | 要求 |
| :--- | :--- |
| `VMQ_FRONTEND_URL` | 买家可达的公网 Origin，如 `https://pay.example.com` |
| `VMQ_TOKEN_SECRET` | `openssl rand -hex 32` 生成，≥32 字符，必填否则启动失败 |
| `VMQ_TOKEN_ISSUER` / `VMQ_TOKEN_TTL` | 必填（无默认值） |
| `VMQ_DB_PASSWORD` / `MYSQL_ROOT_PASSWORD` | 两个不同强随机口令 |
| `VMQ_TRUSTED_PROXY_CIDR` | 有上层反代时必填，否则全站买家共享限流桶 |
| `VMQ_NOTIFY_ALLOW_CIDR` | epay 同机/内网部署时必须放行其地址，否则回调被 SSRF 防护拦截 |

### 2. 启动

```bash
docker compose up -d --build
```

镜像的 `web-builder` 阶段会从 `web/` 源码重新构建 React 前端并校验资源完整性，再编译嵌入 Go 二进制——镜像内嵌的一定是本次提交的产物。

### 3. 初始化数据库与管理者（无需导入 SQL）

一条命令即可完成建表与管理员初始化：`-init-admin` 检测到库表不存在时会**自动创建全部数据表**，无需再手工导入 `database/schema.sql`。

```bash
# 交互式，密码不回显；自动建表 + 设置管理员一步完成
docker compose exec vmqfox-api ./vmqfox-api -init-admin
```

> [!NOTE]
> 服务进程本身启动时不会自动建表，请先完成初始化再对外放流。`database/schema.sql` 仅作为表结构参考或 DBA 特殊场景手工建库用；只想单独建表可运行 `docker compose exec vmqfox-api ./vmqfox-api -init-db`。

### 4. 访问

`http://<主机>:${VMQ_HTTP_PORT:-8000}` 即唯一生产入口：管理台、收银台与 API 全部同源提供。
健康检查：`GET /health`（进程存活）、`GET /ready`（数据库连通）。

## 升级已有 Go-only 订单库

从引入 `publicToken` 之前的版本升级时，**必须先执行迁移再对外放流**，详见[升级与迁移](升级与迁移.md)。

## 常用运维命令

```bash
docker compose logs -f vmqfox-api        # 跟踪 API 日志
docker compose ps                        # 服务状态
docker compose down                      # 停止（保留数据卷）
docker compose exec mysql mysqldump -u root -p vmq > backup.sql   # 备份
```

## 下一步

- 需要 HTTPS / 域名接入：见[反向代理与 HTTPS](反向代理与HTTPS.md)
- 对接商户系统：见[epay 商户插件](https://github.com/hulisang/vmqfox-backend/wiki/epay插件)
- 部署挂机手机：见[监控端 App 配置](https://github.com/hulisang/vmqfox-backend/wiki/监控端App配置)
