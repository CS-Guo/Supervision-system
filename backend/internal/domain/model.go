// Package domain 定义服务层、持久化层和传输层共同使用的业务实体。
// 本包中的类型只表达业务数据，不包含数据库或 HTTP 行为。
package domain

import "time"

const (
	// 角色值会持久化到 MySQL，并作为服务层权限判断的依据。
	RoleLeader     = "leader"
	RoleDepartment = "department"
	RoleAdmin      = "admin"
)

// User 表示可登录系统的账号。只有部门账号必须关联 DepartmentID；
// 领导和管理员账号不限定在某一个部门的数据范围内。
type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	DepartmentID *int64 `json:"departmentId,omitempty"`
	Department   string `json:"department,omitempty"`
	PasswordHash string `json:"-"`
	Enabled      bool   `json:"enabled"`
}

// Department 表示组织部门主数据，供账号、任务和阶段共同引用。
type Department struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// Task 表示任务列表使用的聚合摘要。阶段详情通过 TaskDetail 单独加载，
// 避免列表接口返回不必要的大量阶段和附件数据。
type Task struct {
	ID               int64    `json:"id"`
	Title            string   `json:"title"`
	Source           string   `json:"source"`
	Type             string   `json:"type"`
	LeadDepartmentID int64    `json:"leadDepartmentId"`
	Lead             string   `json:"lead"`
	CollaboratorIDs  []int64  `json:"collaboratorIds"`
	Collaborators    []string `json:"collaborators"`
	Owner            string   `json:"owner"`
	Progress         int      `json:"progress"`
	Expected         int      `json:"expected"`
	Status           string   `json:"status"`
	Stage            string   `json:"stage"`
	NextNode         string   `json:"nextNode"`
	Deadline         string   `json:"deadline"`
	Tags             []string `json:"tags"`
	Description      string   `json:"description"`
	Comment          string   `json:"comment,omitempty"`
	UpdatedAt        string   `json:"updatedAt"`
}

// Stage 表示任务内有顺序的工作阶段。任务牵头部门或该阶段的责任部门
// 可以提交阶段完成情况。
type Stage struct {
	ID                      int64        `json:"id"`
	TaskID                  int64        `json:"taskId"`
	Name                    string       `json:"name"`
	Sequence                int          `json:"sequence"`
	ResponsibleDepartmentID int64        `json:"responsibleDepartmentId"`
	ResponsibleDepartment   string       `json:"responsibleDepartment"`
	Deadline                string       `json:"deadline"`
	Status                  string       `json:"status"`
	Completion              string       `json:"completion,omitempty"`
	CompletedBy             string       `json:"completedBy,omitempty"`
	CompletedAt             *time.Time   `json:"completedAt,omitempty"`
	Attachments             []Attachment `json:"attachments"`
}

// Attachment 保存附件元数据。StorageKey 永远不会序列化给客户端；
// 下载时必须由服务端完成权限校验后再解析实际存储位置。
type Attachment struct {
	ID           int64     `json:"id"`
	StageID      int64     `json:"stageId"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"createdAt"`
	StorageKey   string    `json:"-"`
}

// Audit 表示不可变的关键业务或安全操作记录。
type Audit struct {
	ID     int64  `json:"id"`
	Time   string `json:"time"`
	User   string `json:"user"`
	Action string `json:"action"`
	Target string `json:"target"`
}

// TaskDetail 将任务摘要与按顺序排列的工作阶段组合为任务详情。
type TaskDetail struct {
	Task   Task    `json:"task"`
	Stages []Stage `json:"stages"`
}

// Dashboard 包含当前角色数据范围内的统计结果和部门参考数据。
type Dashboard struct {
	Counts      map[string]int `json:"counts"`
	UpdatedAt   string         `json:"updatedAt"`
	Departments []Department   `json:"departments"`
}

// TaskInput 表示由管理员维护的任务字段。
type TaskInput struct {
	Title            string   `json:"title"`
	Source           string   `json:"source"`
	Type             string   `json:"type"`
	LeadDepartmentID int64    `json:"leadDepartmentId"`
	CollaboratorIDs  []int64  `json:"collaboratorIds"`
	Owner            string   `json:"owner"`
	Status           string   `json:"status"`
	NextNode         string   `json:"nextNode"`
	Deadline         string   `json:"deadline"`
	Tags             []string `json:"tags"`
	Description      string   `json:"description"`
}

// UserInput 用于创建和更新账号。指针字段用于区分“未传值”和
// “明确传入 false 或 null”。
type UserInput struct {
	Username     string `json:"username"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	DepartmentID *int64 `json:"departmentId"`
	Password     string `json:"password"`
	Enabled      *bool  `json:"enabled"`
}

// StageCompletion 表示部门阶段填报需要在同一事务中保存的数据。
type StageCompletion struct {
	Completion  string
	Progress    int
	Completed   bool
	Attachments []NewAttachment
}

// StageInput 表示管理员创建的新任务阶段及其顺序和责任部门。
type StageInput struct {
	Name                    string `json:"name"`
	Sequence                int    `json:"sequence"`
	ResponsibleDepartmentID int64  `json:"responsibleDepartmentId"`
	Deadline                string `json:"deadline"`
}

// NewAttachment 表示文件已经写入附件存储、但元数据尚未提交数据库时的
// 内部临时信息。
type NewAttachment struct {
	OriginalName string
	ContentType  string
	Size         int64
	StorageKey   string
}
