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
	PinotTableControllerCreateSuccess = "PinotSchemaControllerCreateSuccess"
	PinotTableControllerCreateFail    = "PinotTableControllerCreateFail"
	PinotTableControllerUpdateSuccess = "PinotTableControllerUpdateSuccess"
	PinotTableControllerUpdateFail    = "PinotTableControllerUpdateFail"
	PinotTableControllerDeleteSuccess = "PinotTableControllerDeleteSuccess"
	PinotTableControllerDeleteFail    = "PinotTableControllerDeleteFail"
	PinotTableControllerFinalizer     = "pinottable.datainfra.io/finalizer"
	PinotControllerPort               = "9000"
)

func (r *PinotTableReconciler) do(ctx context.Context, table *v1beta1.PinotTable) error {

	svcName, err := r.getControllerSvcUrl(table.Namespace, table.Spec.PinotCluster)
	if err != nil {
		return err
	}

	getOwnerRef := makeOwnerRef(
		table.APIVersion,
		table.Kind,
		table.Name,
		table.UID,
	)
	cm := r.makeTableConfigMap(table, getOwnerRef, table.Spec.PinotTablesJson)

	build := builder.NewBuilder(
		builder.ToNewBuilderConfigMap([]builder.BuilderConfigMap{*cm}),
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinotTableController"}),
		builder.ToNewBuilderContext(builder.BuilderContext{Context: ctx}),
		builder.ToNewBuilderStore(
			*builder.NewStore(r.Client, map[string]string{"table": table.Name}, table.Namespace, table),
		),
	)

	resp, err := build.ReconcileConfigMap()
	if err != nil {
		return err
	}

	switch resp {
	case controllerutil.OperationResultCreated:

		http := internalHTTP.NewHTTPClient(http.MethodPost, makeControllerCreateTablePath(svcName), http.Client{}, []byte(table.Spec.PinotTablesJson))
		resp, err := http.Do()
		if err != nil {
			build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerCreateFail)
			return err
		}

		if getRespCode(resp) != "200" && getRespCode(resp) != "" {
			build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerCreateFail)
		} else {
			build.Recorder.GenericEvent(table, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerCreateSuccess)
		}
	case controllerutil.OperationResultUpdated:
		tableName, err := getTableName(table.Spec.PinotTablesJson)
		if err != nil {
			return err
		}

		http := internalHTTP.NewHTTPClient(http.MethodPut, makeControllerUpdateDeleteTablePath(svcName, tableName), http.Client{}, []byte(table.Spec.PinotTablesJson))
		resp, err := http.Do()
		if err != nil {
			build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerUpdateFail)
			return err
		}

		if getRespCode(resp) != "200" && getRespCode(resp) != "" {
			build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerUpdateFail)
		} else {
			build.Recorder.GenericEvent(table, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerUpdateSuccess)
		}

	default:
		if table.ObjectMeta.DeletionTimestamp.IsZero() {
			// The object is not being deleted, so if it does not have our finalizer,
			// then lets add the finalizer and update the object. This is equivalent
			// registering our finalizer.
			if !controllerutil.ContainsFinalizer(table, PinotTableControllerFinalizer) {
				controllerutil.AddFinalizer(table, PinotTableControllerFinalizer)
				if err := r.Update(ctx, table); err != nil {
					return err
				}
			}
		} else {
			// The object is being deleted
			if controllerutil.ContainsFinalizer(table, PinotTableControllerFinalizer) {
				// our finalizer is present, so lets handle any external dependency
				tableName, err := getTableName(table.Spec.PinotTablesJson)
				if err != nil {
					return err
				}
				http := internalHTTP.NewHTTPClient(http.MethodDelete, makeControllerUpdateDeleteTablePath(svcName, tableName), http.Client{}, []byte{})
				resp, err := http.Do()
				if err != nil {
					build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerDeleteFail)
					return err
				}
				if getRespCode(resp) != "200" && getRespCode(resp) != "" {
					build.Recorder.GenericEvent(table, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerDeleteFail)
				} else {
					build.Recorder.GenericEvent(table, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotTableControllerDeleteSuccess)
				}

				// remove our finalizer from the list and update it.
				controllerutil.RemoveFinalizer(table, PinotTableControllerFinalizer)
				if err := r.Update(ctx, table); err != nil {
					return err
				}
			}
			return nil
		}
	}

	return nil
}

func (r *PinotTableReconciler) makeTableConfigMap(
	table *v1beta1.PinotTable,
	ownerRef *metav1.OwnerReference,
	data interface{},
) *builder.BuilderConfigMap {

	configMap := &builder.BuilderConfigMap{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      table.GetName() + "-" + "table",
				Namespace: table.GetNamespace(),
			},
			Client:   r.Client,
			CrObject: table,
			OwnerRef: *ownerRef,
		},
		Data: map[string]string{
			"tables.json": data.(string),
		},
	}

	return configMap
}

// create owner ref ie pinot table controller
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

func getTableName(tablesJson string) (string, error) {
	var err error

	schema := make(map[string]json.RawMessage)
	if err = json.Unmarshal([]byte(tablesJson), &schema); err != nil {
		return "", err
	}

	return utils.TrimQuote(string(schema["tableName"])), nil
}

func getRespCode(resp []byte) string {
	var err error

	respMap := make(map[string]json.RawMessage)
	if err = json.Unmarshal(resp, &respMap); err != nil {
		return ""
	}

	return utils.TrimQuote(string(respMap["code"]))
}

func makeControllerCreateTablePath(svcName string) string { return svcName + "/tables" }

func makeControllerUpdateDeleteTablePath(svcName, tableName string) string {
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
	var svcName string

	for range svcList.Items {
		svcName = svcList.Items[0].Name
	}

	newName := "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort

	return newName, nil
}
