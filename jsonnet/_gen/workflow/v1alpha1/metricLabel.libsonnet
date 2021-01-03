{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='metricLabel', url='', help='MetricLabel is a single label for a prometheus metric'),
  '#withKey':: d.fn(help='', args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#withValue':: d.fn(help='', args=[d.arg(name='value', type=d.T.string)]),
  withValue(value): { value: value },
  '#mixin': 'ignore',
  mixin: self,
}
