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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExternalServiceType string

const StaticExternalServiceType ExternalServiceType = "static"

// ExternalServicePort contains information on service's port.
type ExternalServicePort struct {
	// The Name of this port within the service. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. When considering
	// the endpoints for a Service, this must match the 'name' field in the
	// EndpointPort.
	// Optional if only one ServicePort is defined on this service.
	//+kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// The IP Protocol for this port. Supports "TCP", "UDP", and "SCTP".
	// Default is TCP.
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Enum=TCP;UDP;SCTP
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// The application protocol for this port.
	// This field follows standard Kubernetes label syntax.
	// Un-prefixed names are reserved for IANA standard service names (as per
	// RFC-6335 and https://www.iana.org/assignments/service-names).
	// Non-standard protocols should use prefixed names such as
	// mycompany.com/my-custom-protocol.
	//+kubebuilder:validation:Optional
	AppProtocol *string `json:"appProtocol,omitempty"`

	// The Port that will be exposed by this service.
	//+kubebuilder:validation:Required
	Port int32 `json:"port"`
}

type StaticEndpoint struct {
	// IP address of the external endpoint
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Pattern=`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	IP string `json:"ip"`
}
type Static struct {
	// Addresses is the list of addresses in the Endpoints object
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinItems=1
	Addresses []StaticEndpoint `json:"addresses"`
}

// ExternalServiceSpec defines the desired state of ExternalService
// +kubebuilder:validation:XValidation:rule=`(self.type == "static") && (has(self.static) && has(self.static.addresses) && size(self.static.addresses)!=0)`,message=`.spec.static.addresses should not be empty when .spec.type is set to static`
type ExternalServiceSpec struct {
	// Type is the type of external service. it could be static and openstack (in future).
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Enum=static
	Type ExternalServiceType `json:"type"`
	// ServiceType is the type of created service. It can be ClusterIP and LoadBalancer
	//+kubebuilder:default="ClusterIP"
	//+kubebuilder:validation:Enum=ClusterIP;LoadBalancer
	ServiceType corev1.ServiceType `json:"serviceType"`
	// Ports are the ports of the created service. minimum number of ports is 1.
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinItems=1
	Ports []ExternalServicePort `json:"ports"`
	// Static is used to create endpoints in StaticExternalServiceType type.
	Static Static `json:"static"`
}

// ExternalServiceStatus defines the observed state of ExternalService
type ExternalServiceStatus struct {
	ServiceAddress string             `json:"serviceAddress,omitempty"`
	Conditions     []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ExternalService is the Schema for the externalservices API
type ExternalService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalServiceSpec   `json:"spec,omitempty"`
	Status ExternalServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExternalServiceList contains a list of ExternalService
type ExternalServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalService{}, &ExternalServiceList{})
}
