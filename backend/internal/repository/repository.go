// Package repository 定义业务层使用的持久化端口。
// 在不修改服务层的情况下，可以使用 MySQL 或其他持久化存储实现该接口。
package repository

import (
	"context"
	"errors"
	"time"

	"supervision-system/internal/domain"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

// Repository 是 Service 所需的完整持久化契约。所有方法接收 context，
// 以便请求取消信号和数据库超时能够继续向下传递。
type Repository interface {
	FindUserByUsername(context.Context, string) (domain.User, error)
	FindUserByID(context.Context, int64) (domain.User, error)
	CreateSession(context.Context, string, int64, time.Time) error
	DeleteSession(context.Context, string) error
	FindSessionUser(context.Context, string, time.Time) (domain.User, error)
	ListTasks(context.Context, domain.User) ([]domain.Task, error)
	GetTask(context.Context, int64) (domain.Task, error)
	GetTaskDetail(context.Context, int64) (domain.TaskDetail, error)
	CreateTask(context.Context, domain.TaskInput) (domain.Task, error)
	UpdateTask(context.Context, int64, domain.TaskInput) (domain.Task, error)
	CreateStage(context.Context, int64, domain.StageInput) (domain.Stage, error)
	SetTaskAction(context.Context, int64, string, string) error
	CompleteStage(context.Context, int64, int64, domain.StageCompletion, domain.User) error
	GetAttachment(context.Context, int64) (domain.Attachment, int64, error)
	ListDepartments(context.Context) ([]domain.Department, error)
	ListUsers(context.Context) ([]domain.User, error)
	CreateUser(context.Context, domain.UserInput, string) (domain.User, error)
	UpdateUser(context.Context, int64, domain.UserInput, string) (domain.User, error)
	ListAudits(context.Context) ([]domain.Audit, error)
	AddAudit(context.Context, domain.User, string, string) error
}
