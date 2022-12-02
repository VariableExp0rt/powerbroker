package fake

import (
	"net/url"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type MockSession struct {
	MockReadTransaction  func(neo4j.TransactionWork, ...func(*neo4j.TransactionConfig)) (interface{}, error)
	MockWriteTransaction func(neo4j.TransactionWork, ...func(*neo4j.TransactionConfig)) (interface{}, error)
	MockBeginTransaction func(configurers ...func(*neo4j.TransactionConfig)) (neo4j.Transaction, error)
	MockRun              func(cypher string, params map[string]interface{}, configurers ...func(*neo4j.TransactionConfig)) (neo4j.Result, error)
	MockLastBookmark     func() string
	MockClose            func() error
}

func (_m MockSession) ReadTransaction(tx neo4j.TransactionWork, configurers ...func(*neo4j.TransactionConfig)) (interface{}, error) {
	return _m.MockReadTransaction(tx, configurers...)
}

func (_m MockSession) WriteTransaction(tx neo4j.TransactionWork, configurers ...func(*neo4j.TransactionConfig)) (interface{}, error) {
	return _m.MockWriteTransaction(tx, configurers...)
}

func (_m MockSession) BeginTransaction(configurers ...func(*neo4j.TransactionConfig)) (neo4j.Transaction, error) {
	return _m.MockBeginTransaction(configurers...)
}

func (_m MockSession) LastBookmark() string {
	return _m.MockLastBookmark()
}

func (_m MockSession) Run(cypher string, params map[string]interface{}, configurers ...func(*neo4j.TransactionConfig)) (neo4j.Result, error) {
	return _m.MockRun(cypher, params, configurers...)
}

func (_m MockSession) Close() error {
	return _m.MockClose()
}

type MockDriver struct {
	MockNewSession         func(neo4j.SessionConfig) neo4j.Session
	MockSession            func(accessMode neo4j.AccessMode, bookmarks ...string) (neo4j.Session, error)
	MockTarget             func() url.URL
	MockVerifyConnectivity func() error
	MockClose              func() error
}

func (_m *MockDriver) NewSession(config neo4j.SessionConfig) neo4j.Session {
	return _m.MockNewSession(config)
}

func (_m *MockDriver) Session(accessMode neo4j.AccessMode, bookmarks ...string) (neo4j.Session, error) {
	return _m.MockSession(accessMode, bookmarks...)
}

func (_m *MockDriver) VerifyConnectivity() error {
	return _m.MockVerifyConnectivity()
}

func (_m *MockDriver) Target() url.URL {
	return _m.MockTarget()
}

func (_m *MockDriver) Close() error {
	return _m.MockClose()
}
