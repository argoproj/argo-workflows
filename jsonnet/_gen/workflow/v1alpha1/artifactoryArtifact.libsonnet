{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='artifactoryArtifact', url='', help='ArtifactoryArtifact is the location of an artifactory artifact'),
  '#passwordSecret':: d.obj(help='SecretKeySelector selects a key of a Secret.'),
  passwordSecret: {
    '#localObjectReference':: d.obj(help='LocalObjectReference contains enough information to let you locate the\nreferenced object inside the same namespace.'),
    localObjectReference: {
      '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
      withName(name): { passwordSecret+: { localObjectReference+: { name: name } } },
    },
    '#withKey':: d.fn(help='The key of the secret to select from.  Must be a valid secret key.', args=[d.arg(name='key', type=d.T.string)]),
    withKey(key): { passwordSecret+: { key: key } },
    '#withOptional':: d.fn(help='', args=[d.arg(name='optional', type=d.T.boolean)]),
    withOptional(optional): { passwordSecret+: { optional: optional } },
  },
  '#usernameSecret':: d.obj(help='SecretKeySelector selects a key of a Secret.'),
  usernameSecret: {
    '#localObjectReference':: d.obj(help='LocalObjectReference contains enough information to let you locate the\nreferenced object inside the same namespace.'),
    localObjectReference: {
      '#withName':: d.fn(help='', args=[d.arg(name='name', type=d.T.string)]),
      withName(name): { usernameSecret+: { localObjectReference+: { name: name } } },
    },
    '#withKey':: d.fn(help='The key of the secret to select from.  Must be a valid secret key.', args=[d.arg(name='key', type=d.T.string)]),
    withKey(key): { usernameSecret+: { key: key } },
    '#withOptional':: d.fn(help='', args=[d.arg(name='optional', type=d.T.boolean)]),
    withOptional(optional): { usernameSecret+: { optional: optional } },
  },
  '#withUrl':: d.fn(help='URL of the artifact', args=[d.arg(name='url', type=d.T.string)]),
  withUrl(url): { url: url },
  '#mixin': 'ignore',
  mixin: self,
}
