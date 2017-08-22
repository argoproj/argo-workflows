placement=$(curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone)
EC2_REGION=${placement:0:-1}
INSTANCE_ID=`curl -s http://169.254.169.254/latest/meta-data/instance-id`
AWS_CMD="/usr/local/bin/aws"

# 1. Configure the minion to *not* reboot.
IS_MASTER=`grep "KUBERNETES_MASTER:" /etc/kubernetes/kube_env.yaml | cut -d " " -f 2 | sed "s/'//g"`
if [[ $IS_MASTER == true ]]; then
    echo "AX: Kubernetes master"
else
    if test -f "/var/log/ax-reboot-test"; then
       echo "AX: Reboot test file exists. Terminating instance 10 minutes from: `date`"
       sleep 600
       $AWS_CMD --region $EC2_REGION ec2 terminate-instances --instance-ids $INSTANCE_ID
    else
       date >> "/var/log/ax-reboot-test"
       echo "AX: Reboot test file created!"
    fi
fi

# 2. Add tag to the volumes connected to this instance.
CLUSTER_NAME=`$AWS_CMD --region $EC2_REGION ec2 --output text describe-tags --filters "Name=resource-id,Values=$INSTANCE_ID" --query 'Tags[?Key==\`KubernetesCluster\`].[Value]'`
volume_ids=`$AWS_CMD --region $EC2_REGION ec2 describe-volumes --filters Name=attachment.instance-id,Values=$INSTANCE_ID --output text --query 'Volumes[].VolumeId'`
for volume in ${volume_ids}; do
    echo "Tagging volume $volume. KubernetesCluster:$CLUSTER_NAME"
    n=0
    while [ $n -le 4 ]
    do
       $AWS_CMD --region $EC2_REGION ec2 create-tags --resources $volume --tags Key=KubernetesCluster,Value=$CLUSTER_NAME && break
       n=$[$n+1]
       sleep 10
    done
done
