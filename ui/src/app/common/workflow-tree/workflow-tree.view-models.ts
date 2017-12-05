import * as models from '../../models';

export interface NodeInfo {
  template: models.Template;
  nodeName: string;
  stepName: string;
  status: models.NodeStatus;
  children: NodeInfo[][];
}

export interface ArtifactInfo extends models.Artifact {
  stepName: string;
  nodeName: string;
  downloadUrl: string;
}

export class WorkflowTree {
  private templateByNames = new Map<string, models.Template>();
  private rootNode: NodeInfo;

  constructor(public workflow: models.Workflow) {
    workflow.spec.templates.forEach(template => this.templateByNames.set(template.name, template));
    this.rootNode = this.createRoot();
  }

  private createRoot(): NodeInfo {
    return this.getNodeInfo(this.workflow.status.nodes[this.workflow.metadata.name], this.workflow.metadata.name, true);
  }

  private getNodeInfo(nodeStatus: models.NodeStatus, nodeName: string, root = false): NodeInfo {
    const name = nodeStatus.name;
    const step = this.getStepByName(name);
    const meta = { template: this.templateByNames.get(step.template), stepName: step.name };
    const info = {
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
    if (info.children.length === 0 && root) {
      info.children.push([Object.assign({}, info, {children: [], stepName: '_'})]);
    }
    return info;
  }

  public getStepByName(name: string): models.WorkflowStep {
    name = name.indexOf('(') > -1 ? name.substring(0, name.indexOf('(')) : name;
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

  public getArtifacts(): ArtifactInfo[] {
    return Object.keys(this.workflow.status.nodes)
    .map(nodeName => {
      const node = this.workflow.status.nodes[nodeName];
      const items = (node.outputs || { artifacts: [] }).artifacts || <models.Artifact[]>[];
      return items.map(item => Object.assign({}, item, {
        downloadUrl: `/api/workflows/${this.workflow.metadata.namespace}/${this.workflow.metadata.name}/artifacts/${nodeName}/${item.name}`,
        stepName: node.name,
        dateCreated: node.finishedAt,
        nodeName
      }));
    })
    .reduce((first, second) => first.concat(second), []) || [];
  }
}
