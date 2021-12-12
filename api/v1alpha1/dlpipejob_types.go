/*
Copyright 2021.

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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DLpipeJobSpec defines the desired state of DLpipeJob
type DLpipeJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of DLpipeJob. Edit dlpipejob_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	WorldSize   *int64         `json:"worldSize"`
	JobTemplate corev1.PodSpec `json:"jobTemplate"`
	Placement   map[string]int `json:"placement,omitempty"`
}

type DLJobPhase string

const (
	JobPending       DLJobPhase = "JobPending"
	WaitingForMaster DLJobPhase = "WaitingForMaster"
	MasterIsReady    DLJobPhase = "MasterIsReady"
	JobRunning       DLJobPhase = "JobRunning"
	Completed        DLJobPhase = "Completed"
	Failed           DLJobPhase = "Failed"
)

// DLpipeJobStatus defines the observed state of DLpipeJob
type DLpipeJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	JobPhase   DLJobPhase   `json:"phase"`
	StartTime  *metav1.Time `json:"startTime"`
	MasterAddr string       `json:"masterAddr,omitempty"`
}

func (jobStatus *DLpipeJobStatus) SetDefault() bool {
	changed := false
	if jobStatus.JobPhase == "" {
		jobStatus.JobPhase = JobPending
		jobStatus.MasterAddr = ""
		now := metav1.NewTime(time.Now())
		jobStatus.StartTime = &now
		changed = true
	}

	return changed
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DLpipeJob is the Schema for the dlpipejobs API
type DLpipeJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DLpipeJobSpec   `json:"spec,omitempty"`
	Status DLpipeJobStatus `json:"status,omitempty"`
}

func (j *DLpipeJob) SetDefaultStatus() bool {
	return j.Status.SetDefault()
}

//+kubebuilder:object:root=true

// DLpipeJobList contains a list of DLpipeJob
type DLpipeJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DLpipeJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DLpipeJob{}, &DLpipeJobList{})
}
