{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='noneStrategy', url='', help='NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately.'),
  '#mixin': 'ignore',
  mixin: self,
}
