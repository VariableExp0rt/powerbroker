package storage

import (
	"fmt"
	"strings"

	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
	"github.com/VariableExp0rt/powerbroker/internal/storage/neo4j/transaction"
	storetypes "github.com/VariableExp0rt/powerbroker/internal/storage/types"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/pkg/errors"
)

type Neo4jDB struct {
	Driver neo4j.Driver
}

func (db *Neo4jDB) CreateUser(userName string, personaRefs []string) (string, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	var out interface{}
	var uuid string
	var err error

	if out, err = session.WriteTransaction(transaction.AddUserTxFunc(userName)); err != nil {
		return "", err
	}

	record := out.(*neo4j.Record)

	uuid, ok := record.Values[0].(string)
	if !ok {
		return "", errors.New("no user was created")
	}

	if _, err = session.WriteTransaction(transaction.AddUserPersonaRelationTxFunc(uuid, personaRefs)); err != nil {
		return "", err
	}

	return uuid, nil
}

func (db *Neo4jDB) GetUser(userUuid string) (*types.GetUserResponse, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	out, err := session.ReadTransaction(transaction.GetUserTxFunc(userUuid))
	if err != nil {
		if strings.Contains(err.Error(), "Result contains no more records") {
			return &types.GetUserResponse{
				NodeID:     userUuid,
				Status:     storetypes.StatusDeleted,
				References: nil,
			}, &storetypes.EntityNotFoundError{}
		}
		return &types.GetUserResponse{
			References: nil,
			NodeID:     userUuid,
			Status:     storetypes.StatusUnavailable,
		}, err
	}

	// TODO: look at get persona to see how to handle edge cases

	switch out.(type) {
	case nil:
		return &types.GetUserResponse{
				NodeID:     userUuid,
				Status:     storetypes.StatusUnavailable,
				References: nil,
			},
			&storetypes.EntityNotFoundError{}
	case *neo4j.Record:
		record := out.(*neo4j.Record)
		references := record.Values[0].([]interface{})

		personaSlc := make([]string, len(references))
		for i, v := range references {
			personaSlc[i] = fmt.Sprint(v)
		}

		if len(personaSlc) == 0 {
			return &types.GetUserResponse{
					NodeID:     userUuid,
					Status:     storetypes.StatusDeleted,
					References: nil,
				},
				&storetypes.EntityNotFoundError{}
		}

		return &types.GetUserResponse{
				NodeID:     userUuid,
				Status:     storetypes.StatusAvailable,
				References: personaSlc,
			},
			nil
	}

	return &types.GetUserResponse{
		References: nil,
		NodeID:     userUuid,
		Status:     storetypes.StatusAvailable,
	}, &transaction.InternalError{Message: "internal server error"}
}

func (db *Neo4jDB) UpdateUser(userName string, userUuid string, personaRefs []string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	if _, err := session.WriteTransaction(transaction.UpdateUserTxFunc(userName, personaRefs)); err != nil {
		return err
	}

	return nil
}

func (db *Neo4jDB) DeleteUser(userUuid string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	if _, err := session.WriteTransaction(transaction.DeleteUserTxFunc(userUuid)); err != nil {
		return err
	}

	return nil
}

func (db *Neo4jDB) CreatePersona(personaName string, permissionSetRefs []string) (string, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	var out interface{}
	var uuid string
	var err error

	if out, err = session.WriteTransaction(transaction.AddPersonaTxFunc(personaName)); err != nil {
		return "", err
	}

	record := out.(*neo4j.Record)

	uuid, ok := record.Values[0].(string)
	if !ok {
		return "", errors.New("no persona was created")
	}

	if _, err := session.WriteTransaction(transaction.AddPersonaPermissionSetRelationTxFunc(uuid, permissionSetRefs)); err != nil {
		return "", err
	}

	return uuid, nil
}

func (db *Neo4jDB) GetPersona(uuid string) (*types.GetPersonaResponse, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	out, err := session.ReadTransaction(transaction.GetPersonaTxFunc(uuid))
	if err != nil {
		if strings.Contains(err.Error(), "Result contains no more records") {
			return &types.GetPersonaResponse{
				NodeID:     uuid,
				Status:     storetypes.StatusDeleted,
				References: nil,
			}, &storetypes.EntityNotFoundError{}
		}
		return &types.GetPersonaResponse{
			NodeID:     uuid,
			Status:     storetypes.StatusUnavailable,
			References: nil,
		}, err
	}

	// In another horrifying story of the way in which the golang driver
	// works, forcing us to range over the returned []interface{} values
	// which cannot be type asserted to strings and appended normally
	// because of the way memory layout works for []iface{}. Please forgive me
	// for this...
	// https://stackoverflow.com/questions/44027826/convert-interface-to-string-in-golang

	// When collect() is used, it will not error on collect(elements_length_zero) as x
	// so 'x' is returned as an empty slice, this means we need to check the returned
	// []iface{} once converted into []string, and if len == 0, deletion request was
	// successful
	switch out.(type) {
	case nil:
		return &types.GetPersonaResponse{
				NodeID:     uuid,
				Status:     storetypes.StatusUnavailable,
				References: nil,
			},
			&storetypes.EntityNotFoundError{}
	case *neo4j.Record:
		record := out.(*neo4j.Record)
		permissionSets, _ := record.Values[0].([]interface{})

		permissionSetSlc := make([]string, len(permissionSets))
		for i, v := range permissionSets {
			permissionSetSlc[i] = fmt.Sprint(v)
		}

		if len(permissionSetSlc) == 0 {
			return &types.GetPersonaResponse{
					NodeID:     uuid,
					Status:     storetypes.StatusDeleted,
					References: nil,
				},
				&storetypes.EntityNotFoundError{}
		}

		return &types.GetPersonaResponse{
			NodeID:     uuid,
			Status:     storetypes.StatusAvailable,
			References: permissionSetSlc,
		}, nil
	}

	return &types.GetPersonaResponse{
			NodeID:     uuid,
			Status:     storetypes.StatusUnavailable,
			References: nil,
		},
		&transaction.InternalError{Message: "internal server error"}

}

func (db *Neo4jDB) UpdatePersona(personaName string, personaUuid string, permissionSetUuids []string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.WriteTransaction(transaction.UpdatePersonaTxFunc(personaName, permissionSetUuids))

	return err
}

func (db *Neo4jDB) DeletePersona(personaUuid string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.WriteTransaction(transaction.DeletePersonaTxFunc(personaUuid))

	return err
}

func (db *Neo4jDB) CreatePermissionSet(name string, binding v1alpha1.AccountRoleBinding) (string, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	var out interface{}
	var uuid string
	var err error

	if _, err = session.WriteTransaction(transaction.AddAccountTxFunc(binding.Account, binding.Alias, binding.AccountClass)); err != nil {
		return "", err
	}

	if _, err = session.WriteTransaction(transaction.AddRoleTxFunc(binding.RoleName)); err != nil {
		return "", err
	}

	if out, err = session.WriteTransaction(transaction.AddPermissionSetTxFunc(name)); err != nil {
		return "", err
	}

	record := out.(*neo4j.Record)

	uuid, ok := record.Values[0].(string)
	if !ok {
		return "", errors.New("no permissionset was created")
	}

	if _, err := session.WriteTransaction(transaction.AddPermissionSetAccountRoleRelationTxFunc(uuid, binding.Account, binding.RoleName)); err != nil {
		return "", err
	}

	return uuid, nil
}

func (db *Neo4jDB) GetPermissionSet(uuid string) (*types.GetPermissionSetResponse, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	out, err := session.ReadTransaction(transaction.GetPermissionSetTxFunc(uuid))
	if err != nil {
		if strings.Contains(err.Error(), "Result contains no more records") {
			return &types.GetPermissionSetResponse{
				Binding: v1alpha1.AccountRoleBinding{},
				Status:  storetypes.StatusDeleted,
				NodeID:  uuid,
			}, &storetypes.EntityNotFoundError{}
		}
		return &types.GetPermissionSetResponse{
			Binding: v1alpha1.AccountRoleBinding{},
			Status:  storetypes.StatusUnavailable,
			NodeID:  uuid,
		}, err
	}

	// NOTE: I feel horrified that the driver doesn't propagate
	// status codes for read only transactions, which is why
	// this hideous code is necessary. Essentially, when we return
	// result.Single() from the neo4j.TransactionWork function, it will
	// not throw `Neo.Neo4jDBError.Statement.EntityNotFound`. Instead, it
	// returns a nil record as an INTERFACE?!?! which makes it
	// impossible to type assert without a panic i.e. nil interface
	// conversion.

	switch out.(type) {
	case nil:
		return &types.GetPermissionSetResponse{
				Binding: v1alpha1.AccountRoleBinding{},
				Status:  storetypes.StatusDeleted,
				NodeID:  uuid,
			},
			&storetypes.EntityNotFoundError{}
	case *neo4j.Record:
		record := out.(*neo4j.Record)
		return &types.GetPermissionSetResponse{
				Binding: v1alpha1.AccountRoleBinding{
					Account:      record.Values[0].(string),
					Alias:        record.Values[1].(string),
					AccountClass: record.Values[2].(string),
					RoleName:     record.Values[3].(string),
				},
				Status: storetypes.StatusAvailable,
				NodeID: uuid,
			},
			nil
	}

	return &types.GetPermissionSetResponse{
			Binding: v1alpha1.AccountRoleBinding{},
			Status:  storetypes.StatusUnavailable,
			NodeID:  uuid,
		},
		&transaction.InternalError{Message: "internal server error"}
}

func (db *Neo4jDB) UpdatePermissionSet(permissionSetUuid, crName string, binding v1alpha1.AccountRoleBinding) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.WriteTransaction(transaction.UpdatePermissionSetTxFunc(permissionSetUuid,
		crName,
		binding.Account,
		binding.Alias,
		binding.AccountClass,
		binding.RoleName))
	if err != nil {
		return err
	}

	return nil
}

func (db *Neo4jDB) DeletePermissionSet(permissionSetUuid string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.WriteTransaction(transaction.DeletePermissionSetTxFunc(permissionSetUuid))
	if err != nil {
		return err
	}

	return nil
}

func (db *Neo4jDB) CreateTeam(teamparams *v1alpha1.TeamParameters) (string, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	var out interface{}
	var uuid string
	var err error

	if out, err = session.WriteTransaction(transaction.AddTeamTxFunc(teamparams.Name)); err != nil {
		return "", err
	}

	record := out.(*neo4j.Record)

	uuid, ok := record.Values[0].(string)
	if !ok {
		return "", errors.New("no team was created")
	}

	if _, err = session.WriteTransaction(transaction.AddTeamManagedByRelationTxFunc(uuid, teamparams.ManagedBy.User)); err != nil {
		return "", err
	}

	if _, err = session.WriteTransaction(transaction.AddTeamMemberRelationTxFunc(uuid, teamparams.Members)); err != nil {
		return "", err
	}

	if _, err = session.WriteTransaction(transaction.AddTeamPersonaRelationTxFunc(uuid, teamparams.Personas)); err != nil {
		return "", err
	}

	return uuid, nil
}

func (db *Neo4jDB) GetTeam(uuid string) (*types.GetTeamResponse, error) {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	out, err := session.ReadTransaction(transaction.GetPersonaTxFunc(uuid))
	if err != nil {
		if strings.Contains(err.Error(), "Result contains no more records") {
			return &types.GetTeamResponse{
				NodeID:    uuid,
				Status:    storetypes.StatusDeleted,
				ManagedBy: "",
				Members:   nil,
				Personas:  nil,
			}, &storetypes.EntityNotFoundError{}
		}
		return &types.GetTeamResponse{
			NodeID:    uuid,
			Status:    storetypes.StatusUnavailable,
			ManagedBy: "",
			Members:   nil,
			Personas:  nil,
		}, err
	}

	// In another horrifying story of the way in which the golang driver
	// works, forcing us to range over the returned []interface{} values
	// which cannot be type asserted to strings and appended normally
	// because of the way memory layout works for []iface{}. Please forgive me
	// for this...
	// https://stackoverflow.com/questions/44027826/convert-interface-to-string-in-golang

	// When collect() is used, it will not error on collect(elements_length_zero) as x
	// so 'x' is returned as an empty slice, this means we need to check the returned
	// []iface{} once converted into []string, and if len == 0, deletion request was
	// successful
	switch out.(type) {
	case nil:
		return &types.GetTeamResponse{
				NodeID:    uuid,
				Status:    storetypes.StatusUnavailable,
				ManagedBy: "",
				Members:   nil,
				Personas:  nil,
			},
			&storetypes.EntityNotFoundError{}
	case *neo4j.Record:
		record := out.(*neo4j.Record)
		members, _ := record.Values[0].([]interface{})
		managedBy, _ := record.Values[1].(string)
		personas, _ := record.Values[2].([]interface{})

		memberSlc := make([]string, len(members))
		for i, v := range members {
			memberSlc[i] = fmt.Sprint(v)
		}

		personaSlc := make([]string, len(personas))
		for i, v := range personas {
			personaSlc[i] = fmt.Sprint(v)
		}

		if len(memberSlc) == 0 && len(personaSlc) == 0 && managedBy == "" {
			return &types.GetTeamResponse{
					NodeID:    uuid,
					Status:    storetypes.StatusDeleted,
					ManagedBy: "",
					Members:   nil,
					Personas:  nil,
				},
				&storetypes.EntityNotFoundError{}
		}

		return &types.GetTeamResponse{
			NodeID:    uuid,
			Status:    storetypes.StatusAvailable,
			ManagedBy: managedBy,
			Members:   memberSlc,
			Personas:  personaSlc,
		}, nil
	}

	return &types.GetTeamResponse{
			NodeID:    uuid,
			Status:    storetypes.StatusUnavailable,
			ManagedBy: "",
			Members:   nil,
			Personas:  nil,
		},
		&transaction.InternalError{Message: "internal server error"}
}

func (db *Neo4jDB) UpdateTeam(uuid string, teamparams *v1alpha1.TeamParameters) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	if _, err := session.WriteTransaction(transaction.UpdateTeamTxFunc(uuid,
		teamparams.ManagedBy.User,
		teamparams.Members,
		teamparams.Personas)); err != nil {
		return err
	}

	return nil
}

func (db *Neo4jDB) DeleteTeam(uuid string) error {
	session := db.Driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	if _, err := session.WriteTransaction(transaction.DeleteTeamTxFunc(uuid)); err != nil {
		return err
	}

	return nil
}
