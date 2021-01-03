{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='archiveStrategy', url='', help='ArchiveStrategy describes how to archive files/directory when saving artifacts'),
  '#tar':: d.obj(help='TarStrategy will tar and gzip the file or directory when saving'),
  tar: {
    '#withCompressionLevel':: d.fn(help='CompressionLevel specifies the gzip compression level to use for the artifact. Defaults to gzip.DefaultCompression.', args=[d.arg(name='compressionLevel', type=d.T.integer)]),
    withCompressionLevel(compressionLevel): { tar+: { compressionLevel: compressionLevel } },
  },
  '#withNone':: d.fn(help='NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately.', args=[d.arg(name='none', type=d.T.object)]),
  withNone(none): { none: none },
  '#withNoneMixin':: d.fn(help='NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately.\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='none', type=d.T.object)]),
  withNoneMixin(none): { none+: none },
  '#withZip':: d.fn(help='ZipStrategy will unzip zipped input artifacts', args=[d.arg(name='zip', type=d.T.object)]),
  withZip(zip): { zip: zip },
  '#withZipMixin':: d.fn(help='ZipStrategy will unzip zipped input artifacts\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='zip', type=d.T.object)]),
  withZipMixin(zip): { zip+: zip },
  '#mixin': 'ignore',
  mixin: self,
}
