{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='gauge', url='', help='Gauge is a Gauge prometheus metric'),
  '#withRealtime':: d.fn(help='Realtime emits this metric in real time if applicable', args=[d.arg(name='realtime', type=d.T.boolean)]),
  withRealtime(realtime): { realtime: realtime },
  '#withValue':: d.fn(help='Value is the value of the metric', args=[d.arg(name='value', type=d.T.string)]),
  withValue(value): { value: value },
  '#mixin': 'ignore',
  mixin: self,
}
