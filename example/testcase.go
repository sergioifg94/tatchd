package example

import (
	"context"
	"errors"
	"fmt"

	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testcase"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodAnnotationTestCase asserts that a pod has an annotation with an expected
// value
type PodAnnotationTestCase struct {
	PodName            string
	ExpectedAnnotation string
	ExpectedValue      string
}

var _ testcase.Interface = &PodAnnotationTestCase{}

func (tc *PodAnnotationTestCase) ShouldRun(s interface{}) bool {
	state := s.(PodSuiteState)
	if ready, ok := state[tc.PodName]; ok && ready {
		return true
	}

	return false
}

func (tc *PodAnnotationTestCase) Run(c client.Client, namespace string) error {
	pod := &v1.Pod{}
	if err := c.Get(context.TODO(), client.ObjectKey{
		Name:      tc.PodName,
		Namespace: namespace,
	}, pod); err != nil {
		return fmt.Errorf("Failed to obtain pod: %v", err)
	}

	annotations := pod.Annotations
	if annotations == nil {
		return errors.New("Pod has no annotations")
	}

	if value, ok := annotations[tc.ExpectedAnnotation]; !ok || value != tc.ExpectedValue {
		return fmt.Errorf("Annotation %s: %s not found in Pod", tc.ExpectedAnnotation, tc.ExpectedValue)
	}

	return nil
}

type PodAnnotationProvider struct{}

var _ strategy.StrategyProvider = &PodAnnotationProvider{}

func (p *PodAnnotationProvider) New(config map[string]string) interface{} {
	return &PodAnnotationTestCase{
		PodName:            config["podName"],
		ExpectedAnnotation: config["expectedAnnotation"],
		ExpectedValue:      config["expectedValue"],
	}
}