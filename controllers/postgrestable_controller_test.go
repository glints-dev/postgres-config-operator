package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	utils "github.com/glints-dev/postgres-config-operator/controllers/utils"
)

var _ = Context("Inside of a new Postgres instance", func() {
	ctx := context.Background()
	SetupPostgresContainer(ctx)
	SetupPostgresConnection(ctx)

	var namespace *corev1.Namespace

	BeforeEach(func() {
		namespace = CreateTestNamespace(ctx)
		CreatePostgresSecret(ctx, namespace.Name)
	})

	AfterEach(func() {
		defer DeleteNamespace(ctx, namespace)
	})

	Describe("Tables", func() {
		tests := []struct {
			name  string
			input []postgresv1alpha1.PostgresColumn
		}{
			{
				name: "should create a table",
				input: []postgresv1alpha1.PostgresColumn{
					{
						Name:       "id",
						PrimaryKey: true,
						DataType:   "uuid",
						Nullable:   false,
					},
				},
			},
			{
				name:  "should create empty table",
				input: []postgresv1alpha1.PostgresColumn{},
			},
			{
				name: "should create table with composite primary keys",
				input: []postgresv1alpha1.PostgresColumn{
					{
						Name:       "foo_id",
						PrimaryKey: true,
						DataType:   "uuid",
						Nullable:   false,
					},
					{
						Name:       "bar_id",
						PrimaryKey: true,
						DataType:   "uuid",
						Nullable:   false,
					},
				},
			},
			{
				name: "should create table with nullable columns",
				input: []postgresv1alpha1.PostgresColumn{
					{
						Name:       "id",
						PrimaryKey: true,
						DataType:   "uuid",
						Nullable:   false,
					},
					{
						Name:       "foo",
						PrimaryKey: false,
						DataType:   "uuid",
						Nullable:   true,
					},
				},
			},
		}

		for _, test := range tests {
			It(test.name, func() {
				tableIdentifier := postgresv1alpha1.PostgresIdentifier{
					Name:   "test",
					Schema: "my_schema",
				}

				table := &postgresv1alpha1.PostgresTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "table-test",
						Namespace: namespace.Name,
					},
					Spec: postgresv1alpha1.PostgresTableSpec{
						PostgresRef:        PostgresContainerRef(ctx),
						PostgresIdentifier: tableIdentifier,
						Columns:            test.input,
					},
				}

				err := k8sClient.Create(ctx, table)
				Expect(err).NotTo(HaveOccurred(), "failed to create PostgresTable resource")

				waitForColumns(ctx, tableIdentifier, test.input)
			})
		}
	})
})

// waitForColumns waits for the given slice of columns to be created in
// PostgresSQL, until timeout is reached.
func waitForColumns(
	ctx context.Context,
	tableIdentifier postgresv1alpha1.PostgresIdentifier,
	columns []postgresv1alpha1.PostgresColumn,
) {
	Eventually(postgresConn).Should(utils.HaveColumns(
		ctx,
		tableIdentifier.Schema,
		tableIdentifier.Name,
		columns,
	), "created column should exist")
}
