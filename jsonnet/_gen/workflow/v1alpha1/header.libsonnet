{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='header', url='', help='Header indicate a key-value request header to be used when fetching artifacts over HTTP'),
  '#withName':: d.fn(help='Name is the header name', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withValue':: d.fn(help='Value is the literal value to use for the header', args=[d.arg(name='value', type=d.T.string)]),
  withValue(value): { value: value },
  '#mixin': 'ignore',
  mixin: self,
}
