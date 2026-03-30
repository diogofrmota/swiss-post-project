# Kubernetes Cluster using GitOps

A production-grade Kubernetes homelab running on three Raspberry Pi 4B nodes, managed fully via GitOps with Argo CD.

<p align="center">
  <img src="media/cluster.jpeg" alt="Cluster" width="350">
</p>

## Hardware

| Node | Hostname | IP | Role |
|------|----------|----|------|
| Raspberry Pi 4B (2GB) | k3s-master | 192.168.1.29 | Control Plane |
| Raspberry Pi 4B (2GB) | k3s-worker-01 | 192.168.1.31 | Worker |
| Raspberry Pi 4B (2GB) | k3s-worker-02 | 192.168.1.32 | Worker |

## Dashboards

| Service | URL | Deployed at |
|---------|-----|-------------|
| Argo CD | https://argocd.diogomota.com | Homelab Cluster |
| Grafana | https://grafana.diogomota.com | Virtual Machine |
| Prometheus | https://prometheus.diogomota.com | Virtual Machine |

Prometheus and Grafana run on a separate local cluster and scrape node-exporter from the Pi nodes over the local network.

All services are exposed via Cilium ingress with TLS certificates issued automatically by cert-manager (Let's Encrypt).

## Tech Stack

| Logo | Tool | Purpose | Status |
|------|------|---------|--------|
| <img src="media/k3s-logo.png" width="35" height="35"> | k3s | Lightweight Kubernetes distribution | Active |
| <img src="media/argo-logo.png" width="35" height="35"> | Argo CD | GitOps continuous delivery | Active |
| <img src="media/helm-logo.png" width="35" height="35"> | Helm | Package manager | Active |
| <img src="media/kustomize-logo.png" width="35" height="35"> | Kustomize | Manifest customisation | Active |
| <img src="media/cilium-logo.png" width="35" height="35"> | Cilium | CNI, network policies, ingress | Active |
| <img src="media/cert-logo.png" width="35" height="35"> | cert-manager | Automatic TLS via Let's Encrypt | Active |
| <img src="media/kyverno-logo.png" width="35" height="35"> | Kyverno | Policy enforcement | Inactive due to RAM usage |
| <img src="media/exporter-logo.png" width="35" height="35"> | Node Exporter | Host-level metrics (scraped remotely) | Active |
| <img src="media/prometheus-logo.png" width="35" height="35"> | Prometheus | Metrics & alerting | Active (separate cluster) |
| <img src="media/grafana-logo.png" width="35" height="35"> | Grafana | Dashboards & observability | Active (separate cluster) |
| <img src="media/gh-actions-logo.png" width="35" height="35"> | GitHub Actions | CI - build & push images | Inactive |
| <img src="media/renovate-logo.png" width="35" height="35"> | Renovate Bot | Automated dependency updates | Inactive |
| <img src="media/golang-logo.png" width="35" height="35"> | Golang | Custom scraper | Inactive |

## Installation

The file [installation.md](installation.md) has the detailed setup instructions covering OS preparation, k3s cluster bootstrap, Argo CD deployment and post-install verification.