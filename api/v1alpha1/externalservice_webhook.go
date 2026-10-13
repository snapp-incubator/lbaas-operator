/*
Copyright 2023.

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
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const ExternalServiceGroup = "networking.snappcloud.io"

// log is for logging in this package.
var externalservicelog = logf.Log.WithName("externalservice-resource")

func (r *ExternalService) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(&ExternalServiceValidator{Client: mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-networking-snappcloud-io-v1alpha1-externalservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.snappcloud.io,resources=externalservices,verbs=create,versions=v1alpha1,name=vexternalservice.kb.io,admissionReviewVersions=v1

// ExternalServiceValidator validates ExternalService objects on admission.
// +kubebuilder:object:generate=false
type ExternalServiceValidator struct {
	Client client.Client
}

var _ admission.Validator[*ExternalService] = &ExternalServiceValidator{}

// ValidateCreate rejects an ExternalService when a Service with the same name already exists.
func (v *ExternalServiceValidator) ValidateCreate(ctx context.Context, r *ExternalService) (admission.Warnings, error) {
	externalservicelog.Info("validating create external service", "name", r.Name)
	var svc corev1.Service
	err := v.Client.Get(ctx, types.NamespacedName{
		Namespace: r.Namespace,
		Name:      r.Name,
	}, &svc)
	if errors.IsNotFound(err) {
		return nil, nil
	}
	return nil, errors.NewInvalid(
		schema.GroupKind{
			Group: ExternalServiceGroup,
			Kind:  "ExternalService",
		},
		r.Name,
		field.ErrorList{
			field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "There should not exist a Service object with the name of ExternalService given"),
		},
	)
}

// ValidateUpdate implements admission.Validator.
func (v *ExternalServiceValidator) ValidateUpdate(_ context.Context, _, _ *ExternalService) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements admission.Validator.
func (v *ExternalServiceValidator) ValidateDelete(_ context.Context, _ *ExternalService) (admission.Warnings, error) {
	return nil, nil
}
