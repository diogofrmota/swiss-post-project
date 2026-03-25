# Installation

Complete guide to set up the Kubernetes homelab from bare metal to a fully GitOps-managed cluster.

## Prerequisites

- 3× Raspberry Pi 4B (2GB) with SD cards
- A running network with access to the internet
- A Cloudflare account managing your domain, with an API token that has `Zone:DNS:Edit` permissions
- A local DNS at `/etc/hosts` to override for the ingress hostnames (`argocd.diogomota.com`)

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
sudo systemctl status ssh
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

All three nodes will show `NotReady` — this is expected because there is no CNI yet. Cilium is installed in the next step.

> **Important:** `kubectl` and `helm` commands must be run from the **master node** (`k3s-master`). The k3s API server only runs on the master, and the kubeconfig is automatically created at `/etc/rancher/k3s/k3s.yaml`. Running `kubectl` on a worker node without configuration will fail with `connection refused` on `localhost:8080`.

### 3.4 Configure kubectl on worker nodes or a laptop (optional)

If you want to run `kubectl` from a worker node or your local machine instead of SSH-ing into the master, copy the kubeconfig and update the server address:

```bash
# On the master — print the kubeconfig
sudo cat /etc/rancher/k3s/k3s.yaml

# On the worker or laptop — save it and fix the server address
mkdir -p ~/.kube
# Paste the kubeconfig content into ~/.kube/config, then replace the
# loopback address with the master's IP:
sed -i 's|server: https://127.0.0.1:6443|server: https://192.168.1.29:6443|' ~/.kube/config
```

Verify it works:

```bash
kubectl get nodes -o wide
```

This is optional — all remaining steps in this guide assume you are running commands on the master node.

---

## 4 — Install Helm

Helm is needed to bootstrap Cilium before Argo CD is available.

```bash
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
helm version
```

---

## 5 — Bootstrap Cilium (CNI)

Since k3s was installed without Flannel and kube-proxy, the nodes will stay `NotReady` until a CNI is installed. Cilium must be bootstrapped manually via Helm before Argo CD can start — Argo CD pods themselves need a working network.

Once Argo CD is running, its Cilium Application (sync-wave 0) will adopt and manage this Helm release going forward.

### 5.1 Install Cilium

The values here must match `applications/cilium/application.yaml` so Argo CD can adopt the release cleanly.

```bash
helm repo add cilium https://helm.cilium.io/
helm repo update

helm install cilium cilium/cilium --version 1.19.2 \
  --namespace kube-system \
  --set kubeProxyReplacement=true \
  --set k8sServiceHost=192.168.1.29 \
  --set k8sServicePort=6443 \
  --set ipam.operator.clusterPoolIPv4PodCIDRList="10.42.0.0/16" \
  --set bpf.masquerade=true \
  --set hubble.enabled=false \
  --set ingressController.enabled=true \
  --set ingressController.default=true \
  --set ingressController.loadbalancerMode=shared \
  --set operator.replicas=1 \
  --set resources.requests.cpu=50m \
  --set resources.requests.memory=100Mi \
  --set resources.limits.cpu=300m \
  --set resources.limits.memory=256Mi \
  --set operator.resources.requests.cpu=20m \
  --set operator.resources.requests.memory=32Mi \
  --set operator.resources.limits.cpu=100m \
  --set operator.resources.limits.memory=128Mi
```

### 5.2 Wait for Cilium and nodes to become Ready

```bash
# Watch Cilium pods come up
kubectl get pods -n kube-system -l app.kubernetes.io/part-of=cilium -w

# Verify Cilium is healthy
kubectl -n kube-system exec ds/cilium -- cilium status --brief

# All nodes should now show Ready
kubectl get nodes -o wide
```

All three nodes should transition from `NotReady` to `Ready` once the Cilium agent is running on each.

---

## 6 — Create Required Secrets

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
  api-token: <CLOUDFLARE_API_TOKEN>
EOF
```

---

## 7 — Install Argo CD and Bootstrap GitOps

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

## 8 — Verify the Deployment

### 8.1 Monitor Argo CD sync

```bash
kubectl get applications -n argocd -w
```

Wait for all applications to show **Synced** and **Healthy**.

### 8.2 Access Argo CD

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

Open `https://localhost:8080`, log in with user `admin` and the password from step 7.

### 8.3 Check Cilium

```bash
kubectl -n kube-system exec ds/cilium -- cilium status --brief
```

### 8.4 Check Cilium Loadbalancer IP allocation

```bash
kubectl get svc -A | grep LoadBalancer
```

IPs should be assigned from the `?????` pool.

### 8.5 Check certificates

```bash
kubectl get certificates -A
kubectl get clusterissuer
```

The `letsencrypt-prod` ClusterIssuer should show `Ready: True`. Certificates use DNS-01 validation via Cloudflare, so they work without exposing any ports to the internet.

### 8.6 Check Kyverno policies

```bash
kubectl get clusterpolicy
```

All four policies should be listed and active.

### 8.7 Check node-exporter

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

## 9 — Configure DNS

Point your ingress hostname at the Cilium loadbalancer IP (or configure `/etc/hosts` for local access):

```
192.168.1.210   argocd.diogomota.com
```

---

## 10 — Remote Prometheus Configuration

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

## 11 — Kyverno Policies (Optional)
Kyverno is disabled in this setup to conserve RAM on the Raspberry Pi nodes. However, the policy configuration files are kept in the repository (policies/) for reference. If you have more memory available, you can enable Kyverno by uncommenting it in applications/kustomization.yaml and updating the sync waves.

## 12 — Accept Self-Signed Certificates (optional)

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
- **cert-manager** runs under 96Mi limits.