# Identity Claim Operator

A Kubernetes operator for workload identity management using SPIFFE-compatible identities and cert-manager integration.

## Overview

The Identity Claim Operator provides a Kubernetes-native approach to managing workload identity. Workloads declare identity requirements through Custom Resources, and the operator handles verification, certificate issuance via cert-manager, and automatic renewal.

## Features

- **Declarative Identity Claims**: Define identity requirements as Kubernetes resources
- **SPIFFE-Compatible**: Generates industry-standard SPIFFE identity URIs
- **cert-manager Integration**: Leverages cert-manager for certificate lifecycle
- **Automatic Renewal**: Certificates are automatically renewed before expiry
- **Status Conditions**: Full observability with Kubernetes-standard conditions

## Quick Start

### Option 1: Helm (Recommended)

```bash
helm install identity-claim-operator ./charts/identity-claim-operator \
  --namespace identity-system --create-namespace
```

### Option 2: Kustomize

```bash
# Install CRDs
make install

# Deploy operator
make deploy IMG=ghcr.io/osagberg/identity-claim-operator:latest
```

### Option 3: Local Development

```bash
make install  # Install CRDs
make run      # Run operator locally
```

## Example

```yaml
apiVersion: identity.cluster.local/v1alpha1
kind: IdentityClaim
metadata:
  name: my-service-identity
  namespace: default
spec:
  selector:
    matchLabels:
      app: my-service
  ttl: 1h
```

After applying, check the status:

```bash
kubectl get identityclaims

NAME                  PHASE   SPIFFE ID                                                  SECRET                         AGE
my-service-identity   Ready   spiffe://cluster.local/ns/default/ic/my-service-identity   my-service-identity-identity   1m
```

## How It Works

1. You create an `IdentityClaim` targeting pods via label selector
2. Operator verifies matching pods exist
3. Operator generates a SPIFFE ID: `spiffe://cluster.local/ns/<namespace>/ic/<name>`
4. Operator creates a cert-manager `Certificate` resource
5. cert-manager issues the certificate and stores it in a `Secret`
6. Pods can mount the Secret for mTLS authentication

## CRD Reference

### IdentityClaimSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `selector` | `LabelSelector` | Yes | Pods matching these labels receive the identity |
| `ttl` | `Duration` | No | Certificate validity period (default: `1h`) |

### IdentityClaimStatus

| Field | Type | Description |
|-------|------|-------------|
| `phase` | `string` | Current phase: `Pending`, `Issuing`, `Ready`, `Failed` |
| `spiffeId` | `string` | Assigned SPIFFE URI |
| `secretName` | `string` | Name of Secret containing TLS certificate |
| `expiresAt` | `Time` | Certificate expiration timestamp |
| `conditions` | `[]Condition` | Standard Kubernetes conditions |

### Status Conditions

| Type | Description |
|------|-------------|
| `Ready` | Overall health of the identity claim |
| `CertificateIssued` | Certificate has been issued by cert-manager |
| `PodsVerified` | Matching pods were found for the selector |

## Prerequisites

- Kubernetes 1.26+
- cert-manager installed with a configured `ClusterIssuer` named `selfsigned-issuer`

## Development

```bash
make generate   # Generate code (DeepCopy methods)
make manifests  # Generate CRD and RBAC manifests
make test       # Run unit tests
make lint       # Run linter
```

## License

Apache 2.0
