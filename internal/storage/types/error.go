package types

const (
	entityNotFound = "requested object not found"
	internalError  = "internal error occurred"
)

type EntityNotFoundError struct{}

func (e *EntityNotFoundError) Error() string {
	return entityNotFound
}

func IsEntityNotFoundNeo4jErr(err error) bool {
	_, ok := err.(*EntityNotFoundError)
	return ok
}

type InternalError struct{}

func (e *InternalError) Error() string {
	return internalError
}

func IsInternalError(err error) bool {
	_, ok := err.(*InternalError)
	return ok
}
