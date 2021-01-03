{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='workflowStopRequest', url='', help=''),
  '#withMessage':: d.fn(help='', args=[d.arg(name='message', type=d.T.string)]),
  withMessage(message): { message: message },
  '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withNamespace':: d.fn(help='', args=[d.arg(name='namespace', type=d.T.string)]),
  withNamespace(namespace): { namespace: namespace },
  '#withNodeFieldSelector':: d.fn(help='', args=[d.arg(name='nodeFieldSelector', type=d.T.string)]),
  withNodeFieldSelector(nodeFieldSelector): { nodeFieldSelector: nodeFieldSelector },
  '#mixin': 'ignore',
  mixin: self,
}
