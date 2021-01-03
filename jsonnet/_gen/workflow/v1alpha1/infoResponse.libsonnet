{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='infoResponse', url='', help=''),
  '#withLinks':: d.fn(help='', args=[d.arg(name='links', type=d.T.array)]),
  withLinks(links): { links: if std.isArray(v=links) then links else [links] },
  '#withLinksMixin':: d.fn(help='\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='links', type=d.T.array)]),
  withLinksMixin(links): { links+: if std.isArray(v=links) then links else [links] },
  '#withManagedNamespace':: d.fn(help='', args=[d.arg(name='managedNamespace', type=d.T.string)]),
  withManagedNamespace(managedNamespace): { managedNamespace: managedNamespace },
  '#mixin': 'ignore',
  mixin: self,
}
