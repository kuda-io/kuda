/*
Copyright 2021 The Kuda Authors.

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

// DataSpec defines the desired state of Data
type DataSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DataItems   []DataItem   `json:"dataItems"`
	Lifecycle   *Lifecycle   `json:"lifecycle,omitempty"`
	DataSources *DataSources `json:"dataSources"`
}

// DataStatus defines the observed state of Data
type DataStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DataItemsStatus DataItemsStatus `json:"dataItemsStatus"`
	DataItems       int             `json:"dataItems"`
	Success         int             `json:"success"`
	Waiting         int             `json:"waiting"`
	Downloading     int             `json:"downloading"`
	Failed          int             `json:"failed"`
	Ready           string          `json:"ready"`
}

type DataItemsStatus []DataItemStatus

// DataItemStatus defines status fields for each data item.
type DataItemStatus struct {
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Version   string      `json:"version"`
	Phase     DataPhase   `json:"phase"`
	StartTime metav1.Time `json:"startTime"`
	Message   string      `json:"message,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=datas
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.ready`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Data is the Schema for the data API
type Data struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the Data.
	Spec DataSpec `json:"spec,omitempty"`

	// Most recently observed status of the Data.
	Status DataStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataList contains a list of Data
type DataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Data `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Data{}, &DataList{})
}
