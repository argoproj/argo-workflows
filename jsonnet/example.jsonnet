local argo = import 'main.libsonnet';
local workflow = argo.workflow.v1alpha1.workflow;
local template = argo.workflow.v1alpha1.template;

workflow.new() +
workflow.metadata.withGenerateName('hello-world-') +
  workflow.spec.withEntrypoint('whalesay') +
    workflow.spec.withTemplates([
        template.withName('whalesay') +
          template.container.withImage('docker/whalesay') +
          template.container.withCommand('cowsay') +
          template.container.withArgs('hello world') +
          template.container.resources.withLimits({memory: '32Mi', cpu: '100m'})
    ])
    