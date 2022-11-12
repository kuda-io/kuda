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

package controllers

import (
	"context"
	v12 "k8s.io/api/core/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kuda-io/kuda/pkg/api/data/v1alpha1"
)

func TestCreateDataResource(t *testing.T) {
	testDataSetReconciler, err := getTestDataSetReconciler()
	assert.NoError(t, err)

	datasetName := "test-ds"
	dataItemName := "test-data"
	podName := "test-pod"

	t.Run("create data resource success", func(t *testing.T) {
		dataset := getTestDataSet(datasetName, dataItemName)
		data, err := testDataSetReconciler.createDataResource(context.Background(), dataset, podName)
		assert.NoError(t, err)
		assert.Equal(t, getDataNameByPod(datasetName, podName), data.Name)
	})

	t.Run("create data resource when it already exists", func(t *testing.T) {
		dataset := getTestDataSet(datasetName, dataItemName)
		_, err := testDataSetReconciler.createDataResource(context.Background(), dataset, podName)
		assert.NoError(t, err)
	})
}

func TestUpdateDataResource(t *testing.T) {
	testDataSetReconciler, err := getTestDataSetReconciler()
	assert.NoError(t, err)

	datasetName := "test-ds"
	dataItemName := "test-data"
	podName := "test-pod"

	t.Run("update data resource when it changes", func(t *testing.T) {
		dataset := getTestDataSet(datasetName, dataItemName)
		data := getTestData(datasetName, dataItemName, podName)

		err = testDataSetReconciler.Create(context.Background(), data)
		assert.NoError(t, err)

		data.Spec.DataItems[0].LocalPath = "/local/test.conf"
		err := testDataSetReconciler.updateDataResource(context.Background(), dataset, data)
		assert.NoError(t, err)
	})
	t.Run("update data resource when it not changes", func(t *testing.T) {
		dataset := getTestDataSet(datasetName, dataItemName)
		data := getTestData(datasetName, dataItemName, podName)
		err := testDataSetReconciler.updateDataResource(context.Background(), dataset, data)
		assert.NoError(t, err)
	})
}

func TestPruneDataResources(t *testing.T) {
	testDataSetReconciler, err := getTestDataSetReconciler()
	assert.NoError(t, err)

	datasetName := "test-ds"
	dataItemName := "test-data"
	podName := "test-pod"

	t.Run("skip delete data resource if the pod exists", func(t *testing.T) {
		data := getTestData(datasetName, dataItemName, podName)
		dataList := &v1alpha1.DataList{
			Items: []v1alpha1.Data{*data},
		}
		podMap := map[string]*v12.Pod{
			podName: &v12.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: podName,
				},
			},
		}

		err := testDataSetReconciler.pruneDataResources(context.Background(), dataList, podMap)
		assert.NoError(t, err)
	})

	t.Run("delete data resource if the pod not exists", func(t *testing.T) {
		data := getTestData(datasetName, dataItemName, podName)
		dataList := &v1alpha1.DataList{
			Items: []v1alpha1.Data{*data},
		}
		podMap := map[string]*v12.Pod{}

		err := testDataSetReconciler.pruneDataResources(context.Background(), dataList, podMap)
		assert.NoError(t, err)
	})
}

func getTestDataSetReconciler() (*DataSetReconciler, error) {
	dsReconciler := &DataSetReconciler{}

	s := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := scheme.AddToScheme(s); err != nil {
		return nil, err
	}
	dsReconciler.Scheme = s

	cli := fake.NewClientBuilder().WithScheme(s).Build()
	dsReconciler.Client = cli

	return dsReconciler, nil
}

func getTestDataSet(dataSetName, dataItemName string) *v1alpha1.DataSet {
	dataset := &v1alpha1.DataSet{
		ObjectMeta: v1.ObjectMeta{
			Name: dataSetName,
		},
		Spec: v1alpha1.DataSetSpec{
			Template: v1alpha1.DataTemplateSpec{
				DataItems: []v1alpha1.DataItem{},
				DataSources: &v1alpha1.DataSources{
					Hdfs: &v1alpha1.HdfsDataSource{
						Addresses: []string{"localhost:8020"},
						UserName:  "root",
					},
				},
			},
			WorkloadSelector: map[string]string{
				"app": "test",
			},
		},
	}

	dataset.Spec.Template.DataItems = []v1alpha1.DataItem{getTestDataItem(dataItemName)}
	return dataset
}

func getTestData(datasetName, dataItemName, podName string) *v1alpha1.Data {
	data := &v1alpha1.Data{
		ObjectMeta: v1.ObjectMeta{
			Name: getDataNameByPod(datasetName, podName),
			Labels: map[string]string{
				v1alpha1.KudaKeyPod: podName,
			},
		},
		Spec: v1alpha1.DataSpec{
			DataSources: &v1alpha1.DataSources{
				Hdfs: &v1alpha1.HdfsDataSource{
					Addresses: []string{"localhost:8020"},
					UserName:  "root",
				},
			},
		},
	}

	data.Spec.DataItems = []v1alpha1.DataItem{getTestDataItem(dataItemName)}

	return data
}

func getTestDataItem(name string) v1alpha1.DataItem {
	return v1alpha1.DataItem{
		Name:           name,
		Namespace:      "test-ns",
		RemotePath:     "/nginx-conf/test.conf",
		Version:        "v1",
		LocalPath:      "/tmp/test.conf",
		DataSourceType: "hdfs",
	}
}
