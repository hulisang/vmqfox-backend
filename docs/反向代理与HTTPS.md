> 本页为 [项目 Wiki](https://github.com/hulisang/vmqfox-backend/wiki) 的离线副本（同步于 2026-08-31），在线版以 Wiki 为准。

# 反向代理与 HTTPS

> 单服务模型下 Go 进程同时提供 `/`（嵌入式前端）、`/assets` 与 `/api`，反代**只做纯转发**：不注入 CORS 头，安全响应头与 CSP 全部由 Go 的 `SecurityHeaders` 中间件统一下发，避免两层重复或冲突。

## 为什么必须配反代

生产环境**不要**把 `VMQ_HTTP_PORT`（默认 8000）直接开放到公网防火墙。TLS 在反代层终止，仅对外暴露 443。

## nginx 参考配置

仓库 `docker/nginx/default.conf` 是官方示例，自建 nginx 时搬运并改 `server_name`/证书路径：

```nginx
upstream vmqfox_api {
    server 127.0.0.1:8080;   # Docker 部署改为 vmqfox-api:8080
    keepalive 16;
}

# 公开支付 URL 中的 publicToken 是 bearer 凭据，访问日志必须脱敏，且不记录查询参数
map $uri $vmqfox_log_path {
    ~^/api/order/(?:get|check|return-url)/ /api/order/[redacted-token];
    default $uri;
}
log_format vmqfox_access '$remote_addr - $remote_user [$time_local] '
                         '"$request_method $vmqfox_log_path $server_protocol" $status $body_bytes_sent '
                         '"$http_user_agent"';

server {
    listen 443 ssl http2;
    server_name pay.your-domain.com;
    # ssl_certificate ...; ssl_certificate_key ...;

    client_max_body_size 9m;
    access_log /var/log/nginx/vmqfox_access.log vmqfox_access;

    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    # 关键：Go 据此决定是否下发 HSTS，必须反映客户端到入口的真实协议
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Authorization $http_authorization;
    proxy_read_timeout 35s;

    location = /health { access_log off; proxy_pass http://vmqfox_api; }
    location = /ready  { access_log off; proxy_pass http://vmqfox_api; }
    location /         { proxy_pass http://vmqfox_api; }
}

server { listen 80; server_name pay.your-domain.com; return 301 https://$host$request_uri; }
```

## 反代后必须同步调整的两个 .env 项

| 变量 | 说明 |
| :--- | :--- |
| `VMQ_TRUSTED_PROXY_CIDR` | **必填**反代所在网段（如 `172.x.0.0/16` 或 `127.0.0.1/32`）。留空时所有请求的 client IP 都被识别为反代地址，全站买家共享同一个限流桶；填了才解析 `X-Forwarded-For`，防止客户端伪造 IP 绕过限流。该配置同时决定是否信任 `X-Forwarded-Proto`，进而决定是否下发 HSTS。 |
| `VMQ_FRONTEND_URL` | 改为买家可达的**最终 https 地址**，结尾不带 `/`。 |

## 1Panel / 宝塔面板用户

- 新建「网站」指向反代域名，开启 HTTPS 并强制跳转。
- **不要**在面板里再添加 CORS 头或 `add_header` 安全头（Go 已统一下发，重复会冲突）。
- 面板伪静态里保留 `X-Forwarded-Proto` 透传，参考上面的 nginx 片段。
- epay 站点若与本服务同机，记得把 epay 内网地址加入 `VMQ_NOTIFY_ALLOW_CIDR`，否则异步回调会被 SSRF 防护拦截，订单永远停在未支付。详见[epay 商户插件](https://github.com/hulisang/vmqfox-backend/wiki/epay插件)。

## 健康检查

`/health`（进程存活）与 `/ready`（数据库连通）都反代到 Go 真实状态，供容器编排与监控使用，已关闭 access_log。
