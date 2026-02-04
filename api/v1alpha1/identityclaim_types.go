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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IdentityClaimSpec defines the desired state of IdentityClaim
type IdentityClaimSpec struct {
	// selector specifies which pods should receive the identity.
	// Pods matching these labels will have access to the generated TLS certificate.
	// +required
	Selector metav1.LabelSelector `json:"selector"`

	// ttl specifies how long the certificate should be valid.
	// Defaults to 1h if not specified.
	// +optional
	// +kubebuilder:default="1h"
	TTL metav1.Duration `json:"ttl,omitempty"`
}

// IdentityClaimPhase represents the current phase of the IdentityClaim
// +kubebuilder:validation:Enum=Pending;Issuing;Ready;Failed
type IdentityClaimPhase string

const (
	// PhasePending means the claim is waiting for processing
	PhasePending IdentityClaimPhase = "Pending"
	// PhaseIssuing means the certificate is being issued
	PhaseIssuing IdentityClaimPhase = "Issuing"
	// PhaseReady means the identity is ready for use
	PhaseReady IdentityClaimPhase = "Ready"
	// PhaseFailed means the identity could not be issued
	PhaseFailed IdentityClaimPhase = "Failed"
)

// Condition types for IdentityClaim
const (
	// ConditionReady indicates the overall readiness of the identity claim
	ConditionReady = "Ready"
	// ConditionCertificateIssued indicates the certificate has been issued
	ConditionCertificateIssued = "CertificateIssued"
	// ConditionPodsVerified indicates matching pods were found
	ConditionPodsVerified = "PodsVerified"
)

// IdentityClaimStatus defines the observed state of IdentityClaim.
type IdentityClaimStatus struct {
	// phase represents the current lifecycle phase of the identity claim.
	// +optional
	Phase IdentityClaimPhase `json:"phase,omitempty"`

	// spiffeId is the SPIFFE identity URI assigned to this claim.
	// Format: spiffe://cluster.local/ns/<namespace>/ic/<name>
	// +optional
	SpiffeID string `json:"spiffeId,omitempty"`

	// secretName is the name of the Secret containing the TLS certificate.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// expiresAt is the timestamp when the current certificate expires.
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// conditions represent the current state of the IdentityClaim resource.
	// Condition types: Ready, CertificateIssued, PodsVerified
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Current phase"
// +kubebuilder:printcolumn:name="SPIFFE ID",type="string",JSONPath=".status.spiffeId",description="Assigned SPIFFE identity"
// +kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".status.secretName",description="Secret name"
// +kubebuilder:printcolumn:name="Expires",type="date",JSONPath=".status.expiresAt",description="Certificate expiration"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// IdentityClaim is the Schema for the identityclaims API
type IdentityClaim struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of IdentityClaim
	// +required
	Spec IdentityClaimSpec `json:"spec"`

	// status defines the observed state of IdentityClaim
	// +optional
	Status IdentityClaimStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// IdentityClaimList contains a list of IdentityClaim
type IdentityClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []IdentityClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IdentityClaim{}, &IdentityClaimList{})
}
