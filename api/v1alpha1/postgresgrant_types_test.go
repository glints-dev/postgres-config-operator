package v1alpha1

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PostgresGrant: Webhooks", func() {
	type testCase struct {
		Name           string
		Spec           PostgresGrantSpec
		ErrExpected    bool
		ExpectedCode   int32
		ExpectedReason metav1.StatusReason
	}

	testCases := []testCase{
		{
			Name: "should fail to validate if ALL is specified along other operations",
			Spec: PostgresGrantSpec{
				Tables: []PostgresIdentifier{
					{Name: "jobs", Schema: "public"},
				},
				TablePrivileges: []string{"SELECT", "ALL"},
			},
			ExpectedCode:   http.StatusForbidden,
			ExpectedReason: "\"ALL\" cannot be specified alongside other privileges",
		},
		{
			Name: "should fail to validate with duplicate table privileges",
			Spec: PostgresGrantSpec{
				Tables: []PostgresIdentifier{
					{Name: "jobs", Schema: "public"},
				},
				TablePrivileges: []string{"SELECT", "SELECT"},
			},
			ExpectedCode:   http.StatusForbidden,
			ExpectedReason: "duplicate table privilege \"SELECT\"",
		},
		{
			Name: "should fail to validate with invalid table privileges",
			Spec: PostgresGrantSpec{
				Tables: []PostgresIdentifier{
					{Name: "jobs", Schema: "public"},
				},
				TablePrivileges: []string{"DUMMY"},
			},
			ExpectedCode:   http.StatusForbidden,
			ExpectedReason: "invalid table privilege \"DUMMY\"",
		},
	}

	It("should succeed validation with valid privilege declaration", func() {

		grant := &PostgresGrant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "grant-test",
				Namespace: "default",
			},
			Spec: PostgresGrantSpec{
				Tables: []PostgresIdentifier{
					{Name: "jobs", Schema: "public"},
				},
				TablePrivileges: []string{"SELECT", "UPDATE"},
			},
		}

		err := k8sClient.Create(ctx, grant)
		Expect(err).NotTo(HaveOccurred())
	})

	for _, testCase := range testCases {
		testCase := testCase
		It(testCase.Name, func() {
			grant := &PostgresGrant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "grant-test",
					Namespace: "default",
				},
				Spec: testCase.Spec,
			}

			err := k8sClient.Create(ctx, grant)
			Expect(err).To(HaveOccurred())

			statusErr, ok := err.(*errors.StatusError)
			Expect(ok).To(BeTrue())

			Expect(statusErr.ErrStatus.Code).To(BeEquivalentTo(testCase.ExpectedCode))
			Expect(statusErr.ErrStatus.Reason).To(Equal(testCase.ExpectedReason))
		})
	}
})
