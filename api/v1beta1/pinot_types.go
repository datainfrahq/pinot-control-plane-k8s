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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PinotSpec defines the desired state of Pinot
type PinotSpec struct {
	DeploymentOrder []PinotNodeType   `json:"deploymentOrder"`
	External        ExternalSpec      `json:"external,omitempty"`
	K8sConfig       []K8sConfig       `json:"k8sConfig"`
	PinotNodeConfig []PinotNodeConfig `json:"pinotNodeConfig"`
	Nodes           []NodeSpec        `json:"nodes"`
}

type ExternalSpec struct {
	Zookeeper ZookeeperSpec `json:"zookeeper"`
}

type ZookeeperSpec struct {
	Spec ZookeeperConfig `json:"spec"`
}

type ZookeeperConfig struct {
	ZkAddress string `json:"zkAddress"`
}

type K8sConfig struct {
	Name               string                  `json:"name"`
	Volumes            []v1.Volume             `json:"volumes,omitempty"`
	VolumeMount        []v1.VolumeMount        `json:"volumeMount,omitempty"`
	Image              string                  `json:"image"`
	ImagePullPolicy    v1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	ServiceAccountName string                  `json:"serviceAccountName,omitempty"`
	Env                []v1.EnvVar             `json:"env,omitempty"`
	Tolerations        []v1.Toleration         `json:"tolerations,omitempty"`
	PodMetadata        Metadata                `json:"podMetadata,omitempty"`
	StorageConfig      []StorageConfig         `json:"storageConfig,omitempty"`
	NodeSelector       map[string]string       `json:"nodeSelector,omitempty"`
	Service            *v1.ServiceSpec         `json:"service,omitempty"`
	Resources          v1.ResourceRequirements `json:"resources,omitempty"`
}

type Metadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type StorageConfig struct {
	Name      string                       `json:"name"`
	MountPath string                       `json:"mountPath"`
	PvcSpec   v1.PersistentVolumeClaimSpec `json:"spec"`
}

type PinotNodeConfig struct {
	Name     string `json:"name"`
	JavaOpts string `json:"java_opts"`
	Data     string `json:"data"`
}

type PinotNodeType string

const (
	Controller PinotNodeType = "controller"
	Broker     PinotNodeType = "broker"
	Server     PinotNodeType = "server"
	Minion     PinotNodeType = "minion"
)

type NodeSpec struct {
	Name            string        `json:"name"`
	Kind            string        `json:"kind"`
	NodeType        PinotNodeType `json:"nodeType"`
	Replicas        int           `json:"replicas"`
	K8sConfig       string        `json:"k8sConfig"`
	PinotNodeConfig string        `json:"pinotNodeConfig"`
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
