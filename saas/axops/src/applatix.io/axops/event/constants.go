// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package event

const (
	TopicContainerUsage  = "container_usages"
	TopicHostUsage       = "host_usages"
	TopicHost            = "hosts"
	TopicContainer       = "containers"
	TopicBitbucketBranch = "devops_branch"
	TopicDevopsTasks     = "devops_task"
	TopicBitBucketCommit = "devops_commit"
	TopicMessage         = "messages"
	TopicDevopsTemplates = "devops_template"
	TopicDevopsCI        = "devops_ci_event"
	TopicRepoGC          = "repo_gc"
)

const (
	KafkaServiceName     = "kafka-zk:9092"
	RestProxyServiceName = "http://rest-proxy:8092"
	//all instances in AX cluster share the same ConsumerGroupID so that they form a group.
	ConsumerGroupID = "AXEvent"
)

const (
	eventAppName = "axevent"
)

const (
	topicColumnName = "topic"
	keyColumnName   = "key"
	dataColumnName  = "data"
	opColumnName    = "op"
)

const NumOfConsumers = 8
