# Kubernetes GitOps Cluster

A production-grade Kubernetes homelab running on three Raspberry Pi 4B nodes, managed fully via GitOps with Argo CD.

## Hardware

| Node | Hostname | IP | Role |
|------|----------|----|------|
| Raspberry Pi 4B (2GB) | k3s-master | 192.168.1.29 | Control Plane |
| Raspberry Pi 4B (2GB) | k3s-worker-01 | 192.168.1.31 | Worker |
| Raspberry Pi 4B (2GB) | k3s-worker-02 | 192.168.1.32 | Worker |

## Stack

| Tool | Purpose |
|------|---------|
| k3s | Lightweight Kubernetes distribution |
| Argo CD | GitOps continuous delivery |
| Helm | Package manager |
| Kustomize | Manifest customisation |
| Cilium | CNI, network policies, ingress |
| MetalLB | Bare-metal load balancer |
| cert-manager | Automatic TLS via Let's Encrypt |
| Kyverno | Policy enforcement |
| Prometheus | Metrics & alerting |
| Grafana | Dashboards & observability |
| GitHub Actions | CI - build & push images |
| Renovate Bot | Automated dependency updates |
| Golang | Custom operator |

## Repository Layout

```
.
├── .github/
│   └── workflows/
│       ├── ci.yaml                   # Build and push operator image on push to main
│       └── renovate.yml              # Scheduled Renovate dependency update bot
├── applications/                     # Argo CD Application manifests (App of Apps pattern)
│   ├── argocd/                       # Argo CD self-manages its own Application
│   ├── cert-manager/                 # TLS certificate management + ClusterIssuer
│   ├── cilium/                       # CNI, network policies, ingress controller
│   ├── custom-operator/              # Golang operator + ingress + network policies
│   ├── kyverno/                      # Policy engine
│   ├── metallb/                      # Bare-metal load balancer + IP pool
│   └── monitoring/                   # Prometheus, Grafana, alerts, dashboards
├── gitops-setup/                     # Bootstrap manifests applied once manually
│   ├── argocd-project.yaml           # AppProject scoping source repos and destinations
│   └── root-app.yaml                 # Root App of Apps - bootstraps everything else
├── infrastructure/
│   ├── base/                         # Base Kustomize manifests for the custom operator
│   └── overlays/production/          # Production overrides (replicas, resource limits, CRD)
├── operator/                         # Source code for the custom Golang operator
│   ├── cmd/main/main.go              # Entrypoint
│   ├── internal/
│   │   ├── api/v1alpha1/             # AppConfig CRD Go types and DeepCopy methods
│   │   └── controller/               # AppConfigReconciler and envtest suite
│   ├── Dockerfile                    # Multi-stage ARM64 build
│   ├── Makefile                      # Developer tasks: build, test, lint, docker-push
│   ├── go.mod
│   └── .golangci.yaml
├── policies/                         # Kyverno ClusterPolicy resources
│   └── tests/                        # Kyverno CLI tests and test pod fixtures
├── renovate.json                     # Renovate Bot configuration
└── scripts/
    └── argocd-setup.sh               # One-time bootstrap script
```

## Setting up the homelab

```bash
# 1. Install k3s on the master node
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable=traefik --disable=flannel --flannel-backend=none --disable-network-policy" sh -

# 2. Join workers - get the token from the master first
cat /var/lib/rancher/k3s/server/node-token
# Then on each worker:
curl -sfL https://get.k3s.io | K3S_URL=https://192.168.1.29:6443 K3S_TOKEN=<token> sh -

# 3. Install Argo CD and apply the root App of Apps
bash scripts/argocd-setup.sh
```

After step 3, Argo CD takes over and reconciles every application defined in `applications/`.

## GitOps Workflow

```
git push -> GitHub Actions builds image -> updates image tag in repo -> Argo CD detects diff -> rolls out new version
```

---

## Folder & Application Reference

### `.github/workflows/`

- **`renovate.yml`** - runs Renovate Bot on a daily cron. Renovate scans the repository for outdated Helm chart versions, container image tags, Go module dependencies, and GitHub Actions versions, then opens pull requests to update them. Configured by `renovate.json` at the repo root.
- **`ci.yaml`** *(to be added)* - builds the operator Docker image on every push to `main`, pushes it to GHCR, and patches the image tag in `infrastructure/overlays/production/kustomization.yaml` so Argo CD detects the change and rolls it out.

---

### `applications/`

Each subdirectory contains an Argo CD `Application` manifest. The root app points at this directory, so adding a folder here automatically registers a new application with Argo CD.

#### `applications/argocd/`
Argo CD manages itself via GitOps. Deploys the `argo-cd` Helm chart (v9.4.15). `server.service.type: LoadBalancer` gives Argo CD a stable IP from MetalLB. TLS is terminated at the Cilium ingress layer so the server runs plain HTTP internally (`--insecure`).

#### `applications/cert-manager/`
Deploys cert-manager (v1.20.0) for automatic TLS certificate issuance. `installCRDs: true` keeps CRDs version-controlled with the chart. Resource limits are deliberately low (32Mi request, 64Mi limit) for the Raspberry Pi memory budget. `cluster-issuer.yaml` defines a `ClusterIssuer` pointing at the Let's Encrypt production ACME endpoint using the HTTP-01 challenge via the Cilium ingress class.

#### `applications/cilium/`
Deploys Cilium (v1.15.3) as the CNI, replacing k3s default Flannel. `kubeProxyReplacement: true` means Cilium handles all packet routing via eBPF. `bpf.masquerade: true` handles outgoing pod traffic at the eBPF layer instead of iptables. Hubble is enabled for real-time network observability. The ingress controller runs in shared load-balancer mode (one MetalLB IP for all ingresses). ARM64 images are pinned explicitly for Raspberry Pi. `network-policy-global.yaml` denies all cross-namespace traffic by default with egress allowances only for the API server and CoreDNS.

#### `applications/custom-operator/`
Deploys the custom Golang operator. `application.yaml` points Argo CD at `infrastructure/overlays/production` for a Kustomize build. `example-appconfig.yaml` is a sample `AppConfig` CR. `ingress.yaml` exposes `hello.diogomota.com` with cert-manager TLS. `network-policy.yaml` has four Cilium policies: default-deny, allow Prometheus scraping on port 8080, allow egress to the API server, and allow DNS.

#### `applications/kyverno/`
Deploys Kyverno (v3.1.4) as the admission controller. Single replica to save memory (256Mi limit). Policies live in `policies/` and are applied as `ClusterPolicy` resources.

#### `applications/metallb/`
Deploys MetalLB (v0.14.4) for LoadBalancer-type services on bare metal. The speaker DaemonSet tolerates the master node NoSchedule taint so ARP announcements come from all nodes. `ip-pool.yaml` assigns IPs from `192.168.1.200-192.168.1.220` via Layer 2 mode.

#### `applications/monitoring/`
Deploys `kube-prometheus-stack` (v58.2.2) with Prometheus, Grafana, and Alertmanager. `ServerSideApply=true` is required for the large CRDs. `kubeEtcd`, `kubeControllerManager`, and `kubeScheduler` are disabled (not exposed by k3s). Prometheus has 7-day retention and a 10Gi PVC. Grafana pre-loads two community dashboards (gnetId 7249 and 1860). `operator-alerts.yaml` defines three PrometheusRules for the custom operator. `operator-dashboard.yaml` is a ConfigMap-based Grafana dashboard auto-loaded by the sidecar.

---

### `gitops-setup/`

Applied manually once during bootstrap. After this Argo CD manages everything.

- **`root-app.yaml`** - the root Argo CD `Application` pointing at `applications/`. Applying this one manifest bootstraps the entire cluster. Uses `prune: true` and `selfHeal: true`.
- **`argocd-project.yaml`** - an `AppProject` named `homelab` restricting which source repos and destination namespaces Argo CD may use. Includes a `syncWindows` deny rule blocking syncs to `kube-system` between 02:00-03:00 UTC.

---

### `infrastructure/`

Kustomize manifests for the custom operator.

- **`base/`** - Namespace, Deployment, ServiceAccount, ClusterRole, ClusterRoleBinding, Service, ServiceMonitor. Deployment uses distroless/nonroot and drops all Linux capabilities.
- **`overlays/production/`** - `crd.yaml` with OpenAPI validation schema, `replica-patch.yaml` pinning replicas to 1, `resource-patch.yaml` with tighter CPU/memory limits, and `kustomization.yaml` wiring it all together.

---

### `operator/`

Go source for the custom Kubernetes operator using controller-runtime.

- **`cmd/main/main.go`** - sets up the manager, registers the AppConfig scheme, wires the reconciler, starts health/metrics servers.
- **`internal/api/v1alpha1/`** - AppConfig type definitions, group/version registration, generated DeepCopy methods.
- **`internal/controller/appconfig_controller.go`** - reconciliation loop: fetch AppConfig, build desired Deployment, create or update it, sync status back. Owner references handle garbage collection on deletion.
- **`internal/controller/appconfig_controller_test.go`** - envtest integration tests covering Deployment creation, owner references, and spec drift.
- **`Dockerfile`** - multi-stage: compiles static ARM64 binary in golang:1.22-alpine, copies into distroless/static:nonroot.
- **`Makefile`** - `make build`, `make test`, `make lint`, `make docker-push`, `make install-crds`, `make run`.

---

### `policies/`

Kyverno `ClusterPolicy` resources, all in `Enforce` mode.

| Policy | Effect |
|--------|--------|
| `require-resource-limits.yaml` | Blocks Pods without explicit CPU and memory limits. Excludes kube-system and kyverno. |
| `disallow-privileged-containers.yaml` | Blocks Pods with `privileged: true`. Excludes kube-system. |
| `require-non-root.yaml` | Blocks Pods without `runAsNonRoot: true`. Excludes kube-system, kyverno, argocd. |
| `add-namespace-labels.yaml` | Mutates new Namespaces to add `managed-by: gitops` and `environment: production`. |

`tests/` contains Kyverno CLI test manifests runnable with `kyverno test policies/tests/`.

---

### `renovate.json`

Groups all Kubernetes infrastructure Helm chart updates into a single Monday/Thursday PR. Auto-merges patch-level Helm and GitHub Actions bumps. Groups Go module updates into a Monday PR. Uses a regexManager to track `ARGOCD_VERSION` in `scripts/argocd-setup.sh`. The `helm-values` file matcher detects chart version bumps inside Argo CD Application manifests.

---

### `scripts/`

- **`argocd-setup.sh`** - creates the argocd namespace, installs Argo CD, waits for the server deployment, applies the root app. Prints the initial admin password and a port-forward command.