#!/bin/sh
# 在开发机构建 x86_64 云服务器的离线包；需要联网下载基础镜像。
set -eu
cd "$(dirname "$0")/.."
bundle=dist/supervision-linux-amd64
mkdir -p "$bundle"
if [ "${EXPORT_ONLY:-0}" != "1" ]; then
  registry=${BASE_REGISTRY:-docker.io/library}
  docker buildx build --build-arg "BASE_REGISTRY=$registry" --platform linux/amd64 --load -t supervision-app:1.0.0-amd64 .
  docker pull --platform linux/amd64 "$registry/mysql:8.0.45"
  docker tag "$registry/mysql:8.0.45" supervision-mysql:8.0.45-amd64
fi
# 显式选择架构，避免 Docker 导出尚未下载的其他架构清单。
# 先导出再压缩，确保导出失败时不会留下伪成功的压缩包。
docker image save --platform linux/amd64 -o "$bundle/images.tar" supervision-app:1.0.0-amd64 supervision-mysql:8.0.45-amd64
gzip -f "$bundle/images.tar"
cp deploy/compose.yaml deploy/deploy.sh deploy/.env.example "$bundle/"
cp docs/docker-deployment.md "$bundle/README.md"
(cd "$bundle" && LC_ALL=C shasum -a 256 images.tar.gz > SHA256SUMS)
tar -czf dist/supervision-linux-amd64.tar.gz -C dist supervision-linux-amd64
printf '离线部署包：dist/supervision-linux-amd64.tar.gz\n'
