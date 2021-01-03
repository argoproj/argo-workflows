{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='backoff', url='', help='Backoff is a backoff strategy to use within retryStrategy'),
  '#factor':: d.obj(help='+protobuf=true\n+protobuf.options.(gogoproto.goproto_stringer)=false\n+k8s:openapi-gen=true'),
  factor: {
    '#withIntVal':: d.fn(help='', args=[d.arg(name='intVal', type=d.T.integer)]),
    withIntVal(intVal): { factor+: { intVal: intVal } },
    '#withStrVal':: d.fn(help='', args=[d.arg(name='strVal', type=d.T.string)]),
    withStrVal(strVal): { factor+: { strVal: strVal } },
    '#withType':: d.fn(help='', args=[d.arg(name='type', type=d.T.string)]),
    withType(type): { factor+: { type: type } },
  },
  '#withDuration':: d.fn(help='Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h")', args=[d.arg(name='duration', type=d.T.string)]),
  withDuration(duration): { duration: duration },
  '#withMaxDuration':: d.fn(help='MaxDuration is the maximum amount of time allowed for the backoff strategy', args=[d.arg(name='maxDuration', type=d.T.string)]),
  withMaxDuration(maxDuration): { maxDuration: maxDuration },
  '#mixin': 'ignore',
  mixin: self,
}
