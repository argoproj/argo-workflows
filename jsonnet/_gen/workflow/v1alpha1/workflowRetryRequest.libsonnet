{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='workflowRetryRequest', url='', help=''),
  '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withNamespace':: d.fn(help='', args=[d.arg(name='namespace', type=d.T.string)]),
  withNamespace(namespace): { namespace: namespace },
  '#withNodeFieldSelector':: d.fn(help='', args=[d.arg(name='nodeFieldSelector', type=d.T.string)]),
  withNodeFieldSelector(nodeFieldSelector): { nodeFieldSelector: nodeFieldSelector },
  '#withRestartSuccessful':: d.fn(help='', args=[d.arg(name='restartSuccessful', type=d.T.boolean)]),
  withRestartSuccessful(restartSuccessful): { restartSuccessful: restartSuccessful },
  '#mixin': 'ignore',
  mixin: self,
}
