# Daemon Containers

Argo workflows can start containers that run in the background (also known as `daemon containers`) while the workflow itself continues execution. Note that the daemons will be *automatically destroyed* when the workflow exits the template scope in which the daemon was invoked. Daemon containers are useful for starting up services to be tested or to be used in testing (e.g., fixtures). We also find it very useful when running large simulations to spin up a database as a daemon for collecting and organizing the results. The big advantage of daemons compared with sidecars is that their existence can persist across multiple steps or even the entire workflow.

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: daemon-step-
spec:
  entrypoint: daemon-example
  templates:
  - name: daemon-example
    steps:
    - - name: influx
        template: influxdb              # start an influxdb as a daemon (see the influxdb template spec below)

    - - name: init-database             # initialize influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl -XPOST 'http://{{steps.influx.ip}}:8086/query' --data-urlencode "q=CREATE DATABASE mydb"

    - - name: producer-1                # add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d "cpu,host=server01,region=uswest load=$i" ; sleep .5 ; done
      - name: producer-2                # add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d "cpu,host=server02,region=uswest load=$((RANDOM % 100))" ; sleep .5 ; done
      - name: producer-3                # add entries to influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d 'cpu,host=server03,region=useast load=15.4'

    - - name: consumer                  # consume intries from influxdb
        template: influxdb-client
        arguments:
          parameters:
          - name: cmd
            value: curl --silent -G http://{{steps.influx.ip}}:8086/query?pretty=true --data-urlencode "db=mydb" --data-urlencode "q=SELECT * FROM cpu"

  - name: influxdb
    daemon: true                        # start influxdb as a daemon
    retryStrategy:
      limit: 10                         # retry container if it fails
    container:
      image: influxdb:1.2
      command:
      - influxd
      readinessProbe:                   # wait for readinessProbe to succeed
        httpGet:
          path: /ping
          port: 8086

  - name: influxdb-client
    inputs:
      parameters:
      - name: cmd
    container:
      image: appropriate/curl:latest
      command: ["/bin/sh", "-c"]
      args: ["{{inputs.parameters.cmd}}"]
      resources:
        requests:
          memory: 32Mi
          cpu: 100m
```

///

/// tab | Python

```python
from hera.workflows import Container, Step, Steps, Workflow
from hera.workflows.models import (
    HTTPGetAction,
    Parameter,
    Probe,
    ResourceRequirements,
    RetryStrategy,
)

with Workflow(
    generate_name="daemon-step-",
    entrypoint="daemon-example",
) as w:
    influxdb = Container(
        name="influxdb",
        image="influxdb:1.2",
        command=["influxd"],
        retry_strategy=RetryStrategy(limit=10),
        readiness_probe=Probe(http_get=HTTPGetAction(path="/ping", port=8086)),
        daemon=True,
    )
    influxdb_client = Container(
        name="influxdb-client",
        image="appropriate/curl:latest",
        command=["/bin/sh", "-c"],
        args=["{{inputs.parameters.cmd}}"],
        inputs=[Parameter(name="cmd")],
        resources=ResourceRequirements(
            requests={
                "memory": "32Mi",
                "cpu": "100m",
            }
        ),
    )

    with Steps(name="daemon-example") as steps:
        influxdb(name="influx")
        influxdb_client(
            name="init-database",
            arguments={"cmd": "curl -XPOST 'http://{{steps.influx.ip}}:8086/query' --data-urlencode \"q=CREATE DATABASE mydb\""},
        )
        with steps.parallel():
            influxdb_client(
                name="producer-1",
                arguments={"cmd": "for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d \"cpu,host=server01,region=uswest load=$i\" ; sleep .5 ; done"},
            )
            influxdb_client(
                name="producer-2",
                arguments={"cmd": "for i in $(seq 1 20); do curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d \"cpu,host=server02,region=uswest load=$((RANDOM % 100))\" ; sleep .5 ; done"},
            )
            influxdb_client(
                name="producer-3",
                arguments={"cmd": "curl -XPOST 'http://{{steps.influx.ip}}:8086/write?db=mydb' -d 'cpu,host=server03,region=useast load=15.4'"},
            )
        influxdb_client(
            name="consumer",
            arguments={"cmd":'curl --silent -G http://{{steps.influx.ip}}:8086/query?pretty=true --data-urlencode "db=mydb" --data-urlencode "q=SELECT * FROM cpu"'},
        )
```

///

Step templates use the `steps` prefix to refer to [certain attributes](../variables.md#steps-templates) of another step: for example `{{steps.influx.ip}}`.
In DAG templates, the `tasks` prefix is used instead: for example `{{tasks.influx.ip}}`.
