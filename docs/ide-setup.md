# IDE Set-Up

## How To Use CRD Validation With Your Editor

Use either VS Code or Kubernetes and load the full Custom Resource Definitions (CRDs) into an IDE that support CRD validation (e.g IntelliJ, VS Code).

CRD URLs:

* [ClusterWorkflowTemplate](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_clusterworkflowtemplates.yaml)
* [WorkflowTemplate](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_workflowtemplates.yaml)
* [CronWorkflow](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_cronworkflows.yaml)
* [Workflow](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_workflows.yaml) 

### IntelliJ

Install the Kubernetes plugin:

![Step 1](assets/ide-step-1.png)

Add the CRD URLs to the Kubernetes configuration panel (choose “IDE” for the scope):

![Step 2](assets/ide-step-2.png)

Finally, open your CRDs and verify no errors appear, example:

![Step 3](assets/ide-step-3.png)

### VSCode

The [Red Hat YAML](https://github.com/redhat-developer/vscode-yaml) plugin will provide error highlighting and autocompletion for Argo resources.

Install the Red Hat YAML plugin in VSCode and open extension settings:

![VSCode Install Plugin](assets/vscode-ide-step-1-install-plugin.png)

Open the YAML schemas settings:

![VSCode YAML Schema Settings](assets/vscode-ide-step-2-schema-settings.png)

Add the Argo schema setting `yaml.schemas`:

![VSCode Specify Argo Schema](assets/vscode-ide-step-3-spec-schema.png)

- The schema is located at [https://tbc/schema.json](https://tbc/schema.json).
- Specify a file glob pattern that locates **your** Argo files. The example glob here is for the Argo Github project!
- Note that other defined schemas with overlapping glob patterns may cause errors.

That's it. Open an Argo YAML file and you should see smarter behaviour, including type errors and context-sensitive autocomplete.

![VScode Example Functionality](assets/vscode-ide-step-4-example-functionality.png)