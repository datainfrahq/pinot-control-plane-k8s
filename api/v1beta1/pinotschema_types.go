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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PinotSchemaSpec defines the desired state of PinotSchema
type PinotSchemaSpec struct {
	// +required
	PinotCluster string `json:"pinotCluster"`
	// +required
	PinotSchemaJson string `json:"schema.json"`
}

// PinotSchemaStatus defines the observed state of PinotSchema
type PinotSchemaStatus struct {
	Type               string             `json:"type,omitempty"`
	Status             v1.ConditionStatus `json:"status,omitempty"`
	Reason             string             `json:"reason,omitempty"`
	Message            string             `json:"message,omitempty"`
	LastUpdateTime     metav1.Time        `json:"lastUpdateTime,omitempty"`
	CurrentSchemasJson string             `json:"currentSchemas.json"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Pinot_Cluster",type="string",JSONPath=".spec.pinotCluster"
// PinotSchema is the Schema for the pinotschemas API
type PinotSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PinotSchemaSpec   `json:"spec,omitempty"`
	Status PinotSchemaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PinotSchemaList contains a list of PinotSchema
type PinotSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PinotSchema `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PinotSchema{}, &PinotSchemaList{})
}
