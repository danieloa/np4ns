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
	"os"
	"reflect"
	"strings"
	"time"

	slices "golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// getNamespaceExceptionList returns the list of namespaces to exclude from network policy enforcement.
// Reads from NS_EXCEPTION_LIST environment variable (comma-separated), or uses defaults.
func getNamespaceExceptionList() []string {
	defaultExceptions := []string{"kube-system", "kube-public", "kube-node-lease", "local-path-storage"}

	envList := os.Getenv("NS_EXCEPTION_LIST")
	if envList == "" {
		return defaultExceptions
	}

	exceptions := strings.Split(envList, ",")
	result := make([]string, 0, len(exceptions))
	for _, ns := range exceptions {
		trimmed := strings.TrimSpace(ns)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getNamespaceTargetList returns the list of namespaces to enforce network policies on.
// Reads from NS_TARGET_FOR_NP environment variable (comma-separated).
// If empty, returns nil (meaning enforce on all except exceptions).
func getNamespaceTargetList() []string {
	envList := os.Getenv("NS_TARGET_FOR_NP")
	if envList == "" {
		return nil
	}

	targets := strings.Split(envList, ",")
	result := make([]string, 0, len(targets))
	for _, ns := range targets {
		trimmed := strings.TrimSpace(ns)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// shouldEnforceNetworkPolicy determines if a namespace should have network policy enforcement.
func shouldEnforceNetworkPolicy(namespaceName string) bool {
	// Check exceptions first
	exceptions := getNamespaceExceptionList()
	if slices.Contains(exceptions, namespaceName) {
		return false
	}

	// Check if there's a target list
	targets := getNamespaceTargetList()
	if targets == nil {
		// No target list means enforce on all (except exceptions)
		return true
	}

	// If target list exists, only enforce on those namespaces
	return slices.Contains(targets, namespaceName)
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

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Check if this namespace should have network policy enforcement
	if !shouldEnforceNetworkPolicy(req.NamespacedName.Name) {
		logger.V(1).Info("Skipping namespace (not in target list or in exception list)", "namespace", req.NamespacedName.Name)
		return ctrl.Result{}, nil
	}
	// Fetch the namespace
	ns := &corev1.Namespace{}
	err := r.Get(ctx, req.NamespacedName, ns)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Define the desired (compliant) network policy spec
	desiredNPSpec := buildCompliantNetworkPolicySpec()

	// Check if the enforced network policy already exists
	enforcedNPName := "enforced-network-policy"
	existingNP := &networkingv1.NetworkPolicy{}
	err = r.Get(ctx, types.NamespacedName{Name: enforcedNPName, Namespace: ns.Name}, existingNP)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "failed to get NetworkPolicy")
			return ctrl.Result{}, err
		}

		// NetworkPolicy doesn't exist, create it
		logger.Info("Creating enforced NetworkPolicy", "namespace", ns.Name)
		np := &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      enforcedNPName,
				Namespace: ns.Name,
			},
			Spec: desiredNPSpec,
		}

		// Set owner reference so the NetworkPolicy is cleaned up when the namespace is deleted
		if err := ctrl.SetControllerReference(ns, np, r.Scheme); err != nil {
			logger.Error(err, "failed to set owner reference on NetworkPolicy")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, np); err != nil {
			logger.Error(err, "failed to create NetworkPolicy")
			return ctrl.Result{}, err
		}

		logger.Info("Successfully created enforced NetworkPolicy", "namespace", ns.Name, "policy", enforcedNPName)

		// Update namespace annotation
		if err := r.updateNamespaceAnnotation(ctx, ns); err != nil {
			logger.Error(err, "failed to update namespace annotation")
			// Don't return error, annotation is not critical
		}

		return ctrl.Result{}, nil
	}

	// NetworkPolicy exists, check if it's compliant
	if !reflect.DeepEqual(existingNP.Spec, desiredNPSpec) {
		logger.Info("NetworkPolicy is not compliant, updating", "namespace", ns.Name, "policy", enforcedNPName)

		// Update the spec to be compliant
		existingNP.Spec = desiredNPSpec
		if err := r.Update(ctx, existingNP); err != nil {
			logger.Error(err, "failed to update NetworkPolicy")
			return ctrl.Result{}, err
		}

		logger.Info("Successfully updated NetworkPolicy to be compliant", "namespace", ns.Name, "policy", enforcedNPName)

		// Update namespace annotation
		if err := r.updateNamespaceAnnotation(ctx, ns); err != nil {
			logger.Error(err, "failed to update namespace annotation")
		}

		return ctrl.Result{}, nil
	}

	// NetworkPolicy exists and is compliant
	logger.V(1).Info("NetworkPolicy is compliant", "namespace", ns.Name, "policy", enforcedNPName)
	return ctrl.Result{}, nil
}

// buildCompliantNetworkPolicySpec returns the desired NetworkPolicy spec that should be enforced.
// This matches the spec defined in helpers/compliant_np.yaml.
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
	protocolTCP := corev1.ProtocolTCP
	port := intstr.FromInt(80)

	return networkingv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{}, // Selects all pods in the namespace
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
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "myapp",
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
	}
}

// updateNamespaceAnnotation adds/updates an annotation on the namespace to track enforcement.
func (r *NamespaceReconciler) updateNamespaceAnnotation(ctx context.Context, ns *corev1.Namespace) error {
	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}

	currentTime := time.Now().Format(time.RFC3339)
	ns.Annotations["network-policy/enforced"] = currentTime

	return r.Update(ctx, ns)
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
		// Watch NetworkPolicies - trigger reconciliation when they are created, updated, or deleted
		// This ensures the operator can recreate the enforced policy if it gets deleted
		Watches(
			&networkingv1.NetworkPolicy{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				np := obj.(*networkingv1.NetworkPolicy)

				// Only watch for our enforced network policy
				if np.Name != "enforced-network-policy" {
					return []reconcile.Request{}
				}

				// Reconcile the Namespace where this NetworkPolicy resides
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: np.Namespace,
						},
					},
				}
			}),
			// No predicate filter - we want to catch all events including deletions
		).
		Complete(r)
}
