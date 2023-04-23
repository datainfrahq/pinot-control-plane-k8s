// /*
// DataInfra Pinot Control Plane (C) 2023 - 2024 DataInfra.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package tablecontroller

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
	PinotTableControllerCreateSuccess      = "PinotTableControllerCreateSuccess"
	PinotTableControllerCreateFail         = "PinotTableControllerCreateFail"
	PinotTableControllerGetSuccess         = "PinotTableControllerGetSuccess"
	PinotTableControllerGetFail            = "PinotTableControllerGetFail"
	PinotTableControllerUpdateSuccess      = "PinotTableControllerUpdateSuccess"
	PinotTableControllerPatchStatusSuccess = "PinotTableControllerPatchStatusSuccess"
	PinotTableControllerPatchStatusFail    = "PinotTableControllerPatchStatusFail"
	PinotTableControllerUpdateFail         = "PinotTableControllerUpdateFail"
	PinotTableControllerDeleteSuccess      = "PinotTableControllerDeleteSuccess"
	PinotTableControllerDeleteFail         = "PinotTableControllerDeleteFail"
	PinotTableControllerFinalizer          = "pinottable.datainfra.io/finalizer"
	PinotControllerPort                    = "9000"
)

func (r *PinotTableReconciler) do(ctx context.Context, table *v1beta1.PinotTable) error {

	build := builder.NewBuilder(
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinorTableController"}),
	)

	svcName, err := r.getControllerSvcUrl(table.Namespace, table.Spec.PinotCluster)
	if err != nil {
		return err
	}

	_, err = r.CreateOrUpdate(table, svcName, *build)
	if err != nil {
		return err
	}
	if table.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// 	then lets add the finalizer and update the object. This is equivalent
		// 	registering our finalizer.
		if !controllerutil.ContainsFinalizer(table, PinotTableControllerFinalizer) {
			controllerutil.AddFinalizer(table, PinotTableControllerFinalizer)
			if err := r.Update(ctx, table); err != nil {
				return err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(table, PinotTableControllerFinalizer) {
			// our finalizer is present, so lets handle any external dependency

			svcName, err := r.getControllerSvcUrl(table.Namespace, table.Spec.PinotCluster)
			if err != nil {
				return err
			}

			tenantName, err := getTableName(table.Spec.PinotTablesJson)
			if err != nil {
				return err
			}
			http := internalHTTP.NewHTTPClient(http.MethodDelete, makeControllerGetUpdateDeleteTablePath(svcName, tenantName), http.Client{}, []byte{})
			resp := http.Do()
			if resp.Err != nil {
				build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerDeleteFail)
				return err
			}
			if resp.StatusCode != 200 {
				build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerDeleteFail)
			} else {
				build.Recorder.GenericEvent(table, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerDeleteSuccess)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(table, PinotTableControllerFinalizer)
			if err := r.Update(ctx, table); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PinotTableReconciler) CreateOrUpdate(
	table *v1beta1.PinotTable,
	svcName string,
	build builder.Builder,
) (controllerutil.OperationResult, error) {

	// get table name
	tableName, err := getTableName(table.Spec.PinotTablesJson)
	if err != nil {
		return controllerutil.OperationResultNone, nil
	}

	// get table
	getHttp := internalHTTP.NewHTTPClient(http.MethodGet, makeControllerGetUpdateDeleteTablePath(svcName, tableName), http.Client{}, []byte{})
	resp := getHttp.Do()
	if resp.Err != nil {
		build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerGetFail)
		return controllerutil.OperationResultNone, nil
	}

	// if not found create table
	if string(resp.RespBody) == "{}" {

		postHttp := internalHTTP.NewHTTPClient(http.MethodPost, makeControllerCreateTablePath(svcName), http.Client{}, []byte(table.Spec.PinotTablesJson))
		resp := postHttp.Do()
		if resp.Err != nil {
			build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerCreateFail)
			return controllerutil.OperationResultNone, err
		}

		if resp.StatusCode == 200 {
			result, err := r.makePatchPinotTableStatus(table, PinotTableControllerCreateSuccess, string(resp.RespBody), v1.ConditionTrue, PinotTableControllerCreateSuccess)
			if err != nil {
				build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s], Result [%s]", string(resp.RespBody), result), PinotTableControllerPatchStatusFail)
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(table, v1.EventTypeNormal, fmt.Sprintf("Resp [%s], Result [%s]", string(resp.RespBody), result), PinotTableControllerCreateSuccess)
			return controllerutil.OperationResultCreated, nil

		} else {
			_, err := r.makePatchPinotTableStatus(table, PinotTableControllerCreateSuccess, string(resp.RespBody), v1.ConditionTrue, PinotTableControllerCreateFail)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerCreateFail)
			return controllerutil.OperationResultCreated, nil
		}
	} else if string(resp.RespBody) != "{}" {
		ok, err := utils.IsEqualJson(table.Status.CurrentTableJson, table.Spec.PinotTablesJson)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}

		if !ok {
			postHttp := internalHTTP.NewHTTPClient(http.MethodPut, makeControllerGetUpdateDeleteTablePath(svcName, tableName), http.Client{}, []byte(table.Spec.PinotTablesJson))
			resp := postHttp.Do()
			if resp.Err != nil {
				build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerCreateFail)
				return controllerutil.OperationResultNone, err
			}

			if resp.StatusCode == 200 {
				build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerUpdateSuccess)
				result, err := r.makePatchPinotTableStatus(table, PinotTableControllerUpdateSuccess, string(resp.RespBody), v1.ConditionTrue, PinotTableControllerUpdateSuccess)
				if err != nil {
					build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s], Result [%s]", string(resp.RespBody), result), PinotTableControllerPatchStatusFail)
					return controllerutil.OperationResultNone, err
				} else {
					build.Recorder.GenericEvent(table, v1.EventTypeNormal, fmt.Sprintf("Resp [%s], Result [%s]", string(resp.RespBody), result), PinotTableControllerPatchStatusSuccess)
					return controllerutil.OperationResultUpdated, nil
				}
			} else {
				build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp.RespBody)), PinotTableControllerUpdateFail)
				return controllerutil.OperationResultNone, err
			}
		}
	}

	return controllerutil.OperationResultNone, nil
}

func getTableName(tablesJson string) (string, error) {
	var err error
	table := make(map[string]json.RawMessage)

	if err = json.Unmarshal([]byte(tablesJson), &table); err != nil {
		return "", err
	}

	return utils.TrimQuote(string(table["tableName"])), nil
}

func makeControllerCreateTablePath(svcName string) string { return svcName + "/tables" }

func makeControllerGetUpdateDeleteTablePath(svcName, tableName string) string {
	return svcName + "/tables/" + tableName
}

func (r *PinotTableReconciler) getControllerSvcUrl(namespace, pinotClusterName string) (string, error) {
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

func (r *PinotTableReconciler) makePatchPinotTableStatus(
	table *v1beta1.PinotTable,
	msg string,
	reason string,
	status v1.ConditionStatus,
	pinotTableConditionType v1beta1.PinotTableConditionType,

) (controllerutil.OperationResult, error) {
	updatedPinotTableStatus := v1beta1.PinotTableStatus{}

	updatedPinotTableStatus.CurrentTableJson = table.Spec.PinotTablesJson
	updatedPinotTableStatus.LastUpdateTime = time.Now().Format(metav1.RFC3339Micro)
	updatedPinotTableStatus.Message = msg
	updatedPinotTableStatus.Reason = reason
	updatedPinotTableStatus.Status = status
	updatedPinotTableStatus.Type = pinotTableConditionType

	patchBytes, err := json.Marshal(map[string]v1beta1.PinotTableStatus{"status": updatedPinotTableStatus})
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if err := r.Client.Status().Patch(
		context.Background(),
		table,
		client.RawPatch(types.MergePatchType,
			patchBytes,
		)); err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.OperationResultUpdatedStatusOnly, nil
}
