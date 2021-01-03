{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='semaphoreRef', url='', help='SemaphoreRef is a reference of Semaphore'),
  '#configMapKeyRef':: d.obj(help='Selects a key from a ConfigMap.'),
  configMapKeyRef: {
    '#localObjectReference':: d.obj(help='LocalObjectReference contains enough information to let you locate the\nreferenced object inside the same namespace.'),
    localObjectReference: {
      '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
      withName(name): { configMapKeyRef+: { localObjectReference+: { name: name } } },
    },
    '#withKey':: d.fn(help='The key to select.', args=[d.arg(name='key', type=d.T.string)]),
    withKey(key): { configMapKeyRef+: { key: key } },
    '#withOptional':: d.fn(help='', args=[d.arg(name='optional', type=d.T.boolean)]),
    withOptional(optional): { configMapKeyRef+: { optional: optional } },
  },
  '#mixin': 'ignore',
  mixin: self,
}
