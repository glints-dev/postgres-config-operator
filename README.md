# PostgreSQL Configuration Operator

The Postgres Configuration Operator allows for declarative configuration of
PostgreSQL instances using Custom Resource Definitions (CRDs).

This operator is written using the [Operator SDK](https://sdk.operatorframework.io/).

## Operator Features

- Creating and maintaining [publications](https://www.postgresql.org/docs/current/sql-createpublication.html)
- Creating [tables](https://www.postgresql.org/docs/current/sql-createtable.html)
- (Coming soon) Creating and maintaining [roles](https://www.postgresql.org/docs/13/sql-createrole.html)

## Getting Started

As this operator is in early development, there are no releases at the moment.

To build a copy of the operator and push the resulting image to a Docker
registry, use the following command:

```
make docker-build docker-push IMG="localhost:5000/postgres-config-operator:latest"
```

To deploy to a running Kubernetes cluster using the current context configured
with `kubectl`, use:

```
make deploy IMG="localhost:5000/postgres-config-operator:latest"
```

## Development & Testing

Testing can be done locally by simply running:

```
make test
```

## Examples

Example manifests can be found under config/samples/.
