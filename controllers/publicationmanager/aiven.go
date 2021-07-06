package publicationmanager

import (
	"context"
	"fmt"
	"strings"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
)

// AivenManager is a publication manager for Aiven PostgreSQL. It uses the
// aiven-extras extension to manage publications. See:
// https://github.com/aiven/aiven-extras
type AivenManager struct {
	StdManager
}

func (m *AivenManager) CreatePublication(
	ctx context.Context,
	publication *postgresv1alpha1.PostgresPublication,
) (bool, error) {
	query := "SELECT * FROM aiven_extras.pg_create_publication($1, $2, VARIADIC $3)"

	var tableIdentifiers []string
	for _, table := range publication.Spec.Tables {
		tableIdentifier := pgx.Identifier{table.Schema, table.Name}
		tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
	}

	operations := defaultPublishOperations
	if len(publication.Spec.Operations) > 0 {
		operations = strings.Join(publication.Spec.Operations, ", ")
	}

	_, err := m.Conn.Exec(ctx, query, publication.Spec.Name, operations, tableIdentifiers)

	publicationCreated := true
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == pgerrcode.DuplicateObject {
			publicationCreated = false
		} else {
			return false, fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return publicationCreated, nil
}

func (m *AivenManager) AlterExistingPublication(
	ctx context.Context,
	publication *postgresv1alpha1.PostgresPublication,
) error {
	return m.StdManager.AlterExistingPublication(ctx, publication)
}

func (m *AivenManager) DropPublication(
	ctx context.Context,
	name string,
) error {
	return m.StdManager.DropPublication(ctx, name)
}
