import * as models from '../../models';

export class NodeInfo {
  private static getNodeInfo(
      templateName: string,
      stepName: string,
      fullStepName: string,
      context: { templateByName: Map<string, models.Template>, statusByFullName: Map<string, models.NodeStatus> }) {
    const template = context.templateByName.get(templateName);
    const children: NodeInfo[][] = (template.steps || []).map((stepGroup, i) =>
        stepGroup.map(step => NodeInfo.getNodeInfo(step.template, step.name, `${fullStepName}[${i}].${step.name}`, context)));

    const nodeStatus = context.statusByFullName.get(fullStepName);
    const nodeInfo = new NodeInfo(
      template,
      stepName,
      nodeStatus,
      children,
    );
    return nodeInfo;
  }

  private constructor(
    public template: models.Template,
    public stepName: string,
    public status: models.NodeStatus,
    public children: NodeInfo[][]) {
  }

  public static create(workflow: models.Workflow): NodeInfo {
    const templateByName = new Map<string, models.Template>();
    const statusByFullName = new Map<string, models.NodeStatus>();
    workflow.spec.templates.forEach(template => templateByName.set(template.name, template));
    Object.keys(workflow.status.nodes)
      .map(name => workflow.status.nodes[name])
      .forEach(status => statusByFullName.set(status.name, status));
    return NodeInfo.getNodeInfo(workflow.spec.entrypoint, '', workflow.metadata.name, { templateByName, statusByFullName });
  }
}
