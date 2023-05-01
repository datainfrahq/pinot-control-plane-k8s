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
)

const (
	ControlPlaneUserName = "CONTROL_PLANE_USERNAME"
	ControlPlanePassword = "CONTROL_PLANE_PASSWORD"
)

const (
	PinotControllerPort = "9000"
)

func (r *PinotSchemaReconciler) do(ctx context.Context, schema *v1beta1.PinotSchema) error {

	build := builder.NewBuilder(
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "PinotSchemaController"}),
	)

	basicAuth, err := r.getAuthCreds(ctx, schema)
	if err != nil {
		return err
	}

	svcName, err := r.getControllerSvcUrl(schema.Namespace, schema.Spec.PinotCluster)
	if err != nil {
		return err
	}

	_, err = r.CreateOrUpdate(schema, svcName, *build, internalHTTP.Auth{BasicAuth: basicAuth})
	if err != nil {
		return err
	}

	if schema.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
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
			http := internalHTTP.NewHTTPClient(
				http.MethodDelete,
				makeControllerGetUpdateDeleteSchemaPath(svcName, schemaName),
				http.Client{},
				[]byte{},
				internalHTTP.Auth{BasicAuth: basicAuth},
			)
			respDeleteSchema := http.Do()
			if respDeleteSchema.Err != nil {
				return respDeleteSchema.Err
			}
			if respDeleteSchema.StatusCode != 200 {
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s]", string(respDeleteSchema.PinotErrorResponse.Error)),
					PinotSchemaControllerDeleteFail,
				)
			} else {
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respDeleteSchema.PinotSuccessResponse.Status)),
					PinotSchemaControllerDeleteSuccess,
				)
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
	auth internalHTTP.Auth,
) (controllerutil.OperationResult, error) {

	// get schema name
	schemaName, err := getSchemaName(schema.Spec.PinotSchemaJson)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	// get schema
	getHttp := internalHTTP.NewHTTPClient(
		http.MethodGet,
		makeControllerGetUpdateDeleteSchemaPath(svcName, schemaName),
		http.Client{},
		[]byte{},
		auth,
	)

	respGetSchema := getHttp.Do()
	if respGetSchema.Err != nil {
		return controllerutil.OperationResultNone, respGetSchema.Err
	}

	// if not found create schema
	// else check for updates
	if respGetSchema.PinotErrorResponse.Code == 404 {

		// create schema
		postHttp := internalHTTP.NewHTTPClient(
			http.MethodPost,
			makeControllerCreateSchemaPath(svcName),
			http.Client{},
			[]byte(schema.Spec.PinotSchemaJson),
			auth,
		)

		respCreatechema := postHttp.Do()
		if respCreatechema.Err != nil {
			return controllerutil.OperationResultNone, respCreatechema.Err
		}

		if respCreatechema.StatusCode == 200 {
			result, err := r.makePatchPinotSchemaStatus(
				schema,
				PinotSchemaControllerCreateSuccess,
				string(respCreatechema.PinotSuccessResponse.Status),
				v1.ConditionTrue,
				v1beta1.PinotSchemaCreateSuccess,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(
				schema,
				v1.EventTypeNormal,
				fmt.Sprintf("Resp [%s]", string(respCreatechema.PinotSuccessResponse.Status)),
				PinotSchemaControllerCreateSuccess,
			)
			build.Recorder.GenericEvent(
				schema,
				v1.EventTypeNormal,
				fmt.Sprintf("Resp [%s], Result [%s]", string(respCreatechema.PinotSuccessResponse.Status), result),
				PinotSchemaControllerPatchStatusSuccess)
			return controllerutil.OperationResultCreated, nil

		} else {
			_, err := r.makePatchPinotSchemaStatus(
				schema,
				PinotSchemaControllerCreateFail,
				string(respCreatechema.PinotErrorResponse.Error),
				v1.ConditionTrue,
				v1beta1.PinotSchemaCreateFail,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(
				schema,
				v1.EventTypeWarning,
				fmt.Sprintf("Resp [%s], Status", string(respCreatechema.PinotErrorResponse.Error)),
				PinotSchemaControllerCreateFail,
			)
			return controllerutil.OperationResultCreated, nil

		}
	} else if respGetSchema.StatusCode == 200 {

		ok, err := utils.IsEqualJson(schema.Status.CurrentSchemasJson, schema.Spec.PinotSchemaJson)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}

		// if desiredstate and currentstate not the same then update
		if !ok {
			postHttp := internalHTTP.NewHTTPClient(
				http.MethodPut,
				makeControllerGetUpdateDeleteSchemaPath(svcName, schemaName),
				http.Client{},
				[]byte(schema.Spec.PinotSchemaJson),
				auth,
			)
			respUpdateSchema := postHttp.Do()
			if respUpdateSchema.Err != nil {
				return controllerutil.OperationResultNone, err
			}
			if respUpdateSchema.PinotSuccessResponse != (internalHTTP.PinotSuccessResponse{}) {
				// patch status to store the current valid schema json
				result, err := r.makePatchPinotSchemaStatus(
					schema,
					PinotSchemaControllerUpdateSuccess,
					string(respUpdateSchema.PinotSuccessResponse.Status),
					v1.ConditionTrue,
					v1beta1.PinotSchemaUpdateSuccess,
				)
				if err != nil {
					return controllerutil.OperationResultNone, err
				}
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respUpdateSchema.PinotSuccessResponse.Status)),
					PinotSchemaControllerUpdateSuccess,
				)
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s], Result [%s]", string(respUpdateSchema.PinotSuccessResponse.Status), result),
					PinotSchemaControllerPatchStatusSuccess)
				return controllerutil.OperationResultUpdated, nil

			} else if respUpdateSchema.PinotErrorResponse != (internalHTTP.PinotErrorResponse{}) {
				// patch status with failure and emit events
				_, err := r.makePatchPinotSchemaStatus(
					schema,
					PinotSchemaControllerUpdateFail,
					string(respUpdateSchema.PinotErrorResponse.Error),
					v1.ConditionTrue,
					v1beta1.PinotSchemaUpdateFail,
				)
				if err != nil {
					return controllerutil.OperationResultNone, err
				}
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s], StatusCode [%d]", string(respUpdateSchema.PinotErrorResponse.Error), respUpdateSchema.PinotErrorResponse.Code),
					PinotSchemaControllerUpdateFail,
				)
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

	updatedPinotSchemaStatus.CurrentSchemasJson = schema.Spec.PinotSchemaJson
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

	return controllerutil.OperationResultUpdatedStatusOnly, nil
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

	_ = "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort

	return "http://localhost:9000", nil
}

func (r *PinotSchemaReconciler) getAuthCreds(ctx context.Context, schema *v1beta1.PinotSchema) (internalHTTP.BasicAuth, error) {
	pinot := v1beta1.Pinot{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: schema.Namespace,
		Name:      schema.Spec.PinotCluster,
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
