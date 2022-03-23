package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
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
		It("should create a table", func() {
			const publicationName = "jobs"
			createBarebonesTable(ctx, "jobs")

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
					Columns: []postgresv1alpha1.PostgresColumn{
						{
							Name:       "id",
							PrimaryKey: true,
							DataType:   "uuid",
							Nullable:   false,
						},
					},
				},
			}

			err := k8sClient.Create(ctx, table)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresTable resource")

			waitForTable(ctx, tableIdentifier)
		})
	})
})

// waitForTable waits for a table with the given name to be created within
// PostgreSQL, until timeout is reached.
func waitForTable(ctx context.Context, tableIdentifier postgresv1alpha1.PostgresIdentifier) {
	Eventually(func() int {
		row := postgresConn.QueryRow(
			ctx,
			"SELECT COUNT(*) AS count FROM pg_tables WHERE schemaname = $1 AND tablename = $2",
			tableIdentifier.Schema,
			tableIdentifier.Name,
		)

		var count int
		err := row.Scan(&count)
		Expect(err).NotTo(HaveOccurred())

		return count
	}).Should(Equal(1), "created table count should be 1")
}
