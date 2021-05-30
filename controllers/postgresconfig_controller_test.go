package controllers

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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

	Describe("Publications", func() {
		It("should create a publication for the given table", func() {
			const publicationName = "jobs"
			createBarebonesTable(ctx, "jobs")

			postgresConfig := &postgresv1alpha1.PostgresConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresConfigSpec{
					PostgresRef: PostgresContainerRef(ctx),
					Publications: []postgresv1alpha1.PostgresPublication{
						{
							Name: publicationName,
							Tables: []postgresv1alpha1.PostgresTableIdentifier{
								{
									Name:   "jobs",
									Schema: "public",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

			waitForPublication(ctx, publicationName)
		})

		It("should create a publication for the given table for subset of operations", func() {
			const publicationName = "jobs"
			createBarebonesTable(ctx, "jobs")

			postgresConfig := &postgresv1alpha1.PostgresConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresConfigSpec{
					PostgresRef: PostgresContainerRef(ctx),
					Publications: []postgresv1alpha1.PostgresPublication{
						{
							Name: publicationName,
							Tables: []postgresv1alpha1.PostgresTableIdentifier{
								{
									Name:   "jobs",
									Schema: "public",
								},
							},
							Operations: []string{"insert", "delete"},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

			waitForPublication(ctx, publicationName)

			Eventually(func() bool {
				row := postgresConn.QueryRow(
					ctx,
					"SELECT pubinsert, pubupdate, pubdelete FROM pg_publication WHERE pubname = $1",
					publicationName,
				)

				var pubinsert bool
				var pubupdate bool
				var pubdelete bool
				err := row.Scan(&pubinsert, &pubupdate, &pubdelete)
				Expect(err).NotTo(HaveOccurred(), "failed to scan row")

				return pubinsert && pubdelete && !pubupdate
			}).Should(BeTrue())
		})

		It("should add new table to existing publication", func() {
			const publicationName = "publication_test"
			createBarebonesTable(ctx, "jobs")
			createBarebonesTable(ctx, "users")

			postgresConfig := &postgresv1alpha1.PostgresConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresConfigSpec{
					PostgresRef: PostgresContainerRef(ctx),
					Publications: []postgresv1alpha1.PostgresPublication{
						{
							Name: publicationName,
							Tables: []postgresv1alpha1.PostgresTableIdentifier{
								{
									Name:   "jobs",
									Schema: "public",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

			waitForPublication(ctx, publicationName)

			// Obtain a fresh copy of the resource, as a finalizer could have been
			// added onto it.
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace.Name,
				Name:      "publication-test",
			}, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to get latest PostgresConfig resource")

			postgresConfig.Spec.Publications[0].Tables = append(
				postgresConfig.Spec.Publications[0].Tables,
				postgresv1alpha1.PostgresTableIdentifier{
					Name:   "users",
					Schema: "public",
				},
			)
			err = k8sClient.Update(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to update PostgresConfig resource")

			Eventually(func() int {
				row := postgresConn.QueryRow(
					ctx,
					`SELECT COUNT(*) AS count FROM pg_publication_tables WHERE
						pubname = $1 AND 
						schemaname = 'public' AND
						tablename = ANY($2::text[])`,
					publicationName,
					[]string{"jobs", "users"},
				)

				var count int
				err := row.Scan(&count)
				Expect(err).NotTo(HaveOccurred())

				return count
			}).Should(Equal(2))
		})

		It("should drop table from existing publication", func() {
			const publicationName = "publication_test"

			createBarebonesTable(ctx, "jobs")
			createBarebonesTable(ctx, "users")

			postgresConfig := &postgresv1alpha1.PostgresConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresConfigSpec{
					PostgresRef: PostgresContainerRef(ctx),
					Publications: []postgresv1alpha1.PostgresPublication{
						{
							Name: publicationName,
							Tables: []postgresv1alpha1.PostgresTableIdentifier{
								{
									Name:   "jobs",
									Schema: "public",
								},
								{
									Name:   "users",
									Schema: "public",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

			waitForPublication(ctx, publicationName)

			// Obtain a fresh copy of the resource, as a finalizer could have been
			// added onto it.
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace.Name,
				Name:      "publication-test",
			}, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to get latest PostgresConfig resource")

			postgresConfig.Spec.Publications[0].Tables = []postgresv1alpha1.PostgresTableIdentifier{
				postgresConfig.Spec.Publications[0].Tables[0],
			}

			err = k8sClient.Update(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to update PostgresConfig resource")

			Eventually(func() int {
				row := postgresConn.QueryRow(
					ctx,
					`SELECT COUNT(*) AS count FROM pg_publication_tables WHERE
						pubname = $1 AND 
						schemaname = 'public' AND
						tablename = ANY($2::text[])`,
					publicationName,
					[]string{"jobs", "users"},
				)

				var count int
				err := row.Scan(&count)
				Expect(err).NotTo(HaveOccurred())

				return count
			}).Should(Equal(1))
		})

		It("should should change subset of operations", func() {
			const publicationName = "publication_test"
			createBarebonesTable(ctx, "jobs")

			postgresConfig := &postgresv1alpha1.PostgresConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresConfigSpec{
					PostgresRef: PostgresContainerRef(ctx),
					Publications: []postgresv1alpha1.PostgresPublication{
						{
							Name: publicationName,
							Tables: []postgresv1alpha1.PostgresTableIdentifier{
								{
									Name:   "jobs",
									Schema: "public",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

			waitForPublication(ctx, publicationName)

			// Obtain a fresh copy of the resource, as a finalizer could have been
			// added onto it.
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace.Name,
				Name:      "publication-test",
			}, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to get latest PostgresConfig resource")

			postgresConfig.Spec.Publications[0].Operations = []string{"insert", "delete"}
			err = k8sClient.Update(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to update PostgresConfig resource")

			Eventually(func() bool {
				row := postgresConn.QueryRow(
					ctx,
					"SELECT pubinsert, pubupdate, pubdelete FROM pg_publication WHERE pubname = $1",
					publicationName,
				)

				var pubinsert bool
				var pubupdate bool
				var pubdelete bool
				err := row.Scan(&pubinsert, &pubupdate, &pubdelete)
				Expect(err).NotTo(HaveOccurred(), "failed to scan row")

				return pubinsert && pubdelete && !pubupdate
			}).Should(BeTrue())
		})

		It("should delete publication", func() {
			const publicationName = "jobs"
			createBarebonesTable(ctx, "jobs")

			postgresConfig := &postgresv1alpha1.PostgresConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresConfigSpec{
					PostgresRef: PostgresContainerRef(ctx),
					Publications: []postgresv1alpha1.PostgresPublication{
						{
							Name: publicationName,
							Tables: []postgresv1alpha1.PostgresTableIdentifier{
								{
									Name:   "jobs",
									Schema: "public",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

			waitForPublication(ctx, publicationName)

			err = k8sClient.Delete(ctx, postgresConfig)
			Expect(err).NotTo(HaveOccurred(), "failed to delete PostgresConfig resource")

			Eventually(func() int {
				row := postgresConn.QueryRow(
					ctx,
					"SELECT COUNT(*) AS count FROM pg_publication WHERE pubname = $1",
					"jobs",
				)

				var count int
				err := row.Scan(&count)
				Expect(err).NotTo(HaveOccurred())

				return count
			}).Should(Equal(0), "created publication count should be more than 0")
		})
	})
})

// createBarebonesTable creates a very minimal table with the given name.
func createBarebonesTable(ctx context.Context, name string) {
	query := fmt.Sprintf(
		`CREATE TABLE %s (
			id UUID NOT NULL DEFAULT gen_random_uuid(),
			CONSTRAINT %s PRIMARY KEY (id)
		)`,
		pgx.Identifier{name}.Sanitize(),
		pgx.Identifier{fmt.Sprintf("%s_pkey", name)}.Sanitize(),
	)

	_, err := postgresConn.Exec(ctx, query)
	Expect(err).NotTo(HaveOccurred(), "failed to create table %s", name)
}

// waitForPublication waits for a publication with the given name to be created
// within PostgreSQL, until timeout is reached.
func waitForPublication(ctx context.Context, name string) {
	Eventually(func() int {
		row := postgresConn.QueryRow(
			ctx,
			"SELECT COUNT(*) AS count FROM pg_publication WHERE pubname = $1",
			name,
		)

		var count int
		err := row.Scan(&count)
		Expect(err).NotTo(HaveOccurred())

		return count
	}).Should(Equal(1), "created publication count should be 1")
}
