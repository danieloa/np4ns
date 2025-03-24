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
	"k8s.io/apimachinery/pkg/api/errors"
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

	// in case we would need to filter to only watch for a specific namespace, we can apply the following filter
	// ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"}}}
	ns := &corev1.Namespace{}

	err := r.Get(ctx, req.NamespacedName, ns)
	if err != nil {
		// logger.Error(err, "unable to fetch Namespace") // this is not needed
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// we want to get notified when a new namespace is created or updated
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, ns, func() error {
		logger.Info("CreateOrUpdate", "ns", ns.Name)

		// because we are using CreateOrUpdate, we need to modify the namespace object to trigger the create/update operation, else we will get a no-op "unchanged"
		// + sets the label on the namespace
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["network-policy"] = "managed"
		return nil
	})

	if err != nil {
		logger.Error(err, "unable to create or update Namespace")
		return ctrl.Result{}, err
	}
	logger.Info("operation:", "op", op)
	switch op {
	case controllerutil.OperationResultCreated:
		// A new namespace is created
		logger.Info("Namespace being created -> network policy enforced", "namespace", ns)

		// lets see if the network policy already exists
		npr := &networkingv1.NetworkPolicy{}

		np_err := r.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      "enforced-network-policy",
		}, npr)

		if np_err != nil {
			if errors.IsNotFound(np_err) {
				// there is no network policy, set controller reference and create it
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
				if np_err = ctrl.SetControllerReference(ns, npc, r.Scheme); np_err != nil {
					logger.Error(np_err, "unable to set owner reference on NetworkPolicy") // this is not needed, the error is already logged by the system
					return ctrl.Result{}, np_err
				}

				if np_err = r.Create(ctx, npc); np_err != nil {
					logger.Error(np_err, "unable to create NetworkPolicy") // this is not needed, the error is already logged by the system
					return ctrl.Result{}, np_err
				}
			}
		}
	case controllerutil.OperationResultUpdated:
		// a namespace is updated
		logger.Info("Namespace being updated -> network policy enforced", "namespace", ns)
		// lets see if the network policy already exists
		npr := &networkingv1.NetworkPolicy{}

		np_err := r.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      "enforced-network-policy",
		}, npr)

		if np_err != nil {
			if errors.IsNotFound(np_err) {
				// there is no network policy, set controller reference and create it
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
				if np_err = ctrl.SetControllerReference(ns, npc, r.Scheme); np_err != nil {
					logger.Error(np_err, "unable to set owner reference on NetworkPolicy") // this is not needed, the error is already logged by the system
					return ctrl.Result{}, np_err
				}

				if np_err = r.Create(ctx, npc); np_err != nil {
					logger.Error(np_err, "unable to create NetworkPolicy") // this is not needed, the error is already logged by the system
					return ctrl.Result{}, np_err
				}
			}
		} else {
			if np_err = ctrl.SetControllerReference(ns, npr, r.Scheme); np_err != nil {
				// sets the owner reference on the network policy
				logger.Error(np_err, "unable to set owner reference on NetworkPolicy") // this is not needed, the error is already logged by the system
				return ctrl.Result{}, np_err
			}
			if np_err = r.Delete(ctx, npr); np_err != nil {
				// assumes np is wrong and deletes the existing one
				logger.Error(np_err, "unable to delete NetworkPolicy") // this is not needed, the error is already logged by the system
				return ctrl.Result{}, np_err
			}
			if np_err = r.Create(ctx, npr); np_err != nil {
				// creates a new network policy
				logger.Error(np_err, "unable to create NetworkPolicy") // this is not needed, the error is already logged by the system
				return ctrl.Result{}, np_err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// SetupWithManager sets up the controller with the Manager
	// + For sets the object type that this controller will watch for
	// + Owns sets the object type that this controller will own
	// + Complete finalizes the controller setup
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}

// TODO:
// + CreateOrUpdate should only modify the ns.labels and return either `nil` or `ctrl.SetControllerReference(ns, npc, r.Scheme)`
// + as the logic to create the network policy shoud be gone bc ^^^. an implementation of it should go inside the respective case statement
// + the logger.Error should be removed from the CreateOrUpdate function
// + the logger.Info should be removed from the CreateOrUpdate function
