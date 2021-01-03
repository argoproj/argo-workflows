{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='continueOn', url='', help='ContinueOn defines if a workflow should continue even if a task or step fails/errors. It can be specified if the workflow should continue when the pod errors, fails or both.'),
  '#withError':: d.fn(help='', args=[d.arg(name='err', type=d.T.boolean)]),
  withError(err): { 'error': err },
  '#withFailed':: d.fn(help='', args=[d.arg(name='failed', type=d.T.boolean)]),
  withFailed(failed): { failed: failed },
  '#mixin': 'ignore',
  mixin: self,
}
