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
package pinotcontroller

import (
	"fmt"

	"github.com/datainfrahq/operator-runtime/builder"
	"github.com/datainfrahq/pinot-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ib interface {
	makeConfigMap(pinotNodeConfigGroupSpec *v1beta1.PinotNodeConfigGroups, pinotNodeSpec *v1beta1.NodeSpec) *builder.BuilderConfigMap
}

type internalBuilder struct {
	pinot        *v1beta1.Pinot
	client       client.Client
	ownerRef     *metav1.OwnerReference
	commonLabels map[string]string
}

func newInternalBuilder(
	pt *v1beta1.Pinot,
	client client.Client,
	nodeSpec *v1beta1.NodeSpec,
	ownerRef *metav1.OwnerReference) *internalBuilder {
	return &internalBuilder{
		pinot:        pt,
		client:       client,
		ownerRef:     ownerRef,
		commonLabels: makeLabels(pt, nodeSpec),
	}
}

func (ib *internalBuilder) makeConfigMap(
	pinotNodeConfigGroupSpec *v1beta1.PinotNodeConfigGroups,
	pinotNodeSpec *v1beta1.NodeSpec,
) *builder.BuilderConfigMap {

	var data map[string]string
	if pinotNodeSpec.NodeType == v1beta1.Controller {
		data = map[string]string{
			"pinot-controller.conf": fmt.Sprintf("%s", pinotNodeConfigGroupSpec.Data),
		}
	} else if pinotNodeSpec.NodeType == v1beta1.Broker {
		data = map[string]string{
			"pinot-broker.conf": fmt.Sprintf("%s", pinotNodeConfigGroupSpec.Data),
		}

	} else if pinotNodeSpec.NodeType == v1beta1.Minion {
		data = map[string]string{
			"pinot-minion.conf": fmt.Sprintf("%s", pinotNodeConfigGroupSpec.Data),
		}
	} else if pinotNodeSpec.NodeType == v1beta1.Server {
		data = map[string]string{
			"pinot-server.conf": fmt.Sprintf("%s", pinotNodeConfigGroupSpec.Data),
		}
	}
	return &builder.BuilderConfigMap{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      makeConfigMapName(ib.pinot.GetName(), pinotNodeConfigGroupSpec.Name),
				Namespace: ib.pinot.GetNamespace(),
				Labels:    ib.commonLabels,
			},
			Client:   ib.client,
			CrObject: ib.pinot,
			OwnerRef: *ib.ownerRef,
		},
		Data: data,
	}
}

func makeLabels(pt *v1beta1.Pinot, nodeSpec *v1beta1.NodeSpec) map[string]string {

	return map[string]string{
		"app":              "pinot",
		"custom_resource":  pt.Name,
		"nodeType":         string(nodeSpec.NodeType),
		"pinotConfigGroup": nodeSpec.PinotNodeConfigGroupName,
		"k8sConfigGroup":   nodeSpec.K8sConfigGroupName,
	}
}

func makeConfigMapName(nodeName, configGroupName string) string {
	return nodeName + "-" + configGroupName + "-" + "config"
}
