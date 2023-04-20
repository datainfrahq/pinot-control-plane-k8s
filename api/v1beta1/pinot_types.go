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

// PinotSpec defines the desired state of Pinot
type PinotSpec struct {
	// +optional
	Plugins []string `json:"plugins"`
	// +required
	DeploymentOrder []PinotNodeType `json:"deploymentOrder"`
	// +required
	External ExternalSpec `json:"external,omitempty"`
	// +required
	K8sConfig []K8sConfig `json:"k8sConfig"`
	// +required
	PinotNodeConfig []PinotNodeConfig `json:"pinotNodeConfig"`
	// +required
	Nodes []NodeSpec `json:"nodes"`
}

type ExternalSpec struct {
	// +required
	Zookeeper ZookeeperSpec `json:"zookeeper"`
	// +optional
	DeepStorage DeepStorageSpec `json:"deepStorage"`
}

type ZookeeperSpec struct {
	// +required
	Spec ZookeeperConfig `json:"spec"`
}

type ZookeeperConfig struct {
	// +required
	ZkAddress string `json:"zkAddress"`
}

type DeepStorageSpec struct {
	// +optional
	Spec []DeepStorageConfig `json:"spec"`
}

type DeepStorageConfig struct {
	// +optional
	NodeType PinotNodeType `json:"nodeType"`
	// +optional
	Data string `json:"data"`
}

type K8sConfig struct {
	// +required
	Name string `json:"name"`
	// +optional
	Volumes []v1.Volume `json:"volumes,omitempty"`
	// +required
	Port []v1.ContainerPort `json:"port"`
	// +optional
	VolumeMount []v1.VolumeMount `json:"volumeMount,omitempty"`
	// +required
	Image string `json:"image"`
	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// +optional
	Env []v1.EnvVar `json:"env,omitempty"`
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
	// +optional
	PodMetadata Metadata `json:"podMetadata,omitempty"`
	// +optional
	StorageConfig []StorageConfig `json:"storageConfig,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +required
	Service *v1.ServiceSpec `json:"service"`
	// +optional
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`
	// +optional
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`
	// +optional
	StartUpProbe *v1.Probe `json:"startUpProbe,omitempty"`
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

type Metadata struct {
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type StorageConfig struct {
	// +required
	Name string `json:"name"`
	// +required
	MountPath string `json:"mountPath"`
	// +required
	PvcSpec v1.PersistentVolumeClaimSpec `json:"spec"`
}

type PinotNodeConfig struct {
	// +required
	Name string `json:"name"`
	// +required
	JavaOpts string `json:"java_opts"`
	// +required
	Data string `json:"data"`
}

type PinotNodeType string

const (
	Controller PinotNodeType = "controller"
	Broker     PinotNodeType = "broker"
	Server     PinotNodeType = "server"
	Minion     PinotNodeType = "minion"
)

type NodeSpec struct {
	// +required
	Name string `json:"name"`
	// +required
	Kind string `json:"kind"`
	// +required
	NodeType PinotNodeType `json:"nodeType"`
	// +required
	Replicas int `json:"replicas"`
	// +required
	K8sConfig string `json:"k8sConfig"`
	// +required
	PinotNodeConfig string `json:"pinotNodeConfig"`
}

// PinotStatus defines the observed state of Pinot
type PinotStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Pinot is the Schema for the pinots API
type Pinot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PinotSpec   `json:"spec,omitempty"`
	Status PinotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// PinotList contains a list of Pinot
type PinotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pinot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pinot{}, &PinotList{})
}
