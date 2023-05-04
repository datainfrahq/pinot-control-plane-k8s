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

const (
	schemaName = "schemaName"
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
			if err := r.Update(ctx, schema.DeepCopyObject().(*v1beta1.PinotSchema)); err != nil {
				return nil
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(schema, PinotSchemaControllerFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			svcName, err := r.getControllerSvcUrl(schema.Namespace, schema.Spec.PinotCluster)
			if err != nil {
				return err
			}

			schemaName, err := utils.GetValueFromJson(schema.Spec.PinotSchemaJson, schemaName)
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
			respDeleteSchema, err := http.Do()
			if err != nil {
				return err
			}
			if respDeleteSchema.StatusCode != 200 {
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s]", string(respDeleteSchema.ResponseBody)),
					PinotSchemaControllerDeleteFail,
				)
			} else {
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respDeleteSchema.ResponseBody)),
					PinotSchemaControllerDeleteSuccess,
				)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(schema, PinotSchemaControllerFinalizer)
			if err := r.Update(ctx, schema.DeepCopyObject().(*v1beta1.PinotSchema)); err != nil {
				return nil
			}
		}
	}

	return nil
}

// Get Schema if does not exist create
// if exists check for update
func (r *PinotSchemaReconciler) CreateOrUpdate(
	schema *v1beta1.PinotSchema,
	svcName string,
	build builder.Builder,
	auth internalHTTP.Auth,
) (controllerutil.OperationResult, error) {

	// get schema name
	schemaName, err := utils.GetValueFromJson(schema.Spec.PinotSchemaJson, schemaName)
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

	respGetSchema, err := getHttp.Do()
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	// if not found create schema
	// else check for updates
	if respGetSchema.StatusCode == 404 {

		// create schema
		postHttp := internalHTTP.NewHTTPClient(
			http.MethodPost,
			makeControllerCreateSchemaPath(svcName),
			http.Client{},
			[]byte(schema.Spec.PinotSchemaJson),
			auth,
		)

		respCreatechema, err := postHttp.Do()
		if err != nil {
			return controllerutil.OperationResultNone, err
		}

		if respCreatechema.StatusCode == 200 {
			result, err := r.makePatchPinotSchemaStatus(
				schema,
				PinotSchemaControllerCreateSuccess,
				string(respCreatechema.ResponseBody),
				v1.ConditionTrue,
				PinotSchemaControllerCreateSuccess,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(
				schema,
				v1.EventTypeNormal,
				fmt.Sprintf("Resp [%s]", string(respCreatechema.ResponseBody)),
				PinotSchemaControllerCreateSuccess,
			)
			build.Recorder.GenericEvent(
				schema,
				v1.EventTypeNormal,
				fmt.Sprintf("Resp [%s], Result [%s]", string(respCreatechema.ResponseBody), result),
				PinotSchemaControllerPatchStatusSuccess)
			return controllerutil.OperationResultCreated, nil

		} else {
			_, err := r.makePatchPinotSchemaStatus(
				schema,
				PinotSchemaControllerCreateFail,
				string(respCreatechema.ResponseBody),
				v1.ConditionTrue,
				PinotSchemaControllerCreateFail,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
			build.Recorder.GenericEvent(
				schema,
				v1.EventTypeWarning,
				fmt.Sprintf("Resp [%s], Status", string(respCreatechema.ResponseBody)),
				PinotSchemaControllerCreateFail,
			)
			return controllerutil.OperationResultCreated, nil

		}
	} else if respGetSchema.StatusCode == 200 {

		// at times of mis-match of state, where resource exists on pinot, but
		// on creation status wasn't updated.
		// get the current state ie schema and patch the status
		if schema.Status.CurrentSchemasJson == "" {
			build.Recorder.GenericEvent(
				schema,
				v1.EventTypeWarning,
				fmt.Sprintf("Schema Exists on Pinot, but status is not updated"),
				PinotSchemaControllerUpdateFail,
			)

			_, err := r.makePatchPinotSchemaStatus(
				schema,
				PinotSchemaControllerCreateSuccess,
				string(respGetSchema.ResponseBody),
				v1.ConditionTrue,
				PinotSchemaControllerUpdateSuccess,
			)
			if err != nil {
				return controllerutil.OperationResultNone, err
			}
		}

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

			respUpdateSchema, err := postHttp.Do()
			if err != nil {
				return controllerutil.OperationResultNone, err
			}

			if respUpdateSchema.StatusCode == 200 {
				// patch status to store the current valid schema json
				result, err := r.makePatchPinotSchemaStatus(
					schema,
					PinotSchemaControllerUpdateSuccess,
					string(respUpdateSchema.ResponseBody),
					v1.ConditionTrue,
					PinotSchemaControllerUpdateSuccess,
				)
				if err != nil {
					return controllerutil.OperationResultNone, err
				}
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s]", string(respUpdateSchema.ResponseBody)),
					PinotSchemaControllerUpdateSuccess,
				)
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeNormal,
					fmt.Sprintf("Resp [%s], Result [%s]", string(respUpdateSchema.ResponseBody), result),
					PinotSchemaControllerPatchStatusSuccess)

				return controllerutil.OperationResultUpdated, nil
			} else {
				// patch status with failure and emit events
				_, err := r.makePatchPinotSchemaStatus(
					schema,
					PinotSchemaControllerUpdateFail,
					string(respGetSchema.ResponseBody),
					v1.ConditionTrue,
					PinotSchemaControllerUpdateFail,
				)
				if err != nil {
					return controllerutil.OperationResultNone, err
				}
				build.Recorder.GenericEvent(
					schema,
					v1.EventTypeWarning,
					fmt.Sprintf("Resp [%s]", string(respUpdateSchema.ResponseBody)),
					PinotSchemaControllerUpdateFail,
				)
				return controllerutil.OperationResultNone, err

			}
		}

	}

	return controllerutil.OperationResultNone, nil
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
	pinotSchemaConditionType string,

) (controllerutil.OperationResult, error) {

	if _, _, err := utils.PatchStatus(context.Background(), r.Client, schema, func(obj client.Object) client.Object {
		in := obj.(*v1beta1.PinotSchema)
		in.Status.CurrentSchemasJson = schema.Spec.PinotSchemaJson
		in.Status.LastUpdateTime = metav1.Time{Time: time.Now()}
		in.Status.Message = msg
		in.Status.Reason = reason
		in.Status.Status = status
		in.Status.Type = pinotSchemaConditionType
		return in
	}); err != nil {
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

	newName := "http://" + svcName + "." + namespace + ".svc.cluster.local:" + PinotControllerPort

	return newName, nil
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
