module github.com/glints-dev/postgres-config-operator

go 1.15

require (
	github.com/go-logr/logr v1.2.3
	github.com/jackc/pgconn v1.11.0
	github.com/jackc/pgerrcode v0.0.0-20201024163028-a0d42d470451
	github.com/jackc/pgx/v4 v4.15.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/testcontainers/testcontainers-go v0.12.0
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	sigs.k8s.io/controller-runtime v0.11.1
)
