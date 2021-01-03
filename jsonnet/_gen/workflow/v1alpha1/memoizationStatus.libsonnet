{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='memoizationStatus', url='', help='MemoizationStatus is the status of this memoized node'),
  '#withCacheName':: d.fn(help='Cache is the name of the cache that was used', args=[d.arg(name='cacheName', type=d.T.string)]),
  withCacheName(cacheName): { cacheName: cacheName },
  '#withHit':: d.fn(help='Hit indicates whether this node was created from a cache entry', args=[d.arg(name='hit', type=d.T.boolean)]),
  withHit(hit): { hit: hit },
  '#withKey':: d.fn(help="Key is the name of the key used for this node's cache", args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#mixin': 'ignore',
  mixin: self,
}
