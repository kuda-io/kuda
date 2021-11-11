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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	datav1alpha1 "github.com/kuda-io/kuda/pkg/api/data/v1alpha1"
)

func TestPodInjector_mutatePod(t *testing.T) {
	type fields struct {
		config  *Config
		client  client.Client
		decoder *admission.Decoder
	}
	type args struct {
		pod     *corev1.Pod
		dataset *datav1alpha1.DataSet
	}

	var (
		dirOrCreate       = corev1.HostPathDirectoryOrCreate
		runtimeImage      = "kuda-runtime:latest"
		hostPath          = "/var/lib/kuda"
		dataPathPrefix    = "/kuda/data"
		runtimeServerPort = 8888
		workloadSelector  = map[string]string{"app": "test"}
	)

	tests := []struct {
		name   string
		fields fields
		args   args
		want   *corev1.Pod
	}{
		{
			name: "mutate",
			fields: fields{
				config: &Config{
					RuntimeImage:      runtimeImage,
					HostPath:          hostPath,
					DataPathPrefix:    dataPathPrefix,
					EnableAffinity:    true,
					RuntimeServerPort: uint(runtimeServerPort),
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test",
								Image: "nginx",
							},
						},
					},
				},
				dataset: &datav1alpha1.DataSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: datav1alpha1.DataSetSpec{WorkloadSelector: workloadSelector},
				},
			},
			want: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{datav1alpha1.KudaKeyDataSet: "test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "nginx",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeNameShareData,
									MountPath: dataPathPrefix,
								},
								{
									Name:      volumeNameHostData,
									MountPath: hostPath,
								},
							},
						},
						{
							Name:  sidecarContainerName,
							Image: runtimeImage,
							Args: []string{
								fmt.Sprintf("--download-root-dir=%s", hostPath),
								fmt.Sprintf("--local-root-dir=%s", dataPathPrefix),
								fmt.Sprintf("--notice-server-port=%d", runtimeServerPort),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeNamePodData,
									MountPath: "/etc/podinfo",
								},
								{
									Name:      volumeNameShareData,
									MountPath: dataPathPrefix,
								},
								{
									Name:      volumeNameHostData,
									MountPath: hostPath,
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
									Value: "test",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name:         volumeNameShareData,
							VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
						},
						{
							Name:         volumeNameHostData,
							VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/kuda", Type: &dirOrCreate}},
						},
						{
							Name: volumeNamePodData,
							VolumeSource: corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{Items: []corev1.DownwardAPIVolumeFile{
								{Path: "annotations", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"}},
							}}},
						},
					},
					Affinity: &corev1.Affinity{
						PodAffinity: &corev1.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 1,
									PodAffinityTerm: corev1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: workloadSelector,
										},
										TopologyKey: affinityTopologyKey,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PodInjector{
				config:  tt.fields.config,
				client:  tt.fields.client,
				decoder: tt.fields.decoder,
			}
			p.mutatePod(tt.args.pod, tt.args.dataset)
			assert.Equal(t, tt.want, tt.args.pod)
		})
	}
}
