<h2 align="center">
  <picture>
    <img alt="DataInfra Logo" src="https://raw.githubusercontent.com/datainfrahq/.github/main/images/logo.svg">
  </picture>
  <br>
  Kubernetes Operator For Apache Pinot
</h2>


<div align="center">

![Build Status](https://github.com/datainfrahq/pinot-operator/actions/workflows/makefile.yml/badge.svg) [![Slack](https://img.shields.io/badge/slack-brightgreen.svg?logo=slack&label=Community&style=flat&color=%2373DC8C&)](https://launchpass.com/datainfra-workspace)
![Docker pull](https://img.shields.io/docker/pulls/datainfrahq/pinot-operator.svg) 
[![Go Reference](https://pkg.go.dev/badge/github.com/datainfrahq/operator-runtime.svg)](https://pkg.go.dev/github.com/datainfrahq/pinot-operator)
![GitHub issues](https://img.shields.io/github/issues/datainfrahq/pinot-operator) [![Go Report Card](https://goreportcard.com/badge/github.com/datainfrahq/pinot-operator)](https://goreportcard.com/report/github.com/datainfrahq/pinot-operator)


</div>

Pinot Kubernetes Operator for deploying and managing heterogenous apache pinot kubernetes clusters. This operator is based on [Dsoi-Spec](https://github.com/datainfrahq/dsoi-spec) and is built using [operator-runtime](https://github.com/datainfrahq/operator-runtime). This is a radical new approach that brings ease of use and decouples application and kubernetes in a way that it becomes easier for day 2 operations. The goal of this project is to build a control plane for managing apache pinot clusters. The underlying controllers are built on conditions ie orthogonal concepts and not state machines.

### Getting Started With Kustomize

```
make deploy
kubectl create ns pinot
helm install zk-operator pravega/zookeeper-operator --version=0.2.15 -n pinot 
helm install zkcr pravega/zookeeper --version=0.2.15 --set replicas=1 --persistence.storageClassName= -n pinot
kubectl apply -f examples/pinot-simple.yaml -n pinot
```
### Note
Apache®, Apache Pinot, Pinot® are either registered trademarks or trademarks of the Apache Software Foundation in the United States and/or other countries. This project, pinot-operator, is not an Apache Software Foundation project.
