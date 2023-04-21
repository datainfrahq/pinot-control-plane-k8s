/*
DataInfra Pinot Control Plane (C) 2023 - 2024 DataInfra.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PinotTenantType string

const (
	BrokerTenant PinotTenantType = "broker"
	ServerTenant PinotTenantType = "server"
)

// PinotTenantSpec defines the desired state of PinotTenant
type PinotTenantSpec struct {
	// +required
	PinotCluster string `json:"pinotCluster"`
	// +required
	PinotTenantType PinotTenantType `json:"pinotTenantType"`
	// +required
	PinotTenantsJson string `json:"tenants.json"`
}

// PinotTenantStatus defines the observed state of PinotTenant
type PinotTenantStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PinotTenant is the Schema for the pinottenants API
type PinotTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PinotTenantSpec   `json:"spec,omitempty"`
	Status PinotTenantStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PinotTenantList contains a list of PinotTenant
type PinotTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PinotTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PinotTenant{}, &PinotTenantList{})
}
