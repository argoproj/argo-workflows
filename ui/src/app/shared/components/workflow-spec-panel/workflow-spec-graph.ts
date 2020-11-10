import {Sequence, TemplateRef, WorkflowSpec} from '../../../../models';
import {Graph} from '../graph/types';
import {icons} from './icons';
import {ID} from './id';

function addCommonDependencies(
    x: {withItems?: string[]; withParam?: string; withSequence?: Sequence; template?: string; when?: string; templateRef?: TemplateRef; onExit?: string},
    id: string,
    g: Graph
) {
    if (x.withItems) {
        const itemsId = id + '#withItems';
        g.nodes.set(itemsId, {
            label: JSON.stringify(x.withItems),
            type: 'items',
            icon: icons.withItems
        });
        g.edges.set({v: id, w: itemsId}, {label: 'loop', classNames: 'related'});
    }
    if (x.withParam) {
        const paramId = id + '#withParam';
        g.nodes.set(paramId, {
            label: x.withParam,
            type: 'param',
            icon: icons.withParam
        });
        g.edges.set({v: id, w: paramId}, {label: 'loop', classNames: 'related'});
    }
    if (x.withSequence) {
        const sequenceId = id + '#withSequence';
        g.nodes.set(sequenceId, {
            label: x.withSequence.count ? '0..' + x.withSequence.count : x.withSequence.start + '..' + x.withSequence.end,
            type: 'sequence',
            icon: icons.withSequence
        });
        g.edges.set({v: id, w: sequenceId}, {label: 'loop', classNames: 'related'});
    }
    if (x.template) {
        g.edges.set({v: id, w: ID.join('Template', x.template)}, {classNames: 'related'});
    }
    if (x.when) {
        const whenId = id + '#when';
        g.nodes.set(whenId, {icon: icons.when, label: x.when, type: 'when'});
        g.edges.set({v: id, w: whenId}, {label: 'when'});
    }
    if (x.templateRef) {
        const templateRefId = ID.join('TemplateRef', x.templateRef.name + '/' + x.templateRef.template);
        g.nodes.set(templateRefId, {
            label: x.templateRef.name,
            type: 'tmpl-ref',
            icon: x.templateRef.clusterScope ? icons.clusterTemplateRef : icons.templateRef
        });
        g.edges.set({v: id, w: templateRefId}, {});
    }
    if (x.onExit) {
        const onExitId = id + '#onExit';
        g.nodes.set(onExitId, {label: 'on-exit', type: 'on-exit', icon: icons.onExit});
        g.edges.set({v: onExitId, w: 'Template/' + x.onExit}, {classNames: 'related'});
        g.edges.set({v: onExitId, w: id}, {});
    }
}

export const workflowSpecGraph = (s: WorkflowSpec): Graph => {
    const g: Graph = new Graph();
    const workflowId = ID.join('Workflow');
    g.nodes.set(workflowId, {label: 'workflow', type: 'workflow', icon: icons.workflow});
    if (s.entrypoint) {
        g.edges.set({v: workflowId, w: ID.join('Template', s.entrypoint)}, {label: 'entrypoint'});
    }
    if (s.arguments) {
        if (s.arguments.parameters) {
            const parametersId = ID.join('Parameters');
            g.nodes.set(parametersId, {
                icon: icons.parameters,
                label: s.arguments.parameters.map(x => x.name).join(','),
                type: 'params'
            });
            g.edges.set({v: 'Workflow', w: parametersId}, {classNames: 'related'});
        }
        if (s.arguments.artifacts) {
            const artifactsId = ID.join('Artifacts');
            g.nodes.set(artifactsId, {
                icon: icons.artifacts,
                label: s.arguments.artifacts.map(x => x.name).join(','),
                type: 'artifacts'
            });
            g.edges.set({v: 'Workflow', w: artifactsId}, {classNames: 'related'});
        }
    }
    if (s.onExit) {
        const onExitId = ID.join('OnExit');
        g.nodes.set(onExitId, {label: 'on-exit', type: 'on-exit', icon: icons.onExit});
        g.edges.set({v: onExitId, w: ID.join('Template', s.onExit)}, {});
        g.edges.set({v: workflowId, w: onExitId}, {classNames: 'related'});
    }
    if (s.workflowTemplateRef) {
        const templateRefId = ID.join('WorkflowTemplateRef');
        g.nodes.set(templateRefId, {
            label: s.workflowTemplateRef.name,
            type: 'tmpl-ref',
            icon: s.workflowTemplateRef.clusterScope ? icons.clusterTemplateRef : icons.templateRef
        });
        g.edges.set({v: 'Workflow', w: templateRefId}, {});
    }
    (s.templates || []).forEach(template => {
        const type = template.dag
            ? 'dag'
            : template.steps
            ? 'steps'
            : template.container
            ? 'container'
            : template.script
            ? 'script'
            : template.resource
            ? 'resource'
            : template.suspend
            ? 'suspend'
            : 'unknown';
        const templateId = ID.join('Template', template.name);
        g.nodes.set(templateId, {label: template.name, type, icon: icons[type]});
        if (template.dag) {
            const inDegree: {[id: string]: boolean} = {};
            template.dag.tasks.filter(task => !!task.dependencies).forEach(task => task.dependencies.forEach(w => (inDegree[w] = true)));
            g.nodeGroups.set(templateId, new Set());
            template.dag.tasks.forEach(task => {
                const taskId = ID.join('Task', template.name + ',' + task.name);
                g.nodes.set(taskId, {label: task.name, type: 'task', icon: icons.task});
                // root node?
                if (!inDegree[task.name]) {
                    g.edges.set({v: templateId, w: taskId}, {});
                }
                if (task.dependencies) {
                    task.dependencies.forEach(dependencyName => {
                        g.edges.set({v: taskId, w: ID.join('Task', template.name + ',' + dependencyName)}, {});
                    });
                }
                addCommonDependencies(task, taskId, g);
                g.nodeGroups.get(templateId).add(taskId);
            });
        } else if (template.steps) {
            template.steps.forEach((group, i) => {
                const groupId = ID.join('StepGroup', template.name + ',' + i);
                g.nodes.set(groupId, {label: 'group ' + i, type: 'group', icon: icons.stepGroup});
                g.nodeGroups.set(groupId, new Set());
                if (i === template.steps.length - 1) {
                    g.edges.set({v: templateId, w: groupId}, {});
                }
                const parentGroupId = ID.join('StepGroup', template.name + ',/' + (i - 1));
                group.forEach((step, j) => {
                    const stepId = ID.join('Step', groupId + ',' + j);
                    g.nodes.set(stepId, {label: step.name, type: 'step', icon: icons.step});
                    g.edges.set({v: groupId, w: stepId}, {});
                    g.nodeGroups.get(groupId).add(stepId);
                    if (i > 0) {
                        g.edges.set({v: stepId, w: parentGroupId}, {});
                    }
                    addCommonDependencies(step, stepId, g);
                });
            });
        }
    });
    return g;
};
