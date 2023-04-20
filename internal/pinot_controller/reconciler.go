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
	"context"

	"github.com/datainfrahq/operator-runtime/builder"
	"github.com/datainfrahq/operator-runtime/utils"
	"github.com/datainfrahq/pinot-control-plane-k8s/api/v1beta1"
	v1 "k8s.io/api/core/v1"
)

func (r *PinotReconciler) do(ctx context.Context, pt *v1beta1.Pinot) error {

	// create ownerRef passed to each object created
	getOwnerRef := makeOwnerRef(
		pt.APIVersion,
		pt.Kind,
		pt.Name,
		pt.UID,
	)

	var ib *internalBuilder

	nodeSpecs := getAllNodeSpecForNodeType(pt)

	pinotConfigMap := []builder.BuilderConfigMap{}
	pinotConfigMapHash := []utils.ConfigMapHash{}
	pinotDeploymentOrStatefulset := []builder.BuilderDeploymentStatefulSet{}
	pinotStorage := []builder.BuilderStorageConfig{}
	pinotService := []builder.BuilderService{}

	// For all the nodeSpec ie nodeType to nodeSpec
	// Get all the config group defined and append to configMap builder
	// For each config group defined create a configmap hash and append to configmaphash builder
	// Get all the k8s config group defined and append to deploymentstatefulset builder
	// For all the storage config defined in k8s config group append

	for _, nodeSpec := range nodeSpecs {

		ib = newInternalBuilder(pt, r.Client, &nodeSpec.NodeSpec, getOwnerRef)
		for _, pinotConfig := range pt.Spec.PinotNodeConfig {

			if nodeSpec.NodeSpec.PinotNodeConfig == pinotConfig.Name {
				cm := *ib.makeConfigMap(&pinotConfig, &nodeSpec.NodeSpec)
				pinotConfigMap = append(pinotConfigMap, cm)
				pinotConfigMapHash = append(pinotConfigMapHash, utils.ConfigMapHash{Object: &v1.ConfigMap{Data: cm.Data, ObjectMeta: cm.ObjectMeta}})
				for _, k8sConfig := range pt.Spec.K8sConfig {
					if nodeSpec.NodeSpec.K8sConfig == k8sConfig.Name {
						pinotDeploymentOrStatefulset = append(pinotDeploymentOrStatefulset, *ib.makeStsOrDeploy(
							ib.pinot,
							&pinotConfig,
							&nodeSpec.NodeSpec,
							&k8sConfig,
							&k8sConfig.StorageConfig,
							pinotConfigMapHash,
						))
						pinotService = append(pinotService, *ib.makeService(&k8sConfig, &nodeSpec.NodeSpec))
						for _, sc := range k8sConfig.StorageConfig {
							pinotStorage = append(pinotStorage, *ib.makePvc(&sc, &k8sConfig, &nodeSpec.NodeSpec))
						}
					}
				}
			}
		}
	}

	// construct builder
	builder := builder.NewBuilder(
		builder.ToNewBuilderConfigMap(pinotConfigMap),
		builder.ToNewBuilderDeploymentStatefulSet(pinotDeploymentOrStatefulset),
		builder.ToNewBuilderStorageConfig(pinotStorage),
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "pinotOperator"}),
		builder.ToNewBuilderContext(builder.BuilderContext{Context: ctx}),
		builder.ToNewBuilderService(pinotService),
		builder.ToNewBuilderStore(*builder.NewStore(ib.client, ib.commonLabels, pt.Namespace, pt)),
	)

	// All builder methods called are responsible for reconciling
	// and triggering reconcilers in case of state change.

	// reconcile configmap
	_, err := builder.ReconcileConfigMap()
	if err != nil {
		return err
	}

	_, err = builder.ReconcileService()
	if err != nil {
		return err
	}

	// reconcile depoyment or statefulset
	_, err = builder.ReconcileDeployOrSts()
	if err != nil {
		return err
	}

	// reconcile store
	if err := builder.ReconcileStore(); err != nil {
		return err
	}

	return nil
}
