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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	datav1alpha1 "github.com/kuda-io/kuda/pkg/api/data/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = datav1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = datav1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("data resource", func() {
	dataset := &datav1alpha1.DataSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dataset",
			Namespace: "default",
		},
		Spec: datav1alpha1.DataSetSpec{
			Template: datav1alpha1.DataTemplateSpec{
				DataItems: []datav1alpha1.DataItem{
					{
						Name:           "test",
						Namespace:      "test",
						RemotePath:     "/remote",
						LocalPath:      "/local",
						Version:        "v1",
						DataSourceType: "hdfs",
					},
				},
				DataSources: &datav1alpha1.DataSources{
					Hdfs: &datav1alpha1.HdfsDataSource{
						Addresses: []string{"127.0.0.1:8020"},
						UserName:  "root",
					},
				},
			},
			WorkloadSelector: map[string]string{"app": "test"},
		},
	}

	data := &datav1alpha1.Data{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-data",
			Namespace: "default",
		},
		Spec: datav1alpha1.DataSpec{
			DataItems: []datav1alpha1.DataItem{
				{
					Name:           "test",
					Namespace:      "test",
					RemotePath:     "/remote",
					LocalPath:      "/local",
					Version:        "v1",
					DataSourceType: "hdfs",
				},
			},
			DataSources: &datav1alpha1.DataSources{
				Hdfs: &datav1alpha1.HdfsDataSource{
					Addresses: []string{"127.0.0.1:8020"},
					UserName:  "root",
				},
			},
		},
	}

	It("dataset", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := k8sClient.Create(ctx, dataset)
		Expect(err).NotTo(HaveOccurred())

		found := &datav1alpha1.DataSet{}
		key := types.NamespacedName{
			Namespace: dataset.Namespace,
			Name:      dataset.Name,
		}
		err = k8sClient.Get(ctx, key, found)
		Expect(err).NotTo(HaveOccurred())
		Expect(found.Spec).Should(Equal(dataset.Spec))

		err = k8sClient.Delete(ctx, dataset)
		Expect(err).NotTo(HaveOccurred())
	})

	It("data", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := k8sClient.Create(ctx, data)
		Expect(err).NotTo(HaveOccurred())

		found := &datav1alpha1.Data{}
		key := types.NamespacedName{
			Namespace: data.Namespace,
			Name:      data.Name,
		}
		err = k8sClient.Get(ctx, key, found)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Delete(ctx, data)
		Expect(err).NotTo(HaveOccurred())
	})
})
