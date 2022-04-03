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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PostgresConfigSpec defines the desired state of PostgresConfig
type PostgresConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PostgresRef is a reference to the PostgreSQL server to configure
	PostgresRef PostgresRef `json:"postgresRef"`

	// Publications is a list of publications to be created
	Publications []Publication `json:"publications,omitempty"`

	// Tables is a list of tables to be created
	Tables []Table `json:"table,omitempty"`
}

// Publication represents a PUBLICATION
// https://www.postgresql.org/docs/current/sql-createpublication.html
type Publication struct {
	// Name is the name of the publication to create
	Name string `json:"name"`

	// Tables is the list of tables to include in the publication. If the list
	// is empty or omitted, publication is created for all tables
	Tables []PostgresIdentifier `json:"tables,omitempty"`

	// Operations determines which DML operations will be published by the
	// publication to subscribers. The allowed operations are insert, update,
	// delete, and truncate. If left empty or omitted, all operations are
	// published
	Operations []string `json:"operations,omitempty"`
}

// Table represents a table
// https://www.postgresql.org/docs/current/sql-createtable.html
type Table struct {
	// Columns is the list of columns to be created
	Columns []PostgresColumn `json:"columns"`

	// Inherit fields from PostgresIdentifier
	PostgresIdentifier `json:""`
}

// PostgresConfigStatus defines the observed state of PostgresConfig
type PostgresConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Configured indicates whether the target PostgreSQL server has been
	// successfully configured according to spec
	Configured bool `json:"configured"`
}

// SecretRef is a reference to a secret that exists in the same namespace.
type SecretRef struct {
	// SecretName is the name of the secret.
	SecretName string `json:"secretName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PostgresConfig is the Schema for the postgresconfigs API
type PostgresConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresConfigSpec   `json:"spec,omitempty"`
	Status PostgresConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostgresConfigList contains a list of PostgresConfig
type PostgresConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresConfig{}, &PostgresConfigList{})
}
