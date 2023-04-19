/*
DataInfra Pinot Operator (C) 2023 - 2024 DataInfra.

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

	"github.com/datainfrahq/operator-runtime/builder"
	"github.com/datainfrahq/pinot-operator/api/v1beta1"
	internalHTTP "github.com/datainfrahq/pinot-operator/internal/http"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *PinotSchemaReconciler) do(ctx context.Context, schema *v1beta1.PinotSchema) error {

	getOwnerRef := makeOwnerRef(
		schema.APIVersion,
		schema.Kind,
		schema.Name,
		schema.UID,
	)

	cm := r.makeSchemaConfigMap(schema, getOwnerRef, schema.Spec.SchemaJson)

	build := builder.NewBuilder(
		builder.ToNewBuilderConfigMap([]builder.BuilderConfigMap{*cm}),
		builder.ToNewBuilderRecorder(builder.BuilderRecorder{Recorder: r.Recorder, ControllerName: "pinotschemacontroller"}),
		builder.ToNewBuilderContext(builder.BuilderContext{Context: ctx}),
		builder.ToNewBuilderStore(
			*builder.NewStore(r.Client, map[string]string{"schema": schema.Name}, schema.Namespace, schema),
		),
	)

	resp, err := build.ReconcileConfigMap()
	if err != nil {
		return err
	}

	if resp == controllerutil.OperationResultCreated {
		if schema.Spec.SchemaJson != "" {
			http := internalHTTP.NewHTTPClient(http.MethodPost, "http://74.220.23.197:9000/schemas", http.Client{}, []byte(schema.Spec.SchemaJson))
			resp, err := http.Do()
			if err != nil {
				return err
			}

			fmt.Println(string(resp))
		}
	} else if resp == controllerutil.OperationResultUpdated {
		if schema.Spec.SchemaJson != "" {
			http := internalHTTP.NewHTTPClient(http.MethodPost, "http://74.220.23.197:9000/schemas/", http.Client{}, []byte(schema.Spec.SchemaJson))
			resp, err := http.Do()
			if err != nil {
				return err
			}

			fmt.Println(string(resp))
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
				Name:      schema.GetName(),
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

// create owner ref ie parseable tenant controller
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
