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

```bash
# Install CRDs
make install

# Run the operator locally
make run

# Or deploy to cluster
make deploy IMG=<your-registry>/identity-claim-operator:tag
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

NAME                  PHASE   SPIFFE ID                                              SECRET                        AGE
my-service-identity   Ready   spiffe://cluster.local/ns/default/ic/my-service-identity   my-service-identity-identity   1m
```

## How It Works

1. You create an `IdentityClaim` targeting pods via label selector
2. Operator verifies matching pods exist
3. Operator generates a SPIFFE ID: `spiffe://cluster.local/ns/<namespace>/ic/<name>`
4. Operator creates a cert-manager `Certificate` resource
5. cert-manager issues the certificate and stores it in a `Secret`
6. Pods can mount the Secret for mTLS authentication

## Prerequisites

- Kubernetes 1.28+
- cert-manager installed with a configured `ClusterIssuer` named `selfsigned-issuer`

## Development

```bash
# Generate code and manifests
make generate
make manifests

# Run tests
make test

# Build container image
make docker-build IMG=<your-registry>/identity-claim-operator:tag
```

## License

Apache 2.0
