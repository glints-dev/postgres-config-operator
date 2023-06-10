package controllers

import (
	"context"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	"github.com/glints-dev/postgres-config-operator/controllers/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Context("Inside of a new Postgres instance", func() {
	ctx := context.Background()
	testutils.SetupPostgresContainer(ctx)
	testutils.SetupPostgresConnection(ctx)

	var namespace *corev1.Namespace

	BeforeEach(func() {
		namespace = CreateTestNamespace(ctx)
		CreatePostgresSecret(ctx, namespace.Name)
	})

	AfterEach(func() {
		defer DeleteNamespace(ctx, namespace)
	})

	Describe("Grants", func() {
		It("should grant permissions to given database", func() {
			_, err := testutils.PostgresConn.Exec(ctx, "CREATE DATABASE test_db")
			Expect(err).NotTo(HaveOccurred())

			_, err = testutils.PostgresConn.Exec(ctx, "CREATE ROLE test_role")
			Expect(err).NotTo(HaveOccurred())

			grant := &postgresv1alpha1.PostgresGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "grant-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresGrantSpec{
					PostgresRef: testutils.PostgresContainerRef(ctx),
					Databases:   []string{"test_db"},
					Role:        "test_role",
				},
			}

			err = k8sClient.Create(ctx, grant)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				row := testutils.PostgresConn.QueryRow(
					ctx,
					"SELECT has_database_privilege($1, $2, $3)",
					"test_role",
					"test_db",
					"CONNECT",
				)

				var hasPrivilege bool
				err := row.Scan(&hasPrivilege)
				Expect(err).NotTo(HaveOccurred())

				return hasPrivilege
			}).Should(BeTrue())
		})
	})
})
