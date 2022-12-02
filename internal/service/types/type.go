package types

import (
	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/storage/types"
)

type Neo4jCredentialObject struct {
	DatabaseURI      string `yaml:"databaseUri"`
	DatabaseUsername string `yaml:"databaseUsername"`
	DatabasePassword string `yaml:"databasePassword"`
}

type GetUserResponse struct {
	References []string
	Status     types.Status
	NodeID     string
}

type GetPersonaResponse struct {
	References []string
	Status     types.Status
	NodeID     string
}

type GetPermissionSetResponse struct {
	Binding v1alpha1.AccountRoleBinding
	Status  types.Status
	NodeID  string
}

type GetTeamResponse struct {
	ManagedBy string
	Members   []string
	Personas  []string
	NodeID    string
	Status    types.Status
}
