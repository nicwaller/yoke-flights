# yoke-flights

This repo contains **Yoke Flights** — Go programs compiled to WebAssembly that act as Kubernetes resource renderers for the [Yoke](https://github.com/yokecd/yoke) package manager (a Helm alternative).

Each flight:
1. Reads configuration values from stdin as YAML
2. Renders Kubernetes resources using those values
3. Outputs resources as JSON to stdout

Flights are analogous to Helm charts, but written in typed Go instead of Go templates.

## Flights

- `argocd/` — deploys ArgoCD (v3.4.1)
- `forgejo/` — deploys Forgejo
- `onetimesecret/` — deploys OneTimeSecret

## Deploying to k3s

Each flight directory has a `Taskfile.yml` with standard tasks. Run from the flight directory (e.g. `cd argocd`):

```sh
task build   # compile to dist/<flight>.wasm
task diff    # preview changes against live cluster (requires build)
task deploy  # build and apply to cluster
task clean   # delete the release and namespace
```

Or from the repo root: `task -d argocd deploy`

Under the hood, `deploy` runs:
```sh
yoke takeoff --namespace <ns> <release-name> dist/<flight>.wasm < values.yaml
```

To destroy a release manually:
```sh
yoke mayday --namespace <ns> <release-name>
```
