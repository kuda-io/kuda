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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type DataPhase string

const (
	DataWaiting     DataPhase = "waiting"
	DataDownloading DataPhase = "downloading"
	DataSuccess     DataPhase = "success"
	DataFailed      DataPhase = "failed"
)

// DataTemplateSpec describes the fields a data resource should have when created from a template.
type DataTemplateSpec struct {
	// List of data items belonging to the data resource.
	DataItems []DataItem `json:"dataItems"`
	// Actions that the kube runtime should take in response to data events.
	Lifecycle *Lifecycle `json:"lifecycle,omitempty"`
	// List of data sources related to data storage.
	DataSources *DataSources `json:"dataSources"`
}

// DataItem describes the fields that each data item should have.
type DataItem struct {
	// Name of the data item. Each data item has a unique name in the same namespace.
	Name string `json:"name"`
	// Namespace defines the space within which each name must be unique.
	Namespace string `json:"namespace"`
	// RemotePath defines the path of data on the remote storage.
	RemotePath string `json:"remotePath"`
	// LocalPath defines the path of data in app container.
	LocalPath string `json:"localPath"`
	// Version defines the version number of the data.
	Version string `json:"version"`
	// The type of data source for the data.
	DataSourceType string `json:"dataSourceType"`
	// Actions should be taken for the data.
	Lifecycle *Lifecycle `json:"lifecycle,omitempty"`
}

// Lifecycle describes actions that the kuda runtime should take in response to data lifecycle events.
type Lifecycle struct {
	// PreDownload is called before data downloaded.
	PreDownload *LifecycleHandler `json:"preDownload,omitempty"`
	// PostDownload is called after data downloaded.
	PostDownload *LifecycleHandler `json:"postDownload,omitempty"`
}

// LifecycleHandler defines a specific action that should be token.
type LifecycleHandler struct {
	Exec    *v1.ExecAction    `json:"exec,omitempty"`
	HTTPGet *v1.HTTPGetAction `json:"httpGet,omitempty"`
}

// UpdateStrategy is currently not implemented
type UpdateStrategy struct {
	Type string `json:"type"`
	Gray int    `json:"gray"`
}

// DataSources defines the attribute information of the related data sources.
type DataSources struct {
	Hdfs    *HdfsDataSource    `json:"hdfs,omitempty"`
	Alluxio *AlluxioDataSource `json:"alluxio,omitempty"`
}

// HdfsDataSource defines the information of the hdfs data source.
type HdfsDataSource struct {
	Addresses []string `json:"addresses"`
	UserName  string   `json:"userName"`
}

// AlluxioDataSource defines the information of the alluxio data source.
type AlluxioDataSource struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout int    `json:"timeout,omitempty"`
}

// DataSetSpec defines the desired state of DataSet
type DataSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Template describes the data resource that will be created.
	Template DataTemplateSpec `json:"template"`

	// Label selector for workloads. The DataSet will be applied to all workloads
	// matching the selector.
	WorkloadSelector map[string]string `json:"workloadSelector"`
}

// DataSetStatus defines the observed state of DataSet
type DataSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DataItems       int    `json:"dataItems"`
	Replicas        int    `json:"replicas"`
	SuccessReplicas int    `json:"success"`
	Ready           string `json:"ready"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="DataItems",type=integer,JSONPath=`.status.dataItems`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.ready`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DataSet is the Schema for the datasets API
type DataSet struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the DataSet.
	Spec DataSetSpec `json:"spec,omitempty"`

	// Most recently observed status of the DataSet.
	Status DataSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataSetList contains a list of DataSet
type DataSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataSet{}, &DataSetList{})
}
