## Getting Started With Schema Management

### Prerequisites
- Create a cluster using the following [doc](./getting_started_local.md)

### Introduction

- Pinot Schema CRD belongs to the following GVK
```
group: datainfra.io
version: v1beta1
kind: PinotSchema
```
- The schema controller reconciles the following spec as a CR.

```
apiVersion: datainfra.io/v1beta1
kind: PinotSchema
metadata:
  name: airlinestats
spec:
  pinotCluster: pinot-basic
  schemas.json: |-
  {
    ....
    ....
  }
```

- The schema controller is responsible for creation, updation and deletion of the schema.

- The schema controller uses finalisers ```pinotschema.datainfra.io/finalizer``` for deletion logic.

### Schema Status

- Get the status of pinotschema
```
kubectl get pinotschema -n <namespace> -o yaml
```

- Schema controller is patches the status on each reconcile in case of a state change.

- Current state of the schema is stored in the status of the schema CR.

```
currentSchema.json: 
{
    ...
    ...
    ...
}
lastUpdateTime: "2023-04-24T17:35:08.241187+05:30"
message: PinotSchemaControllerCreateSuccess
reason: '{"unrecognizedProperties":{},"status":"airlineStats successfully added"}'
status: "True"
type: PinotSchemaCreateSuccess
```
