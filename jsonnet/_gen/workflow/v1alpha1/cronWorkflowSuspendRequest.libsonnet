{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='cronWorkflowSuspendRequest', url='', help=''),
  '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withNamespace':: d.fn(help='', args=[d.arg(name='namespace', type=d.T.string)]),
  withNamespace(namespace): { namespace: namespace },
  '#mixin': 'ignore',
  mixin: self,
}
