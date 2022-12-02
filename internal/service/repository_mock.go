package service

import (
	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
)

type MockRepository struct {
	MockCreateUser          func(string, []string) (string, error)
	MockGetUser             func(string) (*types.GetUserResponse, error)
	MockUpdateUser          func(userName string, userUuid string, personaRefs []string) error
	MockDeleteUser          func(string) error
	MockCreatePersona       func(personaName string, permissionSetRefs []string) (string, error)
	MockGetPersona          func(string) (*types.GetPersonaResponse, error)
	MockUpdatePersona       func(personaName string, personaUuid string, permissionSetUuids []string) error
	MockDeletePersona       func(string) error
	MockCreatePermissionSet func(string, v1alpha1.AccountRoleBinding) (string, error)
	MockGetPermissionSet    func(string) (*types.GetPermissionSetResponse, error)
	MockUpdatePermissionSet func(permissionSetUuid, crName string, binding v1alpha1.AccountRoleBinding) error
	MockDeletePermissionSet func(string) error
	MockCreateTeam          func(*v1alpha1.TeamParameters) (string, error)
	MockGetTeam             func(string) (*types.GetTeamResponse, error)
	MockUpdateTeam          func(string, *v1alpha1.TeamParameters) error
	MockDeleteTeam          func(string) error
}

func (_m MockRepository) CreateUser(name string, personaReferences []string) (string, error) {
	return _m.MockCreateUser(name, personaReferences)
}

func (_m MockRepository) GetUser(uuid string) (*types.GetUserResponse, error) {
	return _m.MockGetUser(uuid)
}

func (_m MockRepository) UpdateUser(userName string, userUuid string, personaRefs []string) error {
	return _m.MockUpdateUser(userName, userUuid, personaRefs)
}

func (_m MockRepository) DeleteUser(uuid string) error {
	return _m.MockDeleteUser(uuid)
}

func (_m MockRepository) CreatePersona(personaName string, permissionSetRefs []string) (string, error) {
	return _m.MockCreatePersona(personaName, permissionSetRefs)
}

func (_m MockRepository) GetPersona(uuid string) (*types.GetPersonaResponse, error) {
	return _m.MockGetPersona(uuid)
}

func (_m MockRepository) UpdatePersona(personaName string, personaUuid string, permissionSetUuids []string) error {
	return _m.MockUpdatePersona(personaName, personaUuid, permissionSetUuids)
}

func (_m MockRepository) DeletePersona(uuid string) error {
	return _m.MockDeletePersona(uuid)
}

func (_m MockRepository) CreatePermissionSet(name string, binding v1alpha1.AccountRoleBinding) (string, error) {
	return _m.MockCreatePermissionSet(name, binding)
}

func (_m MockRepository) GetPermissionSet(uuid string) (*types.GetPermissionSetResponse, error) {
	return _m.MockGetPermissionSet(uuid)
}

func (_m MockRepository) UpdatePermissionSet(permissionSetUuid, crName string, binding v1alpha1.AccountRoleBinding) error {
	return _m.MockUpdatePermissionSet(permissionSetUuid, crName, binding)
}

func (_m MockRepository) DeletePermissionSet(uuid string) error {
	return _m.MockDeletePermissionSet(uuid)
}

func (_m MockRepository) CreateTeam(tp *v1alpha1.TeamParameters) (string, error) {
	return _m.MockCreateTeam(tp)
}

func (_m MockRepository) GetTeam(uuid string) (*types.GetTeamResponse, error) {
	return _m.MockGetTeam(uuid)
}

func (_m MockRepository) UpdateTeam(uuid string, tp *v1alpha1.TeamParameters) error {
	return _m.MockUpdateTeam(uuid, tp)
}
func (_m MockRepository) DeleteTeam(uuid string) error {
	return _m.MockDeleteTeam(uuid)
}
