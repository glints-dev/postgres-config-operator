module github.com/glints-dev/postgres-config-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/jackc/pgconn v1.8.1
	github.com/jackc/pgerrcode v0.0.0-20201024163028-a0d42d470451
	github.com/jackc/pgx/v4 v4.11.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/testcontainers/testcontainers-go v0.10.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)
