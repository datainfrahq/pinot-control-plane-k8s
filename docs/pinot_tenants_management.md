## Getting Started With Tenants Management

### Prerequisites
- Create a cluster using the following [doc](./getting_started_tenants.md.md)

### Introduction

- Pinot tenant CRD belongs to the following GVK
```
group: datainfra.io
version: v1beta1
kind: PinotTenant
```
- The tenant controller reconciles the following spec as a CR.

```
apiVersion: datainfra.io/v1beta1
kind: PinotTenant
metadata:
  name: airlinestats
spec:
  pinotCluster: pinot-basic
  pinotSchema: airlinestats
  pinotTenantType: BROKER
  tenants.json: |- 
    {
      "tenantRole": "BROKER",
      "tenantName": "sampleBrokerTenant",
      "numberOfInstances": 1
    }
```

- Pinot tenant types support the following:
```
type PinotTenantType string

const (
	BrokerTenant PinotTenantType = "BROKER"
	ServerTenant PinotTenantType = "SERVER"
)
```

- The tenant controller is responsible for creation, updation and deletion of the tenant.

- The tenant controller uses finalisers ```pinottenant.datainfra.io/finalizer``` for deletion logic.

### Tenant Status

- Get the status of pinotschema
```
kubectl get pinottenant -n <namespace> -o yaml
```

- Tenant controller is patches the status on each reconcile in case of a state change.

- Current state of the tenant is stored in the status of the tenant CR.

```

```
