# vmqfox-backend 宝塔面板部署教程

> **专门针对ThinkPHP 8版本的vmqfox-backend API服务**

## 🎛️ 宝塔面板部署步骤

### 第一步：安装宝塔面板

#### 1.1 安装宝塔面板
```bash
# Ubuntu/Debian
wget -O install.sh https://download.bt.cn/install/install-ubuntu_6.0.sh && sudo bash install.sh

# CentOS
yum install -y wget && wget -O install.sh https://download.bt.cn/install/install_6.0.sh && sh install.sh
```

#### 1.2 安装LNMP环境
登录宝塔面板后，安装以下组件：
- **Nginx**: 1.20+
- **MySQL**: 8.0 (推荐) 或 5.7
- **PHP**: 8.2 ⚠️ **必须选择8.2版本**
- **phpMyAdmin**: 最新版
- **Redis**: 7.0+ (推荐)

### 第二步：配置PHP 8.2

#### 2.1 安装必需的PHP扩展
宝塔面板 → 软件商店 → PHP 8.2 → 设置 → 安装扩展：

**必装扩展：**
- ✅ `mysqli` - MySQL数据库
- ✅ `pdo_mysql` - PDO MySQL
- ✅ `gd` - 图像处理（二维码）
- ✅ `mbstring` - 多字节字符串
- ✅ `zip` - ZIP压缩
- ✅ `curl` - HTTP请求
- ✅ `xml` - XML解析
- ✅ `bcmath` - 高精度数学
- ✅ `redis` - Redis缓存（如果使用Redis）

#### 2.2 优化PHP配置（可选）
PHP 8.2 → 配置修改 → php.ini：
```ini
memory_limit = 256M
max_execution_time = 300
post_max_size = 50M
upload_max_filesize = 50M
date.timezone = Asia/Shanghai
```

### 第三步：创建API站点（关键配置）

#### 3.1 添加站点
宝塔面板 → 网站 → 添加站点：
- **域名**: `api.yourdomain.com` 或 `your-ip:8000`
- **端口**: `8000` (API专用端口)
- **根目录**: `/www/wwwroot/vmqfox-api`
- **PHP版本**: `PHP-82`
- **数据库**: 创建 `vmq` 数据库

#### 3.2 ⚠️ 重要：运行目录设置
**关键配置**：由于这是纯API服务，配置与传统Web应用不同

网站设置 → 网站目录：
- **运行目录**: 保持为 `/` (根目录)
- **❌ 不要设置为 `public`** 
- **防跨站攻击**: 关闭
- **防盗链**: 关闭

#### 3.3 配置伪静态（ThinkPHP 8 API专用）
网站设置 → 伪静态 → 自定义：

```nginx
# ThinkPHP 8 纯API服务完整伪静态配置

# 处理OPTIONS预检请求（CORS）
location / {
    if ($request_method = "OPTIONS") {
        add_header 'Access-Control-Allow-Origin' '*' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type, X-Requested-With' always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Content-Length' 0;
        add_header 'Content-Type' 'text/plain';
        return 204;
    }
    
    # ThinkPHP 8 路由重写 - 关键配置
    if (!-e $request_filename) {
        rewrite ^(.*)$ /public/index.php?s=$1 last;
    }
}

# API路由专门处理
location ~ ^/api/ {
    if ($request_method = "OPTIONS") {
        add_header 'Access-Control-Allow-Origin' '*' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type, X-Requested-With' always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Content-Length' 0;
        add_header 'Content-Type' 'text/plain';
        return 204;
    }
    
    if (!-e $request_filename) {
        rewrite ^(.*)$ /public/index.php?s=$1 last;
    }
}

# 兼容旧版API路由
location ~ ^/(appHeart|appPush|createOrder|checkOrder|getOrder|login|getMenu|admin) {
    if (!-e $request_filename) {
        rewrite ^(.*)$ /public/index.php?s=$1 last;
    }
}

# 健康检查接口
location = /health {
    access_log off;
    return 200 "healthy\n";
    add_header Content-Type text/plain;
}

# 二维码文件访问
location /qr-code/ {
    alias /www/wwwroot/vmqb/runtime/qrcode/;
    expires 1d;
    add_header Cache-Control "public, immutable";
}

# 上传文件访问
location /uploads/ {
    alias /www/wwwroot/vmqb/public/uploads/;
    expires 1d;
    add_header Cache-Control "public, immutable";
}

# 静态资源缓存
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
    expires 30d;
    add_header Cache-Control "public, immutable";
    access_log off;
}

# 禁止访问敏感文件
location ~ ^/(\.env|\.git|composer\.|\.htaccess|README\.md|\.user\.ini)$ {
    deny all;
    return 404;
}

# 禁止访问敏感目录
location ~ ^/(app|config|vendor|runtime|route)/ {
    deny all;
    return 404;
}

# 禁止访问备份和临时文件
location ~ \.(sql|bak|backup|log)$ {
    deny all;
    return 404;
}
```

#### 3.4 配置CORS跨域支持
网站设置 → 配置文件，在 `server` 块中添加：

```nginx
# CORS跨域配置
add_header 'Access-Control-Allow-Origin' '*' always;
add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type, X-Requested-With' always;
add_header 'Access-Control-Allow-Credentials' 'true' always;
```

### 第四步：部署项目代码

#### 4.1 上传项目
1. 下载vmqfox-backend项目ZIP包
2. 宝塔面板 → 文件 → 上传到 `/www/wwwroot/vmqfox-api/`
3. 解压并确保目录结构正确

#### 4.2 安装Composer依赖
宝塔面板 → 终端：
```bash
cd /www/wwwroot/vmqfox-api

# 确认ThinkPHP 8版本
cat composer.json | grep "topthink/framework"
# 应该显示: "topthink/framework": "^8.0"

# 安装依赖
composer install --no-dev --optimize-autoloader

# 验证安装
php think version
```

#### 4.3 配置环境文件
```bash
# 复制环境配置
cp env.example .env

# 编辑配置文件
nano .env
```

**.env 配置内容：**
```ini
APP_DEBUG = false
APP_TRACE = false
APP_FRONTEND_URL = http://your-frontend-domain.com

[DATABASE]
TYPE = mysql
HOSTNAME = localhost
DATABASE = vmq
USERNAME = vmq_user
PASSWORD = your_database_password
HOSTPORT = 3306
CHARSET = utf8mb4
PREFIX = 
DEBUG = false

[REDIS]
HOST = 127.0.0.1
PORT = 6379
PASSWORD = 
SELECT = 0

[CACHE]
DRIVER = redis

[SESSION]
DRIVER = redis
```

#### 4.4 设置目录权限
宝塔面板 → 文件，设置权限：
- 项目根目录：`755`
- `runtime` 目录：`777`
- `public/qr-code` 目录：`777`

或使用终端：
```bash
chmod -R 755 /www/wwwroot/vmqfox-api
chmod -R 777 /www/wwwroot/vmqfox-api/runtime
chmod -R 777 /www/wwwroot/vmqfox-api/public/qr-code
```

### 第五步：配置数据库

#### 5.1 创建数据库
宝塔面板 → 数据库 → 添加数据库：
- **数据库名**: `vmq`
- **用户名**: `vmq_user`
- **密码**: 设置强密码
- **访问权限**: 本地服务器

#### 5.2 导入数据库结构
宝塔面板 → 数据库 → vmq → 管理 → 导入：
上传项目根目录的 `vmq.sql` 文件

### 第六步：测试部署

#### 6.1 测试API接口
```bash
# 测试健康检查
curl http://your-domain:8000/health

# 测试ThinkPHP 8 API
curl http://your-domain:8000/api/config/status

# 测试登录接口
curl -X POST http://your-domain:8000/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"user":"admin","pass":"admin"}'
```

#### 6.2 查看日志
宝塔面板提供便捷的日志查看：
- 网站 → 日志 → 访问日志
- 网站 → 日志 → 错误日志
- PHP → 错误日志

### 第七步：安全配置

#### 7.1 配置SSL证书（推荐）
宝塔面板 → 网站 → SSL → Let's Encrypt：
- 申请免费SSL证书
- 强制HTTPS

#### 7.2 配置防火墙
宝塔面板 → 安全：
- 开放端口：22, 80, 443, 8000
- 配置SSH安全
- 开启面板SSL

### 第八步：性能优化

#### 8.1 启用OPcache
PHP 8.2 → 性能调整 → OPcache：
- 启用OPcache
- 内存大小：128MB

#### 8.2 配置Redis缓存
如果安装了Redis：
- 启动Redis服务
- 在.env中配置Redis连接

## 🔧 常见问题解决

### 1. 500错误
- 检查PHP错误日志
- 确认目录权限
- 验证.env配置

### 2. 跨域问题
- 检查CORS配置
- 确认前端域名设置

### 3. 数据库连接失败
- 检查数据库用户权限
- 验证.env数据库配置

### 4. 路由不生效
- 检查伪静态规则
- 确认运行目录设置