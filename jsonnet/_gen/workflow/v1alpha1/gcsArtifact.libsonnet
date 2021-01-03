{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='gcsArtifact', url='', help='GCSArtifact is the location of a GCS artifact'),
  '#serviceAccountKeySecret':: d.obj(help='SecretKeySelector selects a key of a Secret.'),
  serviceAccountKeySecret: {
    '#localObjectReference':: d.obj(help='LocalObjectReference contains enough information to let you locate the\nreferenced object inside the same namespace.'),
    localObjectReference: {
      '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
      withName(name): { serviceAccountKeySecret+: { localObjectReference+: { name: name } } },
    },
    '#withKey':: d.fn(help='The key of the secret to select from.  Must be a valid secret key.', args=[d.arg(name='key', type=d.T.string)]),
    withKey(key): { serviceAccountKeySecret+: { key: key } },
    '#withOptional':: d.fn(help='', args=[d.arg(name='optional', type=d.T.boolean)]),
    withOptional(optional): { serviceAccountKeySecret+: { optional: optional } },
  },
  '#withBucket':: d.fn(help='Bucket is the name of the bucket', args=[d.arg(name='bucket', type=d.T.string)]),
  withBucket(bucket): { bucket: bucket },
  '#withKey':: d.fn(help='Key is the path in the bucket where the artifact resides', args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#mixin': 'ignore',
  mixin: self,
}
