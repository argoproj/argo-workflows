{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='cache', url='', help='Cache is the configuration for the type of cache to be used'),
  '#configMap':: d.obj(help='Selects a key from a ConfigMap.'),
  configMap: {
    '#localObjectReference':: d.obj(help='LocalObjectReference contains enough information to let you locate the\nreferenced object inside the same namespace.'),
    localObjectReference: {
      '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
      withName(name): { configMap+: { localObjectReference+: { name: name } } },
    },
    '#withKey':: d.fn(help='The key to select.', args=[d.arg(name='key', type=d.T.string)]),
    withKey(key): { configMap+: { key: key } },
    '#withOptional':: d.fn(help='', args=[d.arg(name='optional', type=d.T.boolean)]),
    withOptional(optional): { configMap+: { optional: optional } },
  },
  '#mixin': 'ignore',
  mixin: self,
}
