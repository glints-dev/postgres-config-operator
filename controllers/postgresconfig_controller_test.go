package controllers

import (
	"context"

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

	It("should create a publication for the given table", func() {
		_, err := postgresConn.Exec(
			ctx,
			`CREATE TABLE "jobs" (
				id UUID NOT NULL DEFAULT gen_random_uuid(),
				CONSTRAINT jobs_pkey PRIMARY KEY (id)
			)`)
		Expect(err).NotTo(HaveOccurred(), "failed to create test table")

		postgresConfig := &postgresv1alpha1.PostgresConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "publication-test",
				Namespace: namespace.Name,
			},
			Spec: postgresv1alpha1.PostgresConfigSpec{
				PostgresRef: PostgresContainerRef(ctx),
				Publications: []postgresv1alpha1.PostgresPublication{
					{
						Name: "jobs",
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

		err = k8sClient.Create(ctx, postgresConfig)
		Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

		waitForPublication(ctx, "jobs")
	})

	It("should add new table to existing publication", func() {
		const publicationName = "publication_test"

		_, err := postgresConn.Exec(
			ctx,
			`CREATE TABLE "jobs" (
				id UUID NOT NULL DEFAULT gen_random_uuid(),
				CONSTRAINT jobs_pkey PRIMARY KEY (id)
			)`)
		Expect(err).NotTo(HaveOccurred(), "failed to create test table")

		_, err = postgresConn.Exec(
			ctx,
			`CREATE TABLE "users" (
				id UUID NOT NULL DEFAULT gen_random_uuid(),
				CONSTRAINT users_pkey PRIMARY KEY (id)
			)`)
		Expect(err).NotTo(HaveOccurred(), "failed to create test table")

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

		err = k8sClient.Create(ctx, postgresConfig)
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

		_, err := postgresConn.Exec(
			ctx,
			`CREATE TABLE "jobs" (
				id UUID NOT NULL DEFAULT gen_random_uuid(),
				CONSTRAINT jobs_pkey PRIMARY KEY (id)
			)`)
		Expect(err).NotTo(HaveOccurred(), "failed to create test table")

		_, err = postgresConn.Exec(
			ctx,
			`CREATE TABLE "users" (
				id UUID NOT NULL DEFAULT gen_random_uuid(),
				CONSTRAINT users_pkey PRIMARY KEY (id)
			)`)
		Expect(err).NotTo(HaveOccurred(), "failed to create test table")

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

		err = k8sClient.Create(ctx, postgresConfig)
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

	It("should delete publication", func() {
		_, err := postgresConn.Exec(
			ctx,
			`CREATE TABLE "jobs" (
				id UUID NOT NULL DEFAULT gen_random_uuid(),
				CONSTRAINT jobs_pkey PRIMARY KEY (id)
			)`)
		Expect(err).NotTo(HaveOccurred(), "failed to create test table")

		postgresConfig := &postgresv1alpha1.PostgresConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "publication-test",
				Namespace: namespace.Name,
			},
			Spec: postgresv1alpha1.PostgresConfigSpec{
				PostgresRef: PostgresContainerRef(ctx),
				Publications: []postgresv1alpha1.PostgresPublication{
					{
						Name: "jobs",
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

		err = k8sClient.Create(ctx, postgresConfig)
		Expect(err).NotTo(HaveOccurred(), "failed to create PostgresConfig resource")

		waitForPublication(ctx, "jobs")

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
