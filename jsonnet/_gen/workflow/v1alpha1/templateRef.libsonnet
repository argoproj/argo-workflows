{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='templateRef', url='', help='TemplateRef is a reference of template resource.'),
  '#withClusterScope':: d.fn(help='ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate).', args=[d.arg(name='clusterScope', type=d.T.boolean)]),
  withClusterScope(clusterScope): { clusterScope: clusterScope },
  '#withName':: d.fn(help='Name is the resource name of the template.', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withRuntimeResolution':: d.fn(help='RuntimeResolution skips validation at creation time. By enabling this option, you can create the referred workflow template before the actual runtime. DEPRECATED: This value is not used anymore and is ignored', args=[d.arg(name='runtimeResolution', type=d.T.boolean)]),
  withRuntimeResolution(runtimeResolution): { runtimeResolution: runtimeResolution },
  '#withTemplate':: d.fn(help='Template is the name of referred template in the resource.', args=[d.arg(name='template', type=d.T.string)]),
  withTemplate(template): { template: template },
  '#mixin': 'ignore',
  mixin: self,
}
