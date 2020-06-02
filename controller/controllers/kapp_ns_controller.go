/*

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
	"github.com/kapp-staging/kapp/controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	KappEnableLabelName  = "kapp-enabled"
	KappEnableLabelValue = "true"
)

// KappNSReconciler watches all namespaces
type KappNSReconciler struct {
	*BaseReconciler
	ctx context.Context
}

func NewKappNSReconciler(mgr ctrl.Manager) *KappNSReconciler {
	return &KappNSReconciler{
		BaseReconciler: NewBaseReconciler(mgr, "KappNS"),
		ctx:            context.Background(),
	}
}

func (r *KappNSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Namespace{}).
		Watches(&source.Kind{Type: &v1alpha1.HttpsCertIssuer{}}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{Type: &v1alpha1.HttpsCert{}}, &handler.EnqueueRequestForObject{}).
		//Owns(&v1alpha1.HttpsCert{}).
		Complete(r)
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
func (r *KappNSReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := r.ctx
	logger := r.Log.WithValues("kappns", req.NamespacedName)

	logger.Info("kapp ns reconciling...")

	var namespaceList v1.NamespaceList
	if err := r.List(ctx, &namespaceList); err != nil {
		err = client.IgnoreNotFound(err)
		return ctrl.Result{}, err
	}

	now := time.Now()

	for _, ns := range namespaceList.Items {
		_, exist := ns.Labels[KappEnableLabelName]
		if !exist {
			continue
		}

		var compList v1alpha1.ComponentList
		if err := r.List(ctx, &compList, client.InNamespace(ns.Name)); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}

		isActive := IsNamespaceKappEnabled(ns)

		for _, item := range compList.Items {
			component := item.DeepCopy()
			if component.Labels == nil {
				component.Labels = map[string]string{}
			}

			var suffix string
			if isActive {
				suffix = "enabled"
			} else {
				suffix = "disabled"
			}

			component.Labels["kapp-namespace-updated-at"] = strconv.Itoa(int(now.Unix())) + "-" + suffix

			if err := r.Update(ctx, component); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// check if default caIssuer & cert is created
	if err := r.reconcileDefaultCAIssuerAndCert(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KappNSReconciler) reconcileDefaultCAIssuerAndCert() error {
	defaultCAIssuerName := "default-cert-issuer"

	expectedCAIssuer := v1alpha1.HttpsCertIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultCAIssuerName,
		},
		Spec: v1alpha1.HttpsCertIssuerSpec{
			CAForTest: &v1alpha1.CAForTestIssuer{},
		},
	}

	currentCAIssuer := v1alpha1.HttpsCertIssuer{}
	err := r.Get(r.ctx, types.NamespacedName{Name: defaultCAIssuerName}, &currentCAIssuer)
	if err != nil {
		if errors.IsNotFound(err) {
			return err
		}

		if err := r.Create(r.ctx, &expectedCAIssuer); err != nil {
			return err
		}
	} else {
		currentCAIssuer.Spec = expectedCAIssuer.Spec
		if err := r.Update(r.ctx, &currentCAIssuer); err != nil {
			return err
		}
	}

	defaultCertName := "default-https-cert"
	expectedCert := v1alpha1.HttpsCert{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultCertName,
		},
		Spec: v1alpha1.HttpsCertSpec{
			HttpsCertIssuer: defaultCAIssuerName,
			Domains:         []string{"*"},
		},
	}

	var currentCert v1alpha1.HttpsCert
	if err = r.Get(r.ctx, types.NamespacedName{Name: defaultCertName}, &currentCert); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		return r.Create(r.ctx, &expectedCert)
	} else {
		currentCert.Spec = expectedCert.Spec
		return r.Update(r.ctx, &currentCert)
	}

	//return r.Patch(r.ctx, &currentCert, client.MergeFrom(&expectedCert))
}