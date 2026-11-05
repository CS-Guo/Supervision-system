# 党委会事项督办系统

前后端分离的党委会事项督办系统。后端使用 Go 和 MySQL，前端为独立单页应用。

## 工程结构

```text
backend/
  cmd/server/       服务启动与依赖装配
  cmd/migrate/      数据库迁移工具
  cmd/seed/         本地演示账号初始化
  internal/domain/  领域模型
  internal/service/ 业务规则与权限校验
  internal/repository/mysql/ MySQL 数据访问与事务
  internal/httpapi/ HTTP 路由、鉴权和参数适配
  internal/storage/ 附件文件存储
  migrations/       版本化数据库结构和演示事项
frontend/           独立前端代码
```

详细表设计见 [数据库设计](docs/database-design.md)。

## 云服务器 Docker 部署

完整离线部署方法见 [Docker 部署说明](docs/docker-deployment.md)。执行 `sh deploy/package.sh` 可构建并导出包含应用和 MySQL 的 Linux amd64 镜像包。服务器导入镜像后运行包内 `sh deploy.sh`，会自动初始化数据库并启动网页服务。云部署使用随机管理员密码，不使用下述本地演示密码。

## 本地启动

1. 创建 MySQL 8 数据库和应用账号，字符集使用 `utf8mb4`。

当前开发机已创建项目专用 Docker 容器：

```bash
docker start supervision-mysql
```

该容器映射 `127.0.0.1:3306`，数据库为 `supervision`，应用账号为 `supervision`。开发密码是 `supervision_dev`，仅用于本机开发环境，不应复制到测试或生产环境。
2. 配置连接串：

```bash
cd backend
export MYSQL_DSN='supervision:your_password@tcp(127.0.0.1:3306)/supervision?parseTime=true&charset=utf8mb4&loc=Local'
```

3. 执行迁移和演示账号初始化：

```bash
go run ./cmd/migrate
go run ./cmd/seed
```

4. 启动服务：

```bash
go run ./cmd/server
```

上述命令也可以分别使用 `make migrate`、`make seed`、`make run`。质量检查使用 `make test` 和 `make vet`。

打开 <http://127.0.0.1:8080>。演示账号为 `leader`、`office`、`political` 和 `admin`，初始密码均为 `12345678`，首次正式部署后应立即修改。

## 业务权限

| 角色 | 权限 |
| --- | --- |
| 领导 | 查看全部事项、统计、批示、确认、归档、重点关注 |
| 部门 | 查看本部门牵头及协办事项；牵头部门或阶段责任部门可提交阶段完成情况和附件 |
| 管理员 | 维护事项、阶段、账号、部门分工并查看审计日志 |

权限同时在服务层校验；隐藏前端按钮不作为安全边界。附件下载也会重新校验当前账号是否有权查看所属任务。

## 配置

复制 `backend/.env.example` 作为配置参考。服务从环境变量读取配置，不会自动读取 `.env`：

- `MYSQL_DSN`：必填，MySQL DSN。
- `APP_ADDR`：监听地址，默认 `:8080`。
- `FRONTEND_DIR`：前端目录，默认 `../frontend`。
- `UPLOAD_DIR`：附件目录，默认 `data/uploads`。
- `MAX_UPLOAD_MB`：单次请求附件总大小，默认 20MB，最大 100MB。

生产环境应把附件目录挂载到持久卷或将 `internal/storage` 替换为对象存储实现，并由反向代理启用 TLS、请求限流和安全响应头。
