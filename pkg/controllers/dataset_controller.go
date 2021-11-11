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
	"fmt"
	"reflect"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	datav1alpha1 "github.com/kuda-io/kuda/pkg/api/data/v1alpha1"
)

// DataSetReconciler reconciles a DataSet object
type DataSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=data.kuda.io,resources=datasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=data.kuda.io,resources=datasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=data.kuda.io,resources=datasets/finalizers,verbs=update
//+kubebuilder:rbac:groups=data.kuda.io,resources=datas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=data.kuda.io,resources=datas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods/exec,verbs=get;list;patch;update;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *DataSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	log.Info("reconcile dataset", "instance", req.NamespacedName)

	// Get the DataSet instance
	instance := &datav1alpha1.DataSet{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("DataSet resource not found. The object has been deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get DataSet")
		return ctrl.Result{}, err
	}

	// Get the pod list that match workloadSelector in DataSet
	podList := &v1.PodList{}
	podListOpts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(instance.Spec.WorkloadSelector),
	}
	if err := r.List(ctx, podList, podListOpts...); err != nil {
		log.Error(err, "failed to list pods")
		return ctrl.Result{}, err
	}

	// Get Data list for the DataSet
	dataList := &datav1alpha1.DataList{}
	dataListOpts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{datav1alpha1.KudaKeyDataSet: instance.Name}),
	}
	if err := r.List(ctx, dataList, dataListOpts...); err != nil {
		log.Error(err, "failed to list data resource")
		return ctrl.Result{}, err
	}

	// Sync DataSet
	if err := r.syncDataSet(ctx, instance, podList, dataList); err != nil {
		log.Error(err, "sync dataset error")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// syncDataSet takes action(create/update/delete) on each data resource by the corresponding pod.
func (r *DataSetReconciler) syncDataSet(ctx context.Context, instance *datav1alpha1.DataSet, podList *v1.PodList, dataList *datav1alpha1.DataList) error {
	log := ctrllog.FromContext(ctx)

	podMap := convertPodListToMap(podList)
	dataMap := convertDataListToMap(dataList)

	for _, pod := range podList.Items {
		dataName := getDataNameByPod(instance.Name, pod.Name)
		if dataOld, ok := dataMap[dataName]; ok {
			if err := r.updateDataResource(ctx, instance, dataOld); err != nil {
				log.Error(err, "failed to update data resource")
				return err
			}
			continue
		}

		// Create if the data resource is not exist.
		data, err := r.createDataResource(ctx, instance, pod.Name)
		if err != nil {
			log.Error(err, "failed to create date resource")
			return err
		}
		dataList.Items = append(dataList.Items, *data)
	}

	// delete data resource if the corresponding pod is not exist
	if err := r.pruneDataResources(ctx, dataList, podMap); err != nil {
		log.Error(err, "failed to delete data resource")
		return err
	}

	// update status of the dataset
	if err := r.updateDataSetStatus(ctx, instance, dataList); err != nil {
		log.Error(err, "failed to update dataset status", "name", instance.Name)
		return err
	}

	return nil
}

// createDataResource create a new data resource for the pod.
func (r *DataSetReconciler) createDataResource(ctx context.Context, instance *datav1alpha1.DataSet, podName string) (*datav1alpha1.Data, error) {
	data := r.newDataResource(instance, podName)
	if err := ctrl.SetControllerReference(instance, data, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, data); err != nil {
		if errors.IsAlreadyExists(err) {
			ctrllog.FromContext(ctx).Info("the data resource is already exist", "data.Name", data.Name, "data.Namespace", data.Namespace)
			return nil, nil
		}
		return nil, err
	}

	ctrllog.FromContext(ctx).Info("create data resource success", "data.Name", data.Name, "data.Namespace", data.Namespace)

	return data, nil
}

// newDataResource returns a data object for the pod.
func (r *DataSetReconciler) newDataResource(instance *datav1alpha1.DataSet, podName string) *datav1alpha1.Data {
	data := &datav1alpha1.Data{
		ObjectMeta: v12.ObjectMeta{
			Name:      getDataNameByPod(instance.Name, podName),
			Namespace: instance.Namespace,
			Labels: map[string]string{
				datav1alpha1.KudaKeyDataSet: instance.Name,
				datav1alpha1.KudaKeyPod:     podName,
			},
		},
		Spec: datav1alpha1.DataSpec{
			DataItems:   instance.Spec.Template.DataItems,
			DataSources: instance.Spec.Template.DataSources,
			Lifecycle:   instance.Spec.Template.Lifecycle,
		},
	}

	return data
}

// updateDataResource will update the data resource if it is not the latest.
func (r *DataSetReconciler) updateDataResource(ctx context.Context, instance *datav1alpha1.DataSet, dataOld *datav1alpha1.Data) error {
	podName := getPodNameByData(dataOld)

	dataNew := r.newDataResource(instance, podName)

	if !reflect.DeepEqual(dataOld.Spec, dataNew.Spec) {
		dataOld.Spec = dataNew.Spec
		if err := r.Update(ctx, dataOld); err != nil {
			return err
		}
		ctrllog.FromContext(ctx).Info("update data resource success", "data.Name", dataOld.Name, "data.Namespace", dataOld.Namespace)
	}

	return nil
}

// pruneDataResources clean up data resource if the pod has been deleted.
func (r *DataSetReconciler) pruneDataResources(ctx context.Context, dataList *datav1alpha1.DataList, podMap map[string]*v1.Pod) error {
	items := make([]datav1alpha1.Data, 0)

	for _, data := range dataList.Items {
		podName := getPodNameByData(&data)
		if _, ok := podMap[podName]; ok {
			items = append(items, data)
			continue
		}

		if err := r.Delete(ctx, &data); err != nil && !errors.IsNotFound(err) {
			items = append(items, data)
			ctrllog.FromContext(ctx).Error(err, "failed to delete data resource", "name", data.Name, "namespace", data.Namespace)
			return err
		}

		ctrllog.FromContext(ctx).Info("delete data resource success", "data.Name", data.Name, "data.Namespace", data.Namespace)
	}

	dataList.Items = items

	return nil
}

// Only when all the data items of an instance are download successfully, the instance is considered to be successful
func (r *DataSetReconciler) updateDataSetStatus(ctx context.Context, instance *datav1alpha1.DataSet, dataList *datav1alpha1.DataList) error {
	dataItemsNum := len(instance.Spec.Template.DataItems)

	newStatus := datav1alpha1.DataSetStatus{
		DataItems: dataItemsNum,
		Replicas:  len(dataList.Items),
	}

	for _, data := range dataList.Items {
		if data.Status.Success == dataItemsNum {
			newStatus.SuccessReplicas += 1
		}
	}
	newStatus.Ready = fmt.Sprintf("%d/%d", newStatus.SuccessReplicas, len(dataList.Items))

	if !reflect.DeepEqual(newStatus, instance.Status) {
		instance.Status = newStatus
		if err := r.Status().Update(ctx, instance); err != nil {
			return err
		}
		ctrllog.FromContext(ctx).Info("update dataset status success", "newStatus", newStatus)
	}

	return nil
}

// Get dataset from pod annotations that injected by webhook.
func (r *DataSetReconciler) getDataSetForPod(object client.Object) (string, bool) {
	if ds, ok := object.GetAnnotations()[datav1alpha1.KudaKeyDataSet]; ok && ds != "" {
		return ds, true
	}

	return "", false
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	podPredicates := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if _, exist := r.getDataSetForPod(e.Object); exist {
				return true
			}
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if _, exist := r.getDataSetForPod(e.Object); exist {
				return true
			}
			return false
		},
	}
	podHandlers := handler.MapFunc(func(object client.Object) []reconcile.Request {
		requests := make([]reconcile.Request, 0)
		if ds, exist := r.getDataSetForPod(object); exist {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      ds,
				Namespace: object.GetNamespace(),
			}})
		}

		return requests
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1alpha1.DataSet{}).
		Owns(&datav1alpha1.Data{}).
		Watches(
			&source.Kind{Type: &v1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(podHandlers),
			builder.WithPredicates(podPredicates)).
		Complete(r)
}

func convertPodListToMap(podList *v1.PodList) map[string]*v1.Pod {
	podMap := make(map[string]*v1.Pod, podList.Size())
	for _, pod := range podList.Items {
		podMap[pod.Name] = &pod
	}
	return podMap
}

func convertDataListToMap(dataList *datav1alpha1.DataList) map[string]*datav1alpha1.Data {
	dataMap := make(map[string]*datav1alpha1.Data, dataList.ListMeta.Size())
	for _, data := range dataList.Items {
		dataMap[data.Name] = &data
	}
	return dataMap
}

func getDataNameByPod(dsName, podName string) string {
	res := strings.Split(podName, "-")
	return fmt.Sprintf("%s-%s", dsName, strings.Join(res[len(res)-2:], "-"))
}

func getPodNameByData(data *datav1alpha1.Data) string {
	return data.GetLabels()[datav1alpha1.KudaKeyPod]
}
