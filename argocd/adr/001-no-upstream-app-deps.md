# ADR 001: Do not import upstream application packages as flight dependencies

## Status

Accepted

## Context

While building the ArgoCD flight, we considered importing `github.com/argoproj/argo-cd/v3/util/settings`
to use its `FilteredResource` type when constructing the `argocd-cm` ConfigMap data. The appeal was
type safety: the struct definitions in that package are the authoritative schema for what the config
values must look like, so using them directly would prevent schema drift.

We ran the experiment: added the import, wired it into `configmaps.go`, and attempted `go mod tidy`.

## What we observed

`go mod tidy` blocked indefinitely at 0% CPU. Investigation revealed:

- The argo-cd module has transitive dependencies that are **not available on the Go module proxy**
  (`https://proxy.golang.org`), specifically `github.com/argoproj/argo-cd/gitops-engine` with a
  pseudo-version.
- Go falls back to `direct` mode, spawning `git-remote-https` to clone
  `https://github.com/argoproj/argo-cd` directly from GitHub.
- The argo-cd repo has ~185,000 git objects. At typical network speeds this takes many minutes.
- `killall go` does not kill the git child processes — they continue cloning as orphans.
- A second `go mod tidy` in a concurrent terminal blocks on the module file lock held by the first.

## Decision

Do not import packages from the applications we are deploying (ArgoCD, Forgejo, etc.) as flight
dependencies. The cost is prohibitive:

1. **Cold `go mod tidy` requires a full git clone** of the upstream repo, which is slow and
   network-dependent. CI environments without a warm module cache would hang.
2. **Transitive dependency bloat**: argo-cd pulls in its entire dependency graph, which would
   conflict with or inflate the k8s library versions already used by the flight.
3. **Version coupling**: the flight's Go module would need to track the upstream app's Go module
   version, creating unnecessary churn.

## Consequences

ConfigMap data values for `argocd-cm` (and equivalent config in other flights) are expressed as
raw YAML strings embedded in Go string literals. The schema for these values is defined by the
upstream application's source code (`util/settings/filtered_resource.go` etc.) and should be
consulted when making changes, but not imported.

The tradeoff accepted is that typos in YAML field names inside those strings will not be caught at
compile time. This is acceptable given the alternative cost.
