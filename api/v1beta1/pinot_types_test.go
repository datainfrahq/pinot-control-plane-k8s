package v1beta1

import (
	"testing"

	"sigs.k8s.io/yaml"
)

var CR = `
apiVersion: datainfra.io/v1beta1
kind: Pinot
spec:

  nodes:
    - name: pinot-controller
      kind: Statefulset
      replicas: 1
      nodeType: controller
      k8sConfigGroupName: pinotcontroller
      parseableConfigGroupName: pinotcontroller

  deploymentOrder:
  - controller

  external:
    zookeeper:
      spec:
        data: |-
          controller.zk.str=pinot-zookeeper:2181

  k8sConfigGroups:
  - name: pinotcontroller
    spec:
      serviceAccountName: "pinotcontroller"
      nodeSelector: {}
      toleration: {}
      affinity: {}
      labels: {}
      service: {}
    storageConfig:
    - name: pinotcontroller
      mountPath: "/var/pinot/controller/config"
      pvcSpec:
        accessModes:
        - ReadWriteOnce
        storageClassName: "civo-volume"
        resources:
          requests:
            storage: 1Gi

  pinotNodeConfigGroups:
  - name: pinotcontroller
    java_opts: "-XX:ActiveProcessorCount=2 -Xms256M -Xmx1G -XX:+UseG1GC -XX:MaxGCPauseMillis=200
	-Xlog:gc*:file=/opt/pinot/gc-pinot-broker.log -Dlog4j2.configurationFile=/opt/pinot/conf/log4j2.xml
	-Dplugins.dir=/opt/pinot/plugins"
    data: |-
        controller.helix.cluster.name=pinot
        controller.port=9000
        controller.data.dir=/var/pinot/controller/data 
        pinot.set.instance.id.to.hostname=true
        controller.task.scheduler.enabled=true 
`

func TestParseableTenant(t *testing.T) {
	var spec Pinot

	t.Logf("%+v", spec.Spec.Nodes)
	err := yaml.Unmarshal([]byte(CR), &spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", spec.Spec)
}
