{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='counter', url='', help='Counter is a Counter prometheus metric'),
  '#withValue':: d.fn(help='Value is the value of the metric', args=[d.arg(name='value', type=d.T.string)]),
  withValue(value): { value: value },
  '#mixin': 'ignore',
  mixin: self,
}
