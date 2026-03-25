# Installation

Complete guide to set up the Kubernetes homelab from bare metal to a fully GitOps-managed cluster.

## Prerequisites

- 3× Raspberry Pi 4B (2GB) with SD cards
- A running network with access to the internet
- A GitHub account with a fork or clone of this repository
- A Cloudflare account managing your domain, with an API token that has `Zone:DNS:Edit` permissions
- A domain (e.g. `diogomota.com`) with DNS A records pointing at the MetalLB IP pool range, or a local DNS / `/etc/hosts` override for the ingress hostnames (`argocd.diogomota.com`)

## Hardware Layout

| Node | Hostname | IP | Role |
|------|----------|----|------|
| Raspberry Pi 4B (2GB) | k3s-master | 192.168.1.29 | Control Plane |
| Raspberry Pi 4B (2GB) | k3s-worker-01 | 192.168.1.31 | Worker |
| Raspberry Pi 4B (2GB) | k3s-worker-02 | 192.168.1.32 | Worker |

---

## 1 — OS Installation (on each node)

Flash **Ubuntu Server 64-bit 22.04 LTS** onto each SD card using the official Raspberry Pi Imager tool. Boot each Pi, then find its IP:

```bash
hostname -I
```

Set the hostname on each node to match the table above:

```bash
sudo hostnamectl set-hostname k3s-master        # on 192.168.1.29
sudo hostnamectl set-hostname k3s-worker-01      # on 192.168.1.31
sudo hostnamectl set-hostname k3s-worker-02      # on 192.168.1.32
```

---

## 2 — System Configuration (on each node)

### 2.1 Install essential utilities

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y curl wget git vim jq openssh-server
sudo systemctl enable ssh
sudo systemctl start ssh
```

### 2.2 Enable cgroups

Append `cgroup_memory=1 cgroup_enable=memory` to the end of the existing line in `/boot/firmware/cmdline.txt`:

```bash
sudo vim /boot/firmware/cmdline.txt
```

### 2.3 Disable swap and reboot

```bash
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
sudo reboot
```

---

## 3 — K3s Installation

k3s is installed with Flannel, Traefik, servicelb, and kube-proxy **disabled** because Cilium replaces all of them.

### 3.1 Master node

```bash
export SETUP_NODEIP=192.168.1.29
export SETUP_CLUSTERTOKEN=<STRONG_TOKEN_HERE>

curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.33.3+k3s1" \
  INSTALL_K3S_EXEC="--node-ip $SETUP_NODEIP \
  --disable=servicelb,traefik \
  --flannel-backend=none \
  --disable-kube-proxy" \
  K3S_TOKEN=$SETUP_CLUSTERTOKEN \
  K3S_KUBECONFIG_MODE=644 sh -s -
```

Retrieve the node token for workers:

```bash
sudo cat /var/lib/rancher/k3s/server/node-token
```

### 3.2 Each worker node

```bash
export MASTER_IP=192.168.1.29
export NODE_IP=<THIS_WORKER_IP>       # 192.168.1.31 or 192.168.1.32
export K3S_TOKEN=<TOKEN_FROM_MASTER>

curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.33.3+k3s1" \
  K3S_URL="https://$MASTER_IP:6443" \
  K3S_TOKEN=$K3S_TOKEN \
  INSTALL_K3S_EXEC="--node-ip $NODE_IP" sh -
```

### 3.3 Verify (on master)

```bash
kubectl get nodes -o wide
```

All three nodes should show `Ready`.

---

## 4 — Create Required Secrets

This secret must exist before Argo CD syncs the applications.

### Cloudflare API token (for DNS-01 challenges)

The Cloudflare API token needs `Zone:DNS:Edit` permissions. cert-manager uses it to create TXT records for Let's Encrypt DNS-01 validation.

```bash
kubectl create namespace cert-manager

kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: cloudflare-api-token
  namespace: cert-manager
type: Opaque
stringData:
  api-token: <YOUR_CLOUDFLARE_API_TOKEN>
EOF
```

---

## 5 — Install Helm

```bash
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
helm version
```

---

## 6 — Install Argo CD and Bootstrap GitOps

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

### Change the Argo CD admin password

Generate a bcrypt hash for your new password, then patch the secret:

```bash
kubectl -n argocd patch secret argocd-secret -p \
  '{"stringData": {
    "admin.password": "<BCRYPT_HASH>",
    "admin.passwordMtime": "'$(date +%FT%T%Z)'"
  }}'
```

---

## 7 — Verify the Deployment

### 7.1 Monitor Argo CD sync

```bash
kubectl get applications -n argocd -w
```

Wait for all applications to show **Synced** and **Healthy**.

### 7.2 Access Argo CD

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

Open `https://localhost:8080`, log in with user `admin` and the password from step 6.

### 7.3 Check Cilium

```bash
kubectl -n kube-system exec ds/cilium -- cilium status --brief
```

### 7.4 Check MetalLB IP allocation

```bash
kubectl get svc -A | grep LoadBalancer
```

IPs should be assigned from the `192.168.1.200-192.168.1.220` pool.

### 7.5 Check certificates

```bash
kubectl get certificates -A
kubectl get clusterissuer
```

The `letsencrypt-prod` ClusterIssuer should show `Ready: True`. Certificates use DNS-01 validation via Cloudflare, so they work without exposing any ports to the internet.

### 7.6 Check Kyverno policies

```bash
kubectl get clusterpolicy
```

All four policies should be listed and active.

### 7.7 Check node-exporter

```bash
kubectl get pods -n monitoring -o wide
```

One pod should be running on each node. Verify metrics are reachable from your network:

```bash
curl http://192.168.1.29:9100/metrics | head
curl http://192.168.1.31:9100/metrics | head
curl http://192.168.1.32:9100/metrics | head
```

---

## 8 — Configure DNS

Point your ingress hostname at the MetalLB IP (or configure `/etc/hosts` for local access):

```
192.168.1.210   argocd.diogomota.com
```

---

## 9 — Remote Prometheus Configuration

Prometheus and Grafana run on a separate cluster on the local network. Add the following scrape targets to your remote Prometheus configuration so it scrapes node-exporter from each Pi:

```yaml
scrape_configs:
  - job_name: 'pi-cluster-node-exporter'
    static_configs:
      - targets:
          - '192.168.1.29:9100'
          - '192.168.1.31:9100'
          - '192.168.1.32:9100'
        labels:
          cluster: pi-homelab
```

---

## 10 — Accept Self-Signed Certificates (optional)

If using self-signed certificates instead of Let's Encrypt, you can add them to your system trust store to avoid browser warnings.

### Linux

```bash
kubectl get secret argocd-tls -n argocd -o jsonpath='{.data.tls\.crt}' | base64 -d > argocd.crt
sudo cp argocd.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

### Windows

1. Export the certificate as shown above
2. Double-click the `.crt` file
3. Click "Install Certificate" → "Local Machine" → "Trusted Root Certification Authorities"

---

## Memory Optimisation Notes

All resource limits have been tuned for 2GB Raspberry Pi 4B nodes. Key decisions:

- **Prometheus and Grafana offloaded** — only node-exporter runs on the Pi cluster (32Mi limit per node). The full monitoring stack runs on a separate local cluster with more resources.
- **Hubble disabled entirely** — no local Prometheus to consume the metrics.
- **Argo CD components reduced** — server and repo-server capped at 192Mi, controller at 384Mi, redis and notifications at 48Mi each.
- **Cilium agent capped at 256Mi**, operator at 128Mi.
- **cert-manager, Kyverno background controller, and MetalLB** all run under 96Mi limits.