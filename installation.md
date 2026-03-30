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

## 1. OS Installation (on each node)

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

## 2. System Configuration (on each node)

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

## 3. Install k3s on the Master Node

```bash
# On k3s-master (192.168.1.29)
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.31.4+k3s1" INSTALL_K3S_EXEC="server \
  --flannel-backend=none \
  --disable-network-policy \
  --disable-kube-proxy \
  --disable=traefik \
  --disable=servicelb \
  --write-kubeconfig-mode 644" sh -

# Get the node token (needed to join workers)
sudo cat /var/lib/rancher/k3s/server/node-token
```

Flannel, kube-proxy, and the built-in service LB are disabled because Cilium replaces all three.

## 4. Join Worker Nodes

```bash
# On k3s-worker-01 (192.168.1.31) and k3s-worker-02 (192.168.1.32)
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.31.4+k3s1" \
  K3S_URL=https://192.168.1.29:6443 \
  K3S_TOKEN=<token-from-master> sh -
```

Verify from the master:

```bash
sudo kubectl get nodes
# All three nodes should appear (status may be NotReady until Cilium is installed)
```

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

## 5. Install Cilium CLI

The Cilium CLI release assets use `arm64` (not `aarch64` from `uname -m`), so the architecture must be hardcoded:

```bash
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
curl -L --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-arm64.tar.gz
sudo tar -C /usr/local/bin -xzf cilium-linux-arm64.tar.gz
rm cilium-linux-arm64.tar.gz
cilium version
```

---

## 6. Bootstrap Cilium (CNI)

> **This step must be done before deploying Argo CD.** Pods cannot start without a CNI, and Argo CD is responsible for managing Cilium going forward — but it needs a working network to come up in the first place.

First, export the kubeconfig so both `kubectl` and the Cilium CLI can reach the cluster. Add it to your shell profile to persist across sessions:

```bash
echo 'export KUBECONFIG=/etc/rancher/k3s/k3s.yaml' >> ~/.bashrc
source ~/.bashrc
```

Verify the nodes are visible:

```bash
kubectl get nodes
```

Then install Cilium manually with the same settings as in `infra/cilium/values.yaml`:

```bash
cilium install --version 1.16.3 \
  --set cgroup.autoMount.enabled=false \
  --set cgroup.hostRoot=/sys/fs/cgroup \
  --set k8sServiceHost=192.168.1.29 \
  --set k8sServicePort=6443 \
  --set kubeProxyReplacement=true \
  --set operator.replicas=1 \
  --set ipam.mode=kubernetes \
  --set l2announcements.enabled=true \
  --set gatewayAPI.enabled=true \
  --set socketLB.hostNamespaceOnly=true
```

Wait for Cilium to be fully ready before continuing:

```bash
cilium status --wait
```

Once Cilium is up, Argo CD will take over managing it via the `infra/cilium` app in Git.

---

## 7. Deploy Argo CD

Use `--server-side` to avoid annotation size errors with large CRDs, and `--force-conflicts` to resolve any ownership conflicts from a previous apply:

```bash
kubectl apply -k apps/argocd/ --server-side --force-conflicts
```

Wait for Argo CD to be ready:

```bash
kubectl -n argocd rollout status deployment argocd-server
```

Get the initial admin password:

```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d
echo
```

---

## 8. Create Required Secrets

These secrets are not stored in Git for security reasons. Create them manually before syncing the relevant applications.

### Cloudflare API Token (for cert-manager DNS01 challenges)

```bash
kubectl create namespace cert-manager --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic cloudflare-api-token \
  --namespace cert-manager \
  --from-literal=api-token=<YOUR_CLOUDFLARE_API_TOKEN>
```

To create the token in Cloudflare: go to My Profile > API Tokens > Create Token > use the "Edit zone DNS" template, scope it to your zone (`diogomota.com`).

---

## 9. Sync Applications via Argo CD

Once Argo CD is running, it will automatically detect the ApplicationSets and begin syncing:

- `infra` project: Cilium (CNI, Gateway API, L2 announcements, IP pool)
- `apps` project: cert-manager, node-exporter, and argocd itself

Monitor sync status:

```bash
# Using the CLI
kubectl -n argocd get applications

# Or log in to the Argo CD UI (once DNS is configured — see next section)
```

---

## 10. Local DNS Configuration

The shared Cilium Gateway gets a LoadBalancer IP from the `CiliumLoadBalancerIPPool`. To access services like `argocd.diogomota.com` from your laptop (which is on the same network), you need to resolve that hostname to the gateway's LB IP.

### Find the Gateway's LoadBalancer IP

```bash
kubectl get gateway shared-gateway -n default -o jsonpath='{.status.addresses[0].value}'
```

This should return an IP from your pool (e.g. `192.168.1.200` if you've updated the pool to use your local subnet).

### Option A: /etc/hosts (quickest, per-machine)

Add entries to `/etc/hosts` on your laptop:

```bash
# Linux / macOS
sudo nano /etc/hosts

# Add a line (replace with your actual LB IP):
192.168.1.200   argocd.diogomota.com
```

On Windows, edit `C:\Windows\System32\drivers\etc\hosts` as Administrator.

### Option B: Local DNS server (recommended for multiple devices)

If your router supports custom DNS entries (e.g. Pi-hole, pfSense, OPNsense, Unbound), add a DNS override:

```
argocd.diogomota.com  →  192.168.1.200
```

This way every device on the network can resolve the hostname without per-machine configuration.

### Option C: Wildcard DNS (best for many services)

If your local DNS supports wildcards, add a single record:

```
*.diogomota.com  →  192.168.1.200
```

This covers `argocd.diogomota.com`, any future services, and avoids updating DNS for each new HTTPRoute.

---

## 11. Post-Install Verification

```bash
# Check all nodes are Ready
kubectl get nodes -o wide

# Check Cilium status
cilium status

# Check that the Gateway has an external IP
kubectl get gateway -A

# Check cert-manager is issuing certificates
kubectl get certificates -A
kubectl get clusterissuer letsencrypt-prod

# Check Argo CD applications are synced
kubectl -n argocd get applications

# Test access from your laptop (after DNS is configured)
curl -k https://argocd.diogomota.com
```

---

## Troubleshooting

**Nodes stuck in NotReady**: Cilium may not be installed yet. Check `kubectl get pods -n kube-system` for Cilium agent pods.

**Pods stuck in ContainerCreating**: No CNI is installed. Complete step 6 (Bootstrap Cilium) before deploying anything else.

**`kubectl` or `cilium` CLI says cluster unreachable**: The kubeconfig is not set. Run `export KUBECONFIG=/etc/rancher/k3s/k3s.yaml` or add it to `~/.bashrc`.

**Certificate not issuing**: Ensure the `cloudflare-api-token` secret exists in the `cert-manager` namespace and that the token has DNS edit permissions for your zone.

**Cannot reach the LB IP from your laptop**: Make sure the IP pool uses addresses in the same subnet as your nodes (`192.168.1.x`), not a different subnet like `192.168.200.x`. L2 announcements only work within the same broadcast domain.

**Argo CD OOM crashes**: If you're using the HA manifest on low-memory nodes, switch to the non-HA manifest in `apps/argocd/kustomization.yaml`.

**`kubectl apply` annotation too large error**: Use `--server-side --force-conflicts` instead of a plain `kubectl apply`.