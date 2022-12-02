package service

import (
	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
)

// Repository is an interface which must be satisfied by storage
// objects that implement the interface. Currently only neo4j - planned
// extension for SpiceDB.
type Repository interface {
	CreateUser(string, []string) (string, error)
	GetUser(string) (*types.GetUserResponse, error)
	UpdateUser(userName string, userUuid string, personaRefs []string) error
	DeleteUser(string) error
	CreatePersona(personaName string, permissionSetRefs []string) (string, error)
	GetPersona(string) (*types.GetPersonaResponse, error)
	UpdatePersona(personaName string, personaUuid string, permissionSetUuids []string) error
	DeletePersona(string) error
	CreatePermissionSet(string, v1alpha1.AccountRoleBinding) (string, error)
	GetPermissionSet(string) (*types.GetPermissionSetResponse, error)
	UpdatePermissionSet(permissionSetUuid, crName string, binding v1alpha1.AccountRoleBinding) error
	DeletePermissionSet(string) error
	CreateTeam(*v1alpha1.TeamParameters) (string, error)
	GetTeam(string) (*types.GetTeamResponse, error)
	UpdateTeam(string, *v1alpha1.TeamParameters) error
	DeleteTeam(string) error
}
