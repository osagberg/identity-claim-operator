# Identity Claim Operator

## CRITICAL: This is a PUBLIC repository

This directory is a git submodule. All commits here are pushed to:
**https://github.com/osagberg/identity-claim-operator** (PUBLIC)

Do NOT commit:
- Secrets or credentials
- Internal infrastructure details
- Cluster-specific configurations
- Anything you wouldn't want public

## Quick Commands

```bash
# Generate CRD manifests from Go types
make generate
make manifests

# Run locally against cluster
make run

# Run tests
make test

# Build container
make docker-build IMG=ghcr.io/osagberg/identity-claim-operator:tag

# Deploy via Helm
helm install identity-claim-operator ./charts/identity-claim-operator \
  --namespace identity-system --create-namespace
```

## Project Structure

```
.
├── api/v1alpha1/           # CRD type definitions
│   └── identityclaim_types.go
├── internal/controller/    # Reconciliation logic
│   └── identityclaim_controller.go
├── charts/                 # Helm chart
│   └── identity-claim-operator/
├── config/
│   ├── crd/               # Generated CRD manifests
│   ├── rbac/              # RBAC for operator
│   ├── manager/           # Deployment manifests
│   └── samples/           # Example CRs
├── cmd/main.go            # Entry point
├── Dockerfile
├── Makefile
└── go.mod
```

## Code Style

- Go 1.22+
- Use `slog` for structured logging
- Table-driven tests
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Kubebuilder markers for RBAC and CRD generation

## Reconciliation Pattern

```go
func (r *IdentityClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the IdentityClaim
    // 2. Check if being deleted (handle finalizers)
    // 3. Validate spec
    // 4. Create/update dependent resources (Certificate CR)
    // 5. Update status with conditions
    // 6. Return (requeue if needed)
}
```

## Status Conditions

Always set status conditions:
- `Ready` - Overall health
- `CertificateIssued` - Certificate status
- `PodsVerified` - Matching pods found

## Testing

- Unit tests: `*_test.go` files alongside code
- Integration tests: `internal/controller/suite_test.go` (envtest)
- E2E tests: `test/e2e/` (requires running cluster)

## Key Files

| File | Purpose |
|------|---------|
| `api/v1alpha1/identityclaim_types.go` | CRD schema definition |
| `internal/controller/identityclaim_controller.go` | Main reconciliation logic |
| `charts/identity-claim-operator/values.yaml` | Helm configuration |
| `config/samples/identity_v1alpha1_identityclaim.yaml` | Example CR |
