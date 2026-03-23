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
| Golang | Custom operator |

## Repository Layout

```
.
├── .github/workflows/        # CI pipelines
├── apps/                     # Argo CD Application manifests (App of Apps)
│   ├── argocd/
│   ├── cilium/
│   ├── cert-manager/
│   ├── metallb/
│   ├── kyverno/
│   ├── monitoring/
│   └── custom-operator/
├── bootstrap/                # Root App of Apps + Argo CD install
├── infrastructure/
│   ├── base/                 # Base Kustomize manifests
│   └── overlays/production/  # Production overrides
├── policies/                 # Kyverno ClusterPolicy resources
├── operator/                 # Custom Golang operator
└── scripts/                  # Bootstrap & utility scripts
```

## Seting up the homelab

```bash
# 1. Install k3s (run on master node)

# 2. Join workers (run on each worker node)

# 3. Install Argo CD and apply root App of Apps
bash scripts/bootstrap-argocd.sh
```

After step 3, Argo CD takes over and reconciles every application defined in `apps/`.

## GitOps Workflow

```
git push → GitHub Actions builds image → updates image tag in repo → Argo CD detects diff → rolls out new version
```