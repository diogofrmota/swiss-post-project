# Show Project
https://github.com/diogofrmota/swiss-post-project

# Showcase Cluster

```bash
# Nodes
kubectl get nodes -o wide

# All pods
kubectl get pods -A
```

# Show ArgoCD CLI
```bash
argocd version

argocd repo list

argocd app list

argocd app get cert-manager
```

# Show ArgoCD UI
https://argocd.diogomota.com/

# Show Cilium

```bash
# Status
cilium status

# Gateway API resources
kubectl get gateway -A
kubectl get httproutes -A

# L2 announcements & IP pool
kubectl get ciliuml2announcementpolicies
kubectl get ciliumloadbalancerippool

# Code — show infra/cilium/values.yaml, gateway.yaml, ip-pool.yaml, announce.yaml
```

# Show cert-manager & TLS

```bash
# Certificates issued by Let's Encrypt
kubectl get certificates -A
kubectl get clusterissuer letsencrypt-prod

# Verify HTTPS is working
curl -v https://argocd.diogomota.com 2>&1 | grep "issuer"
```

# Show Kustomize

```bash
# Helm integration enabled via ArgoCD
kubectl -n argocd get cm argocd-cm -o yaml | grep kustomize

# Example: cert-manager kustomization with remote resource + patches
# Show apps/cert-manager/kustomization.yaml
# Example: Cilium kustomization with Helm chart + Gateway API CRDs
# Show infra/cilium/kustomization.yaml
```

# Show Node Exporter (bare-metal to Kubernetes bridge)

```bash
# Endpoints pointing to host IPs (not pods)
kubectl get endpoints node-exporter -n node-exporter

# Verify metrics from each node
curl -s http://192.168.1.29:9100/metrics | head -3
curl -s http://192.168.1.31:9100/metrics | head -3
curl -s http://192.168.1.32:9100/metrics | head -3
```

# Show Monitoring Stack (separate minikube cluster)

```bash
# Switch context to minikube
kubectl config use-context minikube
kubectl get pods -n monitoring
```

# Show Prometheus
http://localhost:9091/
# Show targets page — all three Pi nodes should be UP

# Show Grafana
http://localhost:3000/
# Dashboards: Pi Homelab > Node Exporter Full / Node Overview

```bash
# Switch back to homelab context
kubectl config use-context default
```

# Show Renovate Bot
https://github.com/diogofrmota/swiss-post-project/actions
# Show a Renovate PR if available:
https://github.com/diogofrmota/swiss-post-project/pulls?q=is%3Apr+author%3Aapp%2Frenovate

# Show Restart Script

```bash
# Ordered rollout script for the full cluster
cat scripts/restart_cluster.sh
# Restarts in dependency order: kube-system/Cilium → cert-manager → ArgoCD
```