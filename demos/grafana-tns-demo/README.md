# Build grafana tns demo in kind 

project: https://github.com/grafana/tns 

# Steps

Related modification is in: https://github.com/fatsheep9146/tns/tree/kind

## 1. install kind 

```
kind create cluster --image=kindest/node:v1.20.7 --config kind_config.yaml
```

## 2. install the tns 

```
rm -rf tanka
./install
```

## 3. remove the grafana configmap root_url config

```
kubectl edit configmap grafana-config
```

delete grafana pod to restart 

## 4. view the grafana dashboard

http://127.0.0.1:3000


