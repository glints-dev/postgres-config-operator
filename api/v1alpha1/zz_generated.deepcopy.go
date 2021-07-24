// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresConfig) DeepCopyInto(out *PostgresConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresConfig.
func (in *PostgresConfig) DeepCopy() *PostgresConfig {
	if in == nil {
		return nil
	}
	out := new(PostgresConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresConfigList) DeepCopyInto(out *PostgresConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PostgresConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresConfigList.
func (in *PostgresConfigList) DeepCopy() *PostgresConfigList {
	if in == nil {
		return nil
	}
	out := new(PostgresConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresConfigSpec) DeepCopyInto(out *PostgresConfigSpec) {
	*out = *in
	out.PostgresRef = in.PostgresRef
	if in.Publications != nil {
		in, out := &in.Publications, &out.Publications
		*out = make([]Publication, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresConfigSpec.
func (in *PostgresConfigSpec) DeepCopy() *PostgresConfigSpec {
	if in == nil {
		return nil
	}
	out := new(PostgresConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresConfigStatus) DeepCopyInto(out *PostgresConfigStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresConfigStatus.
func (in *PostgresConfigStatus) DeepCopy() *PostgresConfigStatus {
	if in == nil {
		return nil
	}
	out := new(PostgresConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresGrant) DeepCopyInto(out *PostgresGrant) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresGrant.
func (in *PostgresGrant) DeepCopy() *PostgresGrant {
	if in == nil {
		return nil
	}
	out := new(PostgresGrant)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresGrant) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresGrantList) DeepCopyInto(out *PostgresGrantList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PostgresGrant, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresGrantList.
func (in *PostgresGrantList) DeepCopy() *PostgresGrantList {
	if in == nil {
		return nil
	}
	out := new(PostgresGrantList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresGrantList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresGrantSpec) DeepCopyInto(out *PostgresGrantSpec) {
	*out = *in
	out.PostgresRef = in.PostgresRef
	if in.Tables != nil {
		in, out := &in.Tables, &out.Tables
		*out = make([]PostgresIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.TablePrivileges != nil {
		in, out := &in.TablePrivileges, &out.TablePrivileges
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Sequences != nil {
		in, out := &in.Sequences, &out.Sequences
		*out = make([]PostgresIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.SequencePrivileges != nil {
		in, out := &in.SequencePrivileges, &out.SequencePrivileges
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Databases != nil {
		in, out := &in.Databases, &out.Databases
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DatabasePrivileges != nil {
		in, out := &in.DatabasePrivileges, &out.DatabasePrivileges
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Schemas != nil {
		in, out := &in.Schemas, &out.Schemas
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.SchemaPrivileges != nil {
		in, out := &in.SchemaPrivileges, &out.SchemaPrivileges
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Functions != nil {
		in, out := &in.Functions, &out.Functions
		*out = make([]PostgresIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.FunctionPrivileges != nil {
		in, out := &in.FunctionPrivileges, &out.FunctionPrivileges
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresGrantSpec.
func (in *PostgresGrantSpec) DeepCopy() *PostgresGrantSpec {
	if in == nil {
		return nil
	}
	out := new(PostgresGrantSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresGrantStatus) DeepCopyInto(out *PostgresGrantStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresGrantStatus.
func (in *PostgresGrantStatus) DeepCopy() *PostgresGrantStatus {
	if in == nil {
		return nil
	}
	out := new(PostgresGrantStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresIdentifier) DeepCopyInto(out *PostgresIdentifier) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresIdentifier.
func (in *PostgresIdentifier) DeepCopy() *PostgresIdentifier {
	if in == nil {
		return nil
	}
	out := new(PostgresIdentifier)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresPublication) DeepCopyInto(out *PostgresPublication) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresPublication.
func (in *PostgresPublication) DeepCopy() *PostgresPublication {
	if in == nil {
		return nil
	}
	out := new(PostgresPublication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresPublication) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresPublicationList) DeepCopyInto(out *PostgresPublicationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PostgresPublication, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresPublicationList.
func (in *PostgresPublicationList) DeepCopy() *PostgresPublicationList {
	if in == nil {
		return nil
	}
	out := new(PostgresPublicationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresPublicationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresPublicationSpec) DeepCopyInto(out *PostgresPublicationSpec) {
	*out = *in
	out.PostgresRef = in.PostgresRef
	if in.Tables != nil {
		in, out := &in.Tables, &out.Tables
		*out = make([]PostgresIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.Operations != nil {
		in, out := &in.Operations, &out.Operations
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresPublicationSpec.
func (in *PostgresPublicationSpec) DeepCopy() *PostgresPublicationSpec {
	if in == nil {
		return nil
	}
	out := new(PostgresPublicationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresPublicationStatus) DeepCopyInto(out *PostgresPublicationStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresPublicationStatus.
func (in *PostgresPublicationStatus) DeepCopy() *PostgresPublicationStatus {
	if in == nil {
		return nil
	}
	out := new(PostgresPublicationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresRef) DeepCopyInto(out *PostgresRef) {
	*out = *in
	out.SecretRef = in.SecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresRef.
func (in *PostgresRef) DeepCopy() *PostgresRef {
	if in == nil {
		return nil
	}
	out := new(PostgresRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Publication) DeepCopyInto(out *Publication) {
	*out = *in
	if in.Tables != nil {
		in, out := &in.Tables, &out.Tables
		*out = make([]PostgresIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.Operations != nil {
		in, out := &in.Operations, &out.Operations
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Publication.
func (in *Publication) DeepCopy() *Publication {
	if in == nil {
		return nil
	}
	out := new(Publication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretRef) DeepCopyInto(out *SecretRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretRef.
func (in *SecretRef) DeepCopy() *SecretRef {
	if in == nil {
		return nil
	}
	out := new(SecretRef)
	in.DeepCopyInto(out)
	return out
}
