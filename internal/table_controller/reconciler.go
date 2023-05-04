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

package tablecontroller

import (
	"context"
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
	PinotTableReloadAllSegments            = "PinotTableReloadAllSegments"
	PinotTableControllerFinalizer          = "pinottable.datainfra.io/finalizer"
)

const (
	PinotControllerPort = "9000"
)

const (
	ControlPlaneUserName = "CONTROL_PLANE_USERNAME"
	ControlPlanePassword = "CONTROL_PLANE_PASSWORD"
)

func (r *PinotTableReconciler) do(ctx context.Context, table *v1beta1.PinotTable) error {

	build := builder.NewBuilder(
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinorTableController"}),
	)

	svcName, err := r.getControllerSvcUrl(table.Namespace, table.Spec.PinotCluster)
	if err != nil {
		return err
	}

	basicAuth, err := r.getAuthCreds(ctx, table)
	if err != nil {
		return err
	}

	_, err = r.CreateOrUpdate(table, svcName, *build, internalHTTP.Auth{BasicAuth: basicAuth})
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
				return nil
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(table, PinotTableControllerFinalizer) {
			svcName, err := r.getControllerSvcUrl(table.Namespace, table.Spec.PinotCluster)
			if err != nil {
				return err
			}

			tenantName, err := utils.GetValueFromJson(table.Spec.PinotTablesJson, utils.TableName)
			if err != nil {
				return err
			}
			http := internalHTTP.NewHTTPClient(
				http.MethodDelete,
				makeControllerGetUpdateDeleteTablePath(svcName, tenantName),
				http.Client{}, []byte{},
				internalHTTP.Auth{BasicAuth: basicAuth},
			)
			respDeleteTable, err := http.Do()
			if err != nil {
				return err
			}
			if respDeleteTable.StatusCode != 200 {
				build.Recorder.GenericEvent(
					table,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s]", string(respDeleteTable.ResponseBody)),
					PinotTableControllerDeleteFail,
				)
			} else {
				build.Recorder.GenericEvent(
					table,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respDeleteTable.ResponseBody)),
					PinotTableControllerDeleteSuccess,
				)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(table, PinotTableControllerFinalizer)
			if err := r.Update(ctx, table); err != nil {
				return nil
			}
		}
	}
	return nil
}

// Get table if does not exist create
// if exists check for update
func (r *PinotTableReconciler) CreateOrUpdate(
	table *v1beta1.PinotTable,
	svcName string,
	build builder.Builder,
	auth internalHTTP.Auth,
) (controllerutil.OperationResult, error) {

	// get table name
	tableName, err := utils.GetValueFromJson(table.Spec.PinotTablesJson, utils.TableName)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	// get table
	getHttp := internalHTTP.NewHTTPClient(
		http.MethodGet,
		makeControllerGetUpdateDeleteTablePath(svcName, tableName),
		http.Client{}, []byte{},
		auth,
	)

	respGetTable, err := getHttp.Do()
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	// get - an empty response
	if respGetTable.ResponseBody == "{}" {

		postHttp := internalHTTP.NewHTTPClient(
			http.MethodPost,
			makeControllerCreateTablePath(svcName),
			http.Client{},
			[]byte(table.Spec.PinotTablesJson),
			auth,
		)
		// create table
		respCreateTable, err := postHttp.Do()
		if err != nil {
			return controllerutil.OperationResultNone, err
		}

		// create success
		if respCreateTable.StatusCode == 200 {

			// patch resource
			_, err := r.makePatchPinotTableStatus(
				table,
				PinotTableControllerCreateSuccess,
				string(respCreateTable.ResponseBody),
				v1.ConditionTrue,
				PinotTableControllerCreateSuccess,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(
				table,
				v1.EventTypeNormal,
				fmt.Sprintf("Resp [%s]", string(respCreateTable.ResponseBody)),
				PinotTableControllerCreateSuccess,
			)
			return controllerutil.OperationResultCreated, nil

		} else {
			_, err := r.makePatchPinotTableStatus(
				table,
				PinotTableControllerCreateSuccess,
				string(respCreateTable.ResponseBody),
				v1.ConditionTrue,
				PinotTableControllerCreateSuccess,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}

			build.Recorder.GenericEvent(
				table,
				v1.EventTypeWarning,
				fmt.Sprintf("Resp [%s]", string(respCreateTable.ResponseBody)),
				PinotTableControllerCreateFail,
			)
			return controllerutil.OperationResultNone, nil
		}
	} else if respGetTable.ResponseBody != "{}" {

		if table.Status.CurrentTableJson == "" {
			build.Recorder.GenericEvent(
				table,
				v1.EventTypeWarning,
				fmt.Sprintf("Table Exists on Pinot, but status is not updated"),
				PinotTableControllerUpdateFail,
			)

			_, err := r.makePatchPinotTableStatus(
				table,
				PinotTableControllerCreateSuccess,
				string(respGetTable.ResponseBody),
				v1.ConditionTrue,
				PinotTableControllerCreateSuccess,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
		}

		ok, err := utils.IsEqualJson(
			table.Status.CurrentTableJson,
			table.Spec.PinotTablesJson,
		)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}

		if !ok {
			postHttp := internalHTTP.NewHTTPClient(
				http.MethodPut,
				makeControllerGetUpdateDeleteTablePath(svcName, tableName),
				http.Client{},
				[]byte(table.Spec.PinotTablesJson),
				auth,
			)
			respUpdateTable, err := postHttp.Do()
			if err != nil {
				return controllerutil.OperationResultNone, err
			}

			if respUpdateTable.StatusCode == 200 {
				_, err := r.makePatchPinotTableStatus(
					table,
					PinotTableControllerUpdateSuccess,
					string(respUpdateTable.ResponseBody),
					v1.ConditionTrue,
					PinotTableControllerUpdateSuccess,
				)
				if err != nil {
					return controllerutil.OperationResultNone, err
				}
				build.Recorder.GenericEvent(
					table,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respUpdateTable.ResponseBody)),
					PinotTableControllerUpdateSuccess,
				)
				build.Recorder.GenericEvent(
					table,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respUpdateTable.ResponseBody)),
					PinotTableControllerPatchStatusSuccess)

				return controllerutil.OperationResultUpdated, nil
			} else {
				// patch status with failure and emit events
				_, err := r.makePatchPinotTableStatus(
					table,
					PinotTableControllerUpdateFail,
					string(respUpdateTable.ResponseBody),
					v1.ConditionTrue,
					PinotTableControllerUpdateFail,
				)
				if err != nil {
					return controllerutil.OperationResultNone, err
				}
				build.Recorder.GenericEvent(
					table,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s]", string(respUpdateTable.ResponseBody)),
					PinotTableControllerUpdateFail,
				)
				return controllerutil.OperationResultNone, err
			}
		}
	}

	return controllerutil.OperationResultNone, nil
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
	var svcName string

	for range svcList.Items {
		svcName = svcList.Items[0].Name
	}

	newName := "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort
	return newName, nil
}

func (r *PinotTableReconciler) makePatchPinotTableStatus(
	table *v1beta1.PinotTable,
	msg string,
	reason string,
	status v1.ConditionStatus,
	pinotTableConditionType string,

) (controllerutil.OperationResult, error) {

	if _, _, err := utils.PatchStatus(context.Background(), r.Client, table, func(obj client.Object) client.Object {
		in := obj.(*v1beta1.PinotTable)
		in.Status.CurrentTableJson = table.Spec.PinotTablesJson
		in.Status.LastUpdateTime = metav1.Time{Time: time.Now()}
		in.Status.Message = msg
		in.Status.Reason = reason
		in.Status.Status = status
		in.Status.Type = pinotTableConditionType
		in.Status.ReloadStatus = []string{}
		return in
	}); err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.OperationResultUpdatedStatusOnly, nil
}

func (r *PinotTableReconciler) getAuthCreds(ctx context.Context, table *v1beta1.PinotTable) (internalHTTP.BasicAuth, error) {
	pinot := v1beta1.Pinot{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: table.Namespace,
		Name:      table.Spec.PinotCluster,
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
