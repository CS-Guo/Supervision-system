# 数据库设计

数据库目标为 MySQL 8.0+，使用 InnoDB、`utf8mb4` 和外键约束。建表语句位于 `backend/migrations/001_init.sql`。

## 核心关系

```text
departments 1 --- n users
departments 1 --- n tasks (lead_department_id)
tasks n --- n departments (task_collaborators)
tasks 1 --- n task_stages
task_stages 1 --- n attachments
users 1 --- n sessions
users 1 --- n audit_logs
```

## 表职责

| 表 | 职责 | 关键约束或索引 |
| --- | --- | --- |
| `departments` | 部门主数据 | 部门编码、名称唯一 |
| `users` | 登录账号和角色 | 用户名唯一；部门账号关联部门 |
| `sessions` | 服务端登录会话 | 只保存令牌 SHA-256 摘要；按过期时间索引 |
| `tasks` | 督办事项主表 | 按状态、截止日期、牵头部门索引 |
| `task_collaborators` | 事项协办部门 | 任务与部门联合主键 |
| `task_tags` | 事项标签 | 任务与标签联合主键 |
| `task_stages` | 任务阶段、责任部门和完成情况 | 任务内阶段序号唯一；责任部门和状态索引 |
| `attachments` | 阶段附件元数据 | 随机存储键唯一；关联任务和阶段 |
| `audit_logs` | 关键业务操作留痕 | 按创建时间索引；保留操作时用户名快照 |
| `schema_migrations` | 已执行迁移版本 | 迁移文件名主键 |

附件二进制不存入 MySQL，避免数据库备份膨胀。文件使用随机存储键保存，原文件名仅作为元数据；任何下载都需要先根据附件反查任务，再执行角色和部门范围校验。

阶段填报和附件元数据使用同一数据库事务提交；如果事务失败，服务会删除已经写入文件存储的文件，避免孤儿文件。
