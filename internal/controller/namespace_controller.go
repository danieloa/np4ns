/*
Copyright 2025.

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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	log "sigs.k8s.io/controller-runtime/pkg/log"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=danieloa.io,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=danieloa.io,resources=namespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=danieloa.io,resources=namespaces/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Namespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	ns := &corev1.Namespace{}
	npc := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "enforced-network-policy",
			Namespace: ns.Name,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{},
		},
	}
	// we want to get notified when a new namespace is created or updated
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, ns, func() error {
		ns.Name = req.Name
		return ctrl.SetControllerReference(ns, ns, r.Scheme)
	})
	if err != nil {
		logger.Error(err, "unable to create or update Namespace")
		return ctrl.Result{}, err
	}
	switch op {
	case controllerutil.OperationResultCreated:
		// new namespace is created
		logger.Info("Namespace created", "namespace", ns)
		np := &networkingv1.NetworkPolicy{}

		err = r.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      "enforced-network-policy",
		}, np)

		if err != nil {
			if errors.IsNotFound(err) {
				// there is no network policy, set controller reference and create it

				if err = ctrl.SetControllerReference(ns, npc, r.Scheme); err != nil {
					// logger.Error(err, "unable to set owner reference on NetworkPolicy") // this is not needed, the error is already logged
					return ctrl.Result{}, err
				}

				if err = r.Create(ctx, npc); err != nil {
					// logger.Error(err, "unable to create NetworkPolicy") // this is not needed, the error is already logged
					return ctrl.Result{}, err
				}

			}
		}
	case controllerutil.OperationResultUpdated:
		// a namespace is updated
		logger.Info("Namespace updated", "namespace", ns)
		np := &networkingv1.NetworkPolicy{}
		err = r.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      "enforced-network-policy",
		}, np)

		if err != nil {
			if errors.IsNotFound(err) {
				// for whatever reasson the network policy does not exist, create it again
				if err = ctrl.SetControllerReference(ns, npc, r.Scheme); err != nil {
					// logger.Error(err, "unable to set owner reference on NetworkPolicy") // this is not needed, the error is already logged
					return ctrl.Result{}, err
				}

				if err = r.Create(ctx, npc); err != nil {
					// logger.Error(err, "unable to create NetworkPolicy") // this is not needed, the error is already logged
					return ctrl.Result{}, err
				}

			}
		} else {
			// np exists so lets check if the network policy is still valid, if not, recreate it
			npc := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "enforced-network-policy",
					Namespace: ns.Name,
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeEgress,
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{},
				},
			}
			if np == npc {
				// the network policy is still valid
				return ctrl.Result{}, nil
			} else {
				// the network policy is not valid, recreate it
				if err = r.Delete(ctx, np); err != nil {
					// logger.Error(err, "unable to delete NetworkPolicy") // this is not needed, the error is already logged
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// SetupWithManager sets up the controller with the Manager
	// + For sets the object type that this controller will watch for
	// + Owns sets the object type that this controller will own
	// + Complete sets the finalizer for the controller
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
