### Getting Started With Local Storage

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
```
envsubst < examples/pinot/pinot-basic.yaml  | kubectl apply -f - -n pinot
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

#### Create Schema
```
kubectl apply -f examples/schema/pinotschema-basic.yaml -n pinot
```

#### Create Table
```
kubectl apply -f examples/table/pinottable-basic.yaml -n pinot
```

#### Check All Custom Resources created by the control plane
```
kubectl get pinot -A
kubectl get pinotSchema -A
kubectl get pinottable -A
```

#### Load Data Into Kafka
```
kubectl apply -f examples/ingestion/pinot-realtime-kafka.yaml
```

#### Port-forward and query on console
```
kubectl port-forward pinot-controller-controller-0 -n pinot 9000
```
