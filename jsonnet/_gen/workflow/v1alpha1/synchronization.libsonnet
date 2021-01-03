{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='synchronization', url='', help='Synchronization holds synchronization lock configuration'),
  '#mutex':: d.obj(help='Mutex holds Mutex configuration'),
  mutex: {
    '#withName':: d.fn(help='name of the mutex', args=[d.arg(name='name', type=d.T.string)]),
    withName(name): { mutex+: { name: name } },
  },
  '#semaphore':: d.obj(help='SemaphoreRef is a reference of Semaphore'),
  semaphore: {
    '#configMapKeyRef':: d.obj(help='Selects a key from a ConfigMap.'),
    configMapKeyRef: {
      '#localObjectReference':: d.obj(help='LocalObjectReference contains enough information to let you locate the\nreferenced object inside the same namespace.'),
      localObjectReference: {
        '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
        withName(name): { semaphore+: { configMapKeyRef+: { localObjectReference+: { name: name } } } },
      },
      '#withKey':: d.fn(help='The key to select.', args=[d.arg(name='key', type=d.T.string)]),
      withKey(key): { semaphore+: { configMapKeyRef+: { key: key } } },
      '#withOptional':: d.fn(help='', args=[d.arg(name='optional', type=d.T.boolean)]),
      withOptional(optional): { semaphore+: { configMapKeyRef+: { optional: optional } } },
    },
  },
  '#mixin': 'ignore',
  mixin: self,
}
