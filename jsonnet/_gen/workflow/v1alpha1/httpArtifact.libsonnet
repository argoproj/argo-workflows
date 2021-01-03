{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='httpArtifact', url='', help='HTTPArtifact allows an file served on HTTP to be placed as an input artifact in a container'),
  '#withHeaders':: d.fn(help='Headers are an optional list of headers to send with HTTP requests for artifacts', args=[d.arg(name='headers', type=d.T.array)]),
  withHeaders(headers): { headers: if std.isArray(v=headers) then headers else [headers] },
  '#withHeadersMixin':: d.fn(help='Headers are an optional list of headers to send with HTTP requests for artifacts\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='headers', type=d.T.array)]),
  withHeadersMixin(headers): { headers+: if std.isArray(v=headers) then headers else [headers] },
  '#withUrl':: d.fn(help='URL of the artifact', args=[d.arg(name='url', type=d.T.string)]),
  withUrl(url): { url: url },
  '#mixin': 'ignore',
  mixin: self,
}
