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
	"net/http"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/datainfrahq/pinot-control-plane-k8s/api/v1beta1"
	datainfraiov1beta1 "github.com/datainfrahq/pinot-control-plane-k8s/api/v1beta1"
	internalHTTP "github.com/datainfrahq/pinot-control-plane-k8s/internal/http"
	schemacontroller "github.com/datainfrahq/pinot-control-plane-k8s/internal/schema_controller"

	"github.com/datainfrahq/pinot-control-plane-k8s/internal/utils"
	"github.com/go-logr/logr"
)

// PinotTableReconciler reconciles a PinotTable object
type PinotTableReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// reconcile time duration, defaults to 10s
	ReconcileWait time.Duration
	Recorder      record.EventRecorder
}

func NewPinotTableReconciler(mgr ctrl.Manager) *PinotTableReconciler {
	initLogger := ctrl.Log.WithName("controllers").WithName("pinot")
	return &PinotTableReconciler{
		Client:        mgr.GetClient(),
		Log:           initLogger,
		Scheme:        mgr.GetScheme(),
		ReconcileWait: lookupReconcileTime(initLogger),
		Recorder:      mgr.GetEventRecorderFor("pinot-control-plane"),
	}
}

// +kubebuilder:rbac:groups=datainfra.io,resources=pinottables,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=datainfra.io,resources=pinottables/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=datainfra.io,resources=pinottables/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secret,verbs=get
func (r *PinotTableReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)

	pinotTableCR := &v1beta1.PinotTable{}
	err := r.Get(context.TODO(), req.NamespacedName, pinotTableCR)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.do(ctx, pinotTableCR); err != nil {
		logr.Error(err, err.Error())
		return ctrl.Result{}, err
	} else {
		return ctrl.Result{RequeueAfter: r.ReconcileWait}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PinotTableReconciler) SetupWithManager(mgr ctrl.Manager) error {

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			if !e.ObjectNew.GetDeletionTimestamp().IsZero() {
				return false
			}
			r.Log.Info("Update Event Recieved, Wait for couple of seconds for schema controller to update status  - table controller")

			time.Sleep(time.Second * time.Duration(2))

			schema := v1beta1.PinotSchema{}

			if err := r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: e.ObjectNew.GetNamespace(),
				Name:      e.ObjectNew.GetName(),
			}, &schema); err != nil {
				r.Log.Error(err, "Error getting schema  - table controller")
				return false
			}

			if time.Now().After(schema.Status.LastUpdateTime.Time) {
				listOpts := []client.ListOption{
					client.InNamespace(e.ObjectNew.GetNamespace()),
				}

				tableList := v1beta1.PinotTableList{}
				if err := r.Client.List(context.TODO(), &tableList, listOpts...); err != nil {
					return false
				}

				for _, table := range tableList.Items {
					if table.Spec.SegmentReload {
						if schema.Status.Message == schemacontroller.PinotSchemaControllerUpdateSuccess {
							svcName, err := r.getControllerSvcUrl(table.Namespace, table.Spec.PinotCluster)
							if err != nil {
								r.Log.Error(err, "Error getting serviceName  - table controller")
								return false
							}

							segmentsConfig, err := utils.GetValueFromJson(table.Spec.PinotTablesJson, utils.SegmentsConfig)
							if err != nil {
								r.Log.Error(err, "Error getting schemaName  - table controller")
								return false
							}
							schemaNameinTable, err := utils.GetValueFromJson(segmentsConfig, utils.SchemaName)
							if err != nil {
								r.Log.Error(err, "Error getting schemaName in table  - table controller")
								return false
							}

							schemaNameinEvent, err := utils.GetValueFromJson(schema.Spec.PinotSchemaJson, utils.SchemaName)
							if err != nil {
								r.Log.Error(err, "Error getting schemaName in Update Event  - table controller")
								return false
							}

							getTableName, err := utils.GetValueFromJson(table.Spec.PinotTablesJson, utils.TableName)
							if err != nil {
								r.Log.Error(err, "Error getting tableName in Update Event  - table controller")
								return false
							}

							basicAuth, err := r.getAuthCreds(context.TODO(), &table)
							if err != nil {
								r.Log.Error(err, "Error getting authCreds in Update Event  - table controller")
								return false
							}

							if schemaNameinTable == schemaNameinEvent {
								postHttp := internalHTTP.NewHTTPClient(
									http.MethodPost,
									makeControllerReloadTable(svcName, getTableName),
									http.Client{},
									[]byte{},
									internalHTTP.Auth{BasicAuth: basicAuth},
								)

								reloadHttp, err := postHttp.Do()
								if err != nil {
									r.Log.Error(err, "Error getting reloading segments in Update Event  - table controller")
									return false
								}
								r.Recorder.Event(&table, v1.EventTypeNormal, reloadHttp.ResponseBody, PinotTableReloadAllSegments)

								if _, _, err := utils.PatchStatus(context.Background(), r.Client, &table, func(obj client.Object) client.Object {
									in := obj.(*v1beta1.PinotTable)
									in.Status.ReloadStatus = append(in.Status.ReloadStatus, reloadHttp.ResponseBody)
									return in
								}); err != nil {
									r.Log.Error(err, "Error patching reloading segments in Update Event  - table controller")
									return false
								}

							}

						}
					} else {
						r.Log.Info("No suitable condition found for reload segments - update controller")
						return false
					}

				}
			}

			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&datainfraiov1beta1.PinotTable{}).
		Watches(
			&source.Kind{Type: &v1beta1.PinotSchema{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(p),
		).
		WithEventFilter(
			GenericPredicates{},
		).
		Complete(r)
}

func lookupReconcileTime(log logr.Logger) time.Duration {
	val, exists := os.LookupEnv("RECONCILE_WAIT")
	if !exists {
		return time.Second * 10
	} else {
		v, err := time.ParseDuration(val)
		if err != nil {
			log.Error(err, err.Error())
			// Exit Program if not valid
			os.Exit(1)
		}
		return v
	}
}
