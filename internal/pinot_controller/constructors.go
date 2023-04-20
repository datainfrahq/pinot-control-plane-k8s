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

package pinotcontroller

import (
	"github.com/datainfrahq/pinot-control-plane-k8s/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// create owner ref
func makeOwnerRef(apiVersion, kind, name string, uid types.UID) *metav1.OwnerReference {
	controller := true

	return &metav1.OwnerReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		UID:        uid,
		Controller: &controller,
	}
}

// NodeType and nodeSpec makes it easier to iterate decisions
// around N nodespec each to a nodeType
type NodeTypeNodeSpec struct {
	NodeType v1beta1.PinotNodeType
	NodeSpec v1beta1.NodeSpec
}

// constructor to nodeTypeNodeSpec. Order is constructed based on the deployment Order
func getAllNodeSpecForNodeType(pt *v1beta1.Pinot) []NodeTypeNodeSpec {

	// add more nodes types
	nodeSpecsByNodeType := map[v1beta1.PinotNodeType][]NodeTypeNodeSpec{
		pt.Spec.DeploymentOrder[0]: make([]NodeTypeNodeSpec, 0, 1),
		pt.Spec.DeploymentOrder[1]: make([]NodeTypeNodeSpec, 0, 1),
		pt.Spec.DeploymentOrder[2]: make([]NodeTypeNodeSpec, 0, 1),
		pt.Spec.DeploymentOrder[3]: make([]NodeTypeNodeSpec, 0, 1),
	}

	for _, nodeSpec := range pt.Spec.Nodes {
		nodeSpecs := nodeSpecsByNodeType[nodeSpec.NodeType]
		nodeSpecsByNodeType[nodeSpec.NodeType] = append(nodeSpecs, NodeTypeNodeSpec{nodeSpec.NodeType, nodeSpec})

	}

	allNodeSpecs := make([]NodeTypeNodeSpec, 0, len(pt.Spec.Nodes))

	allNodeSpecs = append(allNodeSpecs, nodeSpecsByNodeType[pt.Spec.DeploymentOrder[0]]...)
	allNodeSpecs = append(allNodeSpecs, nodeSpecsByNodeType[pt.Spec.DeploymentOrder[1]]...)
	allNodeSpecs = append(allNodeSpecs, nodeSpecsByNodeType[pt.Spec.DeploymentOrder[2]]...)
	allNodeSpecs = append(allNodeSpecs, nodeSpecsByNodeType[pt.Spec.DeploymentOrder[3]]...)

	return allNodeSpecs
}
