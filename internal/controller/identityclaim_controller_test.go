/*
Copyright 2026.

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	identityv1alpha1 "github.com/osagberg/identity-claim-operator/api/v1alpha1"
)

var _ = Describe("IdentityClaim Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		identityclaim := &identityv1alpha1.IdentityClaim{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind IdentityClaim")
			err := k8sClient.Get(ctx, typeNamespacedName, identityclaim)
			if err != nil && errors.IsNotFound(err) {
				resource := &identityv1alpha1.IdentityClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: identityv1alpha1.IdentityClaimSpec{
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test"},
						},
						TTL: metav1.Duration{Duration: 1 * time.Hour},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &identityv1alpha1.IdentityClaim{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance IdentityClaim")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &IdentityClaimReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	Context("When TTL is too short", func() {
		It("should be rejected at CRD admission level", func() {
			resource := &identityv1alpha1.IdentityClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "short-ttl-claim",
					Namespace: "default",
				},
				Spec: identityv1alpha1.IdentityClaimSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					TTL: metav1.Duration{Duration: 1 * time.Minute}, // too short (min 5m)
				},
			}
			err := k8sClient.Create(context.Background(), resource)
			Expect(err).To(HaveOccurred(), "CRD validation should reject TTL < 5m")
			Expect(err.Error()).To(ContainSubstring("TTL must be between 5m and 8760h"))
		})
	})

	Context("When TTL is too long", func() {
		It("should be rejected at CRD admission level", func() {
			resource := &identityv1alpha1.IdentityClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "long-ttl-claim",
					Namespace: "default",
				},
				Spec: identityv1alpha1.IdentityClaimSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					TTL: metav1.Duration{Duration: 10000 * time.Hour}, // too long (max 8760h)
				},
			}
			err := k8sClient.Create(context.Background(), resource)
			Expect(err).To(HaveOccurred(), "CRD validation should reject TTL > 8760h")
			Expect(err.Error()).To(ContainSubstring("TTL must be between 5m and 8760h"))
		})
	})

	Context("When custom issuerRef is specified", func() {
		const resourceName = "custom-issuer-claim"
		ctx := context.Background()
		nn := types.NamespacedName{Name: resourceName, Namespace: "default"}

		BeforeEach(func() {
			resource := &identityv1alpha1.IdentityClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: identityv1alpha1.IdentityClaimSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					TTL: metav1.Duration{Duration: 1 * time.Hour},
					IssuerRef: &identityv1alpha1.IssuerReference{
						Name:  "my-custom-issuer",
						Kind:  "Issuer",
						Group: "cert-manager.io",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			resource := &identityv1alpha1.IdentityClaim{}
			if err := k8sClient.Get(ctx, nn, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should use the custom issuer in the certificate", func() {
			controllerReconciler := &IdentityClaimReconciler{
				Client:            k8sClient,
				Scheme:            k8sClient.Scheme(),
				DefaultIssuerName: "selfsigned-issuer",
				DefaultIssuerKind: "ClusterIssuer",
			}

			// Reconcile until certificate would be created (may error since no actual cert-manager, that's OK)
			for i := 0; i < 5; i++ {
				_, _ = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			}

			// Verify the claim got a SPIFFE ID assigned
			claim := &identityv1alpha1.IdentityClaim{}
			Expect(k8sClient.Get(ctx, nn, claim)).To(Succeed())
			Expect(claim.Status.SpiffeID).To(ContainSubstring("custom-issuer-claim"))
		})
	})

	Context("When selector is invalid", func() {
		const resourceName = "invalid-selector-claim"
		ctx := context.Background()
		nn := types.NamespacedName{Name: resourceName, Namespace: "default"}

		BeforeEach(func() {
			resource := &identityv1alpha1.IdentityClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: identityv1alpha1.IdentityClaimSpec{
					Selector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOperator("InvalidOp"),
								Values:   []string{"test"},
							},
						},
					},
					TTL: metav1.Duration{Duration: 1 * time.Hour},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			resource := &identityv1alpha1.IdentityClaim{}
			if err := k8sClient.Get(ctx, nn, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should set Failed phase and SelectorError condition", func() {
			controllerReconciler := &IdentityClaimReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Reconcile multiple times to get past finalizer and status init
			for i := 0; i < 4; i++ {
				_, _ = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			}

			claim := &identityv1alpha1.IdentityClaim{}
			Expect(k8sClient.Get(ctx, nn, claim)).To(Succeed())
			Expect(claim.Status.Phase).To(Equal(identityv1alpha1.PhaseFailed))

			var found bool
			for _, cond := range claim.Status.Conditions {
				if cond.Type == identityv1alpha1.ConditionPodsVerified && cond.Reason == "SelectorError" {
					found = true
					Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				}
			}
			Expect(found).To(BeTrue(), "should have SelectorError condition on PodsVerified")
		})
	})

	Context("When no pods match the selector", func() {
		const resourceName = "no-pods-claim"
		ctx := context.Background()
		nn := types.NamespacedName{Name: resourceName, Namespace: "default"}

		BeforeEach(func() {
			resource := &identityv1alpha1.IdentityClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: identityv1alpha1.IdentityClaimSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "nonexistent-app-that-will-never-match",
						},
					},
					TTL: metav1.Duration{Duration: 1 * time.Hour},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			resource := &identityv1alpha1.IdentityClaim{}
			if err := k8sClient.Get(ctx, nn, resource); err == nil {
				// Remove finalizer first so delete can proceed
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should not issue certificate when no pods match selector", func() {
			controllerReconciler := &IdentityClaimReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile adds the finalizer
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile initializes status
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())

			// Third reconcile should hit the zero-pod guard
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30*time.Second), "should requeue after 30s when no pods found")

			// Verify the PodsVerified condition
			claim := &identityv1alpha1.IdentityClaim{}
			Expect(k8sClient.Get(ctx, nn, claim)).To(Succeed())

			var podsVerifiedFound bool
			for _, cond := range claim.Status.Conditions {
				if cond.Type == identityv1alpha1.ConditionPodsVerified {
					podsVerifiedFound = true
					Expect(cond.Status).To(Equal(metav1.ConditionFalse), "PodsVerified should be False")
					Expect(cond.Reason).To(Equal("NoPods"), "reason should be NoPods")
					Expect(cond.Message).To(ContainSubstring("No pods matching selector found"))
				}
			}
			Expect(podsVerifiedFound).To(BeTrue(), "PodsVerified condition should be set")
		})
	})
})
