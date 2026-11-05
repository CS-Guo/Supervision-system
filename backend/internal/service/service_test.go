package service

import (
	"supervision-system/internal/domain"
	"testing"
)

func TestDepartmentTaskVisibility(t *testing.T) {
	officeID := int64(1)
	politicalID := int64(2)
	task := domain.Task{LeadDepartmentID: politicalID, CollaboratorIDs: []int64{officeID}}
	if !canView(domain.User{Role: domain.RoleDepartment, DepartmentID: &officeID}, task) {
		t.Fatal("collaborating department should see task")
	}
	other := int64(3)
	if canView(domain.User{Role: domain.RoleDepartment, DepartmentID: &other}, task) {
		t.Fatal("unrelated department should not see task")
	}
	if !canView(domain.User{Role: domain.RoleLeader}, task) || !canView(domain.User{Role: domain.RoleAdmin}, task) {
		t.Fatal("leader and admin should see all tasks")
	}
}
func TestDepartmentWithoutDepartmentCannotViewTask(t *testing.T) {
	if canView(domain.User{Role: domain.RoleDepartment}, domain.Task{LeadDepartmentID: 1}) {
		t.Fatal("unbound department user should not see tasks")
	}
}
