package utils

import (
	"context"
	"testing"

	"github.com/glints-dev/postgres-config-operator/controllers/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Utils Suite")
}

var _ = Context("Inside of a new Postgres instance", func() {
	ctx := context.Background()
	testutils.SetupPostgresContainer(ctx)

	Describe("Configs: Publications", func() {
		It("should use custom username and password keys", func() {
			ref := testutils.PostgresContainerRef(ctx)
			ref.SecretRef.UsernameKey = "username"
			ref.SecretRef.PasswordKey = "password"

			obj := metav1.ObjectMeta{}
			recorder := mockEventRecorder{}

			_, err := SetupPostgresConnection(
				ctx,
				mockReader{},
				recorder,
				ref,
				obj,
			)

			Expect(err).ToNot(HaveOccurred())
		})
	})
})

type mockReader struct{}

func (r mockReader) Get(
	ctx context.Context,
	key types.NamespacedName,
	obj client.Object,
	opts ...client.GetOption,
) error {
	secret, ok := obj.(*corev1.Secret)
	Expect(ok).To(Equal(true))

	secret.Data = map[string][]byte{
		"username": []byte(testutils.PostgresUser),
		"password": []byte(testutils.PostgresPassword),
	}

	return nil
}

func (r mockReader) List(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
	return nil
}

type mockEventRecorder struct{}

func (r mockEventRecorder) Event(
	object runtime.Object,
	eventtype,
	reason,
	message string,
) {
}

func (r mockEventRecorder) Eventf(
	object runtime.Object,
	eventtype,
	reason,
	messageFmt string,
	args ...interface{},
) {
}

func (r mockEventRecorder) AnnotatedEventf(
	object runtime.Object,
	annotations map[string]string,
	eventtype,
	reason,
	messageFmt string,
	args ...interface{},
) {
}
