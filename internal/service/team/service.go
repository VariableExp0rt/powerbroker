package team

import (
	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	powerbroker "github.com/VariableExp0rt/powerbroker/internal/service"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
)

type Service interface {
	CreateTeam(*v1alpha1.TeamParameters) (string, error)
	GetTeam(teamname string) (*types.GetTeamResponse, error)
	UpdateTeam(teamname string, teamparams *v1alpha1.TeamParameters) error
	DeleteTeam(teamname string) error
}

type service struct {
	repository powerbroker.Repository
}

func NewService(repo powerbroker.Repository) Service {
	return &service{repository: repo}
}

func (s *service) CreateTeam(params *v1alpha1.TeamParameters) (string, error) {
	return s.repository.CreateTeam(params)
}

func (s *service) GetTeam(name string) (*types.GetTeamResponse, error) {
	return s.repository.GetTeam(name)
}

func (s *service) UpdateTeam(name string, params *v1alpha1.TeamParameters) error {
	return s.repository.UpdateTeam(name, params)
}

func (s *service) DeleteTeam(name string) error {
	return s.repository.DeleteTeam(name)
}
