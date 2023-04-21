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
	PinotSchemaControllerCreateSuccess = "PinotSchemaControllerCreateSuccess"
	PinotSchemaControllerCreateFail    = "PinotSchemaControllerCreateFail"
	PinotSchemaControllerUpdateSuccess = "PinotSchemaControllerUpdateSuccess"
	PinotSchemaControllerUpdateFail    = "PinotSchemaControllerUpdateFail"
	PinotSchemaControllerDeleteSuccess = "PinotSchemaControllerDeleteSuccess"
	PinotSchemaControllerDeleteFail    = "PinotSchemaControllerDeleteFail"
	PinotSchemaControllerFinalizer     = "pinotschema.datainfra.io/finalizer"
	PinotControllerPort                = "9000"
)

func (r *PinotSchemaReconciler) do(ctx context.Context, schema *v1beta1.PinotSchema) error {

	svcName, err := r.getControllerSvcUrl(schema.Namespace, schema.Spec.PinotCluster)
	if err != nil {
		return err
	}

	fmt.Println(makeControllerCreateSchemaPath(svcName))

	getOwnerRef := makeOwnerRef(
		schema.APIVersion,
		schema.Kind,
		schema.Name,
		schema.UID,
	)
	cm := r.makeSchemaConfigMap(schema, getOwnerRef, schema.Spec.PinotSchemaJson)

	build := builder.NewBuilder(
		builder.ToNewBuilderConfigMap([]builder.BuilderConfigMap{*cm}),
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinotSchemaController"}),
		builder.ToNewBuilderContext(builder.BuilderContext{Context: ctx}),
		builder.ToNewBuilderStore(
			*builder.NewStore(r.Client, map[string]string{"schema": schema.Name}, schema.Namespace, schema),
		),
	)

	resp, err := build.ReconcileConfigMap()
	if err != nil {
		return err
	}

	switch resp {
	case controllerutil.OperationResultCreated:
		http := internalHTTP.NewHTTPClient(http.MethodPost, makeControllerCreateSchemaPath(svcName), http.Client{}, []byte(schema.Spec.PinotSchemaJson))
		resp, err := http.Do()
		if err != nil {
			build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerCreateFail)
			return err
		}

		if getRespCode(resp) != "200" && getRespCode(resp) != "" {
			build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerCreateFail)
		} else {
			build.Recorder.GenericEvent(schema, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerCreateSuccess)
		}
	case controllerutil.OperationResultUpdated:
		schemaName, err := getSchemaName(schema.Spec.PinotSchemaJson)
		if err != nil {
			return err
		}
		http := internalHTTP.NewHTTPClient(http.MethodPut, makeControllerUpdateDeleteSchemaPath(svcName, schemaName), http.Client{}, []byte(schema.Spec.PinotSchemaJson))
		resp, err := http.Do()
		if err != nil {
			build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerUpdateFail)
			return err
		}
		if getRespCode(resp) != "200" && getRespCode(resp) != "" {
			build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerUpdateFail)
		} else {
			build.Recorder.GenericEvent(schema, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerUpdateSuccess)
		}
	default:
		if controllerutil.ContainsFinalizer(schema, PinotSchemaControllerFinalizer) {
			// our finalizer is present, so lets handle any external dependency

			schemaName, err := getSchemaName(schema.Spec.PinotSchemaJson)
			if err != nil {
				return err
			}
			http := internalHTTP.NewHTTPClient(http.MethodDelete, makeControllerUpdateDeleteSchemaPath(svcName, schemaName), http.Client{}, []byte{})
			resp, err := http.Do()
			if err != nil {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerDeleteFail)
				return err
			}
			if getRespCode(resp) != "200" && getRespCode(resp) != "" {
				build.Recorder.GenericEvent(schema, v1.EventTypeWarning, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerDeleteFail)
			} else {
				build.Recorder.GenericEvent(schema, v1.EventTypeNormal, fmt.Sprintf("Resp [%s]", string(resp)), PinotSchemaControllerDeleteSuccess)
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

func (r *PinotSchemaReconciler) makeSchemaConfigMap(
	schema *v1beta1.PinotSchema,
	ownerRef *metav1.OwnerReference,
	data interface{},
) *builder.BuilderConfigMap {

	configMap := &builder.BuilderConfigMap{
		CommonBuilder: builder.CommonBuilder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      schema.GetName() + "-" + "schema",
				Namespace: schema.GetNamespace(),
			},
			Client:   r.Client,
			CrObject: schema,
			OwnerRef: *ownerRef,
		},
		Data: map[string]string{
			"schema.json": data.(string),
		},
	}

	return configMap
}

// create owner ref ie pinot controller
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

func getSchemaName(schemaJson string) (string, error) {
	var err error

	schema := make(map[string]json.RawMessage)
	if err = json.Unmarshal([]byte(schemaJson), &schema); err != nil {
		return "", err
	}

	return utils.TrimQuote(string(schema["schemaName"])), nil
}

func makeControllerCreateSchemaPath(svcName string) string { return svcName + "/schemas" }

func makeControllerUpdateDeleteSchemaPath(svcName, schemaName string) string {
	return svcName + "/schemas/" + schemaName
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
	var svcName string

	for range svcList.Items {
		svcName = svcList.Items[0].Name
	}

	newName := "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort

	return newName, nil
}

func getRespCode(resp []byte) string {
	var err error

	respMap := make(map[string]json.RawMessage)
	if err = json.Unmarshal(resp, &respMap); err != nil {
		return ""
	}

	return utils.TrimQuote(string(respMap["code"]))
}
