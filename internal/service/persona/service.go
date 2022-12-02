package persona

import (
	powerbroker "github.com/VariableExp0rt/powerbroker/internal/service"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
)

type Service interface {
	CreatePersona(personaname string, permsetReferences []string) (string, error)
	GetPersona(personaname string) (*types.GetPersonaResponse, error)
	UpdatePersona(personaname, personaUuid string, permsetReferences []string) error
	DeletePersona(personaname string) error
}

type service struct {
	repository powerbroker.Repository
}

func NewService(repo powerbroker.Repository) Service {
	return &service{repository: repo}
}

func (s *service) CreatePersona(personaname string, personaReferences []string) (string, error) {
	return s.repository.CreatePersona(personaname, personaReferences)
}

func (s *service) GetPersona(name string) (*types.GetPersonaResponse, error) {
	return s.repository.GetPersona(name)
}

func (s *service) UpdatePersona(name, uuid string, references []string) error {
	return s.repository.UpdatePersona(name, uuid, references)
}

func (s *service) DeletePersona(name string) error {
	return s.repository.DeletePersona(name)
}
