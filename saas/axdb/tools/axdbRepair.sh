#! /bin/bash
usage ()
{
  echo 'Usage : axdbRepair.sh -rf | [--replicas <replicas>] --cluster-name <cluster-name>'
  echo '                      -h | --help'
}

REPLICAS="3"
HELP=""
AXDB_SERVICE="axdb"
CLUSTER_NAME=""
if [ "$#" -lt 1 ]; then
    usage
    exit 1
fi

while [[ $# -ge 1 ]]
do
key="$1"
case $key in
    -rf|--replicas)
    REPLICAS="$2"
    echo "alter the replication factor to $REPLICAS"
    shift # past argument
    ;;
    --cluster-name)
    CLUSTER_NAME="$2"
    echo "alter the replication factor of cassandra for cluster $CLUSTER_NAME"
    shift
    ;;
    -h|--help)
    HELP="YES"
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

source ~/Dev/prod1.0/bash-helpers.sh
kdef ${CLUSTER_NAME}
IFS=$'\n'
CMD="alter keyspace axdb with replication={'class': 'SimpleStrategy', 'replication_factor': $REPLICAS};"
for line in `kp | grep ${AXDB_SERVICE} | awk '{print $1}'`
do
echo $line
KUBECONF=cluster_${CLUSTER_NAME}.conf
kubectl --namespace axsys --kubeconfig ~/.kube/$KUBECONF exec -ti $line -- cqlsh -e $CMD
done

for line in `kp | grep ${AXDB_SERVICE} | awk '{print $1}'`
do 
echo "repair for $line..."
OK=
while : ; do
  kshell $line "nodetool repair" 
  RET=$?
  if [ "$RET" == "0" ] ; then
    break
  else
    echo "repair for $line failed, retrying...."
  fi
done
echo "repair for $line is done!"
done

echo "Completed"
