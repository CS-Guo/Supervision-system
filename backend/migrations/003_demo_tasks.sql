INSERT INTO tasks(title,source,task_type,lead_department_id,owner,progress,expected_progress,status,current_stage,next_node,deadline,description)
SELECT '支部调研座谈','6月22日 · 党委会','阶段性任务',id,'李娜',28,100,'overdue','任务执行中','完成第一批座谈','2026-10-19','围绕基层党组织建设开展专题调研，形成问题清单和改进建议。' FROM departments WHERE code='political';
SET @task_id=LAST_INSERT_ID();
INSERT INTO task_collaborators(task_id,department_id) SELECT @task_id,id FROM departments WHERE code='office';
INSERT INTO task_tags(task_id,tag) VALUES(@task_id,'重点关注'),(@task_id,'需要领导批示');
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status,completion,completed_at) SELECT @task_id,'完成调研方案',1,id,'2026-09-30','completed','调研方案已经审定并印发。','2026-09-29 10:00:00' FROM departments WHERE code='political';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) SELECT @task_id,'完成第一批座谈',2,id,'2026-10-19','in_progress' FROM departments WHERE code='political';

INSERT INTO tasks(title,source,task_type,lead_department_id,owner,progress,expected_progress,status,current_stage,next_node,deadline,description)
SELECT '司聘专项','6月22日 · 党委会','短期即办',id,'李娜',75,100,'overdue','征求协办意见','提交专题材料','2026-11-02','完成专项情况摸底并提交专题材料。' FROM departments WHERE code='political';
SET @task_id=LAST_INSERT_ID();
INSERT INTO task_collaborators(task_id,department_id) SELECT @task_id,id FROM departments WHERE code='office';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status,completion,completed_at) SELECT @task_id,'情况摸底',1,id,'2026-10-20','completed','已完成相关人员和岗位情况摸底。','2026-10-20 17:00:00' FROM departments WHERE code='political';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) SELECT @task_id,'提交专题材料',2,id,'2026-11-02','in_progress' FROM departments WHERE code='political';

INSERT INTO tasks(title,source,task_type,lead_department_id,owner,progress,expected_progress,status,current_stage,next_node,deadline,description)
SELECT '离职人员情况分析','6月15日 · 党委会','长期任务',id,'李娜',62,81,'due','任务执行中','形成分析初稿','2026-11-07','分析近年人员离职情况，提出队伍稳定建议。' FROM departments WHERE code='political';
SET @task_id=LAST_INSERT_ID();
INSERT INTO task_collaborators(task_id,department_id) SELECT @task_id,id FROM departments WHERE code='office';
INSERT INTO task_tags(task_id,tag) VALUES(@task_id,'长期任务');
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) SELECT @task_id,'形成分析初稿',1,id,'2026-11-07','in_progress' FROM departments WHERE code='political';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) SELECT @task_id,'专题研究审议',2,id,'2026-11-30','pending' FROM departments WHERE code='political';

INSERT INTO tasks(title,source,task_type,lead_department_id,owner,progress,expected_progress,status,current_stage,next_node,deadline,description)
SELECT '支部检查','6月8日 · 读书班','阶段性任务',id,'李娜',0,0,'pending','待领导确认','确定检查范围','2026-11-10','制定检查方案，明确检查范围和工作标准。' FROM departments WHERE code='education';
SET @task_id=LAST_INSERT_ID();
INSERT INTO task_collaborators(task_id,department_id) SELECT @task_id,id FROM departments WHERE code='political';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) SELECT @task_id,'确定检查范围',1,id,'2026-11-10','in_progress' FROM departments WHERE code='education';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) SELECT @task_id,'实施支部检查',2,id,'2026-11-25','pending' FROM departments WHERE code='education';

INSERT INTO tasks(title,source,task_type,lead_department_id,owner,progress,expected_progress,status,current_stage,next_node,deadline,description)
SELECT '习近平新时代中国特色社会主义思想体系梳理','6月22日 · 党委会','阶段性任务',id,'李娜',100,100,'archived','已归档','节点已完成','2026-10-26','形成体系化学习资料。' FROM departments WHERE code='political';
SET @task_id=LAST_INSERT_ID();
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status,completion,completed_at) SELECT @task_id,'完成体系梳理',1,id,'2026-10-26','completed','已形成体系化学习资料并完成归档。','2026-10-26 16:00:00' FROM departments WHERE code='political';

INSERT INTO tasks(title,source,task_type,lead_department_id,owner,progress,expected_progress,status,current_stage,next_node,deadline,description)
SELECT '西北中心专项','6月29日 · 党委会','阶段性任务',id,'西北中心联络员',0,0,'pending','待填计划','提交任务计划','2026-11-15','落实党委会专项工作安排。' FROM departments WHERE code='northwest';
SET @task_id=LAST_INSERT_ID();
INSERT INTO task_collaborators(task_id,department_id) SELECT @task_id,id FROM departments WHERE code='office';
INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) SELECT @task_id,'提交任务计划',1,id,'2026-11-15','in_progress' FROM departments WHERE code='northwest';
