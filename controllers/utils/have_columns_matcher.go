package utils

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
)

// HaveColumns checks if the PostgreSQL database connected through actual has
// columns in the given schema and table matching the specification.
func HaveColumns(
	ctx context.Context,
	schema string,
	table string,
	columns []postgresv1alpha1.PostgresColumn,
) types.GomegaMatcher {
	return &haveColumnsMatcher{
		ctx:     ctx,
		schema:  schema,
		table:   table,
		columns: columns,
	}
}

type haveColumnsMatcher struct {
	ctx context.Context

	schema  string
	table   string
	columns []postgresv1alpha1.PostgresColumn

	columnsFromPg     map[string]columnFromPg
	primaryKeysFromPg map[string]struct{}
}

type columnFromPg struct {
	isNullable bool
}

func (matcher *haveColumnsMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}

	conn, ok := actual.(*pgx.Conn)
	if !ok {
		return false, fmt.Errorf("Expected *pgx.Conn. Got: \n%s", format.Object(actual, 1))
	}

	if err := matcher.fetchColumnDataFromPg(conn); err != nil {
		return false, fmt.Errorf("Expected to be able to fetch column data from PostgreSQL. Got: \n%s", format.Object(err, 1))
	}

	for _, column := range matcher.columns {
		columnFromPg, ok := matcher.columnsFromPg[column.Name]
		if !ok {
			return false, fmt.Errorf("Expected column %s to be found", column.Name)
		}

		if columnFromPg.isNullable != column.Nullable {
			if column.Nullable {
				return false, fmt.Errorf("Expected column %s to be nullable, but it is not", column.Name)
			} else {
				return false, fmt.Errorf("Expected column %s to be non-nullable, but it is", column.Name)
			}
		}

		_, primaryKeyInPg := matcher.primaryKeysFromPg[column.Name]
		if column.PrimaryKey != primaryKeyInPg {
			if column.PrimaryKey {
				return false, fmt.Errorf("Expected column %s to be part of primary key, but it is not", column.Name)
			} else {
				return false, fmt.Errorf("Expected column %s to not be part of primary key, but it is", column.Name)
			}
		}
	}

	return true, nil
}

func (matcher *haveColumnsMatcher) fetchColumnDataFromPg(conn *pgx.Conn) error {
	var columnNames []string
	for _, column := range matcher.columns {
		columnNames = append(columnNames, column.Name)
	}

	matcher.columnsFromPg = make(map[string]columnFromPg)
	matcher.primaryKeysFromPg = make(map[string]struct{})

	var columnName string
	var isNullable bool
	var isPrimaryKey bool
	if _, err := conn.QueryFunc(
		matcher.ctx,
		`
		SELECT
			information_schema.columns.column_name,
			is_nullable = 'YES',
			information_schema.table_constraints.constraint_type IS NOT NULL AS is_primary_key
		FROM information_schema.columns
		LEFT OUTER JOIN information_schema.constraint_column_usage ON
			information_schema.constraint_column_usage.table_schema = information_schema.columns.table_schema AND
			information_schema.constraint_column_usage.table_name = information_schema.columns.table_name AND
			information_schema.constraint_column_usage.column_name = information_schema.columns.column_name
		LEFT OUTER JOIN information_schema.table_constraints ON
			information_schema.table_constraints.constraint_schema = information_schema.constraint_column_usage.constraint_schema AND
			information_schema.table_constraints.constraint_name = information_schema.constraint_column_usage.constraint_name AND
			information_schema.table_constraints.constraint_type = 'PRIMARY KEY'
		WHERE
			information_schema.columns.table_schema = $1 AND
			information_schema.columns.table_name = $2 AND
			information_schema.columns.column_name = ANY($3)
		`,
		[]interface{}{
			matcher.schema,
			matcher.table,
			columnNames,
		},
		[]interface{}{
			&columnName,
			&isNullable,
			&isPrimaryKey,
		},
		func(row pgx.QueryFuncRow) error {
			matcher.columnsFromPg[columnName] = columnFromPg{
				isNullable,
			}

			if isPrimaryKey {
				matcher.primaryKeysFromPg[columnName] = struct{}{}
			}

			return nil
		},
	); err != nil {
		return fmt.Errorf("Expected query to succeed. Got: \n%w", err)
	}

	return nil
}

func (matcher *haveColumnsMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(
		"Expected columns to match specification. Got:\n%s",
		format.Object(matcher.columnsFromPg, 1),
	)
}

func (matcher *haveColumnsMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(
		"Unexpected match of column specification. Got:\n%s",
		format.Object(matcher.columnsFromPg, 1),
	)
}
