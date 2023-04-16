# pinot-operator
Apache Pinot On Kubernetes


```
- helm install zk pravega/zookeeper-operator --version=0.2.15 -n pinot 
- helm install zkcr pravega/zookeeper --version=0.2.15 --set replicas=1 --persistence.storageClassName=civo-volume
```