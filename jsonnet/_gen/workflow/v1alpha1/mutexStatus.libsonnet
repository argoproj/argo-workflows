{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='mutexStatus', url='', help='MutexStatus contains which objects hold  mutex locks, and which objects this workflow is waiting on to release locks.'),
  '#withHolding':: d.fn(help='Holding is a list of mutexes and their respective objects that are held by mutex lock for this io.argoproj.workflow.v1alpha1.', args=[d.arg(name='holding', type=d.T.array)]),
  withHolding(holding): { holding: if std.isArray(v=holding) then holding else [holding] },
  '#withHoldingMixin':: d.fn(help='Holding is a list of mutexes and their respective objects that are held by mutex lock for this io.argoproj.workflow.v1alpha1.\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='holding', type=d.T.array)]),
  withHoldingMixin(holding): { holding+: if std.isArray(v=holding) then holding else [holding] },
  '#withWaiting':: d.fn(help='Waiting is a list of mutexes and their respective objects this workflow is waiting for.', args=[d.arg(name='waiting', type=d.T.array)]),
  withWaiting(waiting): { waiting: if std.isArray(v=waiting) then waiting else [waiting] },
  '#withWaitingMixin':: d.fn(help='Waiting is a list of mutexes and their respective objects this workflow is waiting for.\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='waiting', type=d.T.array)]),
  withWaitingMixin(waiting): { waiting+: if std.isArray(v=waiting) then waiting else [waiting] },
  '#mixin': 'ignore',
  mixin: self,
}
