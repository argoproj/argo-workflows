# The Structure of Workflow Specs

We now know enough about the basic components of a workflow spec. To review its basic structure:

- Kubernetes header including meta-data
- Spec body
    - Entrypoint invocation with optional arguments
    - List of template definitions

- For each template definition
    - Name of the template
    - Optionally a list of inputs
    - Optionally a list of outputs
    - Container invocation (leaf template) or a list of steps
        - For each step, a template invocation

To summarize, workflow specs are composed of a set of Argo templates where each template consists of an optional input section, an optional output section and either a container invocation or a list of steps where each step invokes another template.

Note that the container section of the workflow spec will accept the same options as the container section of a pod spec, including but not limited to environment variables, secrets, and volume mounts. Similarly, for volume claims and volumes.
