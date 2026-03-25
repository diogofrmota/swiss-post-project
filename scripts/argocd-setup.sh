#!/usr/bin/env bash
# Installs Argo CD via Helm and applies the root App of Apps manifest.
# Using Helm here ensures the bootstrap matches the Argo CD Application
# definition in applications/argocd/application.yaml, so Argo CD can
# adopt and self-manage the release without creating duplicate resources.

set -euo pipefail

ARGOCD_CHART_VERSION='9.4.15'

echo "Creating namespace argocd"
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -

echo "Adding Argo Helm repo"
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

echo "Installing Argo CD ${ARGOCD_CHART_VERSION} via Helm"
helm install argocd argo/argo-cd --version "${ARGOCD_CHART_VERSION}" \
  --namespace argocd \
  --set global.domain=argocd.diogomota.com \
  --set server.service.type=ClusterIP \
  --set server.ingress.enabled=true \
  --set server.ingress.ingressClassName=cilium \
  --set 'server.ingress.annotations.cert-manager\.io/cluster-issuer=letsencrypt-prod' \
  --set server.ingress.tls=true \
  --set configs.params."server\.insecure"=true \
  --set server.resources.requests.cpu=25m \
  --set server.resources.requests.memory=64Mi \
  --set server.resources.limits.cpu=150m \
  --set server.resources.limits.memory=192Mi \
  --set controller.resources.requests.cpu=50m \
  --set controller.resources.requests.memory=128Mi \
  --set controller.resources.limits.cpu=300m \
  --set controller.resources.limits.memory=384Mi \
  --set repoServer.resources.requests.cpu=25m \
  --set repoServer.resources.requests.memory=64Mi \
  --set repoServer.resources.limits.cpu=150m \
  --set repoServer.resources.limits.memory=192Mi \
  --set redis.resources.requests.cpu=10m \
  --set redis.resources.requests.memory=16Mi \
  --set redis.resources.limits.cpu=50m \
  --set redis.resources.limits.memory=48Mi \
  --set notifications.resources.requests.cpu=5m \
  --set notifications.resources.requests.memory=16Mi \
  --set notifications.resources.limits.cpu=25m \
  --set notifications.resources.limits.memory=48Mi \
  --set applicationSet.resources.requests.cpu=5m \
  --set applicationSet.resources.requests.memory=32Mi \
  --set applicationSet.resources.limits.cpu=25m \
  --set applicationSet.resources.limits.memory=64Mi

echo "Waiting for Argo CD server to be ready..."
kubectl rollout status deployment/argocd-server -n argocd --timeout=180s

echo "Applying root App of Apps"
kubectl apply -f gitops-setup/root-app.yaml

echo ""
echo "Argo CD setup complete!"
echo "Argo CD initial admin password:"
kubectl get secret argocd-initial-admin-secret -n argocd \
  -o jsonpath="{.data.password}" | base64 -d
echo ""
echo "Port-forward:  kubectl port-forward svc/argocd-server -n argocd 8080:443"
echo "URL:           https://localhost:8080"