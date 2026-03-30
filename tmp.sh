helm upgrade prometheus prometheus-community/prometheus \
  --namespace monitoring \
  --values prometheus/values.yaml