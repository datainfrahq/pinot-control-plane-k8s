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
package schemacontroller

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
	PinotSchemaControllerCreateSuccess      = "PinotSchemaControllerCreateSuccess"
	PinotSchemaControllerCreateFail         = "PinotSchemaControllerCreateFail"
	PinotSchemaControllerGetSuccess         = "PinotSchemaControllerGetSuccess"
	PinotSchemaControllerGetFail            = "PinotSchemaControllerGetFail"
	PinotSchemaControllerUpdateSuccess      = "PinotSchemaControllerUpdateSuccess"
	PinotSchemaControllerUpdateFail         = "PinotSchemaControllerUpdateFail"
	PinotSchemaControllerDeleteSuccess      = "PinotSchemaControllerDeleteSuccess"
	PinotSchemaControllerDeleteFail         = "PinotSchemaControllerDeleteFail"
	PinotSchemaControllerPatchStatusSuccess = "PinotSchemaControllerPatchStatusSuccess"
	PinotSchemaControllerPatchStatusFail    = "PinotSchemaControllerPatchStatusFail"
	PinotSchemaControllerFinalizer          = "pinotschema.datainfra.io/finalizer"
	PinotControllerPort                     = "9000"
)

func (r *PinotSchemaReconciler) do(ctx context.Context, schema *v1beta1.PinotSchema) error {

	build := builder.NewBuilder(
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinotSchemaController"}),
	)

	// switch resp {

	// case controllerutil.OperationResultCreated:
	svcName, err := r.getControllerSvcUrl(schema.Namespace, schema.Spec.PinotCluster)
	if err != nil {
		return err
	}

	_, err = r.CreateOrUpdate(schema, svcName, *build)
	if err != nil {
		return err
	}
	if schema.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// 			// then lets add the finalizer and update the object. This is equivalent
		// 			// registering our finalizer.
		if !controllerutil.ContainsFinalizer(schema, PinotSchemaControllerFinalizer) {
			controllerutil.AddFinalizer(schema, PinotSchemaControllerFinalizer)
			if err := r.Update(ctx, schema); err != nil {
				return err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(schema, PinotSchemaControllerFinalizer) {
			// our finalizer is present, so lets handle any external dependency

			svcName, err := r.getControllerSvcUrl(schema.Namespace, schema.Spec.PinotCluster)
			if err != nil {
				return err
			}

			schemaName, err := getSchemaName(schema.Spec.PinotSchemaJson)
			if err != nil {
				return err
			}
			http := internalHTTP.NewHTTPClient(http.MethodDelete, makeControllerGetUpdateDeleteSchemaPath(svcName, schemaName), http.Client{}, []byte{})
			resp := http.Do()
			if resp.Err != nil {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerDeleteFail)
				return err
			}
			if resp.StatusCode != 200 {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerDeleteFail)
			} else {
				build.Recorder.GenericEvent(schema, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerDeleteSuccess)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(schema, PinotSchemaControllerFinalizer)
			if err := r.Update(ctx, schema); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *PinotSchemaReconciler) CreateOrUpdate(
	schema *v1beta1.PinotSchema,
	svcName string,
	build builder.Builder,
) (controllerutil.OperationResult, error) {

	// get schema name
	schemaName, err := getSchemaName(schema.Spec.PinotSchemaJson)
	if err != nil {
		return controllerutil.OperationResultNone, nil
	}

	fmt.Println(svcName)

	// get schema
	getHttp := internalHTTP.NewHTTPClient(http.MethodGet, makeControllerGetUpdateDeleteSchemaPath(svcName, schemaName), http.Client{}, []byte{})
	resp := getHttp.Do()
	if resp.Err != nil {
		fmt.Println(resp.StatusCode)

		build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerGetFail)
		return controllerutil.OperationResultNone, err
	}

	// if not found create schema
	if resp.StatusCode == 404 {

		postHttp := internalHTTP.NewHTTPClient(http.MethodPost, makeControllerCreateSchemaPath(svcName), http.Client{}, []byte(schema.Spec.PinotSchemaJson))
		resp := postHttp.Do()
		if resp.Err != nil {
			build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerCreateFail)
			return controllerutil.OperationResultNone, err
		}
		if resp.StatusCode == 200 {
			build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerCreateSuccess)
			result, err := r.makePatchPinotSchemaStatus(schema, PinotSchemaControllerCreateSuccess, string(resp.RespBody), v1.ConditionTrue, v1beta1.PinotSchemaCreateSuccess)
			if err != nil {
				return controllerutil.OperationResultNone, err
			} else {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s], Result [%s]", string(resp.RespBody), result), PinotSchemaControllerPatchStatusSuccess)
				return controllerutil.OperationResultCreated, nil
			}
		}
	} else if resp.StatusCode == 200 {
		ok, err := utils.IsEqualJson(schema.Status.CurrentSchemaJson, schema.Spec.PinotSchemaJson)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}

		if !ok {
			postHttp := internalHTTP.NewHTTPClient(http.MethodPut, makeControllerGetUpdateDeleteSchemaPath(svcName, schemaName), http.Client{}, []byte(schema.Spec.PinotSchemaJson))
			resp := postHttp.Do()
			if resp.Err != nil {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerUpdateFail)
				return controllerutil.OperationResultNone, err
			}
			fmt.Println(string(resp.RespBody))
			if resp.StatusCode == 200 {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerUpdateSuccess)
				result, err := r.makePatchPinotSchemaStatus(schema, PinotSchemaControllerUpdateSuccess, string(resp.RespBody), v1.ConditionTrue, v1beta1.PinotSchemaUpdateSuccess)
				if err != nil {
					return controllerutil.OperationResultNone, err
				} else {
					build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s], Result [%s]", string(resp.RespBody), result), PinotSchemaControllerPatchStatusSuccess)
					return controllerutil.OperationResultUpdated, nil
				}
			} else {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotSchemaControllerUpdateFail)
				return controllerutil.OperationResultNone, err
			}
		}
	}

	return controllerutil.OperationResultNone, nil
}

func getSchemaName(schemaJson string) (string, error) {
	var err error

	schema := make(map[string]json.RawMessage)
	if err = json.Unmarshal([]byte(schemaJson), &schema); err != nil {
		return "", err
	}

	return utils.TrimQuote(string(schema["schemaName"])), nil
}

func makeControllerCreateSchemaPath(svcName string) string { return svcName + "/schemas" }

func makeControllerGetUpdateDeleteSchemaPath(svcName, schemaName string) string {
	return svcName + "/schemas/" + schemaName
}

func (r *PinotSchemaReconciler) makePatchPinotSchemaStatus(
	schema *v1beta1.PinotSchema,
	msg string,
	reason string,
	status v1.ConditionStatus,
	pinotSchemaConditionType v1beta1.PinotSchemaConditionType,

) (controllerutil.OperationResult, error) {
	updatedPinotSchemaStatus := v1beta1.PinotSchemaStatus{}

	updatedPinotSchemaStatus.CurrentSchemaJson = schema.Spec.PinotSchemaJson
	updatedPinotSchemaStatus.LastUpdateTime = time.Now().Format(metav1.RFC3339Micro)
	updatedPinotSchemaStatus.Message = msg
	updatedPinotSchemaStatus.Reason = reason
	updatedPinotSchemaStatus.Status = status
	updatedPinotSchemaStatus.Type = pinotSchemaConditionType

	patchBytes, err := json.Marshal(map[string]v1beta1.PinotSchemaStatus{"status": updatedPinotSchemaStatus})
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if err := r.Client.Status().Patch(
		context.Background(),
		schema,
		client.RawPatch(types.MergePatchType,
			patchBytes,
		)); err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.OperationResultUpdatedStatusOnly, err
}

func (r *PinotSchemaReconciler) getControllerSvcUrl(namespace, pinotClusterName string) (string, error) {
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

	// newName := "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort

	return "http://localhost:9000", nil
}
