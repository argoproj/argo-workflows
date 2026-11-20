# Daemon Containers

Argo workflows can start containers that run in the background (also known as `daemon containers`) while the workflow itself continues execution. Note that the daemons will be *automatically destroyed* when the workflow exits the template scope in which the daemon was invoked. Daemon containers are useful for starting up services to be tested or to be used in testing (e.g., fixtures). We also find it very useful when running large simulations to spin up a database as a daemon for collecting and organizing the results. The big advantage of daemons compared with sidecars is that their existence can persist across multiple steps or even the entire workflow.

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

Step templates use the `steps` prefix to refer to another step: for example `{{steps.influx.ip}}`. In DAG templates, the `tasks` prefix is used instead: for example `{{tasks.influx.ip}}`.

## When a daemon dies

A daemon step can die after its step group has finished.
The step group stays `Succeeded`.
The daemon's node fails.
The Steps template and the workflow fail with the message `child '<node ID>' failed`, and no further step group starts.

In a DAG template, a daemon task that dies once it is running fails the DAG in the same way.
An expanded daemon task has `withItems`, `withParam` or `withSequence`.
When its items die after its `TaskGroup` node has finished, they also fail the DAG.

A daemon step or task without items that dies in this way runs its exit hook.
It also runs each lifecycle hook whose expression matches the phase that the daemon ends in.
The template waits for these hooks to finish, and then ends `Failed`.
The items of an expanded daemon step or task that die in this way do not start new hooks.

When a daemon with a `retryStrategy` dies, the controller retries it while the steps or tasks that use it are running.
A daemon can also die in the same reconciliation in which the controller sees the last of those steps or tasks finish.
The controller then also retries the daemon: the template starts a new attempt and waits for it to be ready before the template ends.
So the workflow runs one more pod and finishes later.
If the new attempt never becomes ready, for example because its pod cannot be scheduled, the workflow stays `Running`.
It stays `Running` until the attempt's `pendingTimeout` or `timeout`, or the workflow's `activeDeadlineSeconds`, passes.
