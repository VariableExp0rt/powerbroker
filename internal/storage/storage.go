package storage

import (
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
	neo4jstore "github.com/VariableExp0rt/powerbroker/internal/storage/neo4j"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func NewNeo4jStorage(creds []byte, conf ...func(*neo4j.Config)) (*neo4jstore.Neo4jDB, error) {
	var co types.Neo4jCredentialObject

	err := yaml.Unmarshal(creds, &co)
	if err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal secret data")
	}

	driver, err := neo4j.NewDriver(co.DatabaseURI, neo4j.BasicAuth(co.DatabaseUsername, co.DatabasePassword, ""), conf...)
	if err != nil {
		return nil, errors.Wrap(err, "cannot establish authenticated session")
	}

	return &neo4jstore.Neo4jDB{
		Driver: driver,
	}, nil
}
