#!/bin/sh
set -eu

# 不生成 .env、不写入数据库，也不打印任何凭据；配置校验由 Go 进程负责。
if [ "$#" -eq 0 ]; then
    set -- /usr/local/bin/vmqfox-api
fi

exec "$@"