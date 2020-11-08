import {Sequence, TemplateRef, WorkflowSpec} from '../../../../models';
import {Graph} from '../graph/types';
import {icons} from './icons';

function addCommonDependencies(
    x: {withItems?: string[]; withParam?: string; withSequence?: Sequence; template?: string; when?: string; templateRef?: TemplateRef; onExit?: string},
    id: string,
    g: Graph
) {
    if (x.withItems) {
        const itemsId = id + '/withItems';
        g.nodes.set(itemsId, {
            label: JSON.stringify(x.withItems),
            type: 'items',
            icon: icons.withItems
        });
        g.edges.set({v: id, w: itemsId}, {label: 'loop', classNames: 'related'});
    }
    if (x.withParam) {
        const paramId = id + '/withParam';
        g.nodes.set(paramId, {
            label: x.withParam,
            type: 'param',
            icon: icons.withParam
        });
        g.edges.set({v: id, w: paramId}, {label: 'loop', classNames: 'related'});
    }
    if (x.withSequence) {
        const sequenceId = id + '/withSequence';
        g.nodes.set(sequenceId, {
            label: x.withSequence.count ? '0..' + x.withSequence.count : x.withSequence.start + '..' + x.withSequence.end,
            type: 'sequence',
            icon: icons.withSequence
        });
        g.edges.set({v: id, w: sequenceId}, {label: 'loop', classNames: 'related'});
    }
    if (x.template) {
        g.edges.set({v: id, w: 'Template/' + x.template}, {classNames: 'related'});
    }
    if (x.when) {
        const whenId = id + '/when';
        g.nodes.set(whenId, {icon: icons.when, label: x.when, type: 'when'});
        g.edges.set({v: id, w: whenId}, {label: 'when'});
    }
    if (x.templateRef) {
        const templateRefId = 'TemplateRef/' + x.templateRef.name + '/' + x.templateRef.template;
        g.nodes.set(templateRefId, {
            label: x.templateRef.name,
            type: 'tmpl-ref',
            icon: x.templateRef.clusterScope ? icons.clusterTemplateRef : icons.templateRef
        });
        g.edges.set({v: id, w: templateRefId}, {});
    }
    if (x.onExit) {
        const onExitId = 'OnExit/' + id;
        g.nodes.set(onExitId, {label: 'on-exit', type: 'on-exit', icon: icons.onExit});
        g.edges.set({v: onExitId, w: 'Template/' + x.onExit}, {classNames: 'related'});
        g.edges.set({v: onExitId, w: id}, {});
    }
}

export const workflowSpecGraph = (s: WorkflowSpec): Graph => {
    const g: Graph = new Graph();
    if (s.entrypoint) {
        g.nodes.set('Workflow', {label: 'workflow', type: 'workflow', icon: icons.workflow});
        g.edges.set({v: 'Workflow', w: 'Template/' + s.entrypoint}, {label: 'entrypoint'});
    }
    if (s.arguments) {
        if (s.arguments.parameters) {
            const id = 'Parameters';
            g.nodes.set(id, {
                icon: icons.parameters,
                label: s.arguments.parameters.map(x => x.name).join(','),
                type: 'params'
            });
            g.edges.set({v: 'Workflow', w: id}, {classNames: 'related'});
        }
        if (s.arguments.artifacts) {
            const id = 'Artifacts';
            g.nodes.set(id, {
                icon: icons.artifacts,
                label: s.arguments.artifacts.map(x => x.name).join(','),
                type: 'artifacts'
            });
            g.edges.set({v: 'Workflow', w: id}, {classNames: 'related'});
        }
    }
    if (s.onExit) {
        g.nodes.set('OnExit', {label: 'on-exit', type: 'on-exit', icon: icons.onExit});
        g.edges.set({v: 'OnExit', w: 'Template/' + s.onExit}, {});
        g.edges.set({v: 'Workflow', w: 'OnExit'}, {classNames: 'related'});
    }
    s.templates.forEach(t => {
        const type = t.dag ? 'dag' : t.steps ? 'steps' : t.container ? 'container' : t.script ? 'script' : t.resource ? 'resource' : t.suspend ? 'suspend' : 'unknown';
        const templateId = 'Template/' + t.name;
        g.nodes.set(templateId, {label: t.name, type, icon: icons[type]});
        if (t.dag) {
            const inDegree: {[id: string]: boolean} = {};
            t.dag.tasks.filter(task => !!task.dependencies).forEach(task => task.dependencies.forEach(w => (inDegree[w] = true)));
            g.nodeGroups.set(templateId, new Set());
            t.dag.tasks.forEach(task => {
                const taskId = templateId + '/' + task.name;
                g.nodes.set(taskId, {label: task.name, type: 'task', icon: icons.task});
                // root node?
                if (!inDegree[task.name]) {
                    g.edges.set({v: templateId, w: taskId}, {});
                }
                if (task.dependencies) {
                    task.dependencies.forEach(dependencyName => {
                        g.edges.set({v: taskId, w: templateId + '/' + dependencyName}, {});
                    });
                }
                addCommonDependencies(task, taskId, g);
                g.nodeGroups.get(templateId).add(taskId);
            });
        } else if (t.steps) {
            t.steps.forEach((parallelStep, i) => {
                const parallelStepId = templateId + '/' + i;
                g.nodeGroups.set(parallelStepId, new Set());
                parallelStep.forEach((step, j) => {
                    const stepId = parallelStepId + '/' + j;
                    g.nodes.set(stepId, {label: step.name, type: 'step', icon: icons.step});
                    g.edges.set({v: templateId, w: stepId}, {label: 'step ' + j});
                    addCommonDependencies(step, stepId, g);
                    g.nodeGroups.get(parallelStepId).add(stepId);
                });
            });
        }
    });
    return g;
};
