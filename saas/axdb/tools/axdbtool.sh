#! /bin/bash
usage ()
{
  echo 'Usage : axdbtool.sh -rf | --replication-factor <replicas> --cluster-name <cluster>'
  echo '                    -h | --help'
}

REPLICAS=""
HELP=""
CLUSTER=""
CASSANDRA_SERVICE="axdb"
PETSET=$CASSANDRA_SERVICE
YAMLPATH="/ax/config/service"
YAML="$YAMLPATH/axdb-svc.yml.in"
if [ "$#" -lt 1 ]; then
    usage
    exit 1
fi

while [[ $# -ge 1 ]]
do
key="$1"
case $key in
    -rf|--replication-factor)
    REPLICAS="$2"
    echo "rf=$REPLICAS"
    shift # past argument
    ;;
    -h|--help)
    HELP="YES"
    ;;
    --cluster-name)
    CLUSTER="$2"
    shift # past argument
    ;;
    *)
    HELP="YES"        # unknown option
    ;;
esac
shift # past argument or value
done

if [ "$HELP" == "YES" ] ; then
    usage
    exit 1
fi

# get the current petset definition from kubernetes
# check if axdb petset is running
LN=`kubectl get petset --kubeconfig /tmp/ax_kube/cluster_${CLUSTER}.conf | grep $PETSET | wc -l`

if [ "$LN" -lt 1 ]; then
   echo "pet set $PETSET doesn't exist"
   exit 1
fi

echo "$YAML"
OLDREPLICAS=`fgrep replicas: $YAML | awk '{print $2}'`
echo "$OLDREPLICAS, $REPLICAS"
# extend the cluster with new nodes, or reduce the cluster size
if [ "$REPLICAS" == "$OLDREPLICAS" ] ; then
   exit 1
fi

kubectl delete -f $YAML --kubeconfig /tmp/ax_kube/cluster_${CLUSTER}.conf
sed -e "s/replicas: $OLDREPLICAS/replicas: $REPLICAS/" $YAML > /tmp/petset.yml
cp /tmp/petset.yml $YAML
kubectl create -f $YAML --kubeconfig /tmp/ax_kube/cluster_${CLUSTER}.conf
