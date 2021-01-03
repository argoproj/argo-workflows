{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='createS3BucketOptions', url='', help='CreateS3BucketOptions options used to determine automatic automatic bucket-creation process'),
  '#withObjectLocking':: d.fn(help='ObjectLocking Enable object locking', args=[d.arg(name='objectLocking', type=d.T.boolean)]),
  withObjectLocking(objectLocking): { objectLocking: objectLocking },
  '#mixin': 'ignore',
  mixin: self,
}
