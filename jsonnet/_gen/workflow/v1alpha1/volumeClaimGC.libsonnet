{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='volumeClaimGC', url='', help='VolumeClaimGC describes how to delete volumes from completed Workflows'),
  '#withStrategy':: d.fn(help='Strategy is the strategy to use. One of "OnWorkflowCompletion", "OnWorkflowSuccess"', args=[d.arg(name='strategy', type=d.T.string)]),
  withStrategy(strategy): { strategy: strategy },
  '#mixin': 'ignore',
  mixin: self,
}
