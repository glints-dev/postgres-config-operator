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

import (
	"github.com/jackc/pgx/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PostgresGrantSpec defines the desired state of PostgresGrant
type PostgresGrantSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PostgresRef is a reference to the PostgreSQL server to configure
	PostgresRef PostgresRef `json:"postgresRef"`

	// Role to grant privileges to
	Role string `json:"role"`

	// Tables is the list of tables to grant privileges for
	Tables []PostgresIdentifier `json:"tables,omitempty"`

	// TablePrivileges is the list of privileges to grant
	// Must be one of: SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER, ALL
	TablePrivileges []string `json:"tablePrivileges,omitempty"`

	// Sequences is the list of sequences to grant privileges for
	Sequences []PostgresIdentifier `json:"sequences,omitempty"`

	// SequencePrivileges is the list of privileges to grant
	// Must be one of: USAGE, SELECT, UPDATE, ALL
	SequencePrivileges []string `json:"sequencePrivileges,omitempty"`

	// Databases is the list of databases to grant privileges for
	Databases []string `json:"databases,omitempty"`

	// DatabasePrivileges is the list of privileges to grant
	// Must be one of: CREATE, CONNECT, TEMPORARY, TEMP, ALL
	DatabasePrivileges []string `json:"databasePrivileges,omitempty"`

	// Schemas  is the list of schemas to grant privileges for
	Schemas []string `json:"schemas,omitempty"`

	// SchemaPrivileges is the list of privileges to grant
	// Must be one of: CREATE, USAGE, ALL
	SchemaPrivileges []string `json:"schemaPrivileges,omitempty"`

	// Functions  is the list of functions to grant privileges for
	Functions []PostgresIdentifier `json:"functions,omitempty"`

	// FunctionPrivileges is the list of privileges to grant
	// Must be one of: EXECUTE, ALL
	FunctionPrivileges []string `json:"functionPrivileges,omitempty"`
}

// PostgresGrantStatus defines the observed state of PostgresGrant
type PostgresGrantStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PostgresGrant is the Schema for the postgresgrants API
type PostgresGrant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresGrantSpec   `json:"spec,omitempty"`
	Status PostgresGrantStatus `json:"status,omitempty"`
}

// PrivilegesForType returns the list of privileges defined for a given object
// type within the grant. If the object type is invalid, nil is returned.
func (g *PostgresGrant) PrivilegesForType(o ObjectType) []string {
	switch o {
	case ObjectTypeTable:
		return g.Spec.TablePrivileges
	case ObjectTypeSequence:
		return g.Spec.SequencePrivileges
	case ObjectTypeFunction:
		return g.Spec.FunctionPrivileges
	case ObjectTypeSchema:
		return g.Spec.SchemaPrivileges
	case ObjectTypeDatabase:
		return g.Spec.DatabasePrivileges
	}

	return nil
}

// IdentifiersForType returns the list of identifiers defined for a given object
// type within the grant. If the object type is invalid, nil is returned.
func (g *PostgresGrant) IdentifiersForType(o ObjectType) []pgx.Identifier {
	var identifiers []pgx.Identifier
	switch o {
	case ObjectTypeTable:
		for _, table := range g.Spec.Tables {
			identifiers = append(identifiers, pgx.Identifier{table.Schema, table.Name})
		}
	case ObjectTypeSequence:
		for _, sequence := range g.Spec.Sequences {
			identifiers = append(identifiers, pgx.Identifier{sequence.Schema, sequence.Name})
		}
	case ObjectTypeFunction:
		for _, function := range g.Spec.Functions {
			identifiers = append(identifiers, pgx.Identifier{function.Schema, function.Name})
		}
	case ObjectTypeSchema:
		for _, schema := range g.Spec.Schemas {
			identifiers = append(identifiers, pgx.Identifier{schema})
		}
	case ObjectTypeDatabase:
		for _, database := range g.Spec.Databases {
			identifiers = append(identifiers, pgx.Identifier{database})
		}
	}

	return identifiers
}

//+kubebuilder:object:root=true

// PostgresGrantList contains a list of PostgresGrant
type PostgresGrantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresGrant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresGrant{}, &PostgresGrantList{})
}
