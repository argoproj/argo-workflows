{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='semaphoreStatus', url='', help=''),
  '#withHolding':: d.fn(help='Holding stores the list of resource acquired synchronization lock for workflows.', args=[d.arg(name='holding', type=d.T.array)]),
  withHolding(holding): { holding: if std.isArray(v=holding) then holding else [holding] },
  '#withHoldingMixin':: d.fn(help='Holding stores the list of resource acquired synchronization lock for workflows.\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='holding', type=d.T.array)]),
  withHoldingMixin(holding): { holding+: if std.isArray(v=holding) then holding else [holding] },
  '#withWaiting':: d.fn(help='Waiting indicates the list of current synchronization lock holders.', args=[d.arg(name='waiting', type=d.T.array)]),
  withWaiting(waiting): { waiting: if std.isArray(v=waiting) then waiting else [waiting] },
  '#withWaitingMixin':: d.fn(help='Waiting indicates the list of current synchronization lock holders.\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='waiting', type=d.T.array)]),
  withWaitingMixin(waiting): { waiting+: if std.isArray(v=waiting) then waiting else [waiting] },
  '#mixin': 'ignore',
  mixin: self,
}
