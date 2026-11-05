-- 三类角色工作台共同使用的基础组织数据。
INSERT INTO departments(code,name) VALUES
('office','办公室'),('confidential','保密办'),('logistics','后保处'),('it','信息中心'),
('cyber','网侦中心'),('education','所学习教育专班'),('political','政治处'),('northwest','西北中心');

-- 演示账号由 scripts/seed 生成 bcrypt 密码后写入，避免在迁移文件中保存明文密码。
-- 初始化一个完整的多阶段流程，其余演示场景位于 003 迁移文件中。
INSERT INTO tasks(title,source,task_type,lead_department_id,owner,progress,expected_progress,status,current_stage,next_node,deadline,description)
SELECT '安全检查','6月1日 · 党委会','阶段性任务',id,'徐超军',45,38,'active','任务执行中','完成第二批园区检查','2026-11-06','组织各园区联合安全检查，跟踪问题整改闭环。' FROM departments WHERE code='office';
SET @task_id=LAST_INSERT_ID();
INSERT INTO task_collaborators(task_id,department_id) SELECT @task_id,id FROM departments WHERE code IN ('confidential','logistics','it');
INSERT INTO task_tags(task_id,tag) VALUES(@task_id,'重点关注'),(@task_id,'跨部门');
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status,completion,completed_at)
SELECT @task_id,'制定检查方案',1,id,'2026-10-10','completed','已形成联合检查工作方案。','2026-10-09 15:00:00' FROM departments WHERE code='office';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status)
SELECT @task_id,'完成第二批园区检查',2,id,'2026-11-06','in_progress' FROM departments WHERE code='office';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status)
SELECT @task_id,'整改闭环及报告',3,id,'2026-11-20','pending' FROM departments WHERE code='office';
