# Docker 离线部署说明

本包适用于 **Linux x86_64 / amd64** 云服务器。服务器只需要 Docker Engine 和 Docker Compose v2，建议至少 2 核、2GB 内存以及 10GB 可用磁盘空间。不需要安装 Go、Node 或 MySQL，也不需要服务器访问镜像仓库。

## 首次部署

把 `supervision-linux-amd64.tar.gz` 上传到服务器，执行：

```bash
tar -xzf supervision-linux-amd64.tar.gz
cd supervision-linux-amd64
sha256sum -c SHA256SUMS
docker load -i images.tar.gz
sh deploy.sh
```

访问 `http://云服务器公网IP/`。云安全组、操作系统防火墙须允许 TCP 80 入站。若 80 已占用，在首次运行时使用 `HTTP_PORT=8080 sh deploy.sh`，并访问 `http://云服务器公网IP:8080/`；以后修改 `.env` 中的 `HTTP_PORT` 再执行脚本即可。

`docker load` 仅导入镜像，`deploy.sh` 才会启动应用和数据库。脚本会等待数据库、应用健康检查通过。MySQL 首次初始化可能需要几分钟。

首次执行自动生成 `.env`，包含随机数据库密码和管理员初始密码，权限为 600。登录账号是 `admin`，密码查看当前目录 `.env` 的 `BOOTSTRAP_ADMIN_PASSWORD`。请妥善保管该文件，不要提交到版本库或发送到公开渠道。不要把 `.env.example` 直接改名使用。

默认只创建管理员。登录后由管理员创建领导、部门账号。初次迁移仍会载入项目现有的 **8 个部门和 7 条演示事项**，用于展示原型业务；正式使用前由管理员核对调整。此包不包含开发机数据库中的已有业务数据、会话、账号或上传附件。如需迁移已有数据，还应单独备份并恢复数据库和附件卷。

## 容器与数据

- 应用镜像包含 Go 服务、数据库迁移工具及独立目录的前端静态文件，通过同一端口提供网页和 API，无需额外的 Nginx。
- MySQL 镜像为 8.0.45；仅在 Compose 内部网络访问，不向公网发布 3306 端口。
- 默认项目名 `supervision`，数据库卷 `supervision_mysql_data`、附件卷 `supervision_uploads` 独立持久化。账号、事项、阶段完成情况和附件元数据保存在 MySQL，附件内容保存在附件卷。
- 容器启动自动执行未应用的数据库迁移，再初始化管理员；已有管理员密码、角色和状态不会被覆盖。调整 `.env` 的初始密码不会修改数据库中的已有账号。
- 应用使用非 root 用户、只读根文件系统、受限权限；日志自动轮转。服务器重启后 Docker 启动时容器自动恢复。

## 日常维护

以下命令均在部署目录执行：

```bash
docker compose ps
docker compose logs --tail=100 app
docker compose logs --tail=100 db
docker compose restart app
docker compose stop
docker compose up -d --wait --wait-timeout 360
```

`docker compose down` 会移除容器与网络，但保留数据卷。**不要执行 `docker compose down -v` 或清理项目数据卷，否则会删除数据库和附件。** 更新镜像前应同时备份两个卷，数据库建议通过 `mysqldump --single-transaction` 进行逻辑备份，并在低写入时段同时备份附件，保留 `.env`。镜像包不是业务数据备份。

若改用独立 Compose 项目名（`docker compose -p 名称`），后续操作必须始终使用相同项目名，否则会建立另一组新数据卷。恢复已有 MySQL 卷时必须使用与该卷一致的数据库密码，单独修改 `.env` 不会更改卷内密码。

本配置提供 HTTP 访问。正式对公网使用时，建议绑定域名并在云负载均衡或反向代理上配置 HTTPS，避免账号和业务数据以明文传输。

## 从源代码重新制作镜像包

在项目根目录执行 `sh deploy/package.sh`，产物为 `dist/supervision-linux-amd64.tar.gz`。打包机需要支持 `docker save --platform`（API 1.48+）的 Docker 版本。若 Docker Hub 不可达，可指定可信镜像代理，例如 `BASE_REGISTRY=docker.m.daocloud.io/library sh deploy/package.sh`。脚本固定生成 amd64 镜像；ARM 云服务器需要另行构建对应架构的应用及 MySQL 镜像，并调整 Compose 中的 `platform` 和镜像标签。

若两个目标镜像已经构建并存在本地，可使用 `EXPORT_ONLY=1 sh deploy/package.sh` 直接离线导出，跳过联网构建和下载。
