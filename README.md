# Kubernetes Cluster using GitOps

A production-grade Kubernetes homelab running on three Raspberry Pi 4B nodes, managed fully via GitOps with Argo CD.

## Hardware

| Node | Hostname | IP | Role |
|------|----------|----|------|
| Raspberry Pi 4B (2GB) | k3s-master | 192.168.1.29 | Control Plane |
| Raspberry Pi 4B (2GB) | k3s-worker-01 | 192.168.1.31 | Worker |
| Raspberry Pi 4B (2GB) | k3s-worker-02 | 192.168.1.32 | Worker |

## Tech Stack

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
| Golang | Custom scraper |

## What is Inside each Folder

### `.github/workflows/`

- **`renovate.yml`** - runs Renovate Bot on a daily cron. Renovate scans the repository for outdated Helm chart versions, container image tags, Go module dependencies, and GitHub Actions versions, then opens pull requests to update them. Configured at `renovate.json`.

---

### `applications/`

Each subdirectory contains an Argo CD `Application` manifest. The root app points at this directory, so adding a folder here automatically registers a new application with Argo CD.

#### `applications/argocd/`
Argo CD manages itself via GitOps. Deploys the `argo-cd` using Helm chart.

#### `applications/cert-manager/`
Deploys cert-manager for automatic TLS certificate issuance.

#### `applications/cilium/`
Deploys Cilium as the CNI, replacing k3s default Flannel. Cilium handles all packet routing via eBPF and handles outgoing pod traffic at the eBPF layer instead of iptables. The ingress controller runs in shared load-balancer mode (one MetalLB IP for all ingresses).

#### `applications/golang-scraper/`
Deploys the custom Golang scraper.

#### `applications/kyverno/`
Deploys Kyverno as the admission controller. Policies live in `policies/` and are applied as `ClusterPolicy` resources.

#### `applications/metallb/`
Deploys MetalLB for LoadBalancer-type services on bare metal. `ip-pool.yaml` assigns IPs from `192.168.1.200-192.168.1.220` via Layer 2 mode.

#### `applications/monitoring/`
Deploys `kube-prometheus-stack` with Prometheus and Grafana. `operator-alerts.yaml` defines three PrometheusRules for the custom operator. `operator-dashboard.yaml` is a ConfigMap-based Grafana dashboard auto-loaded by the sidecar.

---

### `gitops-setup/`

Applied manually once during setup. After this Argo CD manages everything.

- **`root-app.yaml`** - the root Argo CD `Application` pointing at `applications/`.

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

Groups all Kubernetes infrastructure Helm chart updates into a single daily PR. Auto-merges patch-level Helm and GitHub Actions bumps. Groups Go module updates into a Monday PR. Uses a regexManager to track `ARGOCD_VERSION` in `scripts/argocd-setup.sh`. The `helm-values` file matcher detects chart version bumps inside Argo CD Application manifests.

---

### `scripts/`

- **`argocd-setup.sh`** - creates the argocd namespace, installs Argo CD, waits for the server deployment, applies the root app. Prints the initial admin password and a port-forward command.