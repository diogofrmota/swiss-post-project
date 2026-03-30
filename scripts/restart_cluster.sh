#!/bin/bash
# Homelab Restart Script — swiss-post-project (k3s + Cilium + ArgoCD + cert-manager)
set -e

echo "Starting homelab restart..."

# Step 1 — kube-system core components
echo ""
echo "Restarting kube-system (2/4)"
kubectl -n kube-system rollout restart deployment coredns
kubectl -n kube-system rollout restart deployment local-path-provisioner
kubectl -n kube-system rollout restart deployment metrics-server
kubectl -n kube-system rollout restart deployment cilium-operator
echo "  Waiting for kube-system..."
kubectl -n kube-system rollout status deployment coredns --timeout=120s
kubectl -n kube-system rollout status deployment cilium-operator --timeout=120s
sleep 10

# Step 2 — cert-manager
echo ""
echo "Restarting cert-manager (3/4)"
kubectl -n cert-manager rollout restart deployment cert-manager
kubectl -n cert-manager rollout restart deployment cert-manager-cainjector
kubectl -n cert-manager rollout restart deployment cert-manager-webhook
echo "  Waiting for cert-manager..."
kubectl -n cert-manager rollout status deployment cert-manager --timeout=120s
sleep 10

# Step 3 — ArgoCD
echo ""
echo "Restarting argocd (4/4)"
kubectl -n argocd rollout restart deployment argocd-applicationset-controller
kubectl -n argocd rollout restart deployment argocd-dex-server
kubectl -n argocd rollout restart deployment argocd-notifications-controller
kubectl -n argocd rollout restart deployment argocd-redis
kubectl -n argocd rollout restart deployment argocd-repo-server
kubectl -n argocd rollout restart deployment argocd-server
kubectl -n argocd rollout restart statefulset argocd-application-controller
echo "  Waiting for argocd..."
kubectl -n argocd rollout status deployment argocd-server --timeout=180s
sleep 10

echo ""
echo "Waiting for all deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment --all -n kube-system 2>/dev/null || true
kubectl wait --for=condition=available --timeout=300s deployment --all -n cert-manager 2>/dev/null || true
kubectl wait --for=condition=available --timeout=300s deployment --all -n argocd 2>/dev/null || true

echo ""
echo "=== Final pod status ==="
kubectl get pods -A

echo ""
echo "Homelab restart completed!"