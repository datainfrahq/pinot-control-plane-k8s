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
	"github.com/datainfrahq/operator-runtime/utils"
	"github.com/datainfrahq/pinot-operator/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MinionConfigMapVolumeMountPath     = "/var/pinot/minion/config"
	ServerConfigMapVolumeMountPath     = "/var/pinot/server/config"
	ControllerConfigMapVolumeMountPath = "/var/pinot/controller/config"
	BrokerConfigMapVolumeMountPath     = "/var/pinot/broker/config"
	MinionConfName                     = "pinot-minion.conf"
	ServerConfName                     = "pinot-server.conf"
	ControllerConfName                 = "pinot-controller.conf"
	BrokerConfName                     = "pinot-broker.conf"
	StartBroker                        = "StartBroker"
	StartController                    = "StartController"
	StartServer                        = "StartServer"
	StartMinion                        = "StartMinion"
)

type ib interface {
	makeConfigMap(
		pinotNodeConfigGroupSpec *v1beta1.PinotNodeConfig,
		pinotNodeSpec *v1beta1.NodeSpec) *builder.BuilderConfigMap
	makeStsOrDeploy(
		pinot *v1beta1.Pinot,
		pinotNodeConfig *v1beta1.PinotNodeConfig,
		pinotNodeSpec *v1beta1.NodeSpec,
		k8sConfig *v1beta1.K8sConfig,
		storageConfig *[]v1beta1.StorageConfig,
		configHash []utils.ConfigMapHash,
	) *builder.BuilderDeploymentStatefulSet
	makePvc(
		sc *v1beta1.StorageConfig,
		k8sConfig *v1beta1.K8sConfig,
		pinotNodeSpec *v1beta1.NodeSpec,
	) *builder.BuilderStorageConfig
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
	pinotNodeConfig *v1beta1.PinotNodeConfig,
	pinotNodeSpec *v1beta1.NodeSpec,
) *builder.BuilderConfigMap {

	var data map[string]string

	if pinotNodeSpec.NodeType == v1beta1.Controller {
		data = map[string]string{
			ControllerConfName: fmt.Sprintf("%s\n%s\n%s", pinotNodeConfig.Data, "controller.zk.str="+ib.pinot.Spec.External.Zookeeper.Spec.ZkAddress, "controller.helix.cluster.name="+ib.pinot.Name),
		}
	} else if pinotNodeSpec.NodeType == v1beta1.Broker {
		data = map[string]string{
			BrokerConfName: fmt.Sprintf("%s", pinotNodeConfig.Data),
		}

	} else if pinotNodeSpec.NodeType == v1beta1.Minion {
		data = map[string]string{
			MinionConfName: fmt.Sprintf("%s", pinotNodeConfig.Data),
		}
	} else if pinotNodeSpec.NodeType == v1beta1.Server {
		data = map[string]string{
			ServerConfName: fmt.Sprintf("%s", pinotNodeConfig.Data),
		}
	}
	return &builder.BuilderConfigMap{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      makeConfigMapName(ib.pinot.GetName(), pinotNodeConfig.Name),
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

func (ib *internalBuilder) makeStsOrDeploy(
	pinot *v1beta1.Pinot,
	pinotNodeConfig *v1beta1.PinotNodeConfig,
	pinotNodeSpec *v1beta1.NodeSpec,
	k8sConfig *v1beta1.K8sConfig,
	storageConfig *[]v1beta1.StorageConfig,
	configHash []utils.ConfigMapHash,
) *builder.BuilderDeploymentStatefulSet {

	podSpec := v1.PodSpec{
		NodeSelector: k8sConfig.NodeSelector,
		Containers: []v1.Container{
			{
				Name:            pinotNodeSpec.Name + "-" + string(pinotNodeSpec.NodeType),
				Image:           k8sConfig.Image,
				Args:            makeArgs(ib.pinot, pinotNodeSpec.NodeType),
				ImagePullPolicy: k8sConfig.ImagePullPolicy,
				Ports:           makePorts(k8sConfig, pinotNodeSpec.NodeType),
				Env:             getEnv(ib.pinot, pinotNodeConfig, k8sConfig, configHash),
				VolumeMounts:    getVolumeMounts(pinot, k8sConfig, pinotNodeSpec, storageConfig),
				Resources:       k8sConfig.Resources,
			},
		},
		Volumes:            getVolume(ib.pinot, k8sConfig, storageConfig, pinotNodeSpec),
		ServiceAccountName: k8sConfig.ServiceAccountName,
	}

	deployment := builder.BuilderDeploymentStatefulSet{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pinotNodeSpec.K8sConfig + "-" + pinotNodeSpec.Name,
				Namespace: ib.pinot.GetNamespace(),
				Labels:    ib.commonLabels,
			},
			Client:   ib.client,
			CrObject: ib.pinot,
			OwnerRef: *ib.ownerRef,
		},
		Replicas: int32(pinotNodeSpec.Replicas),
		Labels:   ib.commonLabels,
		Kind:     pinotNodeSpec.Kind,
		PodSpec:  &podSpec,
	}

	return &deployment
}

func (ib *internalBuilder) makePvc(
	sc *v1beta1.StorageConfig,
	k8sConfig *v1beta1.K8sConfig,
	pinotNodeSpec *v1beta1.NodeSpec,
) *builder.BuilderStorageConfig {
	return &builder.BuilderStorageConfig{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      makePvcName(pinotNodeSpec.Name, k8sConfig.Name, sc.Name),
				Namespace: ib.pinot.GetNamespace()},
			Client:   ib.client,
			CrObject: ib.pinot,
			Labels:   ib.commonLabels,
			OwnerRef: *ib.ownerRef,
		},
		PvcSpec: &sc.PvcSpec,
	}
}

func makeLabels(pt *v1beta1.Pinot, nodeSpec *v1beta1.NodeSpec) map[string]string {

	return map[string]string{
		"app":              "pinot",
		"custom_resource":  pt.Name,
		"nodeType":         string(nodeSpec.NodeType),
		"pinotConfigGroup": nodeSpec.PinotNodeConfig,
		"k8sConfigGroup":   nodeSpec.K8sConfig,
	}
}

func (ib *internalBuilder) makeService(
	k8sConfig *v1beta1.K8sConfig,
	nodeSpec *v1beta1.NodeSpec,
) *builder.BuilderService {
	return &builder.BuilderService{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      makeSvcName(ib.pinot.Name, k8sConfig.Name),
				Namespace: ib.pinot.GetNamespace()},
			Client:   ib.client,
			CrObject: ib.pinot,
			OwnerRef: *ib.ownerRef,
			Labels:   ib.commonLabels,
		},
		SelectorLabels: ib.commonLabels,
		ServiceSpec:    k8sConfig.Service,
	}
}

func makeConfigMapName(pinotName, pinotNodeConfig string) string {
	return pinotName + "-" + pinotNodeConfig + "-" + "config"
}

func makeSvcName(pinotName, k8sConfig string) string {
	return pinotName + "-" + k8sConfig
}

func makePvcName(nodeName, k8sConfig, storageConfig string) string {
	return nodeName + "-" + k8sConfig + "-" + storageConfig
}

func getVolumeMounts(
	pinot *v1beta1.Pinot,
	k8sConfig *v1beta1.K8sConfig,
	pinotNodeSpec *v1beta1.NodeSpec,
	storageConfig *[]v1beta1.StorageConfig,
) []v1.VolumeMount {

	var volumeMount = []v1.VolumeMount{}
	for _, sc := range *storageConfig {
		volumeMount = append(
			volumeMount,
			v1.VolumeMount{
				MountPath: sc.MountPath,
				Name:      sc.Name + "-" + "pvc",
			},
		)
	}

	var mountPath string

	switch pinotNodeSpec.NodeType {
	case v1beta1.Broker:
		mountPath = BrokerConfigMapVolumeMountPath
	case v1beta1.Controller:
		mountPath = ControllerConfigMapVolumeMountPath
	case v1beta1.Server:
		mountPath = ServerConfigMapVolumeMountPath
	case v1beta1.Minion:
		mountPath = MinionConfigMapVolumeMountPath
	}

	volumeMount = append(
		volumeMount,
		v1.VolumeMount{
			MountPath: mountPath,
			Name:      k8sConfig.Name + "-" + "cm",
		},
	)

	volumeMount = append(volumeMount, k8sConfig.VolumeMount...)
	return volumeMount
}

func getVolume(
	pinot *v1beta1.Pinot,
	k8sConfig *v1beta1.K8sConfig,
	storageConfig *[]v1beta1.StorageConfig,
	pinotNodeSpec *v1beta1.NodeSpec,
) []v1.Volume {
	var volumeHolder = []v1.Volume{}

	for _, sc := range *storageConfig {
		volumeHolder = append(volumeHolder,
			v1.Volume{
				Name: sc.Name + "-" + "pvc",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: makePvcName(pinotNodeSpec.Name, k8sConfig.Name, sc.Name),
					},
				},
			},
			v1.Volume{
				Name: k8sConfig.Name + "-" + "cm",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: makeConfigMapName(pinot.Name, pinotNodeSpec.PinotNodeConfig),
						},
					},
				},
			},
		)
	}

	volumeHolder = append(volumeHolder, k8sConfig.Volumes...)
	return volumeHolder
}

func makeArgs(
	pinot *v1beta1.Pinot,
	nodeType v1beta1.PinotNodeType,
) []string {
	switch nodeType {
	case v1beta1.Broker:
		return []string{
			StartBroker,
			"-clusterName",
			pinot.Name,
			"-zkAddress",
			pinot.Spec.External.Zookeeper.Spec.ZkAddress,
			"-configFileName",
			BrokerConfigMapVolumeMountPath + "/" + BrokerConfName,
		}
	case v1beta1.Controller:
		return []string{
			StartController,
			"-clusterName",
			pinot.Name,
			"-configFileName",
			ControllerConfigMapVolumeMountPath + "/" + ControllerConfName,
		}
	case v1beta1.Server:
		return []string{
			StartServer,
			"-clusterName",
			pinot.Name,
			"-zkAddress",
			pinot.Spec.External.Zookeeper.Spec.ZkAddress,
			"-configFileName",
			ServerConfigMapVolumeMountPath + "/" + ServerConfName,
		}
	case v1beta1.Minion:
		return []string{
			StartMinion,
			"-clusterName",
			pinot.Name,
			"-zkAddress",
			pinot.Spec.External.Zookeeper.Spec.ZkAddress,
			"-configFileName",
			MinionConfigMapVolumeMountPath + "/" + MinionConfName,
		}
	default:
		return nil
	}
}

func getEnv(
	pinot *v1beta1.Pinot,
	pinotNodeConfig *v1beta1.PinotNodeConfig,
	k8sConfigGroup *v1beta1.K8sConfig,
	configHash []utils.ConfigMapHash,
) []v1.EnvVar {

	var envs, hashHolder []v1.EnvVar

	jvmOpts := v1.EnvVar{Name: "JAVA_OPTS", Value: pinotNodeConfig.JavaOpts}

	envs = append(envs, k8sConfigGroup.Env...)

	envs = append(envs, jvmOpts)

	hashes, _ := utils.MakeConfigMapHash(configHash)

	for _, cmhash := range hashes {
		if makeConfigMapName(pinot.Name, pinotNodeConfig.Name) == cmhash.Name {
			hashHolder = append(hashHolder, v1.EnvVar{Name: cmhash.Name, Value: cmhash.HashVaule})
		}
	}

	envs = append(envs, hashHolder...)
	return envs
}

func makePorts(k8sConfig *v1beta1.K8sConfig, nodeType v1beta1.PinotNodeType) []v1.ContainerPort {
	switch nodeType {
	case v1beta1.Broker:
		return []v1.ContainerPort{
			{
				Name:          string(v1beta1.Broker),
				ContainerPort: k8sConfig.Port,
				Protocol:      v1.ProtocolTCP,
			},
		}
	case v1beta1.Controller:
		return []v1.ContainerPort{
			{
				Name:          string(v1beta1.Controller),
				ContainerPort: k8sConfig.Port,
				Protocol:      v1.ProtocolTCP,
			},
		}
	case v1beta1.Server:
		return []v1.ContainerPort{
			{
				Name:          "netty",
				ContainerPort: 8098,
				Protocol:      v1.ProtocolTCP,
			},
			{
				Name:          "admin",
				ContainerPort: k8sConfig.Port,
				Protocol:      v1.ProtocolTCP,
			},
		}
	case v1beta1.Minion:
		return []v1.ContainerPort{
			{
				Name:          string(v1beta1.Minion),
				ContainerPort: k8sConfig.Port,
				Protocol:      v1.ProtocolTCP,
			},
		}
	}
	return nil
}
