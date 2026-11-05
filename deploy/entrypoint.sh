#!/bin/sh
set -eu
# 数据库就绪后，依次升级表结构、初始化管理员，再启动服务。
/app/bin/migrate -dir /app/migrations
/app/bin/bootstrap
exec /app/bin/server
