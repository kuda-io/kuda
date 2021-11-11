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

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	datav1alpha1 "github.com/kuda-io/kuda/pkg/api/data/v1alpha1"
	"github.com/kuda-io/kuda/pkg/utils"
)

const (
	affinityTopologyKey = "kubernetes.io/hostname"

	sidecarContainerName = "kuda-runtime"

	volumeNameShareData = "share-data"
	volumeNameHostData  = "host-data"
	volumeNamePodData   = "pod-data"
)

var (
	log = ctrl.Log.WithName("webhook")
)

// PodInjector defines fields for the sidecar container injected to the pod.
type PodInjector struct {
	config  *Config
	client  client.Client
	decoder *admission.Decoder
}

// NewPodInjector returns PodInjector object by the config and client.
func NewPodInjector(config *Config, client client.Client) *PodInjector {
	return &PodInjector{
		config: config,
		client: client,
	}
}

// Handle handles an pod creation request, and mutates the pod spec if the dataset is exist.
func (p *PodInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := p.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string, 0)
	}

	if _, ok := pod.Annotations[datav1alpha1.KudaKeyDataSet]; !ok {
		ds, err := p.findDataSetForPod(ctx, pod)
		if err != nil {
			log.Error(err, "failed to find dataset for pod", "pod.Name", pod.Name)
			return admission.Errored(http.StatusInternalServerError, err)
		}

		if ds != nil {
			p.mutatePod(pod, ds)
		}
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		log.Error(err, "marshal pod error", "pod.Name", pod.Name)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// mutatePod add config for the pod.
func (p *PodInjector) mutatePod(pod *corev1.Pod, dataset *datav1alpha1.DataSet) {
	p.patchSidecar(pod)

	p.patchVolumes(pod)

	p.patchAffinity(pod, dataset.Spec.WorkloadSelector)

	p.patchAnnotations(pod, dataset.Name)
}

// patch kuda runtime container as sidecar for the app.
func (p *PodInjector) patchSidecar(pod *corev1.Pod) {
	sidecar := &corev1.Container{
		Name:  sidecarContainerName,
		Image: p.config.RuntimeImage,
		Args: []string{
			fmt.Sprintf("--download-root-dir=%s", p.config.HostPath),
			fmt.Sprintf("--local-root-dir=%s", p.config.DataPathPrefix),
			fmt.Sprintf("--notice-server-port=%d", p.config.RuntimeServerPort),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeNamePodData,
				MountPath: "/etc/podinfo",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name: datav1alpha1.KudaRuntimeEnvDataSetName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: fmt.Sprintf("metadata.annotations['%s']", datav1alpha1.KudaKeyDataSet),
					},
				},
			},
			{
				Name: datav1alpha1.KudaRuntimeEnvDataSetNamespace,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: datav1alpha1.KudaRuntimeEnvPodName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  datav1alpha1.KudaRuntimeEnvMainContainerName,
				Value: pod.Spec.Containers[0].Name,
			},
		},
	}

	pod.Spec.Containers = append(pod.Spec.Containers, *sidecar)
}

// patch volumes for the pod.
func (p *PodInjector) patchVolumes(pod *corev1.Pod) {
	dirOrCreate := corev1.HostPathDirectoryOrCreate

	volumes := []corev1.Volume{
		{
			Name:         volumeNameShareData,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name:         volumeNameHostData,
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: p.config.HostPath, Type: &dirOrCreate}},
		},
		{
			Name: volumeNamePodData,
			VolumeSource: corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{Items: []corev1.DownwardAPIVolumeFile{
				{Path: "annotations", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"}},
			}}},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	for i, _ := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, []corev1.VolumeMount{
			{
				Name:      volumeNameShareData,
				MountPath: p.config.DataPathPrefix,
			},
			{
				Name:      volumeNameHostData,
				MountPath: p.config.HostPath,
			},
		}...)
	}
}

// patch affinity for the pod.
func (p *PodInjector) patchAffinity(pod *corev1.Pod, workloadSelector map[string]string) {
	if !p.config.EnableAffinity {
		return
	}

	wpat := corev1.WeightedPodAffinityTerm{
		Weight: 1,
		PodAffinityTerm: corev1.PodAffinityTerm{
			LabelSelector: &v1.LabelSelector{
				MatchLabels: workloadSelector,
			},
			TopologyKey: affinityTopologyKey,
		},
	}

	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}

	if pod.Spec.Affinity.PodAffinity == nil {
		pod.Spec.Affinity.PodAffinity = &corev1.PodAffinity{}
	}

	pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, wpat)
}

// patch annotations for the pod.
func (p *PodInjector) patchAnnotations(pod *corev1.Pod, datasetName string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string, 0)
	}

	pod.Annotations[datav1alpha1.KudaKeyDataSet] = datasetName
}

// get the dataset resource for the pod.
func (p *PodInjector) findDataSetForPod(ctx context.Context, pod *corev1.Pod) (*datav1alpha1.DataSet, error) {
	dsList := &datav1alpha1.DataSetList{}
	if err := p.client.List(ctx, dsList, []client.ListOption{client.InNamespace(pod.Namespace)}...); err != nil {
		return nil, err
	}

	// todo: multi dataset for the pod
	for _, ds := range dsList.Items {
		if utils.ContainsAll(pod.GetLabels(), ds.Spec.WorkloadSelector) {
			return &ds, nil
		}
	}

	return nil, nil
}

// PodInjector implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (p *PodInjector) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}
