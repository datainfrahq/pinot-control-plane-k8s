### Getting Started 

```
export STORAGE_CLASS_NAME=civo-volume
make helm-install-pinot-operator
make helm-install-zk-operator
envsubst < examples/pinot-simple.yaml  | kubectl apply -f - -n pinot
```

### Getting Started With DeepStorage Minio

- An e2e getting started from kafka > pinot > minio s3.

#### Install Pinot, Zookeeper and Minio Operator
```
export STORAGE_CLASS_NAME=civo-volume
# Install Pinot Operator
make helm-install-pinot-operator
# Install Zookeeper Operator and ZK Custom Resource
make helm-install-zk-operator
# Install Minio Operator and minion Custom Resource
make helm-install-minio-operator
```

#### Deploy Pinot Cluster
```
envsubst < examples/pinot-s3.yaml  | kubectl apply -f - -n pinot
```
- Once all pods are up and running, get Pinot UI on ```localhost:9000```
```
kubectl port-forward pinot-controller-controller-0 -n pinot 9000
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

#### Ingest Data to kafka
```
# Load Data and Create pinot schema and table
kubectl apply -f examples/pinot/pinot-realtime-kafka.yaml -n pinot
```

#### Check for segments in deepstorage using minio client
```
# Use minio mc client (https://github.com/minio/mc) to check segments in minio
kubectl port-forward svc/myminio-hl -n pinot 9000
mc alias set pinot http://localhost:9000 minio minio123 
mc ls pinot  --recursive
```

#### Clean Environment
```
make clean
```
