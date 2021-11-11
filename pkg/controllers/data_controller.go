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

	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	datav1alpha1 "github.com/kuda-io/kuda/pkg/api/data/v1alpha1"
	"github.com/kuda-io/kuda/pkg/utils"
)

const (
	dataFinalizer = "kuda.io/finalizer"

	runtimeRole        = "kuda-runtime-role"
	runtimeRoleBinding = "kuda-runtime-rolebinding"
)

// DataReconciler reconciles a Data object
type DataReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=data.kuda.io,resources=datas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=data.kuda.io,resources=datas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=data.kuda.io,resources=datas/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;update;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *DataReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	log.Info("reconcile data resource", "instance", req.NamespacedName)

	// Get Data instance
	instance := &datav1alpha1.Data{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "failed to get Data resource")
			return ctrl.Result{}, err
		}

		log.Info("Data resource not found. The object has been deleted")
		return ctrl.Result{}, nil
	}

	// Get pod for the data resource
	pod := &v1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: getPodNameByData(instance), Namespace: instance.Namespace}, pod); err != nil {
		if errors.IsNotFound(err) {
			controllerutil.RemoveFinalizer(instance, dataFinalizer)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "failed to remove finalizer")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get pod")
		return ctrl.Result{}, err
	}

	// Sync Data resource
	if err := r.syncData(ctx, instance, pod); err != nil {
		log.Error(err, "failed to sync data resource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// syncData takes actions on pod by the data resource.
func (r *DataReconciler) syncData(ctx context.Context, instance *datav1alpha1.Data, pod *v1.Pod) error {
	log := ctrllog.FromContext(ctx)

	dataTag, err := utils.MD5(instance.Spec)
	if err != nil {
		log.Error(err, "failed to get data digest")
		return err
	}

	if err := r.updateDataStatus(ctx, instance, pod, dataTag); err != nil {
		log.Error(err, "failed to update status")
		return err
	}

	if err := r.updateRoleBinding(ctx, instance, pod); err != nil {
		log.Error(err, "failed to update rolebinding")
		return err
	}

	if err := r.updatePodAnnotations(ctx, pod, dataTag); err != nil {
		log.Error(err, "failed to update pod annotations")
		return err
	}

	if instance.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(instance, dataFinalizer) {
			delete(pod.Annotations, datav1alpha1.KudaKeyDigest)
			if err := r.Update(ctx, pod); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "failed to update pod")
				return err
			}

			controllerutil.RemoveFinalizer(instance, dataFinalizer)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "failed to remove finalizer")
				return err
			}
		}
		return nil
	}
	if !controllerutil.ContainsFinalizer(instance, dataFinalizer) {
		controllerutil.AddFinalizer(instance, dataFinalizer)
		if err := r.Update(ctx, instance); err != nil {
			log.Error(err, "failed to add finalizer")
			return err
		}
	}

	return nil
}

// update status for the data resource.
func (r *DataReconciler) updateDataStatus(ctx context.Context, instance *datav1alpha1.Data, pod *v1.Pod, dataTag string) error {
	var (
		diff = false
		err  error
	)

	newStatus := genLatestStatus(instance)
	if v, ok := pod.Annotations[datav1alpha1.KudaKeyDigest]; !ok || v != dataTag {
		newStatus = genDefaultStatus(instance)
		diff = true
	}

	if !reflect.DeepEqual(newStatus, instance.Status) {
		patch := client.MergeFrom(instance.DeepCopy())
		instance.Status = *newStatus
		if diff {
			err = r.Status().Update(ctx, instance)
		} else {
			err = r.Status().Patch(ctx, instance, patch)
		}
		if err != nil {

			return err
		}
		ctrllog.FromContext(ctx).Info("update data status success")
	}

	return nil
}

// update role binding for the service account of pod.
func (r *DataReconciler) updateRoleBinding(ctx context.Context, instance *datav1alpha1.Data, pod *v1.Pod) error {
	roleBinding := &v13.RoleBinding{
		ObjectMeta: v12.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", pod.Spec.ServiceAccountName, runtimeRoleBinding),
			Namespace: instance.Namespace,
		},
		RoleRef: v13.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     runtimeRole,
		},
		Subjects: []v13.Subject{
			{
				Kind:      v13.ServiceAccountKind,
				Name:      pod.Spec.ServiceAccountName,
				Namespace: instance.Namespace,
			},
		},
	}

	if err := r.Get(ctx, types.NamespacedName{Name: roleBinding.Name, Namespace: roleBinding.Namespace}, roleBinding); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, roleBinding); err != nil {
				return err
			}
		}
		return err
	}

	return nil
}

// update pod annotations to add data digest value.
func (r *DataReconciler) updatePodAnnotations(ctx context.Context, pod *v1.Pod, dataTag string) error {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string, 0)
	}

	if v, ok := pod.Annotations[datav1alpha1.KudaKeyDigest]; !ok || v != dataTag {
		pod.Annotations[datav1alpha1.KudaKeyDigest] = dataTag
		if err := r.Update(ctx, pod); err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1alpha1.Data{}).
		Complete(r)
}

func genDefaultStatus(d *datav1alpha1.Data) *datav1alpha1.DataStatus {
	status := make(datav1alpha1.DataItemsStatus, 0, len(d.Spec.DataItems))
	for _, data := range d.Spec.DataItems {
		status = append(status, datav1alpha1.DataItemStatus{
			Name:      data.Name,
			Namespace: data.Namespace,
			Version:   data.Version,
			Phase:     datav1alpha1.DataWaiting,
			StartTime: metav1.Now(),
		})
	}

	return &datav1alpha1.DataStatus{
		DataItemsStatus: status,
		DataItems:       len(d.Spec.DataItems),
		Waiting:         len(d.Spec.DataItems),
		Ready:           fmt.Sprintf("0/%d", len(d.Spec.DataItems)),
	}
}

func genLatestStatus(d *datav1alpha1.Data) *datav1alpha1.DataStatus {
	status := &datav1alpha1.DataStatus{
		DataItemsStatus: d.Status.DataItemsStatus,
		DataItems:       len(d.Spec.DataItems),
	}

	for _, item := range d.Status.DataItemsStatus {
		switch item.Phase {
		case datav1alpha1.DataWaiting:
			status.Waiting += 1
		case datav1alpha1.DataSuccess:
			status.Success += 1
		case datav1alpha1.DataDownloading:
			status.Downloading += 1
		case datav1alpha1.DataFailed:
			status.Failed += 1
		}
	}

	status.Ready = fmt.Sprintf("%d/%d", status.Success, len(d.Spec.DataItems))

	return status
}
