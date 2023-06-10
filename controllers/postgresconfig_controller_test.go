package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	Describe("Configs: Publications", func() {
		It("should create publication resources", func() {
			result := createConfigForPublications(ctx, namespace)

			spec := result.Publications.Items[0].Spec
			Expect(spec.Name).To(Equal(result.PublicationName))

			Expect(len(spec.Tables)).To(Equal(1))
			Expect(spec.Tables[0].Name).To(Equal("jobs"))
			Expect(spec.Tables[0].Schema).To(Equal("public"))
		})

		It("should delete publication resources", func() {
			result := createConfigForPublications(ctx, namespace)

			// Replace publication list with an empty array.
			result.Config.Spec.Publications = []postgresv1alpha1.Publication{}

			Expect(k8sClient.Update(ctx, result.Config)).NotTo(
				HaveOccurred(),
				"failed to update PostgresConfig resource",
			)

			Eventually(func() int {
				var publications postgresv1alpha1.PostgresPublicationList
				err := k8sClient.List(
					ctx,
					&publications,
					client.InNamespace(namespace.Name),
				)
				Expect(err).NotTo(HaveOccurred())

				return len(publications.Items)
			}, time.Second).Should(Equal(0))
		})

		It("should update publication resources", func() {
			result := createConfigForPublications(ctx, namespace)

			// Rename a publication in the list.
			result.Config.Spec.Publications[0].Name = "jobs-updated"

			Expect(k8sClient.Update(ctx, result.Config)).NotTo(
				HaveOccurred(),
				"failed to update PostgresConfig resource",
			)

			Eventually(func() int {
				var publications postgresv1alpha1.PostgresPublicationList
				err := k8sClient.List(
					ctx,
					&publications,
					client.InNamespace(namespace.Name),
				)
				Expect(err).NotTo(HaveOccurred())

				return len(publications.Items)
			}, time.Second).Should(Equal(1))
		})
	})
})

// createConfigForPublications creates a PostgresConfig on the test Kubernetes
// cluster for the purpose of testing out publication functionality.
//
// The created config is named "publication-test" and it contains exactly 1
// publication named "jobs" on a test table named "jobs".
func createConfigForPublications(
	ctx context.Context,
	namespace *corev1.Namespace,
) createConfigForPublicationsResult {
	const publicationName = "jobs"
	const tableName = "jobs"

	createBarebonesTable(ctx, tableName)

	config := &postgresv1alpha1.PostgresConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "publication-test",
			Namespace: namespace.Name,
		},
		Spec: postgresv1alpha1.PostgresConfigSpec{
			PostgresRef: testutils.PostgresContainerRef(ctx),
			Publications: []postgresv1alpha1.Publication{
				{
					Name: publicationName,
					Tables: []postgresv1alpha1.PostgresIdentifier{
						{Name: tableName, Schema: "public"},
					},
				},
			},
		},
	}

	Expect(k8sClient.Create(ctx, config)).NotTo(
		HaveOccurred(),
		"failed to create PostgresConfig resource",
	)

	var publications postgresv1alpha1.PostgresPublicationList
	Eventually(func() int {
		err := k8sClient.List(
			ctx,
			&publications,
			client.InNamespace(namespace.Name),
		)
		Expect(err).NotTo(HaveOccurred())

		return len(publications.Items)
	}, time.Second).Should(Equal(1))

	return createConfigForPublicationsResult{
		Config:          config,
		Publications:    publications,
		PublicationName: publicationName,
	}
}

type createConfigForPublicationsResult struct {
	Config          *postgresv1alpha1.PostgresConfig
	Publications    postgresv1alpha1.PostgresPublicationList
	PublicationName string
}
