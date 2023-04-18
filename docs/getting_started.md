# Getting started with Kafka

- Deploy the following 
1. pinot operator 
2. zookeeper operator
3. zookeeper CR 
4. pinot CR.
```
export STORAGE_CLASS_NAME=civo-volume
make helm-install-pinot-operator
make helm-install-zk-operator
envsubst < examples/pinot-simple.yaml  | kubectl apply -f - -n pinot
```
- Deploy kafka
```
helm install -n pinot kafka kafka/kafka --set replicas=1,zookeeper.image.tag=latest
```

- Create Kafka Topics
```
kubectl -n pinot exec kafka-0 -- kafka-topics.sh --bootstrap-server kafka-0:9092 --topic flights-realtime --create --partitions 1 --replication-factor 1
kubectl -n pinot exec kafka-0 -- kafka-topics.sh --bootstrap-server kafka-0:9092 --topic flights-realtime-avro --create --partitions 1 --replication-factor 1
```

- Run Pinot Specs
```
kubectl apply -f examples/pinot-realtime-kafka.yaml 
```

# Getting started with Deepstorage for minio

- Deploy the following 
1. pinot operator 
2. zookeeper operator
3. zookeeper CR 
4. pinot CR.
```
export STORAGE_CLASS_NAME=civo-volume
make helm-install-pinot-operator
make helm-install-zk-operator
make helm-minio-install
envsubst < examples/pinot-s3.yaml  | kubectl apply -f - -n pinot
```

- Deploy kafka
```
helm install -n pinot kafka kafka/kafka --set replicas=1,zookeeper.image.tag=latest
```

- Create Kafka Topics
```
kubectl -n pinot exec kafka-0 -- kafka-topics.sh --bootstrap-server kafka-0:9092 --topic flights-realtime --create --partitions 1 --replication-factor 1
kubectl -n pinot exec kafka-0 -- kafka-topics.sh --bootstrap-server kafka-0:9092 --topic flights-realtime-avro --create --partitions 1 --replication-factor 1
```

- Run Pinot Specs
```
kubectl apply -f examples/pinot-realtime-kafka.yaml 
```
