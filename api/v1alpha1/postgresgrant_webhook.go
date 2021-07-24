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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var postgresgrantlog = logf.Log.WithName("postgresgrant-resource")

type ObjectType string

const (
	ObjectTypeTable    ObjectType = "table"
	ObjectTypeSequence ObjectType = "sequence"
	ObjectTypeDatabase ObjectType = "database"
	ObjectTypeSchema   ObjectType = "schema"
	ObjectTypeFunction ObjectType = "function"
)

func (o ObjectType) ValidPrivileges() map[string]struct{} {
	switch o {
	case ObjectTypeTable:
		return map[string]struct{}{
			"SELECT":     {},
			"INSERT":     {},
			"UPDATE":     {},
			"DELETE":     {},
			"TRUNCATE":   {},
			"REFERENCES": {},
			"TRIGGER":    {},
			"ALL":        {},
		}
	case ObjectTypeSequence:
		return map[string]struct{}{
			"USAGE":  {},
			"SELECT": {},
			"UPDATE": {},
			"ALL":    {},
		}
	case ObjectTypeDatabase:
		return map[string]struct{}{
			"CREATE":    {},
			"CONNECT":   {},
			"TEMPORARY": {},
			"TEMP":      {},
			"ALL":       {},
		}
	case ObjectTypeSchema:
		return map[string]struct{}{
			"CREATE": {},
			"USAGE":  {},
			"ALL":    {},
		}
	case ObjectTypeFunction:
		return map[string]struct{}{
			"EXECUTE": {},
			"ALL":     {},
		}
	}

	return nil
}

func (r *PostgresGrant) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-postgres-glints-com-v1alpha1-postgresgrant,mutating=false,failurePolicy=fail,sideEffects=None,groups=postgres.glints.com,resources=postgresgrants,verbs=create;update,versions=v1alpha1,name=vpostgresgrant.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &PostgresGrant{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PostgresGrant) ValidateCreate() error {
	postgresgrantlog.Info("validate create", "name", r.Name)

	if err := validatePrivileges(r); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PostgresGrant) ValidateUpdate(old runtime.Object) error {
	postgresgrantlog.Info("validate update", "name", r.Name)

	if err := validatePrivileges(r); err != nil {
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PostgresGrant) ValidateDelete() error {
	postgresgrantlog.Info("validate delete", "name", r.Name)

	return nil
}

func validatePrivileges(grant *PostgresGrant) error {
	validationInputs := map[ObjectType][]string{
		ObjectTypeTable:    grant.Spec.TablePrivileges,
		ObjectTypeSequence: grant.Spec.SequencePrivileges,
		ObjectTypeDatabase: grant.Spec.DatabasePrivileges,
		ObjectTypeSchema:   grant.Spec.SchemaPrivileges,
		ObjectTypeFunction: grant.Spec.FunctionPrivileges,
	}

	for objectType, input := range validationInputs {
		if err := validatePrivilegesForType(input, objectType); err != nil {
			return err
		}
	}

	return nil
}

func validatePrivilegesForType(privilegesFromGrant []string, oType ObjectType) error {
	validPrivileges := oType.ValidPrivileges()

	hasAll := false
	wantedPrivileges := make(map[string]struct{})
	for _, privilege := range privilegesFromGrant {
		if _, ok := validPrivileges[privilege]; !ok {
			return fmt.Errorf("invalid %s privilege \"%s\"", oType, privilege)
		}

		if _, ok := wantedPrivileges[privilege]; ok {
			return fmt.Errorf("duplicate %s privilege \"%s\"", oType, privilege)
		}

		wantedPrivileges[privilege] = struct{}{}

		if privilege == "ALL" {
			hasAll = true
		}
	}

	if hasAll && len(privilegesFromGrant) > 1 {
		return fmt.Errorf("\"ALL\" cannot be specified alongside other privileges")
	}

	return nil
}
