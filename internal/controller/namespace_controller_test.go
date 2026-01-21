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
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Namespace Controller", func() {
	Context("When reconciling a namespace", func() {
		const (
			npName = "enforced-network-policy"
		)

		ctx := context.Background()

		// Helper to generate unique namespace names
		getUniqueNamespace := func() string {
			return fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
		}

		It("should create a NetworkPolicy for a new namespace", func() {
			namespaceName := getUniqueNamespace()

			// Create a test namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			// Create reconciler
			reconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Trigger reconciliation
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: namespaceName},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify NetworkPolicy was created
			np := &networkingv1.NetworkPolicy{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      npName,
					Namespace: namespaceName,
				}, np)
			}, "10s", "1s").Should(Succeed())

			// Verify the NetworkPolicy spec is compliant
			Expect(np.Spec.PodSelector).To(Equal(metav1.LabelSelector{}))
			Expect(np.Spec.PolicyTypes).To(ContainElement(networkingv1.PolicyTypeEgress))
		})

		It("should skip namespaces in the exception list", func() {
			// Create a namespace that's in the exception list
			exceptionNS := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
				},
			}

			// Create reconciler
			reconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Trigger reconciliation
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: exceptionNS.Name},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify no NetworkPolicy was created
			np := &networkingv1.NetworkPolicy{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      npName,
				Namespace: exceptionNS.Name,
			}, np)
			Expect(err).To(HaveOccurred())
			Expect(client.IgnoreNotFound(err)).To(BeNil())
		})

		It("should update non-compliant NetworkPolicy", func() {
			namespaceName := getUniqueNamespace()

			// Create a test namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			// Create a non-compliant NetworkPolicy
			nonCompliantNP := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      npName,
					Namespace: namespaceName,
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeIngress, // Wrong policy type
					},
				},
			}
			Expect(k8sClient.Create(ctx, nonCompliantNP)).To(Succeed())

			// Create reconciler
			reconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Trigger reconciliation
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: namespaceName},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify NetworkPolicy was updated to be compliant
			np := &networkingv1.NetworkPolicy{}
			Eventually(func() []networkingv1.PolicyType {
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      npName,
					Namespace: namespaceName,
				}, np)
				return np.Spec.PolicyTypes
			}, "10s", "1s").Should(ContainElement(networkingv1.PolicyTypeEgress))
		})
	})

	Context("Environment variable configuration", func() {
		It("should use default exception list when NS_EXCEPTION_LIST is not set", func() {
			os.Unsetenv("NS_EXCEPTION_LIST")
			exceptions := getNamespaceExceptionList()
			Expect(exceptions).To(ContainElement("kube-system"))
			Expect(exceptions).To(ContainElement("kube-public"))
		})

		It("should use custom exception list from NS_EXCEPTION_LIST", func() {
			os.Setenv("NS_EXCEPTION_LIST", "custom-ns1,custom-ns2")
			defer os.Unsetenv("NS_EXCEPTION_LIST")

			exceptions := getNamespaceExceptionList()
			Expect(exceptions).To(ContainElement("custom-ns1"))
			Expect(exceptions).To(ContainElement("custom-ns2"))
		})

		It("should enforce on all namespaces when NS_TARGET_FOR_NP is not set", func() {
			os.Unsetenv("NS_TARGET_FOR_NP")
			os.Unsetenv("NS_EXCEPTION_LIST")

			Expect(shouldEnforceNetworkPolicy("test-ns")).To(BeTrue())
			Expect(shouldEnforceNetworkPolicy("kube-system")).To(BeFalse())
		})

		It("should only enforce on target namespaces when NS_TARGET_FOR_NP is set", func() {
			os.Setenv("NS_TARGET_FOR_NP", "target-ns1,target-ns2")
			os.Unsetenv("NS_EXCEPTION_LIST")
			defer os.Unsetenv("NS_TARGET_FOR_NP")

			Expect(shouldEnforceNetworkPolicy("target-ns1")).To(BeTrue())
			Expect(shouldEnforceNetworkPolicy("target-ns2")).To(BeTrue())
			Expect(shouldEnforceNetworkPolicy("other-ns")).To(BeFalse())
		})
	})
})
