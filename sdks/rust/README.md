# Argo Rust SDK

This is the Rust SDK for Argo Workflows.

## Installation

Please run the following:
```
cargo add argo_workflows
```

## Getting Started

You can submit a workflow from a raw YAML like the following:

```toml
# Cargo.toml
reqwest = { version = "0.11", features = ["json"] }
tokio = { version = "1", features = ["full"] }
serde_yaml = "0.9"
```

```rust
// main.rs
use argo_workflows::apis::configuration;
use argo_workflows::apis::workflow_service_api;
use argo_workflows::models::IoArgoprojWorkflowV1alpha1Workflow as Workflow;
use argo_workflows::models::IoArgoprojWorkflowV1alpha1WorkflowCreateRequest as WorkflowCreateRequest;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let resp =
        reqwest::get("https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/hello-world.yaml")
            .await?
            .text_with_charset("utf-8")
            .await?;

    let manifest: Workflow = serde_yaml::from_str(&resp)?;
    let config = configuration::Configuration::new();

    let create_workflow_params = workflow_service_api::CreateWorkflowParams {
        namespace: String::from("argo"),
        body: WorkflowCreateRequest {
            create_options: None,
            instance_id: None,
            namespace: None,
            server_dry_run: None,
            workflow: Some(Box::new(manifest)),
        },
    };

    match workflow_service_api::create_workflow(&config, create_workflow_params).await {
        Ok(success) => {
            let created_workflow: Workflow = serde_yaml::from_str(&success.content)?;
            println!(
                "successfully created workflow!\n{}",
                serde_yaml::to_string(&created_workflow).unwrap()
            );
        }
        Err(err) => panic!("error: \n{:#?}", &err.to_string()),
    }

    Ok(())
}
```
Alternatively, you can submit a workflow using `IoArgoprojWorkflowV1alpha1Workflow` constructed via the SDK like the following:

```rust
use argo_workflows::apis::configuration;
use argo_workflows::apis::workflow_service_api;
use argo_workflows::models::Container;
use argo_workflows::models::IoArgoprojWorkflowV1alpha1Template as WorkflowTemplate;
use argo_workflows::models::IoArgoprojWorkflowV1alpha1Workflow as Workflow;
use argo_workflows::models::IoArgoprojWorkflowV1alpha1WorkflowCreateRequest as WorkflowCreateRequest;
use argo_workflows::models::IoArgoprojWorkflowV1alpha1WorkflowSpec as WorkflowSpec;
use argo_workflows::models::ObjectMeta;

#[tokio::main]
async fn main() {
    let config = configuration::Configuration::new();

    let mut metadata = ObjectMeta::new();
    metadata.generate_name = Some(String::from("hello-world-"));

    let mut container = Container::new(String::from("argoproj/argosay:v2"));
    container.command = Some(vec![String::from("cowsay")]);
    container.args = Some(vec![String::from("hello, world!")]);

    let mut template = WorkflowTemplate::new();
    template.name = Some(String::from("whalesay"));
    template.container = Some(Box::new(container));

    let mut spec = WorkflowSpec::new();
    spec.entrypoint = Some(String::from("whalesay"));
    spec.templates = Some(vec![template]);

    let manifest = Workflow::new(metadata, spec);

    let request = WorkflowCreateRequest {
        create_options: None,
        instance_id: None,
        namespace: None,
        server_dry_run: None,
        workflow: Some(Box::new(manifest)),
    };

    let create_workflow_params = workflow_service_api::CreateWorkflowParams {
        namespace: String::from("argo"),
        body: request,
    };

    match workflow_service_api::create_workflow(&config, create_workflow_params).await {
        Ok(success) => {
            let created_workflow: Workflow = serde_yaml::from_str(&success.content).unwrap();
            println!(
                "successfully created workflow!\n{}",
                serde_yaml::to_string(&created_workflow).unwrap()
            );
        }
        Err(err) => panic!("error: \n{:#?}", &err.to_string()),
    }
}
```

## Examples

You can find additional examples [here](tests).

## API Reference

You can find the API reference [here](client/docs).
