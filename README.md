# Kubernetes Cluster using GitOps

A production-grade Kubernetes homelab running on three Raspberry Pi 4B nodes, managed fully via GitOps with Argo CD.

<p align="center">
  <img src="assets/cluster.jpeg" alt="Cluster" width="350">
</p>

## Hardware

| Node | Hostname | IP | Role |
|------|----------|----|------|
| Raspberry Pi 4B (2GB) | k3s-master | 192.168.1.29 | Control Plane |
| Raspberry Pi 4B (2GB) | k3s-worker-01 | 192.168.1.31 | Worker |
| Raspberry Pi 4B (2GB) | k3s-worker-02 | 192.168.1.32 | Worker |

## Dashboards

| Service | URL |
|---------|-----|
| Argo CD | https://argocd.diogomota.com |
| Grafana | https://grafana.diogomota.com |
| Prometheus | https://prometheus.diogomota.com |

Prometheus and Grafana run on a separate local cluster and scrape node-exporter from the Pi nodes over the local network.

All services are exposed via Cilium ingress with TLS certificates issued automatically by cert-manager (Let's Encrypt).

## Tech Stack

| Tool | Purpose |
|------|---------|
| k3s | Lightweight Kubernetes distribution |
| Argo CD | GitOps continuous delivery |
| Helm | Package manager |
| Kustomize | Manifest customisation |
| Cilium | CNI, network policies, ingress |
| cert-manager | Automatic TLS via Let's Encrypt |
| Kyverno | Policy enforcement |
| Node Exporter | Host-level metrics (scraped remotely) |
| Prometheus | Metrics & alerting (separate cluster) |
| Grafana | Dashboards & observability (separate cluster) |
| GitHub Actions | CI - build & push images |
| Renovate Bot | Automated dependency updates |
| Golang | Custom scraper |

## Installation

The file [installation.md](installation.md) has the detailed setup instructions covering OS preparation, k3s cluster bootstrap, Argo CD deployment and post-install verification.