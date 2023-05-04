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

package tenantcontroller

import (
	"context"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/datainfrahq/pinot-control-plane-k8s/api/v1beta1"
	datainfraiov1beta1 "github.com/datainfrahq/pinot-control-plane-k8s/api/v1beta1"
	"github.com/go-logr/logr"
)

// PinotTenantReconciler reconciles a PinotTenant object
type PinotTenantReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// reconcile time duration, defaults to 10s
	ReconcileWait time.Duration
	Recorder      record.EventRecorder
}

func NewPinotTenantReconciler(mgr ctrl.Manager) *PinotTenantReconciler {
	initLogger := ctrl.Log.WithName("controllers").WithName("pinot-tenant")
	return &PinotTenantReconciler{
		Client:        mgr.GetClient(),
		Log:           initLogger,
		Scheme:        mgr.GetScheme(),
		ReconcileWait: lookupReconcileTime(initLogger),
		Recorder:      mgr.GetEventRecorderFor("pinot-control-plane"),
	}
}

// +kubebuilder:rbac:groups=datainfra.io,resources=pinottenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=datainfra.io,resources=pinottenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=datainfra.io,resources=pinottenants/finalizers,verbs=update
// +kubebuilder:rbac:groups=datainfra.io,resources=pinotschemas/finalizers,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secret,verbs=get
func (r *PinotTenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)

	pinotTenantCR := &v1beta1.PinotTenant{}
	err := r.Get(context.TODO(), req.NamespacedName, pinotTenantCR)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.do(ctx, pinotTenantCR); err != nil {
		logr.Error(err, err.Error())
		return ctrl.Result{}, err
	} else {
		return ctrl.Result{RequeueAfter: r.ReconcileWait}, nil
	}

}

// SetupWithManager sets up the controller with the Manager.
func (r *PinotTenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datainfraiov1beta1.PinotTenant{}).
		WithEventFilter(predicate.Or(
			GenericPredicates{},
			predicate.GenerationChangedPredicate{},
			predicate.LabelChangedPredicate{},
		)).
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
