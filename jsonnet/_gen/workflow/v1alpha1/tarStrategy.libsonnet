{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='tarStrategy', url='', help='TarStrategy will tar and gzip the file or directory when saving'),
  '#withCompressionLevel':: d.fn(help='CompressionLevel specifies the gzip compression level to use for the artifact. Defaults to gzip.DefaultCompression.', args=[d.arg(name='compressionLevel', type=d.T.integer)]),
  withCompressionLevel(compressionLevel): { compressionLevel: compressionLevel },
  '#mixin': 'ignore',
  mixin: self,
}
