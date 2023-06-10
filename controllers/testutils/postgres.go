package testutils

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
)

// PostgresUser holds the username that can be used to connect to the
// PostgreSQL container.
const PostgresUser = "glints"

// PostgresPassword holds the password that can be used to connect to the
// PostgreSQL container.
const PostgresPassword = "glints"

// PostgresContainer
var PostgresContainer testcontainers.Container

const PostgresSecretName = "postgres"

var PostgresConn *pgx.Conn

// SetupPostgresContainer creates a PostgreSQL container for testing purposes.
// The created container is accessible through the PostgresContainer global.
func SetupPostgresContainer(ctx context.Context) {
	BeforeEach(func() {
		const port = "5432/tcp"
		req := testcontainers.ContainerRequest{
			Image: "postgres:14-alpine",
			Env: map[string]string{
				"POSTGRES_USER":     PostgresUser,
				"POSTGRES_PASSWORD": PostgresPassword,
			},
			ExposedPorts: []string{port},
			WaitingFor:   wait.ForListeningPort(port),
		}

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		Expect(err).NotTo(HaveOccurred())

		retVal, err := container.Exec(ctx, []string{
			"psql", "-U", PostgresUser, "-c", "CREATE EXTENSION pgcrypto;",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(retVal).To(Equal(0))

		PostgresContainer = container
	})

	AfterEach(func() {
		PostgresContainer.Terminate(ctx)
	})
}

// PostgresContainerRef returns a reference to the test Postgres container,
// which can be used within a PostgresConfig resource.
func PostgresContainerRef(ctx context.Context) postgresv1alpha1.PostgresRef {
	endpoint, err := PostgresContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred())

	hostPort := strings.SplitN(endpoint, ":", 2)
	port, err := strconv.Atoi(hostPort[1])
	Expect(err).NotTo(HaveOccurred())

	return postgresv1alpha1.PostgresRef{
		Host:     hostPort[0],
		Port:     uint16(port),
		Database: "glints",
		SecretRef: postgresv1alpha1.SecretRef{
			SecretName: PostgresSecretName,
		},
	}
}

// SetupPostgresConnection creates a connection to the test PostgreSQL
// container. This must be run after SetupPostgresContainer.
func SetupPostgresConnection(ctx context.Context) {
	BeforeEach(func() {
		endpoint, err := PostgresContainer.Endpoint(ctx, "")
		Expect(err).NotTo(HaveOccurred())

		conn, err := pgx.Connect(ctx, fmt.Sprintf(
			"postgres://%s:%s@%s/glints",
			PostgresUser,
			PostgresPassword,
			endpoint,
		))
		Expect(err).NotTo(HaveOccurred())

		PostgresConn = conn
	})

	AfterEach(func() {
		PostgresConn.Close(ctx)
	})
}
