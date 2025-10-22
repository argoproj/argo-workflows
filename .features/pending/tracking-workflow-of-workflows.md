Description: Add labels to track workflow of workflows
Author: [Chengwei Guo](https://github.com/cw-Guo) & [Eduardo Rodrigues](https://github.com/eduardodbr)
Component: General
Issues: 6922

When creating a workflow resource in resouce template, attach some labels to track the parent workflow name and parent node id. Such labels can be used to to jump between parent workflow and children workflow using new UI buttons that only appear if the labels exist.
It only works when `setOwnerReference` is set to True.