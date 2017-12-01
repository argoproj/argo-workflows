import * as models from '../../models';

export interface NodeInfo {
  template: models.Template;
  nodeName: string;
  stepName: string;
  status: models.NodeStatus;
  children: NodeInfo[][];
}

export class WorkflowTree {
  private templateByNames = new Map<string, models.Template>();
  private rootNode: NodeInfo;

  constructor(public workflow: models.Workflow) {
    workflow.spec.templates.forEach(template => this.templateByNames.set(template.name, template));
    this.rootNode = this.createRoot();
  }

  private createRoot(): NodeInfo {
    return this.getNodeInfo(this.workflow.status.nodes[this.workflow.metadata.name], this.workflow.metadata.name);
  }

  private getNodeInfo(nodeStatus: models.NodeStatus, nodeName: string): NodeInfo {
    let name = nodeStatus.name;
    name = name.indexOf('(') > -1 ? name.substring(0, name.indexOf('(')) : name;
    const step = this.getStepByName(name);
    const meta = { template: this.templateByNames.get(step.template), stepName: step.name };
    return {
      nodeName,
      template: meta.template,
      stepName: meta.stepName,
      status: nodeStatus,
      children: (nodeStatus.children || []).map(groupName => {
        const groupStatus = this.workflow.status.nodes[groupName];
        return (groupStatus.children || []).map(childName => ({
          status: this.workflow.status.nodes[childName], nodeName: childName
        })).map(item => this.getNodeInfo(item.status, item.nodeName));
      })
    };
  }

  public getStepByName(name: string): models.WorkflowStep {
    const rootStep: models.WorkflowStep = {
      name: '', template: this.workflow.spec.entrypoint, arguments: null, withItems: null,  when: ''};
    const queue = [{ fullName: this.workflow.metadata.name, step: rootStep }];
    while (queue.length > 0) {
      const next = queue.shift();
      if (next.fullName === name) {
        return next.step;
      } else {
        const template = this.templateByNames.get(next.step.template);
        for (let i = 0; i < (template.steps || []).length; i++) {
          for (const childStep of template.steps[i]) {
            queue.push({fullName: `${next.fullName}[${i}].${childStep.name}`, step: childStep});
          }
        }
      }
    }
    return null;
  }

  public get root(): NodeInfo {
    return this.rootNode;
  }
}
