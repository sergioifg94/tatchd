/*


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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	thatchdv1alpha1 "github.com/thatchd/thatchd/api/v1alpha1"
	"github.com/thatchd/thatchd/pkg/thatchd/strategy"
	"github.com/thatchd/thatchd/pkg/thatchd/testcase"
	"github.com/thatchd/thatchd/pkg/thatchd/testsuite"
	"github.com/thatchd/thatchd/pkg/thatchd/testworker"
)

// TestSuiteReconciler reconciles a TestSuite object
type TestSuiteReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	StrategyProviders map[string]strategy.StrategyProvider
}

// +kubebuilder:rbac:groups=testing.thatchd.io,resources=testsuites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=testing.thatchd.io,resources=testsuites/status,verbs=get;update;patch

func (r *TestSuiteReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("testsuite", req.NamespacedName)

	instance := &thatchdv1alpha1.TestSuite{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	var currentState string
	if instance.Status.CurrentState != "" {
		currentState = instance.Status.CurrentState
	} else if instance.Spec.InitialState != "" {
		currentState = instance.Spec.InitialState
	} else {
		currentState = "{}"
	}

	str := strategy.Strategy(instance.Spec.StateStrategy.Strategy)

	programReconciler, err := testsuite.FromStrategy(&str, r.StrategyProviders)
	if err != nil {
		return r.withErrorStatus(ctx, instance, fmt.Errorf("error obtaining program reconciler: %v", err))
	}

	parsedState, err := programReconciler.ParseState(currentState)
	if err != nil {
		return r.withErrorStatus(ctx, instance, fmt.Errorf("failed to parse current state: %w", err))
	}

	// Reconcile the program state
	updatedState, err := programReconciler.Reconcile(r.Client, req.Namespace, parsedState)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error reconciling program state: %v", err)
	}

	marshalledState, err := json.MarshalIndent(updatedState, "", "  ")
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error marshalling state: %v", err)
	}

	instance.Status.CurrentState = string(marshalledState)
	if err := r.Status().Update(context.TODO(), instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating state: %v", err)
	}

	if err := r.dispatchTestCases(ctx, req.Namespace, updatedState); err != nil {
		return ctrl.Result{}, fmt.Errorf("error dispatching test cases: %v", err)
	}

	if err := r.dispatchTestWorkers(ctx, req.Namespace, updatedState); err != nil {
		return ctrl.Result{}, fmt.Errorf("error dispatching test workers: %w", err)
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Second,
	}, nil
}

func (r *TestSuiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&thatchdv1alpha1.TestSuite{}).
		Complete(r)
}

func (r *TestSuiteReconciler) dispatchTestCases(ctx context.Context, namespace string, currentState interface{}) error {
	testCases := &thatchdv1alpha1.TestCaseList{}
	if err := r.List(ctx, testCases); err != nil {
		return err
	}

	for _, testCase := range testCases.Items {
		str := testCase.Spec.Strategy.Strategy

		testCaseInterface, err := testcase.FromStrategy(&str, r.StrategyProviders)
		if err != nil {
			return err
		}

		// Skip tests that aren't meant to be run yet
		if !testCaseInterface.ShouldRun(currentState) {
			continue
		}

		// Skip tests that have already been dispatched
		if testCase.Status.DispatchedAt != nil {
			continue
		}

		// Dispatch by setting the DispatchedAt field to the current time
		testCase.Status.DispatchedAt = thatchdv1alpha1.TimeString(time.Now())
		testCase.Status.Status = thatchdv1alpha1.TestCaseDispatched
		if err := r.Status().Update(ctx, &testCase); err != nil {
			return fmt.Errorf("error dispatching TestCase %s", testCase.Name)
		}
	}

	return nil
}

func (r *TestSuiteReconciler) dispatchTestWorkers(ctx context.Context, namespace string, currentState interface{}) error {
	testWorkers := &thatchdv1alpha1.TestWorkerList{}
	if err := r.List(ctx, testWorkers); err != nil {
		return err
	}

	for _, testWorker := range testWorkers.Items {
		str := testWorker.GetStrategy().Strategy

		testWorkerInterface, err := testworker.FromStrategy(&str, r.StrategyProviders)
		if err != nil {
			return err
		}

		if !testWorkerInterface.ShouldRun(currentState) {
			continue
		}

		if testWorker.Status.DispatchedAt != nil {
			continue
		}

		testWorker.Status.DispatchedAt = thatchdv1alpha1.TimeString(time.Now())
		if err := r.Status().Update(ctx, &testWorker); err != nil {
			return fmt.Errorf("error dispatching TestWorker %s: %v", testWorker.Name, err)
		}
	}

	return nil
}

func (r *TestSuiteReconciler) withErrorStatus(ctx context.Context, instance *thatchdv1alpha1.TestSuite, errorStatus error) (ctrl.Result, error) {
	instance.Status.Error = errorStatus.Error()
	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update to error status \"%v\": %v", errorStatus, err)
	}

	return ctrl.Result{
		RequeueAfter: time.Second,
		Requeue:      true,
	}, nil
}
