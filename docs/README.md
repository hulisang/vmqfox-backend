# 部署文档（docs）

本目录收录 V3.0（Go-only 单用户版）的部署与升级文档，是 [项目 Wiki](https://github.com/hulisang/vmqfox-backend/wiki) 部署相关页面的离线副本，方便随源码仓库直接查阅；**内容以 Wiki 为准**。

| 文档 | 适用场景 |
| :--- | :--- |
| [Docker 一键部署](docker一键部署.md) | 使用 docker-compose 快速上线（推荐） |
| [裸机二进制部署](裸机二进制部署.md) | 直接下载 Release 产物 + systemd 托管 |
| [反向代理与 HTTPS](反向代理与HTTPS.md) | nginx 反代、脱敏与转发头配置 |
| [升级与迁移](升级与迁移.md) | 旧版数据不兼容说明与公开令牌迁移步骤 |

其余对接类文档（API 契约、签名协议、epay 插件、监控端 App、安全清单）请直接访问 [Wiki](https://github.com/hulisang/vmqfox-backend/wiki)。
