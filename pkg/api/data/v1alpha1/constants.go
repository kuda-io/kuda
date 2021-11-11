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

const (
	KudaKeyPod     = "kuda.io/pod"
	KudaKeyDataSet = "kuda.io/dataset"
	KudaKeyDigest  = "kuda.io/data-digest"

	KudaRuntimeEnvDataSetName       = "KUDA_DATASET_NAME"
	KudaRuntimeEnvDataSetNamespace  = "KUDA_DATASET_NAMESPACE"
	KudaRuntimeEnvPodName           = "MY_POD_NAME"
	KudaRuntimeEnvMainContainerName = "MAIN_CONTAINER_NAME"
)
