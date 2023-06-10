package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	"github.com/glints-dev/postgres-config-operator/controllers/testutils"
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

	Describe("Publications", func() {
		It("should create a publication for the given table", func() {
			const publicationName = "jobs"
			createBarebonesTable(ctx, "jobs")

			publication := &postgresv1alpha1.PostgresPublication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresPublicationSpec{
					PostgresRef: testutils.PostgresContainerRef(ctx),
					Name:        publicationName,
					Tables: []postgresv1alpha1.PostgresIdentifier{
						{
							Name:   "jobs",
							Schema: "public",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, publication)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresPublication resource")

			waitForPublication(ctx, publicationName)
		})

		It("should create a publication for the given table for subset of operations", func() {
			const publicationName = "jobs"
			createBarebonesTable(ctx, "jobs")

			publication := &postgresv1alpha1.PostgresPublication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresPublicationSpec{
					PostgresRef: testutils.PostgresContainerRef(ctx),
					Name:        publicationName,
					Tables: []postgresv1alpha1.PostgresIdentifier{
						{
							Name:   "jobs",
							Schema: "public",
						},
					},
					Operations: []string{"insert", "delete"},
				},
			}

			err := k8sClient.Create(ctx, publication)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresPublication resource")

			waitForPublication(ctx, publicationName)

			Eventually(func() bool {
				row := testutils.PostgresConn.QueryRow(
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

			publication := &postgresv1alpha1.PostgresPublication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresPublicationSpec{
					PostgresRef: testutils.PostgresContainerRef(ctx),
					Name:        publicationName,
					Tables: []postgresv1alpha1.PostgresIdentifier{
						{
							Name:   "jobs",
							Schema: "public",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, publication)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresPublication resource")

			waitForPublication(ctx, publicationName)

			Eventually(func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: namespace.Name,
					Name:      "publication-test",
				}, publication)
				Expect(err).NotTo(HaveOccurred(), "failed to get latest PostgresPublication resource")

				publication.Spec.Tables = append(
					publication.Spec.Tables,
					postgresv1alpha1.PostgresIdentifier{
						Name:   "users",
						Schema: "public",
					},
				)
				return k8sClient.Update(ctx, publication)
			}).ShouldNot(HaveOccurred(), "failed to update PostgresPublication resource")

			Eventually(func() int {
				row := testutils.PostgresConn.QueryRow(
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

			publication := &postgresv1alpha1.PostgresPublication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresPublicationSpec{
					PostgresRef: testutils.PostgresContainerRef(ctx),
					Name:        publicationName,
					Tables: []postgresv1alpha1.PostgresIdentifier{
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
			}

			err := k8sClient.Create(ctx, publication)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresPublication resource")

			waitForPublication(ctx, publicationName)

			Eventually(func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: namespace.Name,
					Name:      "publication-test",
				}, publication)
				Expect(err).NotTo(HaveOccurred(), "failed to get latest PostgresPublication resource")

				publication.Spec.Tables = []postgresv1alpha1.PostgresIdentifier{
					publication.Spec.Tables[0],
				}

				return k8sClient.Update(ctx, publication)
			}).ShouldNot(HaveOccurred(), "failed to update PostgresPublication resource")

			Eventually(func() int {
				row := testutils.PostgresConn.QueryRow(
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

			publication := &postgresv1alpha1.PostgresPublication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresPublicationSpec{
					PostgresRef: testutils.PostgresContainerRef(ctx),
					Name:        publicationName,
					Tables: []postgresv1alpha1.PostgresIdentifier{
						{
							Name:   "jobs",
							Schema: "public",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, publication)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresPublication resource")

			waitForPublication(ctx, publicationName)

			Eventually(func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: namespace.Name,
					Name:      "publication-test",
				}, publication)
				Expect(err).NotTo(HaveOccurred(), "failed to get latest PostgresPublication resource")

				publication.Spec.Operations = []string{"insert", "delete"}
				return k8sClient.Update(ctx, publication)
			}).ShouldNot(HaveOccurred(), "failed to update PostgresPublication resource")

			Eventually(func() bool {
				row := testutils.PostgresConn.QueryRow(
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

			publication := &postgresv1alpha1.PostgresPublication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "publication-test",
					Namespace: namespace.Name,
				},
				Spec: postgresv1alpha1.PostgresPublicationSpec{
					PostgresRef: testutils.PostgresContainerRef(ctx),
					Name:        publicationName,
					Tables: []postgresv1alpha1.PostgresIdentifier{
						{
							Name:   "jobs",
							Schema: "public",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, publication)
			Expect(err).NotTo(HaveOccurred(), "failed to create PostgresPublication resource")

			waitForPublication(ctx, publicationName)

			err = k8sClient.Delete(ctx, publication)
			Expect(err).NotTo(HaveOccurred(), "failed to delete PostgresPublication resource")

			Eventually(func() int {
				row := testutils.PostgresConn.QueryRow(
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

// waitForPublication waits for a publication with the given name to be created
// within PostgreSQL, until timeout is reached.
func waitForPublication(ctx context.Context, name string) {
	Eventually(func() int {
		row := testutils.PostgresConn.QueryRow(
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
