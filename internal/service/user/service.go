package user

import (
	powerbroker "github.com/VariableExp0rt/powerbroker/internal/service"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
)

type Service interface {
	CreateUser(username string, personaReferences []string) (string, error)
	GetUser(username string) (*types.GetUserResponse, error)
	UpdateUser(username, userUuid string, personaReferences []string) error
	DeleteUser(username string) error
}

type service struct {
	repository powerbroker.Repository
}

func NewService(repo powerbroker.Repository) Service {
	return &service{repository: repo}
}

func (s *service) CreateUser(username string, personaReferences []string) (string, error) {
	return s.repository.CreateUser(username, personaReferences)
}

func (s *service) GetUser(name string) (*types.GetUserResponse, error) {
	return s.repository.GetUser(name)
}
func (s *service) UpdateUser(name, uuid string, references []string) error {
	return s.repository.UpdateUser(name, uuid, references)
}
func (s *service) DeleteUser(name string) error {
	return s.repository.DeleteUser(name)
}
