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

// StdManager is a standard publication manager. It uses regular
// CREATE/ALTER/DROP PUBLICATION commands to manipulate publications on a
// PostgreSQL server.
type StdManager struct {
	Conn conn
}

// conn interface allows us to mock pgx.
type conn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	PgConn() *pgconn.PgConn
}

func (m *StdManager) CreatePublication(
	ctx context.Context,
	publication *postgresv1alpha1.PostgresPublication,
) (bool, error) {
	query, err := m.buildCreatePublicationQuery(publication)
	if err != nil {
		return false, fmt.Errorf("failed to build create publication query: %w", err)
	}

	_, err = m.Conn.Exec(ctx, query)

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

func (m *StdManager) buildCreatePublicationQuery(
	publication *postgresv1alpha1.PostgresPublication,
) (string, error) {
	publicationIdentifer := pgx.Identifier{publication.Spec.Name}
	conn := m.Conn.PgConn()

	var forTablePart string
	if len(publication.Spec.Tables) == 0 {
		forTablePart = "FOR ALL TABLES"
	} else {
		var tableIdentifiers []string
		for _, table := range publication.Spec.Tables {
			tableIdentifier := pgx.Identifier{table.Schema, table.Name}
			tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
		}

		forTablePart = fmt.Sprintf("FOR TABLE %s", strings.Join(tableIdentifiers, ", "))
	}

	var withPublicationParameterPart string
	if len(publication.Spec.Operations) > 0 {
		escapedOperations, err := conn.EscapeString(strings.Join(publication.Spec.Operations, ", "))
		if err != nil {
			return "", fmt.Errorf("failed to escape string: %w", err)
		}

		withPublicationParameterPart = fmt.Sprintf(
			"WITH (publish = '%s')",
			escapedOperations,
		)
	}

	return strings.Join(
		[]string{
			"CREATE PUBLICATION",
			publicationIdentifer.Sanitize(),
			forTablePart,
			withPublicationParameterPart,
		},
		" ",
	), nil
}

func (m *StdManager) AlterExistingPublication(
	ctx context.Context,
	publication *postgresv1alpha1.PostgresPublication,
) error {
	conn := m.Conn

	var tableIdentifiers []string
	for _, table := range publication.Spec.Tables {
		tableIdentifier := pgx.Identifier{table.Schema, table.Name}
		tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
	}

	setTableQuery := fmt.Sprintf(
		"ALTER PUBLICATION %s SET TABLE %s",
		pgx.Identifier{publication.Spec.Name}.Sanitize(),
		strings.Join(tableIdentifiers, ", "),
	)

	if _, err := conn.Exec(ctx, setTableQuery); err != nil {
		return fmt.Errorf("failed to set table to publication: %w", err)
	}

	var joinedOperations string
	if len(publication.Spec.Operations) > 0 {
		joinedOperations = strings.Join(publication.Spec.Operations, ", ")
	} else {
		joinedOperations = defaultPublishOperations
	}

	sanitizedOperations, err := conn.PgConn().EscapeString(joinedOperations)
	if err != nil {
		return fmt.Errorf("failed to escape operations: %w", err)
	}

	setParamQuery := fmt.Sprintf(
		"ALTER PUBLICATION %s SET (publish = '%s')",
		pgx.Identifier{publication.Spec.Name}.Sanitize(),
		sanitizedOperations,
	)

	if _, err := conn.Exec(
		ctx,
		setParamQuery,
	); err != nil {
		return fmt.Errorf("failed to set publication parameter: %w", err)
	}

	return nil
}

func (m *StdManager) DropPublication(
	ctx context.Context,
	name string,
) error {
	query := fmt.Sprintf("DROP PUBLICATION %s", pgx.Identifier{name}.Sanitize())

	if _, err := m.Conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to drop publication: %w", err)
	}

	return nil
}
