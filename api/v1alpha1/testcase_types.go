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

package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=Created;Canceled;Dispatched;Running;Finished;Failed
type TestCaseCurrentStatus string

var (
	TestCaseCreated    TestCaseCurrentStatus = "Created"
	TestCaseCanceled   TestCaseCurrentStatus = "Canceled"
	TestCaseDispatched TestCaseCurrentStatus = "Dispatched"
	TestCaseRunning    TestCaseCurrentStatus = "Running"
	TestCaseFinished   TestCaseCurrentStatus = "Finished"
	TestCaseFailed     TestCaseCurrentStatus = "Failed"
)

const (
	DateTimeFormat = time.RFC822
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestCaseSpec defines the desired state of TestCase
type TestCaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Timeout  *string  `json:"timeout,omitempty"`
	Strategy Strategy `json:"strategy"`
}

// TestCaseStatus defines the observed state of TestCase
type TestCaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DispatchedAt   *string               `json:"dispatchedAt,omitempty"`
	StartedAt      *string               `json:"startedAt,omitempty"`
	FinishedAt     *string               `json:"finishedAt,omitempty"`
	FailureMessage *string               `json:"failureMessage,omitempty"`
	Status         TestCaseCurrentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TestCase is the Schema for the testcases API
type TestCase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestCaseSpec   `json:"spec,omitempty"`
	Status TestCaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestCaseList contains a list of TestCase
type TestCaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestCase `json:"items"`
}

func TimeString(t time.Time) *string {
	result := t.Format(DateTimeFormat)
	return &result
}

func init() {
	SchemeBuilder.Register(&TestCase{}, &TestCaseList{})
}
