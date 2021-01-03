{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='mutexHolding', url='', help='MutexHolding describes the mutex and the object which is holding it.'),
  '#withHolder':: d.fn(help="Holder is a reference to the object which holds the Mutex. Holding Scenario:\n  1. Current workflow's NodeID which is holding the lock.\n     e.g: ${NodeID}\nWaiting Scenario:\n  1. Current workflow or other workflow NodeID which is holding the lock.\n     e.g: ${WorkflowName}/${NodeID}", args=[d.arg(name='holder', type=d.T.string)]),
  withHolder(holder): { holder: holder },
  '#withMutex':: d.fn(help='Reference for the mutex e.g: ${namespace}/mutex/${mutexName}', args=[d.arg(name='mutex', type=d.T.string)]),
  withMutex(mutex): { mutex: mutex },
  '#mixin': 'ignore',
  mixin: self,
}
