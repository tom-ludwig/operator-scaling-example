# Operator Scaling Package Example

Example Kubernetes operator demonstrating horizontal scaling with the [operator-scaling](https://github.com/tom-ludwig/operator-scaling) package.

## What it does

This operator uses **consistent hashing** to distribute workloads across multiple replicas. Each replica only reconciles resources assigned to its shard, enabling horizontal scaling without leader election.

## Key Components

| File | Purpose |
|------|---------|
| `cmd/main.go` | Sets up sharding with `sharding.SetupWithDefaults(mgr)` |
| `internal/controller/guestbook_controller.go` | Uses `sharding.NewShardingPredicate` to filter events |
| `config/manager/manager.yaml` | Deployment with `POD_NAME` and `POD_NAMESPACE` env vars |
| `config/rbac/role.yaml` | Includes pod list/watch permissions for shard coordination |

## Quick Start

```bash
# Build and load image (for minikube)
make docker-build
minikube image load operator-scaling-example:latest

# Deploy
make install deploy

# Create test resources
kubectl apply -f config/samples/guestbooks_test.yaml
```

## Verify Sharding

```bash
# Watch which pod processes each resource
kubectl get guestbooks -n operator-scaling-package-example-system \
  -o custom-columns=NAME:.metadata.name,PROCESSED_BY:.status.processedBy,COUNT:.status.reconcileCount -w

# Scale to multiple replicas
kubectl scale deployment -n operator-scaling-package-example-system \
  operator-scaling-package-example-controller-manager --replicas=3
```

Each Guestbook should be consistently handled by the same pod based on its hash.

## Required RBAC

The sharding package needs to watch pods for shard coordination:

```yaml
- apiGroups: [""]
  resources: [pods]
  verbs: [get, list, watch]
- apiGroups: [coordination.k8s.io]
  resources: [leases]
  verbs: [get, list, watch, create, update, patch, delete]
```

## Predicates

When using the sharding predicate, combine it with `GenerationChangedPredicate` to avoid infinite reconcile loops from status updates:

```go
WithEventFilter(predicate.And(
    predicate.GenerationChangedPredicate{},
    sharding.NewShardingPredicate(orchestrator, nil),
))
```

## License

Apache License 2.0
