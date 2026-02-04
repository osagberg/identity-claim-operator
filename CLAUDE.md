# Identity Claim Operator

## CRITICAL: This is a PUBLIC repository

This directory is a git submodule. All commits here are pushed to:
**https://github.com/osagberg/identity-claim-operator** (PUBLIC)

Do NOT commit:
- Secrets or credentials
- Internal infrastructure details
- Cluster-specific configurations
- Anything you wouldn't want public

## Project Structure (after kubebuilder scaffold)

```
operator/
├── api/v1alpha1/           # CRD type definitions
│   └── identityclaim_types.go
├── controllers/            # Reconciliation logic
│   └── identityclaim_controller.go
├── config/
│   ├── crd/               # Generated CRD manifests
│   ├── rbac/              # RBAC for operator
│   ├── manager/           # Deployment manifests
│   └── samples/           # Example CRs
├── Dockerfile
├── Makefile
└── go.mod
```

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
make docker-build IMG=<registry>/identity-claim-operator:tag

# Deploy to cluster
make deploy IMG=<registry>/identity-claim-operator:tag
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
    // 4. Create/update dependent resources (secrets, certs)
    // 5. Update status with conditions
    // 6. Return (requeue if needed)
}
```

## Status Conditions

Always set status conditions:
- `Ready` - Overall health
- `CertificateIssued` - Certificate status
- `Verified` - Identity verification status

## Testing

- Unit tests: `*_test.go` files alongside code
- Integration tests: `controllers/suite_test.go` (envtest)
- E2E tests: Require running cluster
