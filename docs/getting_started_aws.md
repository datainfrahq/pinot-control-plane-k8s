#### Export your StorageClassName 
```
export STORAGE_CLASS_NAME=gp2
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
```
envsubst < examples/05-pinot-aws/pinot-aws.yaml  | kubectl apply -f - -n pinot
```
#### Deploy Kafka Cluster and Create Topics
```
# Add Kafka
helm repo add kafka https://charts.bitnami.com/bitnami
# Deploy kafka
helm install -n pinot kafka kafka/kafka --set replicas=1,zookeeper.image.tag=latest
# Create topics
kubectl -n pinot exec kafka-0 -- kafka-topics.sh --bootstrap-server kafka-0:9092 --topic flights-realtime --create --partitions 1 --replication-factor 1
kubectl -n pinot exec kafka-0 -- kafka-topics.sh --bootstrap-server kafka-0:9092 --topic flights-realtime-avro --create --partitions 1 --replication-factor 1
```

#### Access Pinot Console

```
kubectl port-forward svc/pinot-controller-controller-svc -n pinot 9000
```

### Once Pinot Cluster is up and running

#### Create Schema
```
kubectl apply -f examples/05-pinot-aws/pinotschema-aws.yaml -n pinot
```

#### Create Table
```
kubectl apply -f examples/05-pinot-aws/pinottable-aws.yaml -n pinot
```

#### Check All Custom Resources created by the control plane
```
kubectl get pinot -A
kubectl get pinotschema -A
kubectl get pinottable -A
```

#### Load Data Into Kafka
```
kubectl apply -f examples/03-pinot-minio/pinot-realtime-kafka.yaml
```

#### Port-forward and query on console
```
kubectl port-forward pinot-controller-controller-0 -n pinot 9000
```
