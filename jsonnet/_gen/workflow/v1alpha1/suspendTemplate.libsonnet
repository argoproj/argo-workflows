{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='suspendTemplate', url='', help='SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time'),
  '#withDuration':: d.fn(help='Duration is the seconds to wait before automatically resuming a template', args=[d.arg(name='duration', type=d.T.string)]),
  withDuration(duration): { duration: duration },
  '#mixin': 'ignore',
  mixin: self,
}
