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
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	identityv1alpha1 "github.com/osagberg/identity-claim-operator/api/v1alpha1"
)

const (
	finalizerName = "identity.cluster.local/finalizer"
	trustDomain   = "cluster.local"
)

// IdentityClaimReconciler reconciles a IdentityClaim object
type IdentityClaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=identity.cluster.local,resources=identityclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=identity.cluster.local,resources=identityclaims/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=identity.cluster.local,resources=identityclaims/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile implements the reconciliation loop for IdentityClaim resources
func (r *IdentityClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the IdentityClaim
	claim := &identityv1alpha1.IdentityClaim{}
	if err := r.Get(ctx, req.NamespacedName, claim); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("IdentityClaim not found, ignoring")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !claim.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, claim)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(claim, finalizerName) {
		controllerutil.AddFinalizer(claim, finalizerName)
		if err := r.Update(ctx, claim); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Initialize status if needed
	if claim.Status.Phase == "" {
		claim.Status.Phase = identityv1alpha1.PhasePending
		claim.Status.SpiffeID = r.generateSpiffeID(claim)
		claim.Status.SecretName = fmt.Sprintf("%s-identity", claim.Name)
		if err := r.Status().Update(ctx, claim); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Verify pods matching selector exist
	podsFound, err := r.verifyMatchingPods(ctx, claim)
	if err != nil {
		return ctrl.Result{}, err
	}
	if podsFound == 0 {
		r.setCondition(claim, identityv1alpha1.ConditionPodsVerified, metav1.ConditionFalse, "NoPods",
			"No pods matching selector found")
		if err := r.Status().Update(ctx, claim); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	r.setCondition(claim, identityv1alpha1.ConditionPodsVerified, metav1.ConditionTrue, "PodsFound",
		fmt.Sprintf("Found %d matching pod(s)", podsFound))

	// Create or update the Certificate resource
	if err := r.reconcileCertificate(ctx, claim); err != nil {
		claim.Status.Phase = identityv1alpha1.PhaseFailed
		r.setCondition(claim, identityv1alpha1.ConditionCertificateIssued, metav1.ConditionFalse,
			"CertificateFailed", err.Error())
		if statusErr := r.Status().Update(ctx, claim); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Check certificate status
	cert := &certmanagerv1.Certificate{}
	certName := claim.Status.SecretName
	if err := r.Get(ctx, client.ObjectKey{Namespace: claim.Namespace, Name: certName}, cert); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		// Certificate not found yet, requeue
		claim.Status.Phase = identityv1alpha1.PhaseIssuing
		r.setCondition(claim, identityv1alpha1.ConditionCertificateIssued, metav1.ConditionFalse,
			"Issuing", "Waiting for certificate to be created")
		if err := r.Status().Update(ctx, claim); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Check if certificate is ready
	certReady := false
	for _, cond := range cert.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady && cond.Status == cmmeta.ConditionTrue {
			certReady = true
			break
		}
	}

	if !certReady {
		claim.Status.Phase = identityv1alpha1.PhaseIssuing
		r.setCondition(claim, identityv1alpha1.ConditionCertificateIssued, metav1.ConditionFalse,
			"Issuing", "Certificate is being issued")
		if err := r.Status().Update(ctx, claim); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Certificate is ready
	claim.Status.Phase = identityv1alpha1.PhaseReady
	if cert.Status.NotAfter != nil {
		claim.Status.ExpiresAt = cert.Status.NotAfter
	}
	r.setCondition(claim, identityv1alpha1.ConditionCertificateIssued, metav1.ConditionTrue,
		"Issued", "Certificate has been issued")
	r.setCondition(claim, identityv1alpha1.ConditionReady, metav1.ConditionTrue,
		"Ready", "Identity is ready for use")

	if err := r.Status().Update(ctx, claim); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue before certificate expires to trigger renewal
	if claim.Status.ExpiresAt != nil {
		renewAt := claim.Status.ExpiresAt.Add(-10 * time.Minute)
		if time.Now().Before(renewAt) {
			return ctrl.Result{RequeueAfter: time.Until(renewAt)}, nil
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Minute}, nil
}

// reconcileDelete handles cleanup when an IdentityClaim is deleted
func (r *IdentityClaimReconciler) reconcileDelete(ctx context.Context, claim *identityv1alpha1.IdentityClaim) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Delete the associated Certificate
	cert := &certmanagerv1.Certificate{}
	certName := claim.Status.SecretName
	if certName != "" {
		if err := r.Get(ctx, client.ObjectKey{Namespace: claim.Namespace, Name: certName}, cert); err == nil {
			log.Info("Deleting Certificate", "name", certName)
			if err := r.Delete(ctx, cert); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(claim, finalizerName)
	if err := r.Update(ctx, claim); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// generateSpiffeID creates the SPIFFE ID for the claim
func (r *IdentityClaimReconciler) generateSpiffeID(claim *identityv1alpha1.IdentityClaim) string {
	return fmt.Sprintf("spiffe://%s/ns/%s/ic/%s", trustDomain, claim.Namespace, claim.Name)
}

// verifyMatchingPods checks if pods matching the selector exist
func (r *IdentityClaimReconciler) verifyMatchingPods(ctx context.Context, claim *identityv1alpha1.IdentityClaim) (int, error) {
	selector, err := metav1.LabelSelectorAsSelector(&claim.Spec.Selector)
	if err != nil {
		return 0, fmt.Errorf("invalid selector: %w", err)
	}

	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(claim.Namespace),
		client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return 0, fmt.Errorf("failed to list pods: %w", err)
	}

	return len(podList.Items), nil
}

// reconcileCertificate creates or updates the cert-manager Certificate
func (r *IdentityClaimReconciler) reconcileCertificate(ctx context.Context, claim *identityv1alpha1.IdentityClaim) error {
	certName := claim.Status.SecretName

	// Calculate duration from TTL
	duration := claim.Spec.TTL.Duration
	if duration == 0 {
		duration = time.Hour // default 1h
	}

	cert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: claim.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cert, func() error {
		// Set owner reference for garbage collection
		if err := controllerutil.SetControllerReference(claim, cert, r.Scheme); err != nil {
			return err
		}

		cert.Spec = certmanagerv1.CertificateSpec{
			SecretName: certName,
			Duration:   &metav1.Duration{Duration: duration},
			// Renew at 2/3 of the duration
			RenewBefore: &metav1.Duration{Duration: duration / 3},
			URIs:        []string{claim.Status.SpiffeID},
			CommonName:  claim.Name,
			IssuerRef: cmmeta.ObjectReference{
				Name:  "selfsigned-issuer",
				Kind:  "ClusterIssuer",
				Group: "cert-manager.io",
			},
			PrivateKey: &certmanagerv1.CertificatePrivateKey{
				Algorithm: certmanagerv1.ECDSAKeyAlgorithm,
				Size:      256,
			},
		}
		return nil
	})

	return err
}

// setCondition updates or adds a condition to the claim
func (r *IdentityClaimReconciler) setCondition(claim *identityv1alpha1.IdentityClaim, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&claim.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: claim.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *IdentityClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&identityv1alpha1.IdentityClaim{}).
		Owns(&certmanagerv1.Certificate{}).
		Named("identityclaim").
		Complete(r)
}
