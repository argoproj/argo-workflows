import {Sequence, TemplateRef, WorkflowSpec} from '../../../../models';
import {Graph} from '../graph/types';
import {icons} from './icons';
import {artifactsId, idForStepGroup, idForSteps, idForTask, idForTemplate, idForTemplateRef, onExitId, parametersId, workflowId, workflowTemplateRefId} from './id';

function addCommonDependencies(
    x: {withItems?: string[]; withParam?: string; withSequence?: Sequence; template?: string; when?: string; templateRef?: TemplateRef; onExit?: string; depends?: string},
    id: string,
    g: Graph
) {
    if (x.withItems) {
        const itemsId = id + '#withItems';
        g.nodes.set(itemsId, {
            label: JSON.stringify(x.withItems),
            genre: 'items',
            icon: icons.withItems
        });
        g.edges.set({v: itemsId, w: id}, {label: 'loop', classNames: 'related'});
    }
    if (x.withParam) {
        const paramId = id + '#withParam';
        g.nodes.set(paramId, {
            label: x.withParam,
            genre: 'param',
            icon: icons.withParam
        });
        g.edges.set({v: paramId, w: id}, {label: 'loop', classNames: 'related'});
    }
    if (x.withSequence) {
        const sequenceId = id + '#withSequence';
        g.nodes.set(sequenceId, {
            label: x.withSequence.count ? '0..' + x.withSequence.count : x.withSequence.start + '..' + x.withSequence.end,
            genre: 'sequence',
            icon: icons.withSequence
        });
        g.edges.set({v: sequenceId, w: id}, {label: 'loop', classNames: 'related'});
    }
    if (x.template) {
        g.edges.set({v: idForTemplate(x.template), w: id}, {classNames: 'related'});
    }
    if (x.when) {
        const whenId = id + '#when';
        g.nodes.set(whenId, {icon: icons.when, label: x.when, genre: 'when'});
        g.edges.set({v: whenId, w: id}, {label: 'when', classNames: 'related'});
    }
    if (x.depends) {
        const dependsId = id + '#depends';
        g.nodes.set(dependsId, {icon: icons.depends, label: x.depends, genre: 'depends'});
        g.edges.set({v: dependsId, w: id}, {label: 'depends'});
    }
    if (x.templateRef) {
        const templateRefId = idForTemplateRef(x.templateRef.name, x.templateRef.template);
        g.nodes.set(templateRefId, {
            label: x.templateRef.name,
            genre: 'tmpl-ref',
            icon: x.templateRef.clusterScope ? icons.clusterTemplateRef : icons.templateRef
        });
        g.edges.set({v: templateRefId, w: id}, {});
    }
    if (x.onExit) {
        const exitId = id + '#onExit';
        g.nodes.set(exitId, {label: 'on-exit', genre: 'on-exit', icon: icons.onExit});
        g.edges.set({v: 'Template/' + x.onExit, w: exitId}, {classNames: 'related'});
        g.edges.set({v: id, w: exitId}, {});
    }
}

export const workflowSpecGraph = (s: WorkflowSpec): Graph => {
    const g = new Graph();
    g.nodes.set(workflowId, {label: 'workflow', genre: 'workflow', icon: icons.workflow});
    if (s.entrypoint) {
        g.edges.set({v: idForTemplate(s.entrypoint), w: workflowId}, {label: 'entrypoint'});
    }
    if (s.arguments) {
        if (s.arguments.parameters) {
            g.nodes.set(parametersId, {
                icon: icons.parameters,
                label: s.arguments.parameters.map(x => x.name).join(','),
                genre: 'params'
            });
            g.edges.set({v: parametersId, w: workflowId}, {classNames: 'related'});
        }
        if (s.arguments.artifacts) {
            g.nodes.set(artifactsId, {
                icon: icons.artifacts,
                label: s.arguments.artifacts.map(x => x.name).join(','),
                genre: 'artifacts'
            });
            g.edges.set({v: artifactsId, w: workflowId}, {classNames: 'related'});
        }
    }
    if (s.onExit) {
        g.nodes.set(onExitId, {label: 'on-exit', genre: 'on-exit', icon: icons.onExit});
        g.edges.set({v: idForTemplate(s.onExit), w: onExitId}, {});
        g.edges.set({v: workflowId, w: onExitId}, {classNames: 'related'});
    }
    if (s.workflowTemplateRef) {
        g.nodes.set(workflowTemplateRefId, {
            label: s.workflowTemplateRef.name,
            genre: 'tmpl-ref',
            icon: s.workflowTemplateRef.clusterScope ? icons.clusterTemplateRef : icons.templateRef
        });
        g.edges.set({v: workflowTemplateRefId, w: 'Workflow'}, {});
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
        const templateId = idForTemplate(template.name);
        g.nodes.set(templateId, {label: template.name, genre: type, icon: icons[type]});
        if (template.dag) {
            const inDegree: {[id: string]: boolean} = {};
            template.dag.tasks.filter(task => !!task.dependencies).forEach(task => task.dependencies.forEach(w => (inDegree[w] = true)));
            g.nodeGroups.set(templateId, new Set());
            template.dag.tasks.forEach(task => {
                const taskId = idForTask(template.name, task.name);
                g.nodes.set(taskId, {
                    label: task.name,
                    genre: 'task',
                    icon: icons.task
                });
                // root node?
                if (!inDegree[task.name]) {
                    g.edges.set({v: taskId, w: templateId}, {});
                }
                if (task.dependencies) {
                    task.dependencies.forEach(dependencyName => {
                        g.edges.set({v: idForTask(template.name, dependencyName), w: taskId}, {});
                    });
                }
                addCommonDependencies(task, taskId, g);
                g.nodeGroups.get(templateId).add(taskId);
            });
        } else if (template.steps) {
            template.steps.forEach((group, i) => {
                const groupId = idForStepGroup(template.name, i);
                g.nodes.set(groupId, {label: 'group ' + i, genre: 'group', icon: icons.stepGroup});
                g.nodeGroups.set(groupId, new Set());
                const firstGroup = i === 0;
                const lastGroup = i === template.steps.length - 1;
                if (lastGroup) {
                    g.edges.set({v: groupId, w: templateId}, {});
                }
                const lastGroupId = idForStepGroup(template.name, i - 1);
                group.forEach((step, j) => {
                    const stepId = idForSteps(template.name, i, j);
                    g.nodes.set(stepId, {
                        label: step.name + (step.template ? ': ' + step.template : ''),
                        genre: 'step',
                        icon: icons.step
                    });
                    g.edges.set({v: stepId, w: groupId}, {});
                    g.nodeGroups.get(groupId).add(stepId);
                    if (!firstGroup) {
                        g.edges.set({v: lastGroupId, w: stepId}, {});
                    }
                    addCommonDependencies(step, stepId, g);
                });
            });
        }
    });
    return g;
};
