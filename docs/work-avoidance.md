# Work Avoidance

![GA](assets/ga.svg)

> v2.9 and after

You can make workflows faster and more robust by employing **work avoidance**. A workflow that utilizes this is simply a workflow containing steps that do not run if the work has already been done. This simplest way to do this is to use **marker files**.

Use cases:

* An expensive step appears across multiple workflows - you want to avoid repeating them.
* A workflow has unreliable tasks - you want to be able resubmit the workflow.

A **marker file** is a file on that indicates the work has already been done, before doing the work you check to see if the marker has already been done:

```sh
if [ -e /work/markers/name-of-task ]; then
    echo "work already done"
    exit 0
fi
echo "working very hard"
touch /work/markers/name-of-task
```
 
Choose a name for the file that is unique for the task, e.g. the template name and all the parameters:

```sh
touch /work/markers/$(date +%Y-%m-%d)-echo-{{inputs.parameters.num}}
``` 
 
You need to store the marker files between workflows and this can be achieved using [a PVC](fields.md#persistentvolumeclaim) and [optional input artifact](fields.md#artifact). 

[This complete work avoidance example](examples/work-avoidance.yaml) has the following:

* A PVC to store the markers on.
* A `load-markers` step that loads the marker files from artifact storage.
* Multiple `echo` tasks that avoid work using marker files.
* A `save-markers` exit handler to save the marker files, even if they are not needed. 
