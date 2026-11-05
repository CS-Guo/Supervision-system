// Package mysqlrepo 使用 MySQL 和 database/sql 实现持久化。
// 涉及多张表的聚合写入统一在本包中使用事务处理。
package mysqlrepo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"supervision-system/internal/domain"
	"supervision-system/internal/repository"

	_ "github.com/go-sql-driver/mysql"
)

// Repo 使用共享数据库连接池实现 repository.Repository 接口。
type Repo struct{ db *sql.DB }

// New 封装已经初始化的数据库连接池。
func New(db *sql.DB) *Repo { return &Repo{db: db} }

// Open 配置有界连接池，并在服务接收请求前验证数据库连接。
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect mysql: %w", err)
	}
	return db, nil
}

// tokenHash 避免数据库内容泄漏时直接暴露可用的 Bearer 令牌。
func tokenHash(t string) string { v := sha256.Sum256([]byte(t)); return hex.EncodeToString(v[:]) }

const userSelect = `SELECT u.id,u.username,u.name,u.password_hash,u.role,u.department_id,COALESCE(d.name,''),u.enabled FROM users u LEFT JOIN departments d ON d.id=u.department_id`

func scanUser(row interface{ Scan(...any) error }) (domain.User, error) {
	var u domain.User
	var dep sql.NullInt64
	var enabled bool
	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.PasswordHash, &u.Role, &dep, &u.Department, &enabled)
	if dep.Valid {
		u.DepartmentID = &dep.Int64
	}
	u.Enabled = enabled
	if errors.Is(err, sql.ErrNoRows) {
		err = repository.ErrNotFound
	}
	return u, err
}
func (r *Repo) FindUserByUsername(c context.Context, n string) (domain.User, error) {
	return scanUser(r.db.QueryRowContext(c, userSelect+` WHERE u.username=?`, n))
}
func (r *Repo) FindUserByID(c context.Context, id int64) (domain.User, error) {
	return scanUser(r.db.QueryRowContext(c, userSelect+` WHERE u.id=?`, id))
}
func (r *Repo) CreateSession(c context.Context, t string, id int64, e time.Time) error {
	_, err := r.db.ExecContext(c, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, tokenHash(t), id, e)
	return err
}
func (r *Repo) DeleteSession(c context.Context, t string) error {
	_, err := r.db.ExecContext(c, `DELETE FROM sessions WHERE token_hash=?`, tokenHash(t))
	return err
}
func (r *Repo) FindSessionUser(c context.Context, t string, now time.Time) (domain.User, error) {
	return scanUser(r.db.QueryRowContext(c, userSelect+` JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=? AND s.expires_at>? AND u.enabled=1`, tokenHash(t), now))
}

const taskSelect = `SELECT t.id,t.title,t.source,t.task_type,t.lead_department_id,d.name,t.owner,t.progress,t.expected_progress,t.status,t.current_stage,t.next_node,COALESCE(DATE_FORMAT(t.deadline,'%Y-%m-%d'),''),t.description,COALESCE(t.leader_comment,''),DATE_FORMAT(t.updated_at,'%Y-%m-%d %H:%i') FROM tasks t JOIN departments d ON d.id=t.lead_department_id`

func scanTask(row interface{ Scan(...any) error }) (domain.Task, error) {
	t := domain.Task{CollaboratorIDs: []int64{}, Collaborators: []string{}, Tags: []string{}}
	err := row.Scan(&t.ID, &t.Title, &t.Source, &t.Type, &t.LeadDepartmentID, &t.Lead, &t.Owner, &t.Progress, &t.Expected, &t.Status, &t.Stage, &t.NextNode, &t.Deadline, &t.Description, &t.Comment, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = repository.ErrNotFound
	}
	return t, err
}
func (r *Repo) ListTasks(c context.Context, u domain.User) ([]domain.Task, error) {
	q := taskSelect
	args := []any{}
	if u.Role == domain.RoleDepartment {
		// 部门账号既能查看本部门牵头任务，也能查看明确分配给本部门的协办任务。
		q += ` WHERE t.lead_department_id=? OR EXISTS(SELECT 1 FROM task_collaborators tc WHERE tc.task_id=t.id AND tc.department_id=?)`
		args = append(args, *u.DepartmentID, *u.DepartmentID)
	}
	q += ` ORDER BY FIELD(t.status,'overdue','due','pending','active','archived'),t.deadline`
	rows, err := r.db.QueryContext(c, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Task{}
	for rows.Next() {
		t, e := scanTask(rows)
		if e != nil {
			return nil, e
		}
		if e = r.enrichTask(c, &t); e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (r *Repo) GetTask(c context.Context, id int64) (domain.Task, error) {
	t, e := scanTask(r.db.QueryRowContext(c, taskSelect+` WHERE t.id=?`, id))
	if e == nil {
		e = r.enrichTask(c, &t)
	}
	return t, e
}

// enrichTask 将规范化存储的协办部门和标签关系展开到 API 聚合对象中。
// 空集合保持为 [] 而不是 null，确保 JSON 接口类型稳定。
func (r *Repo) enrichTask(c context.Context, t *domain.Task) error {
	rows, e := r.db.QueryContext(c, `SELECT d.id,d.name FROM task_collaborators tc JOIN departments d ON d.id=tc.department_id WHERE tc.task_id=?`, t.ID)
	if e != nil {
		return e
	}
	for rows.Next() {
		var id int64
		var n string
		if e = rows.Scan(&id, &n); e != nil {
			rows.Close()
			return e
		}
		t.CollaboratorIDs = append(t.CollaboratorIDs, id)
		t.Collaborators = append(t.Collaborators, n)
	}
	rows.Close()
	rows, e = r.db.QueryContext(c, `SELECT tag FROM task_tags WHERE task_id=?`, t.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var x string
		if e = rows.Scan(&x); e != nil {
			return e
		}
		t.Tags = append(t.Tags, x)
	}
	return rows.Err()
}
func (r *Repo) GetTaskDetail(c context.Context, id int64) (domain.TaskDetail, error) {
	t, e := r.GetTask(c, id)
	if e != nil {
		return domain.TaskDetail{}, e
	}
	rows, e := r.db.QueryContext(c, `SELECT s.id,s.task_id,s.name,s.sequence_no,s.responsible_department_id,d.name,COALESCE(DATE_FORMAT(s.deadline,'%Y-%m-%d'),''),s.status,COALESCE(s.completion,''),COALESCE(u.name,''),s.completed_at FROM task_stages s JOIN departments d ON d.id=s.responsible_department_id LEFT JOIN users u ON u.id=s.completed_by WHERE s.task_id=? ORDER BY s.sequence_no`, id)
	if e != nil {
		return domain.TaskDetail{}, e
	}
	defer rows.Close()
	detail := domain.TaskDetail{Task: t, Stages: []domain.Stage{}}
	for rows.Next() {
		var s domain.Stage
		var completed sql.NullTime
		if e = rows.Scan(&s.ID, &s.TaskID, &s.Name, &s.Sequence, &s.ResponsibleDepartmentID, &s.ResponsibleDepartment, &s.Deadline, &s.Status, &s.Completion, &s.CompletedBy, &completed); e != nil {
			return domain.TaskDetail{}, e
		}
		if completed.Valid {
			s.CompletedAt = &completed.Time
		}
		s.Attachments, e = r.attachments(c, s.ID)
		if e != nil {
			return domain.TaskDetail{}, e
		}
		detail.Stages = append(detail.Stages, s)
	}
	return detail, rows.Err()
}
func (r *Repo) attachments(c context.Context, stageID int64) ([]domain.Attachment, error) {
	rows, e := r.db.QueryContext(c, `SELECT id,stage_id,original_name,content_type,file_size,created_at,storage_key FROM attachments WHERE stage_id=? ORDER BY id`, stageID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Attachment{}
	for rows.Next() {
		var a domain.Attachment
		if e = rows.Scan(&a.ID, &a.StageID, &a.OriginalName, &a.ContentType, &a.Size, &a.CreatedAt, &a.StorageKey); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) CreateTask(c context.Context, in domain.TaskInput) (domain.Task, error) {
	// 任务主记录、关联关系和初始阶段必须作为一个整体同时对外可见。
	tx, e := r.db.BeginTx(c, nil)
	if e != nil {
		return domain.Task{}, e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(c, `INSERT INTO tasks(title,source,task_type,lead_department_id,owner,status,next_node,deadline,description) VALUES(?,?,?,?,?,?,?,?,?)`, in.Title, in.Source, in.Type, in.LeadDepartmentID, in.Owner, value(in.Status, "pending"), in.NextNode, nullDate(in.Deadline), in.Description)
	if e != nil {
		return domain.Task{}, e
	}
	id, _ := res.LastInsertId()
	if e = syncRelations(c, tx, id, in.CollaboratorIDs, in.Tags); e != nil {
		return domain.Task{}, e
	}
	if _, e = tx.ExecContext(c, `INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) VALUES(?,?,?,?,?,'in_progress')`, id, value(in.NextNode, "任务执行"), 1, in.LeadDepartmentID, nullDate(in.Deadline)); e != nil {
		return domain.Task{}, e
	}
	if e = tx.Commit(); e != nil {
		return domain.Task{}, e
	}
	return r.GetTask(c, id)
}
func (r *Repo) UpdateTask(c context.Context, id int64, in domain.TaskInput) (domain.Task, error) {
	if _, e := r.GetTask(c, id); e != nil {
		return domain.Task{}, e
	}
	tx, e := r.db.BeginTx(c, nil)
	if e != nil {
		return domain.Task{}, e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(c, `UPDATE tasks SET title=?,source=?,task_type=?,lead_department_id=?,owner=?,status=?,next_node=?,deadline=?,description=? WHERE id=?`, in.Title, in.Source, in.Type, in.LeadDepartmentID, in.Owner, in.Status, in.NextNode, nullDate(in.Deadline), in.Description, id)
	if e != nil {
		return domain.Task{}, e
	}
	_, _ = res.RowsAffected()
	if e = syncRelations(c, tx, id, in.CollaboratorIDs, in.Tags); e != nil {
		return domain.Task{}, e
	}
	if e = tx.Commit(); e != nil {
		return domain.Task{}, e
	}
	return r.GetTask(c, id)
}
func (r *Repo) CreateStage(c context.Context, taskID int64, in domain.StageInput) (domain.Stage, error) {
	res, e := r.db.ExecContext(c, `INSERT INTO task_stages(task_id,name,sequence_no,responsible_department_id,deadline,status) VALUES(?,?,?,?,?,'pending')`, taskID, in.Name, in.Sequence, in.ResponsibleDepartmentID, nullDate(in.Deadline))
	if e != nil {
		return domain.Stage{}, e
	}
	id, _ := res.LastInsertId()
	var s domain.Stage
	e = r.db.QueryRowContext(c, `SELECT s.id,s.task_id,s.name,s.sequence_no,s.responsible_department_id,d.name,COALESCE(DATE_FORMAT(s.deadline,'%Y-%m-%d'),''),s.status FROM task_stages s JOIN departments d ON d.id=s.responsible_department_id WHERE s.id=?`, id).Scan(&s.ID, &s.TaskID, &s.Name, &s.Sequence, &s.ResponsibleDepartmentID, &s.ResponsibleDepartment, &s.Deadline, &s.Status)
	s.Attachments = []domain.Attachment{}
	return s, e
}
func syncRelations(c context.Context, tx *sql.Tx, id int64, deps []int64, tags []string) error {
	// 在事务内整体替换规模较小的管理关系集合，避免差异更新只成功一部分。
	if _, e := tx.ExecContext(c, `DELETE FROM task_collaborators WHERE task_id=?`, id); e != nil {
		return e
	}
	for _, d := range deps {
		if _, e := tx.ExecContext(c, `INSERT INTO task_collaborators(task_id,department_id) VALUES(?,?)`, id, d); e != nil {
			return e
		}
	}
	if _, e := tx.ExecContext(c, `DELETE FROM task_tags WHERE task_id=?`, id); e != nil {
		return e
	}
	for _, t := range tags {
		if _, e := tx.ExecContext(c, `INSERT INTO task_tags(task_id,tag) VALUES(?,?)`, id, t); e != nil {
			return e
		}
	}
	return nil
}
func (r *Repo) SetTaskAction(c context.Context, id int64, a, comment string) error {
	var q string
	var args []any
	switch a {
	case "approve":
		q = `UPDATE tasks SET status='active',current_stage='任务执行中' WHERE id=?`
		args = []any{id}
	case "archive":
		q = `UPDATE tasks SET status='archived',current_stage='已归档' WHERE id=?`
		args = []any{id}
	case "comment":
		q = `UPDATE tasks SET leader_comment=? WHERE id=?`
		args = []any{comment, id}
	case "star":
		q = `INSERT INTO task_tags(task_id,tag) VALUES(?,'重点关注') ON DUPLICATE KEY UPDATE tag=tag`
		args = []any{id}
	case "unstar":
		q = `DELETE FROM task_tags WHERE task_id=? AND tag='重点关注'`
		args = []any{id}
	default:
		return fmt.Errorf("invalid action")
	}
	res, e := r.db.ExecContext(c, q, args...)
	if e == nil {
		_, _ = res.RowsAffected()
	}
	return e
}
func (r *Repo) CompleteStage(c context.Context, taskID, stageID int64, in domain.StageCompletion, u domain.User) error {
	// 阶段完成情况、附件元数据和任务总进度必须原子提交。
	// 如果事务失败，服务层负责删除已经写入的附件文件。
	tx, e := r.db.BeginTx(c, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	status := "in_progress"
	var at any = nil
	if in.Completed {
		status = "completed"
		at = time.Now()
	}
	res, e := tx.ExecContext(c, `UPDATE task_stages SET completion=?,status=?,completed_by=?,completed_at=? WHERE id=? AND task_id=?`, in.Completion, status, u.ID, at, stageID, taskID)
	if e != nil {
		return e
	}
	_, _ = res.RowsAffected()
	for _, a := range in.Attachments {
		if _, e = tx.ExecContext(c, `INSERT INTO attachments(task_id,stage_id,storage_key,original_name,content_type,file_size,uploaded_by) VALUES(?,?,?,?,?,?,?)`, taskID, stageID, a.StorageKey, a.OriginalName, a.ContentType, a.Size, u.ID); e != nil {
			return e
		}
	}
	stageName := ""
	// 将当前阶段名称冗余到任务主表，避免任务列表渲染时额外关联阶段表。
	_ = tx.QueryRowContext(c, `SELECT name FROM task_stages WHERE id=?`, stageID).Scan(&stageName)
	_, e = tx.ExecContext(c, `UPDATE tasks SET progress=?,current_stage=?,status=IF(?=100,'pending','active') WHERE id=?`, in.Progress, stageName, in.Progress, taskID)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (r *Repo) GetAttachment(c context.Context, id int64) (domain.Attachment, int64, error) {
	var a domain.Attachment
	var taskID int64
	e := r.db.QueryRowContext(c, `SELECT id,stage_id,original_name,content_type,file_size,created_at,storage_key,task_id FROM attachments WHERE id=?`, id).Scan(&a.ID, &a.StageID, &a.OriginalName, &a.ContentType, &a.Size, &a.CreatedAt, &a.StorageKey, &taskID)
	if errors.Is(e, sql.ErrNoRows) {
		e = repository.ErrNotFound
	}
	return a, taskID, e
}

func (r *Repo) ListDepartments(c context.Context) ([]domain.Department, error) {
	rows, e := r.db.QueryContext(c, `SELECT id,name,code FROM departments WHERE enabled=1 ORDER BY id`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Department{}
	for rows.Next() {
		var d domain.Department
		if e = rows.Scan(&d.ID, &d.Name, &d.Code); e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *Repo) ListUsers(c context.Context) ([]domain.User, error) {
	rows, e := r.db.QueryContext(c, userSelect+` ORDER BY u.id`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		u, e := scanUser(rows)
		if e != nil {
			return nil, e
		}
		u.PasswordHash = ""
		out = append(out, u)
	}
	return out, rows.Err()
}
func (r *Repo) CreateUser(c context.Context, in domain.UserInput, h string) (domain.User, error) {
	res, e := r.db.ExecContext(c, `INSERT INTO users(username,name,password_hash,role,department_id,enabled) VALUES(?,?,?,?,?,1)`, in.Username, in.Name, h, in.Role, in.DepartmentID)
	if isDuplicate(e) {
		return domain.User{}, repository.ErrConflict
	}
	if e != nil {
		return domain.User{}, e
	}
	id, _ := res.LastInsertId()
	return r.FindUserByID(c, id)
}
func (r *Repo) UpdateUser(c context.Context, id int64, in domain.UserInput, h string) (domain.User, error) {
	if _, e := r.FindUserByID(c, id); e != nil {
		return domain.User{}, e
	}
	parts := []string{"name=?", "role=?", "department_id=?"}
	args := []any{in.Name, in.Role, in.DepartmentID}
	if in.Enabled != nil {
		parts = append(parts, "enabled=?")
		args = append(args, *in.Enabled)
	}
	if h != "" {
		parts = append(parts, "password_hash=?")
		args = append(args, h)
	}
	args = append(args, id)
	res, e := r.db.ExecContext(c, `UPDATE users SET `+strings.Join(parts, ",")+` WHERE id=?`, args...)
	if e != nil {
		return domain.User{}, e
	}
	_, _ = res.RowsAffected()
	return r.FindUserByID(c, id)
}
func (r *Repo) ListAudits(c context.Context) ([]domain.Audit, error) {
	rows, e := r.db.QueryContext(c, `SELECT id,DATE_FORMAT(created_at,'%Y-%m-%d %H:%i'),user_name,action,target FROM audit_logs ORDER BY id DESC LIMIT 500`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Audit{}
	for rows.Next() {
		var a domain.Audit
		if e = rows.Scan(&a.ID, &a.Time, &a.User, &a.Action, &a.Target); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *Repo) AddAudit(c context.Context, u domain.User, a, t string) error {
	_, e := r.db.ExecContext(c, `INSERT INTO audit_logs(user_id,user_name,action,target) VALUES(?,?,?,?)`, u.ID, u.Name, a, t)
	return e
}
func value(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
func nullDate(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func isDuplicate(e error) bool { return e != nil && strings.Contains(e.Error(), "Duplicate entry") }
