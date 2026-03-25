#!/usr/bin/env bash
# Installs Argo CD and applies the root App of Apps manifest.

set -euo pipefail

ARGOCD_VERSION='v3.3.4'
REPO_URL='https://github.com/diogofrmota/swiss-post-project'

echo "Creating namespace argocd"
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -

echo "Installing Argo CD ${ARGOCD_VERSION}"
kubectl create -n argocd -f \
  "https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml"

echo "Waiting for Argo CD server to be ready..."
kubectl rollout status deployment/argocd-server -n argocd --timeout=120s

echo "Applying root App of Apps"
kubectl apply -f gitops-setup/root-app.yaml

echo "Argo CD setup complete!"
echo "Argo CD initial admin password:"
kubectl get secret argocd-initial-admin-secret -n argocd \
  -o jsonpath="{.data.password}" | base64 -d
echo ""
echo "Port-forward:  kubectl port-forward svc/argocd-server -n argocd 8080:443"
echo "URL:           https://localhost:8080"