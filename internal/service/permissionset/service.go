package permissionset

import (
	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	powerbroker "github.com/VariableExp0rt/powerbroker/internal/service"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
)

type Service interface {
	CreatePermissionSet(name string, binding v1alpha1.AccountRoleBinding) (string, error)
	GetPermissionSet(name string) (*types.GetPermissionSetResponse, error)
	UpdatePermissionSet(uuid, name string, binding v1alpha1.AccountRoleBinding) error
	DeletePermissionSet(name string) error
}

type service struct {
	repository powerbroker.Repository
}

func NewService(repo powerbroker.Repository) Service {
	return &service{repository: repo}
}

func (s *service) CreatePermissionSet(name string, binding v1alpha1.AccountRoleBinding) (string, error) {
	return s.repository.CreatePermissionSet(name, binding)
}

func (s *service) GetPermissionSet(name string) (*types.GetPermissionSetResponse, error) {
	return s.repository.GetPermissionSet(name)
}
func (s *service) UpdatePermissionSet(uuid, name string, binding v1alpha1.AccountRoleBinding) error {
	return s.repository.UpdatePermissionSet(uuid, name, binding)
}
func (s *service) DeletePermissionSet(name string) error {
	return s.repository.DeletePermissionSet(name)
}
