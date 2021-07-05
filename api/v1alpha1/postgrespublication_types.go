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

// PostgresPublicationSpec defines the desired state of PostgresPublication
type PostgresPublicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PostgresRef is a reference to the PostgreSQL server to configure
	PostgresRef PostgresRef `json:"postgresRef"`

	// Name is the name of the publication to create
	Name string `json:"name"`

	// Tables is the list of tables to include in the publication. If the list
	// is empty or omitted, publication is created for all tables
	Tables []PostgresTableIdentifier `json:"tables"`

	// Operations determines which DML operations will be published by the
	// publication to subscribers. The allowed operations are insert, update,
	// delete, and truncate. If left empty or omitted, all operations are
	// published
	Operations []string `json:"operations,omitempty"`
}

// PostgresPublicationStatus defines the observed state of PostgresPublication
type PostgresPublicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PostgresPublication is the Schema for the postgrespublications API
type PostgresPublication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresPublicationSpec   `json:"spec,omitempty"`
	Status PostgresPublicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostgresPublicationList contains a list of PostgresPublication
type PostgresPublicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresPublication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresPublication{}, &PostgresPublicationList{})
}
