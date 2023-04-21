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
package tenantcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/datainfrahq/operator-runtime/builder"
	"github.com/datainfrahq/pinot-control-plane-k8s/api/v1beta1"
	internalHTTP "github.com/datainfrahq/pinot-control-plane-k8s/internal/http"
	"github.com/datainfrahq/pinot-control-plane-k8s/internal/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PinotTenantControllerCreateSuccess = "PinotTenantControllerCreateSuccess"
	PinotTenantControllerCreateFail    = "PinotTenantControllerCreateFail"
	PinotTenantControllerUpdateSuccess = "PinotTenantControllerUpdateSuccess"
	PinotTenantControllerUpdateFail    = "PinotTenantControllerUpdateFail"
	PinotTenantControllerDeleteSuccess = "PinotTenantControllerDeleteSuccess"
	PinotTenantControllerDeleteFail    = "PinotTenantControllerDeleteFail"
	PinotTenantControllerFinalizer     = "pinottenant.datainfra.io/finalizer"
	PinotControllerPort                = "9000"
)

func (r *PinotTenantReconciler) do(ctx context.Context, tenant *v1beta1.PinotTenant) error {
	svcName, err := r.getControllerSvcUrl(tenant.Namespace, tenant.Spec.PinotCluster)
	if err != nil {
		return err
	}

	getOwnerRef := makeOwnerRef(
		tenant.APIVersion,
		tenant.Kind,
		tenant.Name,
		tenant.UID,
	)
	cm := r.makeTenantConfigMap(tenant, getOwnerRef, tenant.Spec.PinotTenantsJson)

	build := builder.NewBuilder(
		builder.ToNewBuilderConfigMap([]builder.BuilderConfigMap{*cm}),
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinotTenantController"}),
		builder.ToNewBuilderContext(builder.BuilderContext{Context: ctx}),
		builder.ToNewBuilderStore(
			*builder.NewStore(r.Client, map[string]string{"Tenant": tenant.Name}, tenant.Namespace, tenant),
		),
	)

	resp, err := build.ReconcileConfigMap()
	if err != nil {
		return err
	}

	switch resp {
	case controllerutil.OperationResultCreated:

		http := internalHTTP.NewHTTPClient(http.MethodPost, makeControllerCreateUpdateTenantPath(svcName), http.Client{}, []byte(tenant.Spec.PinotTenantsJson))
		resp, err := http.Do()
		if err != nil {
			build.Recorder.GenericEvent(tenant, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerCreateFail)
			return err
		}

		if getRespCode(resp) != "200" && getRespCode(resp) != "" {
			build.Recorder.GenericEvent(tenant, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerCreateFail)
		} else {
			build.Recorder.GenericEvent(tenant, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerCreateSuccess)
		}
	case controllerutil.OperationResultUpdated:

		http := internalHTTP.NewHTTPClient(http.MethodPut, makeControllerCreateUpdateTenantPath(svcName), http.Client{}, []byte(tenant.Spec.PinotTenantsJson))
		resp, err := http.Do()
		if err != nil {
			build.Recorder.GenericEvent(tenant, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerUpdateFail)
			return err
		}

		if getRespCode(resp) != "200" && getRespCode(resp) != "" {
			build.Recorder.GenericEvent(tenant, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerUpdateFail)
		} else {
			build.Recorder.GenericEvent(tenant, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerCreateSuccess)
		}

	default:
		if tenant.ObjectMeta.DeletionTimestamp.IsZero() {
			// The object is not being deleted, so if it does not have our finalizer,
			// then lets add the finalizer and update the object. This is equivalent
			// registering our finalizer.
			if !controllerutil.ContainsFinalizer(tenant, PinotTenantControllerFinalizer) {
				controllerutil.AddFinalizer(tenant, PinotTenantControllerFinalizer)
				if err := r.Update(ctx, tenant); err != nil {
					return err
				}
			}
		} else {
			// The object is being deleted
			if controllerutil.ContainsFinalizer(tenant, PinotTenantControllerFinalizer) {
				// our finalizer is present, so lets handle any external dependency
				tenantName, err := getTenantName(tenant.Spec.PinotTenantsJson)
				if err != nil {
					return err
				}
				http := internalHTTP.NewHTTPClient(http.MethodDelete, makeControllerDeleteTenantPath(svcName, tenantName), http.Client{}, []byte{})
				resp, err := http.Do()
				if err != nil {
					build.Recorder.GenericEvent(tenant, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerDeleteFail)
					return err
				}
				if getRespCode(resp) != "200" && getRespCode(resp) != "" {
					build.Recorder.GenericEvent(tenant, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerDeleteFail)
				} else {
					build.Recorder.GenericEvent(tenant, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotTenantControllerDeleteSuccess)
				}

				// remove our finalizer from the list and update it.
				controllerutil.RemoveFinalizer(tenant, PinotTenantControllerFinalizer)
				if err := r.Update(ctx, tenant); err != nil {
					return err
				}
			}
			return nil
		}
	}

	return nil
}

func (r *PinotTenantReconciler) makeTenantConfigMap(
	tenant *v1beta1.PinotTenant,
	ownerRef *metav1.OwnerReference,
	data interface{},
) *builder.BuilderConfigMap {

	configMap := &builder.BuilderConfigMap{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tenant.GetName() + "-" + "tenant",
				Namespace: tenant.GetNamespace(),
			},
			Client:   r.Client,
			CrObject: tenant,
			OwnerRef: *ownerRef,
		},
		Data: map[string]string{
			"tenants.json": data.(string),
		},
	}

	return configMap
}

// create owner ref ie pinot tenant controller
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

func getTenantName(tenantsJson string) (string, error) {
	var err error

	schema := make(map[string]json.RawMessage)
	if err = json.Unmarshal([]byte(tenantsJson), &schema); err != nil {
		return "", err
	}

	return utils.TrimQuote(string(schema["tenantName"])), nil
}

func getRespCode(resp []byte) string {
	var err error

	respMap := make(map[string]json.RawMessage)
	if err = json.Unmarshal(resp, &respMap); err != nil {
		return ""
	}

	return utils.TrimQuote(string(respMap["code"]))
}

func makeControllerCreateUpdateTenantPath(svcName string) string { return svcName + "/tenants" }

func makeControllerDeleteTenantPath(svcName, tenantName string) string {
	return svcName + "/tenants/" + tenantName
}

func (r *PinotTenantReconciler) getControllerSvcUrl(namespace, pinotClusterName string) (string, error) {
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(map[string]string{
			"custom_resource": pinotClusterName,
			"nodeType":        "controller",
		}),
	}
	svcList := &v1.ServiceList{}
	if err := r.Client.List(context.Background(), svcList, listOpts...); err != nil {
		return "", err
	}
	// var svcName string

	// for range svcList.Items {
	// 	svcName = svcList.Items[0].Name
	// }

	//newName := "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort

	return "http://localhost:9000", nil
}
