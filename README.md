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

Based on Kubernetes operators, this control Plane for apche pinot is responsible for deploying and managing heterogenous apache pinot kubernetes clusters and its operations including schema, table and tenants management. This control plane is based on [Dsoi-Spec](https://github.com/datainfrahq/dsoi-spec) and is built using [operator-runtime](https://github.com/datainfrahq/operator-runtime). This is a radical new approach that brings ease of use and decouples application and kubernetes in a way that it becomes easier for day 2 operations. The underlying controllers are built on conditions ie orthogonal concepts and not state machines.

## :rocket: Features

- Installation of heterogeneous pinot clusters.
- Rolling Upgrades and Ordered Deployment
- Seperation of pinot specific configurations with k8s configurations.
- Table Management
- Schema Mangement

## :books: Documentation

- [Getting Started With Local Storage](./docs/getting_started_local.md)
- [Getting Started With Deep Storage](./docs/getting_started_deepstorage.md)

## :dart: Motivation

We believe that Kubernetes can serve as a control plane for any application, including those with data and stateful sets. While Helm charts are useful for configuration management, they don't maintain the state of the application. That's why we've built a control plane based on kubernetes operator pattern that acts as a bridge between your application's requirements and Kubernetes infrastructure. 

Pinot control plane for k8s is specifically designed to improve the user experience of running Apache Pinot clusters. As a distributed database, Pinot can be challenging to run on Kubernetes without the right tools. Our project is based on the [DSOI Spec](https://github.com/datainfrahq/dsoi-spec) and built using the [Operator-Runtime](https://github.com/datainfrahq/operator-runtime) library to provide a more user-friendly and Kubernetes-friendly experience.



## :stethoscope: Support

- For questions and feedback please feel free to reach out to us on [Slack ↗︎](https://launchpass.com/datainfra-workspace).
- For bugs, please create issue on [GitHub ↗︎](https://github.com/datainfrahq/pinot-control-plane-k8s/issues).
- For commercial support and consultation, please reach out to us at [`hi@datainfra.io` ↗︎](mailto:hi@datainfra.io).

## :trophy: Contributing

### Contributors

<a href="https://github.com/datainfrahq/pinot-control-plane-k8s/graphs/contributors"><img src="https://contrib.rocks/image?repo=datainfrahq/pinot-control-plane-k8s" /></a>

## Note
Apache®, [Apache Pinot](https://pinot.apache.org), Pinot® are either registered trademarks or trademarks of the Apache Software Foundation in the United States and/or other countries. This project, pinot-control-plane-k8s, is not an Apache Software Foundation project.
