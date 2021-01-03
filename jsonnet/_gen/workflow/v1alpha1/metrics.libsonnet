{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='metrics', url='', help='Metrics are a list of metrics emitted from a Workflow/Template'),
  '#withPrometheus':: d.fn(help='Prometheus is a list of prometheus metrics to be emitted', args=[d.arg(name='prometheus', type=d.T.array)]),
  withPrometheus(prometheus): { prometheus: if std.isArray(v=prometheus) then prometheus else [prometheus] },
  '#withPrometheusMixin':: d.fn(help='Prometheus is a list of prometheus metrics to be emitted\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='prometheus', type=d.T.array)]),
  withPrometheusMixin(prometheus): { prometheus+: if std.isArray(v=prometheus) then prometheus else [prometheus] },
  '#mixin': 'ignore',
  mixin: self,
}
