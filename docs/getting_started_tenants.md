### Pinot Tenants ( Experimental )

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

#### Install Pinot Cluster

- In this example we have set ```cluster.tenant.isolation.enable=false```
- The pinot brokers/server will join the cluster untagged as mentioned [here](https://docs.pinot.apache.org/basics/getting-started/frequent-questions/operations-faq#how-can-i-make-brokers-servers-join-the-cluster-without-the-defaulttenant-tag)

```
envsubst < examples/02-pinot-tenant/pinot-tenant.yaml  | kubectl apply -f - -n pinot 
```

### Create Broker Tenant

- Create broker tenant by creating a tenant CR.

```
kubectl apply -f  02-pinot-tenant/pinottenant-broker.yaml -n pinot 
```
