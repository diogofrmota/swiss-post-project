# Installation

Step-by-step guide to bootstrap GitOps on an existing k3s cluster using Argo CD.

## Prerequisites

- A running k3s cluster with Flannel, Traefik, servicelb, and kube-proxy disabled (Cilium replaces all of these).
- `kubectl` configured and pointing at the cluster.
- A GitHub account with a fork or clone of this repository.
- A domain (e.g. `diogomota.com`) with DNS A records pointing at the MetalLB IP pool range, or a local DNS / `/etc/hosts` override for the ingress hostnames (`argocd.diogomota.com`, `grafana.diogomota.com`, `prometheus.diogomota.com`, `hubble.diogomota.com`).

## 1 — Create Required Secrets

These secrets must exist before Argo CD syncs the applications.

### Grafana admin password

```bash
kubectl create namespace monitoring

kubectl create secret generic grafana-admin-secret \
  -n monitoring \
  --from-literal=admin-password='<YOUR_GRAFANA_PASSWORD>'
```

## 2 — Install Argo CD and Bootstrap GitOps

The `scripts/argocd-setup.sh` script handles everything:

```bash
chmod +x scripts/argocd-setup.sh
./scripts/argocd-setup.sh
```

What the script does:

1. Creates the `argocd` namespace.
2. Installs the Argo CD manifests for the version pinned in the script.
3. Waits for the `argocd-server` deployment to become ready.
4. Applies `gitops-setup/root-app.yaml` — the App of Apps that points at `applications/`.
5. Prints the initial admin password and a port-forward command.

After the root app syncs, Argo CD will automatically deploy every application defined under `applications/` in dependency order (controlled by `argocd.argoproj.io/sync-wave` annotations).

## 3 — Verify the Deployment

### 3.1 Access Argo CD

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

Open https://localhost:8080, log in with user `admin` and the password printed by the setup script. All applications should eventually show **Synced** and **Healthy**.

### 3.2 Check Cilium

```bash
kubectl -n kube-system exec ds/cilium -- cilium status --brief
```

### 3.3 Check MetalLB IP allocation

```bash
kubectl get svc -A | grep LoadBalancer
```

IPs should be assigned from the `192.168.1.200-192.168.1.220` pool.

### 3.4 Check certificates

```bash
kubectl get certificates -A
```

All certificates should show `Ready: True` once Let's Encrypt issues them. If using local DNS only (no public domain), the HTTP-01 challenge will fail — switch to a self-signed `ClusterIssuer` or use DNS-01 for internal setups.

### 3.5 Check Kyverno policies

```bash
kubectl get clusterpolicy
```

All four policies should be listed and active.

### 3.6 Access Grafana

Navigate to https://grafana.diogomota.com (or port-forward `svc/monitoring-grafana -n monitoring 3000:80`) and log in with user `admin` and the password you set in step 1. The Kubernetes cluster and node-exporter dashboards are pre-provisioned.