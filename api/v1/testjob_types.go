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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestjobSpec defines the desired state of Testjob
type TestjobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// TestjobStatus defines the observed state of Testjob
type TestjobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State JobState `json:"state"`
}

// +kubebuilder:object:root=true

// Testjob is the Schema for the testjobs API
type Testjob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestjobSpec   `json:"spec,omitempty"`
	Status TestjobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestjobList contains a list of Testjob
type TestjobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Testjob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Testjob{}, &TestjobList{})
}

type JobState string

const (
	JobStart  JobState = "JobStarting"
	JobStop   JobState = "JobStoping"
	JobRun    JobState = "JobRunning"
	JobFinish JobState = "JobFinished"
)
