package transaction

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j/db"
)

var (
	StatusAvailable   = "available"
	StatusUnavailable = "unavailable"
	StatusDeleted     = "deleted"
)

/*

Transactions are split into two functions
- functions suffixed with TxFunc satisfy neo4j.TransactionWork.
- functions prefixed with CRUD ops without the suffix which invoke a collection of the above.
- the latter functions are exported.

Some considerations:
- check*TxFunc is the only set of functions which return values
	used for comparison in the providers *.Observe() method. All
	others return summaries which are largely ignored.
- error handling is done by creating function satisfying resource.ErrorIs interface
	and checking that specific error in the ExternalClient interface function of
		the provider.
*/

func IsConstraintViolationNeo4jErr(err error) bool {
	nerr, _ := err.(*db.Neo4jError)
	return nerr.Code == "Neo.ClientError.Schema.ConstraintViolation"
}

type EntityNotFoundError struct {
	Entity string
}

func (e *EntityNotFoundError) Error() string {
	return fmt.Sprintf("entity with uuid = %s not found", e.Entity)
}

func IsEntityNotFoundNeo4jErr(err error) bool {
	_, ok := err.(*EntityNotFoundError)
	return ok
}

type InternalError struct {
	Message string
}

func (e *InternalError) Error() string {
	return e.Message
}

type AtProviderResponse struct {
	ID     string
	Status string
}

func (ap *AtProviderResponse) GetID() string {
	return ap.ID
}

func (ap *AtProviderResponse) GetStatus() string {
	return ap.Status
}

func AddTeamTxFunc(teamName string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		CREATE (t:Team {uuid: apoc.create.uuid(), name: $teamName})
		RETURN t.uuid as uuid
		`, map[string]interface{}{
			"teamName": teamName,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

func AddUserTxFunc(userName string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		CREATE (u:User {uuid: apoc.create.uuid(), name: $userName})
		RETURN u.uuid as uuid
		`, map[string]interface{}{
			"userName": userName,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

func AddPersonaTxFunc(name string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		CREATE (p:Persona {uuid: apoc.create.uuid(), name: $name})
		RETURN p.uuid as uuid
		`, map[string]interface{}{
			"name": name,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

func AddPermissionSetTxFunc(name string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		CREATE (p:PermissionSet {uuid: apoc.create.uuid(), name: $name})
		RETURN p.uuid as uuid
		`, map[string]interface{}{
			"name": name,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

func AddAccountTxFunc(accountId, accountAlias, accountClass string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MERGE (:Account {id: $accountId, alias: $accountAlias, class: $accountClass})
		`, map[string]interface{}{
			"accountId":    accountId,
			"accountAlias": accountAlias,
			"accountClass": accountClass,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func AddRoleTxFunc(name string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MERGE (:Role {name: $roleName})
		`, map[string]interface{}{
			"roleName": name,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func DeleteUserTxFunc(userUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (u:User {uuid: $userUuid})
		DETACH DELETE u
		`, map[string]interface{}{
			"userUuid": userUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func DeletePersonaTxFunc(personaUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (p:Persona {uuid: $personaUuid})
		DETACH DELETE p
		`, map[string]interface{}{
			"personaUuid": personaUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func DeleteTeamTxFunc(teamUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (t:Team {uuid: $teamUuid})
		DETACH DELETE t
		`, map[string]interface{}{
			"teamUuid": teamUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func AddTeamPersonaRelationTxFunc(teamUuid string, personaRefs []string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (t:Team {uuid: $teamUuid})
		UNWIND personaRefs as persona
		WITH t, persona
		MATCH (p:Persona {uuid: persona})
		MERGE (t)-[:INHERITS]->(p)
		`, map[string]interface{}{
			"teamUuid":    teamUuid,
			"personaRefs": personaRefs,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

// Adds a relationship edge from a user to a team if that user isn't already managing said team
func AddTeamMemberRelationTxFunc(teamUuid string, memberRefs []string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (t:Team {uuid: $teamUuid})
		UNWIND memberRefs as member
		WITH member, t
		MATCH (u:User {uuid: member})
		WITH
		CASE
			WHEN (t)-[:MANAGED_BY]->(u) THEN 1
			ELSE 0
		END AS is_already_manager
		WHERE is_already_manager = 0
		MERGE (t)<-[:MEMBER_OF]-(u)
		`, map[string]interface{}{
			"teamUuid":   teamUuid,
			"memberRefs": memberRefs,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

// Adds a relationship edge from a manager (also a user) of a team if said user is not already
// a member of said team.
// TODO: if adding team managed by, also create a relationship to each persona referenced
// by the team from the user expressing [:NO_INHERITS]
func AddTeamManagedByRelationTxFunc(teamUuid, managerUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (t:Team {uuid: $teamUuid}), (u:User {uuid: $managerUuid})
		WITH
		CASE
			WHEN (u)-[:MEMBER_OF]->(t) THEN 1
			ELSE 0
		END AS is_already_member
		WHERE is_already_member = 0
		SET u.isManager = true
		MERGE (t)-[:MANAGED_BY]->(u)
		`, map[string]interface{}{
			"teamUuid":    teamUuid,
			"managerUuid": managerUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func AddManagerPersonaNoInheritRelationTxFunc(teamUuid, managerUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (p:Persona)<-[:INHERITS]-(t:Team {uuid: $teamUuid})-[:MANAGED_BY]->(u:User {uuid: $managerUuid})
		MERGE (u)-[:NO_INHERITS]->(p)
		`, map[string]interface{}{
			"teamUuid":    teamUuid,
			"managerUuid": managerUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func UpdateTeamTxFunc(teamUuid, manager string, personaRefs, memberRefs []string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (t:Team {uuid: $teamUuid})
		CALL {
    		WITH t
    		MATCH p=()-[*]-(t)-[*]-()
    		FOREACH (x IN relationships(p) | DELETE x)
		}

		UNWIND personaRefs AS persona
		CALL {
    		WITH t, persona
    		MATCH (p:Persona {uuid: persona})
    		MERGE (t)-[:INHERITS]-(p)
		}

		UNWIND memberRefs AS member
		CALL {
    		WITH t, member
    		MATCH (u:User {uuid: member})
    		MERGE (t)<-[:MEMBER_OF]-(u)
		}

		CALL {
			WITH t
			MATCH (u:User {uuid: $managerUuid})
			MERGE (t)-[:MANAGED_BY]->(u)
		}
		`, map[string]interface{}{
			"teamUuid":    teamUuid,
			"memberRefs":  memberRefs,
			"personaRefs": personaRefs,
			"managerUuid": manager,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func GetTeamTxFunc(teamUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (team:Team {uuid: $teamUuid})
		WITH team
		MATCH (member:User)-[:MEMBER_OF]->(team),
    		(manager:User)<-[:MANAGED_BY]-(team),
    		(persona:Persona)<-[:INHERITS]-(team)

		RETURN collect(member.uuid) AS members,
			manager.uuid AS manager,
			collect(persona.uuid) AS personas
		`, map[string]interface{}{
			"teamUuid": teamUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

// Creates a relationship between the provided permissionSet with the corresponding
// account and role
func AddPermissionSetAccountRoleRelationTxFunc(permissionSetUuid, accountId, roleName string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (p:PermissionSet {uuid: $permissionSetUuid}), (ac:Account {id: $accountId}), (r:Role {name: $roleName})
		WITH p as permissionset, ac as account, r as role
		MERGE (account)<-[:DELEGATES_ACCESS_TO]-(permissionset)-[:DELEGATES_ACCESS_WITH]->(role)
		`, map[string]interface{}{
			"permissionSetUuid": permissionSetUuid,
			"accountId":         accountId,
			"roleName":          roleName,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

// Creates a relationship between the provided User and the referenced
// Personas (one or many)
func AddUserPersonaRelationTxFunc(userUuid string, personaRefs []string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		UNWIND $personaRefs as persona
		WITH persona
		MATCH (u:User {uuid: $userUuid}), (p:Persona {uuid: persona})
		MERGE (u)-[:GRANTED]->(p)
		`, map[string]interface{}{
			"userUuid":    userUuid,
			"personaRefs": personaRefs,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

// Creates one or more relationships between the provided Persona
// and the permissionSet it should be delegated (one or many)
func AddPersonaPermissionSetRelationTxFunc(personaUuid string, permissionSetRefs []string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		UNWIND $permissionSetRefs as ref
		WITH ref
		MATCH (p:Persona {uuid: $personaUuid}), (pe:PermissionSet {uuid: ref})
		MERGE (pe)-[:ATTACHED_TO]->(p)
		`, map[string]interface{}{
			"personaUuid":       personaUuid,
			"permissionSetRefs": permissionSetRefs,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func UpdatePermissionSetTxFunc(permissionSetUuid, name, accountId, accountAlias, accountClass, roleName string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (p:PermissionSet {uuid: $permissionSetUuid}), (ac:Account {id: $accountId}), (ro:Role {name: $roleName})
		WITH p as permissionset, ac as account, ro as role
		SET account.id = $accountId, role.name = $roleName, permissionset.name = $name,
			account.alias = $accountAlias, account.class = $accountClass
		MERGE (account)<-[:DELEGATES_ACCESS_TO]-(PermissionSet)-[:DELEGATES_ACCESS_WITH]->(role)
		`, map[string]interface{}{
			"permissionSetUuid": permissionSetUuid,
			"name":              name,
			"accountId":         accountId,
			"accountAlias":      accountAlias,
			"accountClass":      accountClass,
			"roleName":          roleName,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func UpdatePersonaTxFunc(personaName string, permissionSetRefs []string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (p:Persona {name: $personaName})<-[r:ATTACHED_TO]-(pe:PermissionSet)
		DELETE r

		WITH p
		UNWIND $permissionSetUuids as permissionSet
		MATCH (pe:PermissionSet {uuid: permissionSet})
		MERGE (p)<-[:ATTACHED_TO]-(pe)
		`, map[string]interface{}{
			"permissionSetUuids": permissionSetRefs,
			"personaName":        personaName,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func GetPersonaTxFunc(personaUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (:Persona {uuid: $personaUuid})<--(p:PermissionSet)
		RETURN collect(p.uuid) as permissionSetRefs
		`, map[string]interface{}{
			"personaUuid": personaUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

func UpdateUserTxFunc(userName string, personaRefs []string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (u:User {name: $userName})-[r:DESIGNATED]->(:Persona)
		DELETE r

		WITH u
		UNWIND $personaUuids as persona
		MATCH (p:Persona {uuid: persona})
		MERGE (u)-[:DESIGNATED]->(p)
		`, map[string]interface{}{
			"personaUuids": personaRefs,
			"userName":     userName,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func GetUserTxFunc(userUuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (:User {uuid: $userUuid})-->(p:Persona)
		RETURN collect(p.uuid) as personaRefs
		`, map[string]interface{}{
			"userUuid": userUuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

func DeletePermissionSetTxFunc(uuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (i:PermissionSet {uuid: $uuid})
		DETACH DELETE i
		`, map[string]interface{}{
			"uuid": uuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	}
}

func GetPermissionSetTxFunc(uuid string) neo4j.TransactionWork {
	return func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run(`
		MATCH (r:Role)<--(p:PermissionSet {uuid: $uuid})-->(ac:Account)
		RETURN 	ac.id as id,
			ac.alias as alias,
			ac.class as class,
			r.name as roleName`, map[string]interface{}{
			"uuid": uuid,
		})
		if err != nil {
			return nil, err
		}

		return result.Single()
	}
}

// TODO: PLACEHOLDER, as the tool needs new features
// likely one will be defining teams of users who directly
// inherit personas by default (which may not be too)
// difficult to implement
// func makeTeamRefTxFunc() {}
