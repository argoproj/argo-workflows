use argo_workflows::apis::configuration;
use argo_workflows::models::IoArgoprojWorkflowV1alpha1Workflow;
use argo_workflows::models::ObjectMeta;

#[test]
fn test_create_workflow() {
    let metadata = ObjectMeta::new();
    metadata.cluster_name.as_deref().unwrap_or("hello, world!");
    metadata.cluster_name.map("string");
    let mut client = configuration::Configuration::new();

    let manifest = IoArgoprojWorkflowV1alpha1Workflow::new(metadata, spec)
}
