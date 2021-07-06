package publicationmanager

import (
	"context"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
)

// defaultPublishOperations is the default value of the "publish" parameter for
// convenience of the actual manager implementations. It's documented here:
// https://www.postgresql.org/docs/current/sql-createpublication.html
const defaultPublishOperations = "insert, update, delete, truncate"

// Manager represents something that is able to manage publications on a
// PostgreSQL server. This interface is created to cater for the different ways
// to manage PostgreSQL publications depending on the specific
// database-as-a-service provider.
type Manager interface {
	CreatePublication(
		ctx context.Context,
		publication *postgresv1alpha1.PostgresPublication,
	) (created bool, err error)

	AlterExistingPublication(
		ctx context.Context,
		publication *postgresv1alpha1.PostgresPublication,
	) error

	DropPublication(
		ctx context.Context,
		name string,
	) error
}
