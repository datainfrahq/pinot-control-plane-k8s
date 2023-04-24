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

	"github.com/datainfrahq/operator-runtime/utils"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ignoreAnnotation = "pinottable.datainfra.io/reconcile"
)

// All methods to implement GenericPredicates type
// GenericPredicates to be passed to manager
type GenericPredicates struct {
	predicate.GenerationChangedPredicate
}

// create() to filter create events
func (GenericPredicates) Create(e event.CreateEvent) bool {
	return Create(e, log.FromContext(context.TODO()))
}

// update() to filter update events
func (GenericPredicates) Update(e event.UpdateEvent) bool {
	return Update(e, log.FromContext(context.TODO()))
}

func Create(e event.CreateEvent, log logr.Logger) bool {
	predicates := utils.NewCommonPredicates("pinottable-controller", ignoreAnnotation, log)

	return predicates.IgnoreObjectPredicate(e.Object) &&
		predicates.IgnoreNamespacePredicate(e.Object)
}

func Update(e event.UpdateEvent, log logr.Logger) bool {
	predicates := utils.NewCommonPredicates("pinottable-controller", ignoreAnnotation, log)

	return predicates.IgnoreObjectPredicate(e.ObjectNew) &&
		predicates.IgnoreNamespacePredicate(e.ObjectNew) &&
		predicates.IgnoreUpdate(e)
}
