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

	err := r.Get(ctx, req.NamespacedName, ns)
	if err != nil {
		logger.Error(err, "unable to fetch Namespace")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	np := &networkingv1.NetworkPolicy{}

	err = r.Get(ctx, client.ObjectKey{
		Namespace: ns.Name,
		Name:      "enforced-network-policy",
	}, np)

	if err != nil {
		if errors.IsNotFound(err) {
			np := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "enforced-network-policy",
					Namespace: ns.Name,
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					// if we'd like to seleck based on Label selector...
					// PodSelector: metav1.LabelSelector{
					// 	 MatchLabels: map[string]string{
					// 	 	 "app": "myapp",
					// 	 },
					// },
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeEgress,
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{},
					// if we'd like to block based on namespace...
					// Egress: []networkingv1.NetworkPolicyEgressRule{
					// 	{
					// 		To: []networkingv1.NetworkPolicyPeer{
					// 			{
					// 				NamespaceSelector: &metav1.LabelSelector{
					// 					MatchLabels: map[string]string{
					// 						"project": "myproject",
					// 					},
					// 				},
					// 			},
					// 		},
					// 	}
					// },
				},
			}

			if err = ctrl.SetControllerReference(ns, np, r.Scheme); err != nil {
				logger.Error(err, "unable to set owner reference on NetworkPolicy")
				return ctrl.Result{}, err
			}

			if err = r.Create(ctx, np); err != nil {
				logger.Error(err, "unable to create NetworkPolicy")
				return ctrl.Result{}, err
			}

		}
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&corev1.Namespace{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
