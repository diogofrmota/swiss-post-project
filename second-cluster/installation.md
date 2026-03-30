# Monitoring Stack — Prometheus & Grafana on Minikube

Prometheus and Grafana run on a local minikube cluster on your laptop, scraping node-exporter directly from the Raspberry Pi nodes over the LAN.

## Prerequisites

- minikube installed on your laptop
- helm installed on your laptop
- Your laptop is on the same network as the Pi nodes (`192.168.1.x`)
- Node exporter running on all three Pi nodes (see main `installation.md` step 9)


## 1. Start Minikube

```bash
minikube start --cpus=2 --memory=2048
```

Verify it is running:

```bash
kubectl get nodes
# Should show a single minikube node as Ready
```


## 2. Add Helm Repositories

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
```


## 3. Create the Monitoring Namespace

```bash
kubectl create namespace monitoring
```

## 4. Install Prometheus

```bash
helm install prometheus prometheus-community/prometheus \
  --namespace monitoring \
  --values prometheus/values.yaml
```

Verify Prometheus is up:

```bash
kubectl get pods -n monitoring
# Wait for prometheus-server pod to be Running
```

Test that it can reach the Pi nodes (run from your laptop — minikube must be able to route to 192.168.1.x):

```bash
kubectl port-forward -n monitoring svc/prometheus-server 9090:80 &
open http://localhost:9090/targets
# All three Pi targets should show state=UP
```


## 5. Install Grafana

```bash
helm install grafana grafana/grafana \
  --namespace monitoring \
  --values grafana/values.yaml
```

Verify Grafana is up:

```bash
kubectl get pods -n monitoring
# Wait for grafana pod to be Running
```

## 6. Access the Dashboards

### Option A — minikube service URL (recommended)

```bash
# Open Prometheus
minikube service prometheus-server -n monitoring

# Open Grafana
minikube service grafana -n monitoring
```

### Option B — port-forward

```bash
# Prometheus
kubectl port-forward -n monitoring svc/prometheus-server 9090:80

# Grafana (in a separate terminal)
kubectl port-forward -n monitoring svc/grafana 3000:80
```

Then open `http://localhost:3000` in your browser.

### Grafana login

- Username: `admin`
- Password: `admin`

> Change the password on first login.


## 7. Verify Dashboards

Two dashboards are pre-installed under the **Pi Homelab** folder:

- **Node Exporter Full** (ID 1860) — detailed CPU, memory, disk, network per node
- **Node Overview** (ID 13978) — simple at-a-glance per-node summary

If the dashboards show "No data", check:

```bash
# Confirm Prometheus is scraping successfully
kubectl port-forward -n monitoring svc/prometheus-server 9090:80
# Then open http://localhost:9090/targets — all three Pi IPs should be UP
```

## Teardown

```bash
helm uninstall prometheus -n monitoring
helm uninstall grafana -n monitoring
kubectl delete namespace monitoring
minikube stop
```