### Getting Started With Auth Enabled Cluster

- Control Plane supports basic auth only.

#### Export your StorageClassName 
```
export STORAGE_CLASS_NAME=standard
```

#### Install Pinot Control Plane
```
make helm-install-pinot-control-plane
```

#### Install Zookeeper Opoerator and CR
```
make helm-install-zk-operator
```

### Install Pinot Cluster

```
envsubst < examples/04-pinot-auth/pinotauth-basic.yaml  | kubectl apply -f - -n pinot
```

### Create a K8 secret in the namespace where pinot cluster is deployed


- add secrets to file
```
cat << EOF > pinot-control-plane-secret
CONTROL_PLANE_USERNAME=controlplane
CONTROL_PLANE_PASSWORD=controlplane
EOF
```

- create secret

```
kubectl create secret generic pinot-control-plane-secret --from-env-file=pinot-control-plane-secret -n pinot
```

### create schema

```
kubectl apply -f examples/04-pinot-auth/pinotauth-schema.yaml -n pinot
```

### create table

```
kubectl apply -f examples/04-pinot-auth/pinotauth-table.yaml -n pinot
```
