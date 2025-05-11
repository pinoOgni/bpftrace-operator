/*
Copyright 2025 pinoOgni.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BpfTraceJobSpec defines the desired state of BpfTraceJob.
type BpfTraceJobSpec struct {
	// The hook point to attach to (e.g., "kprobe:vfs_read", "tracepoint:syscalls:sys_enter_execve")
	Hook string `json:"hook"`
}

// BpfTraceJobStatus defines the observed state of BpfTraceJob.
type BpfTraceJobStatus struct {
	// The phase of the bpftracejob
	Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BpfTraceJob is the Schema for the bpftracejobs API.
type BpfTraceJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BpfTraceJobSpec   `json:"spec,omitempty"`
	Status BpfTraceJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BpfTraceJobList contains a list of BpfTraceJob.
type BpfTraceJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BpfTraceJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BpfTraceJob{}, &BpfTraceJobList{})
}
