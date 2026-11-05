// Package service 包含权限校验和业务规则。HTTP 处理器与数据仓储不应重复
// 实现这些判断，确保所有入口使用一致的权限边界。
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"supervision-system/internal/domain"
	"supervision-system/internal/repository"
	"supervision-system/internal/security"
	"supervision-system/internal/storage"
)

// 哨兵错误用于让 HTTP 适配层将业务失败稳定映射为对应状态码。
var ErrUnauthorized = errors.New("unauthorized")
var ErrForbidden = errors.New("forbidden")
var ErrInvalid = errors.New("invalid input")

// Service 协调数据仓储、附件存储和安全策略。
type Service struct {
	repo       repository.Repository
	files      storage.Local
	sessionTTL time.Duration
	maxUpload  int64
}

// New 使用显式传入的基础设施依赖创建应用服务。
func New(repo repository.Repository, files storage.Local, ttl time.Duration, maxUpload int64) *Service {
	return &Service{repo: repo, files: files, sessionTTL: ttl, maxUpload: maxUpload}
}

// Login 校验已启用账号并创建服务端会话。原始令牌只返回给客户端，
// 数据库中仅保存令牌的 SHA-256 摘要。
func (s *Service) Login(c context.Context, username, password string) (string, domain.User, error) {
	u, e := s.repo.FindUserByUsername(c, strings.TrimSpace(username))
	if e != nil || !u.Enabled || !security.CheckPassword(u.PasswordHash, password) {
		return "", domain.User{}, ErrUnauthorized
	}
	token := security.Token()
	if e = s.repo.CreateSession(c, token, u.ID, time.Now().Add(s.sessionTTL)); e != nil {
		return "", domain.User{}, e
	}
	u.PasswordHash = ""
	return token, u, nil
}

// Logout 使当前服务端会话失效。
func (s *Service) Logout(c context.Context, token string) error {
	return s.repo.DeleteSession(c, token)
}

// Session 将未过期的令牌解析为当前仍处于启用状态的账号。
func (s *Service) Session(c context.Context, token string) (domain.User, error) {
	u, e := s.repo.FindSessionUser(c, token, time.Now())
	if errors.Is(e, repository.ErrNotFound) {
		return domain.User{}, ErrUnauthorized
	}
	u.PasswordHash = ""
	return u, e
}

// Tasks 返回指定用户权限范围内可见的任务。
func (s *Service) Tasks(c context.Context, u domain.User) ([]domain.Task, error) {
	return s.repo.ListTasks(c, u)
}

// Dashboard 使用与任务列表相同的数据范围计算角色看板统计。
func (s *Service) Dashboard(c context.Context, u domain.User) (domain.Dashboard, error) {
	tasks, e := s.repo.ListTasks(c, u)
	if e != nil {
		return domain.Dashboard{}, e
	}
	deps, e := s.repo.ListDepartments(c)
	if e != nil {
		return domain.Dashboard{}, e
	}
	counts := map[string]int{"all": len(tasks)}
	for _, t := range tasks {
		counts[t.Status]++
	}
	return domain.Dashboard{Counts: counts, UpdatedAt: time.Now().Format("2006-01-02 15:04"), Departments: deps}, nil
}

// TaskDetail 在返回阶段和附件前先执行部门数据范围校验。
func (s *Service) TaskDetail(c context.Context, u domain.User, id int64) (domain.TaskDetail, error) {
	d, e := s.repo.GetTaskDetail(c, id)
	if e != nil {
		return d, e
	}
	if !canView(u, d.Task) {
		return domain.TaskDetail{}, ErrForbidden
	}
	return d, nil
}

// CreateTask 允许管理员创建任务聚合。
func (s *Service) CreateTask(c context.Context, u domain.User, in domain.TaskInput) (domain.Task, error) {
	if u.Role != domain.RoleAdmin {
		return domain.Task{}, ErrForbidden
	}
	if strings.TrimSpace(in.Title) == "" || in.LeadDepartmentID == 0 {
		return domain.Task{}, ErrInvalid
	}
	t, e := s.repo.CreateTask(c, in)
	if e == nil {
		_ = s.repo.AddAudit(c, u, "新增事项", t.Title)
	}
	return t, e
}

// UpdateTask 允许管理员修改任务主数据和部门分工。
func (s *Service) UpdateTask(c context.Context, u domain.User, id int64, in domain.TaskInput) (domain.Task, error) {
	if u.Role != domain.RoleAdmin {
		return domain.Task{}, ErrForbidden
	}
	t, e := s.repo.UpdateTask(c, id, in)
	if e == nil {
		_ = s.repo.AddAudit(c, u, "更新事项", t.Title)
	}
	return t, e
}

// CreateStage 为现有任务新增一个由管理员定义的阶段。
func (s *Service) CreateStage(c context.Context, u domain.User, taskID int64, in domain.StageInput) (domain.Stage, error) {
	if u.Role != domain.RoleAdmin {
		return domain.Stage{}, ErrForbidden
	}
	if strings.TrimSpace(in.Name) == "" || in.Sequence < 1 || in.ResponsibleDepartmentID == 0 {
		return domain.Stage{}, ErrInvalid
	}
	t, e := s.repo.GetTask(c, taskID)
	if e != nil {
		return domain.Stage{}, e
	}
	stage, e := s.repo.CreateStage(c, taskID, in)
	if e == nil {
		_ = s.repo.AddAudit(c, u, "新增任务阶段", t.Title+" / "+stage.Name)
	}
	return stage, e
}

// TaskAction 是领导和管理员触发任务状态流转的统一入口。
// 数据仓储只负责持久化，不负责判断谁可以触发状态变化。
func (s *Service) TaskAction(c context.Context, u domain.User, id int64, action, comment string) error {
	t, e := s.repo.GetTask(c, id)
	if e != nil {
		return e
	}
	switch action {
	case "approve", "archive":
		if u.Role != domain.RoleLeader && u.Role != domain.RoleAdmin {
			return ErrForbidden
		}
	case "comment":
		if u.Role != domain.RoleLeader || strings.TrimSpace(comment) == "" {
			return ErrForbidden
		}
	case "star", "unstar":
		if u.Role == domain.RoleDepartment {
			return ErrForbidden
		}
		if action == "star" {
			for _, tag := range t.Tags {
				if tag == "重点关注" {
					action = "unstar"
					break
				}
			}
		}
	default:
		return ErrInvalid
	}
	if e = s.repo.SetTaskAction(c, id, action, comment); e == nil {
		_ = s.repo.AddAudit(c, u, "办理: "+action, t.Title)
	}
	return e
}

// CompleteStage 保存部门阶段填报及可选证明材料。只有任务牵头部门或
// 当前阶段责任部门可以提交该阶段的完成情况。
func (s *Service) CompleteStage(c context.Context, u domain.User, taskID, stageID int64, completion string, progress int, completed bool, headers []*multipart.FileHeader) error {
	if u.Role != domain.RoleDepartment && u.Role != domain.RoleAdmin {
		return ErrForbidden
	}
	if strings.TrimSpace(completion) == "" || progress < 0 || progress > 100 {
		return ErrInvalid
	}
	d, e := s.TaskDetail(c, u, taskID)
	if e != nil {
		return e
	}
	var stage *domain.Stage
	for i := range d.Stages {
		if d.Stages[i].ID == stageID {
			stage = &d.Stages[i]
			break
		}
	}
	if stage == nil {
		return repository.ErrNotFound
	}
	if u.Role == domain.RoleDepartment && (u.DepartmentID == nil || (*u.DepartmentID != d.Task.LeadDepartmentID && *u.DepartmentID != stage.ResponsibleDepartmentID)) {
		return ErrForbidden
	}
	attachments := []domain.NewAttachment{}
	saved := []string{}
	// 数据库写入附件元数据前，必须先把文件写入存储。记录所有已写入的
	// 存储键，便于后续校验或事务失败时执行补偿清理。
	for _, h := range headers {
		if h.Size > s.maxUpload {
			cleanup(s.files, saved)
			return fmt.Errorf("%w: attachment too large", ErrInvalid)
		}
		f, e := h.Open()
		if e != nil {
			cleanup(s.files, saved)
			return e
		}
		key, size, e := s.files.Save(f)
		f.Close()
		if e != nil {
			cleanup(s.files, saved)
			return e
		}
		saved = append(saved, key)
		attachments = append(attachments, domain.NewAttachment{OriginalName: filepathBase(h.Filename), ContentType: value(h.Header.Get("Content-Type"), "application/octet-stream"), Size: size, StorageKey: key})
	}
	e = s.repo.CompleteStage(c, taskID, stageID, domain.StageCompletion{Completion: completion, Progress: progress, Completed: completed, Attachments: attachments}, u)
	if e != nil {
		// 数据库是附件关系的事实来源。删除未成功提交为附件记录的文件，
		// 避免文件存储中持续积累孤立文件。
		cleanup(s.files, saved)
		return e
	}
	_ = s.repo.AddAudit(c, u, "填报阶段完成情况", d.Task.Title+" / "+stage.Name)
	return nil
}

// Attachment 重新校验用户对所属任务的访问权限后才解析实际文件。
// 即使知道或猜中附件编号，也不会因此获得下载权限。
func (s *Service) Attachment(c context.Context, u domain.User, id int64) (domain.Attachment, io.ReadCloser, error) {
	a, taskID, e := s.repo.GetAttachment(c, id)
	if e != nil {
		return a, nil, e
	}
	t, e := s.repo.GetTask(c, taskID)
	if e != nil {
		return a, nil, e
	}
	if !canView(u, t) {
		return a, nil, ErrForbidden
	}
	f, e := s.files.Open(a.StorageKey)
	return a, f, e
}

// Departments 返回已启用的组织部门参考数据。
func (s *Service) Departments(c context.Context) ([]domain.Department, error) {
	return s.repo.ListDepartments(c)
}

// Users 向管理员返回账号列表，但不返回密码摘要。
func (s *Service) Users(c context.Context, u domain.User) ([]domain.User, error) {
	if u.Role != domain.RoleAdmin {
		return nil, ErrForbidden
	}
	return s.repo.ListUsers(c)
}

// CreateUser 校验角色与部门的一致性，并对初始密码进行哈希处理。
func (s *Service) CreateUser(c context.Context, u domain.User, in domain.UserInput) (domain.User, error) {
	if u.Role != domain.RoleAdmin {
		return domain.User{}, ErrForbidden
	}
	if len(in.Password) < 8 || !validUserInput(in) || in.Username == "" {
		return domain.User{}, ErrInvalid
	}
	h, e := security.HashPassword(in.Password)
	if e != nil {
		return domain.User{}, e
	}
	created, e := s.repo.CreateUser(c, in, h)
	if e == nil {
		_ = s.repo.AddAudit(c, u, "创建账号", created.Name)
	}
	return created, e
}

// UpdateUser 修改账号资料；只有明确传入新密码时才重新计算密码摘要。
func (s *Service) UpdateUser(c context.Context, u domain.User, id int64, in domain.UserInput) (domain.User, error) {
	if u.Role != domain.RoleAdmin {
		return domain.User{}, ErrForbidden
	}
	if !validUserInput(in) {
		return domain.User{}, ErrInvalid
	}
	h := ""
	var e error
	if in.Password != "" {
		if len(in.Password) < 8 {
			return domain.User{}, ErrInvalid
		}
		h, e = security.HashPassword(in.Password)
		if e != nil {
			return domain.User{}, e
		}
	}
	updated, e := s.repo.UpdateUser(c, id, in, h)
	if e == nil {
		_ = s.repo.AddAudit(c, u, "更新账号", updated.Name)
	}
	return updated, e
}

// Audits 向管理员返回最近的不可变审计记录。
func (s *Service) Audits(c context.Context, u domain.User) ([]domain.Audit, error) {
	if u.Role != domain.RoleAdmin {
		return nil, ErrForbidden
	}
	return s.repo.ListAudits(c)
}

// canView 是任务详情和附件下载共同使用的唯一任务可见性规则。
func canView(u domain.User, t domain.Task) bool {
	if u.Role != domain.RoleDepartment {
		return true
	}
	if u.DepartmentID == nil {
		return false
	}
	if *u.DepartmentID == t.LeadDepartmentID {
		return true
	}
	for _, id := range t.CollaboratorIDs {
		if id == *u.DepartmentID {
			return true
		}
	}
	return false
}
func validRole(r string) bool {
	return r == domain.RoleLeader || r == domain.RoleDepartment || r == domain.RoleAdmin
}
func validUserInput(in domain.UserInput) bool {
	if !validRole(in.Role) || strings.TrimSpace(in.Name) == "" {
		return false
	}
	return in.Role != domain.RoleDepartment || in.DepartmentID != nil
}
func cleanup(l storage.Local, keys []string) {
	for _, k := range keys {
		l.Delete(k)
	}
}
func filepathBase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return "attachment"
	}
	return parts[len(parts)-1]
}
func value(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
