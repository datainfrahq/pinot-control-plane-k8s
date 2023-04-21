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

// PinotTableSpec defines the desired state of PinotTable
type PinotTableSpec struct {
	Foo string `json:"foo,omitempty"`
}

// PinotTableStatus defines the observed state of PinotTable
type PinotTableStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PinotTable is the Schema for the pinottables API
type PinotTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PinotTableSpec   `json:"spec,omitempty"`
	Status PinotTableStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PinotTableList contains a list of PinotTable
type PinotTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PinotTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PinotTable{}, &PinotTableList{})
}
