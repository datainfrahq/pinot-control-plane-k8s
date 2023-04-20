<h2 align="center">
  <picture>
    <img alt="DataInfra Logo" src="https://raw.githubusercontent.com/datainfrahq/.github/main/images/logo.svg" width="500" height="100">
  </picture>
  <br>
  Control Plane For Apache Pinot On Kubernetes
  </br>
</h2>


<div align="center">

![Build Status](https://github.com/datainfrahq/pinot-control-plane-k8s/actions/workflows/makefile.yml/badge.svg) [![Slack](https://img.shields.io/badge/slack-brightgreen.svg?logo=slack&label=Community&style=flat&color=%2373DC8C&)](https://launchpass.com/datainfra-workspace)
![Docker pull](https://img.shields.io/docker/pulls/datainfrahq/pinot-control-plane.svg) 
[![Go Reference](https://pkg.go.dev/badge/github.com/datainfrahq/operator-runtime.svg)](https://pkg.go.dev/github.com/datainfrahq/pinot-control-plane-k8s)
![GitHub issues](https://img.shields.io/github/issues/datainfrahq/pinot-control-plane-k8s) [![Go Report Card](https://goreportcard.com/badge/github.com/datainfrahq/pinot-control-plane-k8s)](https://goreportcard.com/report/github.com/datainfrahq/pinot-control-plane-k8s)


</div>

Control Plane for deploying and managing heterogenous apache pinot kubernetes clusters and its operations including schema, table and tenants management. This control plane is based on [Dsoi-Spec](https://github.com/datainfrahq/dsoi-spec) and is built using [operator-runtime](https://github.com/datainfrahq/operator-runtime). This is a radical new approach that brings ease of use and decouples application and kubernetes in a way that it becomes easier for day 2 operations. The underlying controllers are built on conditions ie orthogonal concepts and not state machines.

### Getting Started 

```
export STORAGE_CLASS_NAME=civo-volume
make helm-install-pinot-control-plane
make helm-install-zk-operator
envsubst < examples/pinot-simple.yaml  | kubectl apply -f - -n pinot
```
### Getting Started With DeepStorage Minio

- An e2e getting started from kafka > pinot > minio s3.

#### Install Pinot, Zookeeper and Minio Operator
```
export STORAGE_CLASS_NAME=civo-volume
# Install Pinot Operator
make helm-install-pinot-control-plane
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

### Note
Apache®, [Apache Pinot](https://pinot.apache.org), Pinot® are either registered trademarks or trademarks of the Apache Software Foundation in the United States and/or other countries. This project, pinot-control-plane-k8s, is not an Apache Software Foundation project.
