// /*
// DataInfra Pinot Control Plane (C) 2023 - 2024 DataInfra.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package tenantcontroller

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"time"

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
	PinotTenantControllerGetSuccess         = "PinotTenantControllerGetSuccess"
	PinotTenantControllerGetFail            = "PinotTenantControllerGetFail"
	PinotTenantControllerCreateSuccess      = "PinotTenantControllerCreateSuccess"
	PinotTenantControllerCreateFail         = "PinotTenantControllerCreateFail"
	PinotTenantControllerUpdateSuccess      = "PinotTenantControllerUpdateSuccess"
	PinotTenantControllerUpdateFail         = "PinotTenantControllerUpdateFail"
	PinotTenantControllerDeleteSuccess      = "PinotTenantControllerDeleteSuccess"
	PinotTenantControllerDeleteFail         = "PinotTenantControllerDeleteFail"
	PinotTenantControllerPatchStatusSuccess = "PinotTenantControllerPatchStatusSuccess"
	PinotTenantControllerPatchStatusFail    = "PinotTenantControllerPatchStatusFail"
	PinotTenantControllerFinalizer          = "pinottenant.datainfra.io/finalizer"
)

const (
	PinotControllerPort = "9000"
)

const (
	ControlPlaneUserName = "CONTROL_PLANE_USERNAME"
	ControlPlanePassword = "CONTROL_PLANE_PASSWORD"
)

func (r *PinotTenantReconciler) do(ctx context.Context, tenant *v1beta1.PinotTenant) error {
	build := builder.NewBuilder(
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinorTableController"}),
	)

	svcName, err := r.getControllerSvcUrl(tenant.Namespace, tenant.Spec.PinotCluster)
	if err != nil {
		return err
	}

	basicAuth, err := r.getAuthCreds(ctx, tenant)
	if err != nil {
		return err
	}

	_, err = r.CreateOrUpdate(tenant, svcName, *build, internalHTTP.Auth{BasicAuth: basicAuth})
	if err != nil {
		return err
	}

	if tenant.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// 	then lets add the finalizer and update the object. This is equivalent
		// 	registering our finalizer.
		if !controllerutil.ContainsFinalizer(tenant, PinotTenantControllerFinalizer) {
			controllerutil.AddFinalizer(tenant, PinotTenantControllerFinalizer)
			if err := r.Update(ctx, tenant); err != nil {
				return err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(tenant, PinotTenantControllerFinalizer) {
			// our finalizer is present, so lets handle any external dependency

			svcName, err := r.getControllerSvcUrl(tenant.Namespace, tenant.Spec.PinotCluster)
			if err != nil {
				return err
			}

			tenantName, err := getTenantName(tenant.Spec.PinotTenantsJson)
			if err != nil {
				return err
			}
			http := internalHTTP.NewHTTPClient(
				http.MethodDelete,
				makeControllerDeleteTenantPath(svcName, tenantName,
					string(tenant.Spec.PinotTenantType)),
				http.Client{},
				[]byte{},
				internalHTTP.Auth{BasicAuth: basicAuth},
			)
			respDeleteTenant := http.Do()
			if respDeleteTenant.Err != nil {
				return respDeleteTenant.Err
			}
			if respDeleteTenant.StatusCode != 200 {
				build.Recorder.GenericEvent(
					tenant,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s]", string(respDeleteTenant.PinotErrorResponse.Error)),
					PinotTenantControllerDeleteFail,
				)
			} else {
				build.Recorder.GenericEvent(
					tenant,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respDeleteTenant.PinotSuccessResponse.Status)),
					PinotTenantControllerDeleteSuccess,
				)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(tenant, PinotTenantControllerFinalizer)
			if err := r.Update(ctx, tenant); err != nil {
				return err
			}
		}
	}
	return nil
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

func makeControllerGetTenantPath(svcName, tenantName string) string {
	return svcName + "/tenants/" + tenantName
}

func makeControllerDeleteTenantPath(svcName, tenantName, pinotTenantType string) string {
	return svcName + "/tenants/" + tenantName + "?type=" + pinotTenantType
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
	var svcName string

	for range svcList.Items {
		svcName = svcList.Items[0].Name
	}

	newName := "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort

	return newName, nil
}

func (r *PinotTenantReconciler) CreateOrUpdate(
	tenant *v1beta1.PinotTenant,
	svcName string,
	build builder.Builder,
	auth internalHTTP.Auth,
) (controllerutil.OperationResult, error) {

	// get tenant name
	tenantName, err := getTenantName(tenant.Spec.PinotTenantsJson)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	// get tenant
	getHttp := internalHTTP.NewHTTPClient(
		http.MethodGet,
		makeControllerGetTenantPath(svcName, tenantName),
		http.Client{},
		[]byte{},
		auth,
	)
	respGetTenant := getHttp.Do()
	if respGetTenant.Err != nil {
		return controllerutil.OperationResultNone, respGetTenant.Err
	}

	// if not found create tenant
	if respGetTenant.StatusCode == 404 {

		postHttp := internalHTTP.NewHTTPClient(
			http.MethodPost,
			makeControllerCreateUpdateTenantPath(svcName),
			http.Client{},
			[]byte(tenant.Spec.PinotTenantsJson),
			auth,
		)
		respCreateTenant := postHttp.Do()
		if respCreateTenant.Err != nil {
			return controllerutil.OperationResultNone, respCreateTenant.Err
		}
		if respCreateTenant.StatusCode == 200 {
			_, err := r.makePatchPinotTenantStatus(
				tenant,
				PinotTenantControllerCreateSuccess,
				string(respCreateTenant.PinotSuccessResponse.Status),
				v1.ConditionTrue,
				PinotTenantControllerCreateSuccess,
			)
			if err != nil {
				return controllerutil.OperationResultNone, respCreateTenant.Err
			}
			build.Recorder.GenericEvent(
				tenant,
				v1.EventTypeNormal,
				fmt.Sprintf("Resp [%s]", string(respCreateTenant.PinotSuccessResponse.Status)),
				PinotTenantControllerCreateSuccess,
			)
			return controllerutil.OperationResultCreated, nil
		} else {
			_, err := r.makePatchPinotTenantStatus(
				tenant,
				PinotTenantControllerCreateFail,
				string(respCreateTenant.PinotErrorResponse.Error),
				v1.ConditionTrue,
				PinotTenantControllerCreateFail,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(
				tenant, v1.EventTypeWarning,
				fmt.Sprintf("Resp [%s]", string(respCreateTenant.PinotErrorResponse.Error)),
				PinotTenantControllerCreateFail,
			)
			return controllerutil.OperationResultNone, nil
		}

	} else if respGetTenant.StatusCode == 200 {

		ok, err := utils.IsEqualJson(tenant.Status.CurrentTenantsJson, tenant.Spec.PinotTenantsJson)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
		if !ok {
			postHttp := internalHTTP.NewHTTPClient(
				http.MethodPut,
				makeControllerCreateUpdateTenantPath(svcName),
				http.Client{},
				[]byte(tenant.Spec.PinotTenantsJson),
				auth,
			)
			respUpdateTenant := postHttp.Do()
			if respUpdateTenant.Err != nil {
				return controllerutil.OperationResultNone, respUpdateTenant.Err
			}

			if respUpdateTenant.StatusCode == 200 {
				build.Recorder.GenericEvent(
					tenant,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respUpdateTenant.PinotSuccessResponse.Status)),
					PinotTenantControllerUpdateSuccess,
				)
				_, err := r.makePatchPinotTenantStatus(
					tenant,
					PinotTenantControllerUpdateSuccess,
					string(respUpdateTenant.PinotSuccessResponse.Status),
					v1.ConditionTrue,
					PinotTenantControllerUpdateSuccess,
				)
				if err != nil {
					return controllerutil.OperationResultNone, err
				}
				build.Recorder.GenericEvent(
					tenant,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respUpdateTenant.PinotSuccessResponse.Status)),
					PinotTenantControllerPatchStatusSuccess,
				)
				return controllerutil.OperationResultUpdated, nil

			} else {
				build.Recorder.GenericEvent(
					tenant,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s]", string(respUpdateTenant.PinotErrorResponse.Error)),
					PinotTenantControllerUpdateFail,
				)
				return controllerutil.OperationResultNone, err
			}
		}

	}
	return controllerutil.OperationResultNone, nil
}

func (r *PinotTenantReconciler) makePatchPinotTenantStatus(
	tenant *v1beta1.PinotTenant,
	msg string,
	reason string,
	status v1.ConditionStatus,
	pinotTenantConditionType v1beta1.PinotTenantConditionType,

) (controllerutil.OperationResult, error) {
	updatedPinotTenantStatus := v1beta1.PinotTenantStatus{}

	updatedPinotTenantStatus.CurrentTenantsJson = tenant.Spec.PinotTenantsJson
	updatedPinotTenantStatus.LastUpdateTime = time.Now().Format(metav1.RFC3339Micro)
	updatedPinotTenantStatus.Message = msg
	updatedPinotTenantStatus.Reason = reason
	updatedPinotTenantStatus.Status = status
	updatedPinotTenantStatus.Type = pinotTenantConditionType

	patchBytes, err := json.Marshal(map[string]v1beta1.PinotTenantStatus{
		"status": updatedPinotTenantStatus})
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if err := r.Client.Status().Patch(
		context.TODO(),
		tenant,
		client.RawPatch(types.MergePatchType,
			patchBytes,
		)); err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.OperationResultUpdatedStatusOnly, nil
}

func (r *PinotTenantReconciler) getAuthCreds(ctx context.Context, tenant *v1beta1.PinotTenant) (internalHTTP.BasicAuth, error) {
	pinot := v1beta1.Pinot{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: tenant.Namespace,
		Name:      tenant.Spec.PinotCluster,
	},
		&pinot,
	); err != nil {
		return internalHTTP.BasicAuth{}, err
	}

	if pinot.Spec.Auth != (v1beta1.Auth{}) {
		secret := v1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{
			Namespace: pinot.Spec.Auth.SecretRef.Namespace,
			Name:      pinot.Spec.Auth.SecretRef.Name,
		},
			&secret,
		); err != nil {
			return internalHTTP.BasicAuth{}, err
		}

		creds := internalHTTP.BasicAuth{
			UserName: string(secret.Data[ControlPlaneUserName]),
			Password: string(secret.Data[ControlPlanePassword]),
		}

		return creds, nil

	}

	return internalHTTP.BasicAuth{}, nil
}
