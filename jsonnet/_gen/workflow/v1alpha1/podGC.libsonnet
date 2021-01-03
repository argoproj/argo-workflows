{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podGC', url='', help='PodGC describes how to delete completed pods as they complete'),
  '#withStrategy':: d.fn(help='Strategy is the strategy to use. One of "OnPodCompletion", "OnPodSuccess", "OnWorkflowCompletion", "OnWorkflowSuccess"', args=[d.arg(name='strategy', type=d.T.string)]),
  withStrategy(strategy): { strategy: strategy },
  '#mixin': 'ignore',
  mixin: self,
}
