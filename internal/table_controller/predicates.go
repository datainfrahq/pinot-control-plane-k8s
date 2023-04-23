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
