{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='workflowTemplateRef', url='', help='WorkflowTemplateRef is a reference to a WorkflowTemplate resource.'),
  '#withClusterScope':: d.fn(help='ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate).', args=[d.arg(name='clusterScope', type=d.T.boolean)]),
  withClusterScope(clusterScope): { clusterScope: clusterScope },
  '#withName':: d.fn(help='Name is the resource name of the workflow template.', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
