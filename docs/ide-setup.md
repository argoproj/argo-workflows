# IDE Set-Up

## Validating Argo YAML against the JSON Schema

Argo provides a [JSON Schema](https://raw.githubusercontent.com/argoproj/argo-workflows/master/api/jsonschema/schema.json) that enables validation of YAML resources in your IDE.

### JetBrains IDEs (Community & Ultimate Editions)

YAML validation is supported natively in IDEA.

Configure your IDE to reference the Argo schema and map it to your Argo YAML files:

![JetBrains IDEs Configure Schema](assets/jetbrains-ide-step-1-config.png)

- The schema is located [here](https://raw.githubusercontent.com/argoproj/argo-workflows/master/api/jsonschema/schema.json).
- Specify a file glob pattern that locates **your** Argo files. The example glob here is for the Argo Github project!
- Note that you may need to restart IDEA to pick up the changes.

That's it. Open an Argo YAML file and you should see smarter behavior, including type errors and context-sensitive auto-complete.

![JetBrains IDEs Example Functionality](assets/jetbrains-ide-step-1-example-functionality.png)

### JetBrains IDEs (Community & Ultimate Editions) + Kubernetes Plugin

If you have the [JetBrains Kubernetes Plugin](https://plugins.jetbrains.com/plugin/10485-kubernetes)
installed in your IDE, the validation can be configured in the Kubernetes plugin settings
instead of using the internal JSON schema file validator.

![JetBrains IDEs Configure Schema with Kubernetes Plugin](assets/jetbrains-ide-step-1-kubernetes-config.png)

Unlike the previous JSON schema validation method, the plugin detects the necessary validation
based on Kubernetes resource definition keys and does not require a file glob pattern.
Like the previously described method:

- The schema is located [here](https://raw.githubusercontent.com/argoproj/argo-workflows/master/api/jsonschema/schema.json).
- Note that you may need to restart IDEA to pick up the changes.

### VSCode

The [Red Hat YAML](https://github.com/redhat-developer/vscode-yaml) plugin will provide error highlighting and auto-completion for Argo resources.

Install the Red Hat YAML plugin in VSCode and open extension settings:

![VSCode Install Plugin](assets/vscode-ide-step-1-install-plugin.png)

Open the YAML schema settings:

![VSCode YAML Schema Settings](assets/vscode-ide-step-2-schema-settings.png)

Add the Argo schema setting `yaml.schemas`:

![VSCode Specify Argo Schema](assets/vscode-ide-step-3-spec-schema.png)

- The schema is located [here](https://raw.githubusercontent.com/argoproj/argo-workflows/master/api/jsonschema/schema.json).
- Specify a file glob pattern that locates **your** Argo files. The example glob here is for the Argo Github project!
- Note that other defined schema with overlapping glob patterns may cause errors.

That's it. Open an Argo YAML file and you should see smarter behavior, including type errors and context-sensitive auto-complete.

![VScode Example Functionality](assets/vscode-ide-step-4-example-functionality.png)
