/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

// PostgresRef is a reference to a PostgreSQL server
type PostgresRef struct {
	// Host is the host of the PostgreSQL server to configure
	Host string `json:"host,omitempty"`

	// Port is the port of the PostgreSQL server to configure
	Port uint16 `json:"port,omitempty"`

	// Database is the name of the database to configure
	Database string `json:"database,omitempty"`

	// SecretRef is a reference to a secret in the same namespace that contains
	// credentials to authenticate against the PostgreSQL server
	SecretRef SecretRef `json:"secretRef"`

	// Variant is for specific database-as-a-service providers. Valid values
	// are: aiven, standard. The default value is "standard"
	Variant string `json:"variant,omitempty"`
}

// PostgresTableIdentifier represents an identifier for a table, e.g. a pair
// of schema and table name.
type PostgresTableIdentifier struct {
	// Name is the name of the table
	Name string `json:"name"`

	// Schema is the name of the schema the table resides in
	Schema string `json:"schema"`
}
