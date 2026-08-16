

# IoArgoprojWorkflowV1alpha1DAGTemplate

DAGTemplate is a template subtype for directed acyclic graph templates

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**failFast** | **Boolean** | This flag is for DAG logic. The DAG logic has a built-in \&quot;fail fast\&quot; feature to stop scheduling new steps, as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed before failing the DAG itself. The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG to completion (either success or failure), regardless of the failed outcomes of branches in the DAG. More info and example about this feature at https://github.com/argoproj/argo-workflows/issues/1442 |  [optional]
**target** | **String** | Target are one or more names of targets to execute in a DAG |  [optional]
**tasks** | [**List&lt;IoArgoprojWorkflowV1alpha1DAGTask&gt;**](IoArgoprojWorkflowV1alpha1DAGTask.md) | Tasks are a list of DAG tasks MaxItems is an artificial limit to limit CEL validation costs - see note at top of file | 



