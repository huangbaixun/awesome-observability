# Prometheus Trace Feature

ref:

- https://prometheus.io/docs/prometheus/latest/configuration/configuration/#tracing_config

# How to use

1. start whole stack

```
kind create cluster --config kind_config.yaml
```

2. expose prometheus UI

```
kubectl -n kube-system port-forward pod/prometheus-kind-control-plane 9090:9090
```

3. expose jaeger UI

```
kubectl -n kube-system port-forward pod/jaeger-kind-control-plane 16686:16686
```