{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='histogram', url='', help='Histogram is a Histogram prometheus metric'),
  '#withBuckets':: d.fn(help='Buckets is a list of bucket divisors for the histogram', args=[d.arg(name='buckets', type=d.T.array)]),
  withBuckets(buckets): { buckets: if std.isArray(v=buckets) then buckets else [buckets] },
  '#withBucketsMixin':: d.fn(help='Buckets is a list of bucket divisors for the histogram\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='buckets', type=d.T.array)]),
  withBucketsMixin(buckets): { buckets+: if std.isArray(v=buckets) then buckets else [buckets] },
  '#withValue':: d.fn(help='Value is the value of the metric', args=[d.arg(name='value', type=d.T.string)]),
  withValue(value): { value: value },
  '#mixin': 'ignore',
  mixin: self,
}
