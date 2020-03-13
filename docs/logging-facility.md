# Logging Facility

![alpha](assets/alpha.svg)

> v2.7 and after

You can configure Argo Server to show deep-links to your logging facility in the user interface. 

```
argo server --help
...
      --logging-facility-name string                 The name of your logging facility
      --logging-facility-templates-pod string        The templates for your logging facility for pods
      --logging-facility-templates-workflow string   The templates for your logging facility for workflows
...```