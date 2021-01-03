{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='nodeSynchronizationStatus', url='', help='NodeSynchronizationStatus stores the status of a node'),
  '#withWaiting':: d.fn(help='Waiting is the name of the lock that this node is waiting for', args=[d.arg(name='waiting', type=d.T.string)]),
  withWaiting(waiting): { waiting: waiting },
  '#mixin': 'ignore',
  mixin: self,
}
