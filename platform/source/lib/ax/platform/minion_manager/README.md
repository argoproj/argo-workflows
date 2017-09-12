## Synopsis

The minion-manager component in Argo does the following:

1. Allows usage of AWS spot-instances in the Kubernetes cluster.
2. Monitors the health of minions in the Kubernetes cluster and terminates unhealthy ones.

## API Reference

The minion-manager configuration can be modified from AXMON. The minion-manager config currently includes two fields:

1. enabled: True/False :: Indicates whether the minion-manager is enabled on not.
2. spot_instances_option: "all"/"partial"/"none". The autoscaling groups to use for spot-instances.

Endpoint: http://axmon.axsys/v1/axmon/cluster/spot_instance_config

Method: GET

Output:
{
  "spot_instances_option": "all",
  "enabled": "True"
}

Method: PUT

Example: curl -H "Content-Type: application/json" -X PUT -d '{"enabled":"True","spot_instances_option":"partial"}' http://axmon.axsys:8901/v1/axmon/cluster/spot_instance_config

## Tests

Minion-manager tests run as part of argo-ci.

