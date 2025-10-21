Description: Add input/output specific artifact repository
Author: [YAMADA Yutaka](https://github.com/yamadayutaka)
Component: General
Issues: 14883

Enables specifying separate artifact repositories for inputs and outputs.
This allows you to separate the output destination for archive logs from the input/output artifact repositories.
By enabling configuration through Artifact Repository Ref, redundant repository configuration descriptions can be avoided.