package utils

import "fmt"

type ErrGetSecret struct {
	Err error
}

func (e *ErrGetSecret) Error() string {
	return fmt.Sprintf("failed to get Secret: %s", e.Err.Error())
}

type ErrFailedConnectPostgres struct {
	Err error
}

func (e *ErrFailedConnectPostgres) Error() string {
	return fmt.Sprintf("failed to connect to PostgreSQL: %s", e.Err.Error())
}
