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
	"reflect"
	"time"

	slices "golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Namespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
//

// +kubebuilder:rbac:groups=danieloa.io,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=danieloa.io,resources=namespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=danieloa.io,resources=namespaces/finalizers,verbs=update
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	nsExceptionList := []string{"kube-system", "kube-public", "kube-node-lease", "local-path-storage"}

	if slices.Contains(nsExceptionList, req.NamespacedName.Name) {
		// errorMsg := "NamespaceReconciler only works for the platform namespace"
		// logger.Error(nil, errorMsg, "namespace", req.NamespacedName.Name)
		return ctrl.Result{}, nil
	}
	// in case we would need to filter to only watch for a specific namespace, we can apply the following filter
	// ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "platform", Namespace: "platform"}}
	ns := &corev1.Namespace{}

	err := r.Get(ctx, req.NamespacedName, ns)
	if err != nil {
		// logger.Error(err, "unable to fetch Namespace") // this is not needed
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// lets see if the network policy already exists
	npl := &networkingv1.NetworkPolicyList{}

	np_err := r.List(ctx, npl, client.InNamespace(ns.Name))

	if np_err != nil {
		// logger.Error(np_err, "unable to list NetworkPolicies") // this is not needed, the error is already logged by the system
		return ctrl.Result{}, np_err
	}

	// define compliant network policy
	protocolTCP := corev1.ProtocolTCP
	port := intstr.FromInt(80)

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "enforced-network-policy",
			Namespace: ns.Name,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "target-namespace",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port:     &port,
						},
					},
				},
			},
		},
	}

	// we want to get notified when a new namespace is created or updated
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, ns, func() error {

		if len(npl.Items) == 0 {
			logger.Info("No network policies found in namespace", "namespace", ns.Name)
			// create a new network policy
			// sets the owner reference on the network policy
			if np_err := ctrl.SetControllerReference(ns, np, r.Scheme); np_err != nil {
				logger.Error(np_err, "unable to set owner reference on NetworkPolicy") // this is not needed, the error is already logged by the system
				return np_err
			}
			// create the network policy
			if np_err := r.Create(ctx, np); np_err != nil {
				logger.Error(np_err, "unable to create NetworkPolicy") // this is not needed, the error is already logged by the system
				return np_err
			}
			logger.Info("NetworkPolicy created", "name", np.Name)
			return nil
		} else {
			logger.Info("Network policies found", "count", len(npl.Items))
		}

		if ns.Annotations == nil {
			ns.Annotations = map[string]string{}
		}

		currentTime := time.Now().Format(time.RFC3339)
		ns.Annotations["network-policy/enforced"] = currentTime

		return nil
	})

	if err != nil {
		logger.Error(err, "unable to create or update Namespace")
		return ctrl.Result{}, err
	}

	logger.Info("CreateOrUpdate", "namespace", ns.Name, "operation", op)

	switch op {
	case controllerutil.OperationResultCreated:
		// this clock never executes,
		// the namespace is created in another, so when the controller catches the 1st event for this ns, it will be in the "unchanged"
		// hence this never prints
		logger.Info("Namespace being created -> network policy enforced", "namespace", ns.Name)

	case controllerutil.OperationResultUpdated:
		// A namespace is updated
		// in the case there were no enforced network policy when entering the Reconcile loop, at least one is already created by the code above
		// so we need to do something else, like ensuring the network policy is appropriate?
		logger.Info("Namespace ", ns.Name, "being updated -> checking for network policies", "count", len(npl.Items))
		for _, netpol := range npl.Items {
			if reflect.DeepEqual(netpol.Spec, np.Spec) {
				// compliant
				logger.Info("NetworkPolicy is compliant", "name", netpol.Name, "namespace", ns.Name)
			} else {
				// non-compliant
				logger.Info("NetworkPolicy is not compliant", "name", netpol.Name, "namespace", ns.Name)

				// update the network policy
				if err := r.Update(ctx, np); err != nil {
					logger.Error(err, "unable to update NetworkPolicy", "name", np.Name, "namespace", ns.Name)
					return ctrl.Result{}, err
				}
				logger.Info("NetworkPolicy updated", "name", np.Name, "namespace", ns.Name)
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// SetupWithManager sets up the controller with the Manager
	// + For sets the object type that this controller will watch for
	// + Watches sets the object type that this controller will watch for
	// + Owns sets the object type that this controller will own
	// + Complete finalizes the controller setup
	// ==================
	// If you are writing a controller for a custom resource MyApp that creates Deployment, Service:
	// 🟢 Use Owns() for Deployment and Service.
	// If your controller should respond to changes in NetworkPolicies in a namespace:
	// 🟢 Use Watches() and map back to the namespace.
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}). // Primary resource
		// Secondary watch: NetworkPolicies
		Watches(
			&networkingv1.NetworkPolicy{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				np := obj.(*networkingv1.NetworkPolicy)

				// Reconcile the Namespace where this NetworkPolicy resides
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: np.Namespace,
						},
					},
				}
			}),
			// Optional: Only respond on create/update (filter out deletes, if needed)
			builder.WithPredicates(
				predicate.Or(
					predicate.ResourceVersionChangedPredicate{},
					predicate.GenerationChangedPredicate{},
				),
			),
		).
		// Owns(&networkingv1.NetworkPolicy{}).
		Complete(r)
}
