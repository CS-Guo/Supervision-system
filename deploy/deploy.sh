#!/bin/sh
# 在离线包目录运行；不会拉取镜像，不会删除持久化数据。
set -eu
cd "$(dirname "$0")"
umask 077
docker compose version >/dev/null
if [ ! -f .env ]; then
  # 依赖 Linux 标准工具生成密码，无需云服务器安装 Go 或 Node。
  random_hex() { od -An -N24 -tx1 /dev/urandom | tr -d ' \n'; }
  {
    printf 'HTTP_PORT=%s\n' "${HTTP_PORT:-80}"
    printf 'MYSQL_ROOT_PASSWORD=%s\n' "$(random_hex)"
    printf 'MYSQL_PASSWORD=%s\n' "$(random_hex)"
    printf 'BOOTSTRAP_ADMIN_PASSWORD=%s\n' "$(random_hex)"
  } > .env
  printf '已生成随机密码，保存在当前目录 .env（仅文件所有者可读写）。\n'
fi
docker compose --env-file .env -f compose.yaml up -d --wait --wait-timeout 360
printf '\n部署完成。登录账号：admin；初始密码见 .env 中 BOOTSTRAP_ADMIN_PASSWORD。\n'
printf '访问 http://云服务器IP:端口（默认80）；请确认云安全组已开放该端口。\n'
