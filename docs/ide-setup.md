# IDE Set-Up

## Validating Argo YAML againt the JSON Schema

Argo provides a [JSON Schema](https://tbc/schema.json) that enables validation of YAML resources in your IDE.

### IntelliJ IDEA (Community & Utimate Editions)

YAML validation is supported natively in IDEA.

Configure your IDE to reference the Argo schema and map it to your Argo YAML files:

![IDEA Configure Schema](assets/intellij-ide-step-1-config.png)
- The schema is located at [https://tbc/schema.json](https://tbc/schema.json).
- Specify a file glob pattern that locates **your** Argo files. The example glob here is for the Argo Github project!
- Note that you may need to restart IDEA to pick up the changes.

That's it. Open an Argo YAML file and you should see smarter behaviour, including type errors and context-sensitive autocomplete.

![IDEA Example Functionality](assets/intellij-ide-step-1-example-functionality.png)

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

## Validating Argo YAML using CRD Validation in IntelliJ Ultimate Edition

:warning: This method is no longer recommended due to challenges with CRD validation, relatively complex setup and limited support across IDEs.

Load the full Custom Resource Definitions (CRDs) into IntelliJ Ultimate Edition, an IDE that support CRD validation via the Kubernetes plugin.

CRD URLs:

* [ClusterWorkflowTemplate](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_clusterworkflowtemplates.yaml)
* [WorkflowTemplate](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_workflowtemplates.yaml)
* [CronWorkflow](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_cronworkflows.yaml)
* [Workflow](https://raw.githubusercontent.com/argoproj/argo/master/manifests/base/crds/full/argoproj.io_workflows.yaml) 

Install the Kubernetes plugin:

![Step 1](assets/ide-step-1.png)

Add the CRD URLs to the Kubernetes configuration panel (choose “IDE” for the scope):

![Step 2](assets/ide-step-2.png)

Finally, open your CRDs and verify no errors appear, example:

![Step 3](assets/ide-step-3.png)
