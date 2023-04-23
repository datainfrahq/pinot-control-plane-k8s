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

type PinotTableType string

const (
	RealTimeTable    PinotTableType = "realtime"
	OfflineTimeTable PinotTableType = "offline"
)

// PinotTableSpec defines the desired state of PinotTable
type PinotTableSpec struct {
	// +required
	PinotCluster string `json:"pinotCluster"`
	// +required
	PinotSchema string `json:"pinotSchema"`
	// +required
	PinotTableType PinotTableType `json:"pinotTableType"`
	// +required
	PinotTablesJson string `json:"tables.json"`
}

type PinotTableConditionType string

const (
	PinotTableCreateSuccess PinotSchemaConditionType = "PinotTableCreateSuccess"
	PinotTableUpdateSuccess PinotSchemaConditionType = "PinotTableUpdateSuccess"
	PinotTableCreateFail    PinotSchemaConditionType = "PinotTableCreateFail"
	PinotTableUpdateFail    PinotSchemaConditionType = "PinotTableUpdateFail"
)

// PinotTableStatus defines the observed state of PinotTable
type PinotTableStatus struct {
	Type             PinotTableConditionType `json:"type,omitempty"`
	Status           v1.ConditionStatus      `json:"status,omitempty"`
	Reason           string                  `json:"reason,omitempty"`
	Message          string                  `json:"message,omitempty"`
	LastUpdateTime   string                  `json:"lastUpdateTime,omitempty"`
	CurrentTableJson string                  `json:"currentTable.json"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Pinot_Cluster",type="string",JSONPath=".spec.pinotCluster"
// +kubebuilder:printcolumn:name="Pinot_Schema",type="string",JSONPath=".spec.pinotSchema"
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
