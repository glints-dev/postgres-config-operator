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

// PostgresTableSpec defines the desired state of PostgresTable
type PostgresTableSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PostgresRef is a reference to the PostgreSQL server to configure
	PostgresRef PostgresRef `json:"postgresRef"`

	// Columns is the list of columns to be created
	Columns []PostgresColumn `json:"columns"`

	// Inherit fields from PostgresIdentifier
	PostgresIdentifier `json:""`
}

// PostgresTableStatus defines the observed state of PostgresTable
type PostgresTableStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PostgresTable is the Schema for the postgrestables API
type PostgresTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresTableSpec   `json:"spec,omitempty"`
	Status PostgresTableStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostgresTableList contains a list of PostgresTable
type PostgresTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresTable{}, &PostgresTableList{})
}
