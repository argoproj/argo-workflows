Description: Load and save the input and output artifacts of executor plugin templates
Authors: [Yanis Bendahmane](https://github.com/MrStrikes)
Component: General
Issues: 17190

Plugin templates can now have input and output artifacts, as container templates do, so steps that run on an executor plugin instead of in a Pod of their own can pass files with artifacts.
The Agent loads the input artifacts into a workspace it shares with the executor plugins before it calls them, and each artifact's `path` in the template a plugin gets is where that artifact is in the workspace.
Once the plugin replies, the Agent saves the output artifacts it wrote there to the artifact repository.
Archive strategies, keys, optional artifacts and artifact garbage collection work as they do for container templates.
See the Artifacts section of Executor Plugins for how a plugin reads and writes artifacts, with the `executor-plugin-artifacts.yaml` example.
