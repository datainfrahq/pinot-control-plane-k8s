### Getting Started With Table Management

## prerequisite
- Create a cluster using the following [doc](./getting_started_local.md)

## Introduction

- Pinot Table CRD belongs to the following GVK
```
group: datainfra.io
version: v1beta1
kind: PinotTable
```
- The table controller reconciles the following spec as a CR.

```
apiVersion: datainfra.io/v1beta1
kind: PinotSchema
metadata:
  name: airlinestats
spec:
  pinotCluster: pinot-basic
  pinotSchema: airlinestats
  pinotTableType: REALTIME
  tables.json: |-
    {
        ....
        ....
    }
```

- Pinot table types support the following:
```
type PinotTableType string

const (
	RealTimeTable    PinotTableType = "realtime"
	OfflineTimeTable PinotTableType = "offline"
	HybridTable      PinotTableType = "hybrid"
)
```

- The table controller is responsible for creation, updation and deletion of the table.

- The table controller uses finalisers ```pinottable.datainfra.io/finalizer``` for deletion logic.

### Table Status

- Get the status of pinotschema
```
kubectl get pinottable -n <namespace> -o yaml
```

- Table controller is patches the status on each reconcile in case of a state change.

- Current state of the table is stored in the status of the table CR.

```
currentTables.json: 
{
    ...
    ...
    ...
}
lastUpdateTime: "2023-04-24T17:35:19.019234+05:30"
message: PinotTableControllerCreateSuccess
reason: '{"unrecognizedProperties":{},"status":"Table airlineStats_REALTIME successfully added"}'
status: "True"
type: PinotTableControllerCreateSuccess
```
