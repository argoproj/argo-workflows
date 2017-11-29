import * as models from '../../models';

export interface NodeInfo {
  template: models.Template;
  stepName: string;
  status: models.NodeStatus;
  children: NodeInfo[][];
}

export function getWorkflowTree(workflow: models.Workflow): NodeInfo {
  const templateByNames = new Map<string, models.Template>();
  workflow.spec.templates.forEach(template => templateByNames.set(template.name, template));

  function getStepByName(name: string): models.WorkflowStep {
    const rootStep: models.WorkflowStep = { name: '', template: workflow.spec.entrypoint, arguments: null, withItems: null,  when: ''};
    const queue = [{ fullName: workflow.metadata.name, step: rootStep }];
    while (queue.length > 0) {
      const next = queue.shift();
      if (next.fullName === name) {
        return next.step;
      } else {
        const template = templateByNames.get(next.step.template);
        for (let i = 0; i < (template.steps || []).length; i++) {
          for (const childStep of template.steps[i]) {
            queue.push({fullName: `${next.fullName}[${i}].${childStep.name}`, step: childStep});
          }
        }
      }
    }
    return null;
  }

  function getStepMeta(nodeStatus: models.NodeStatus) {
    let name = nodeStatus.name;
    name = name.indexOf('(') > -1 ? name.substring(0, name.indexOf('(')) : name;
    const step = getStepByName(name);
    return {
      template: templateByNames.get(step.template),
      stepName: step.name,
    };
  }

  function getNodeInfo(nodeStatus: models.NodeStatus): NodeInfo {
    const meta = getStepMeta(nodeStatus);
    return {
      template: meta.template,
      stepName: meta.stepName,
      status: nodeStatus,
      children: (nodeStatus.children || []).map(groupName => {
        const groupStatus = workflow.status.nodes[groupName];
        return (groupStatus.children || []).map(name => workflow.status.nodes[name]).map(status => getNodeInfo(status));
      })
    };
  }

  return getNodeInfo(workflow.status.nodes[workflow.metadata.name]);
}
