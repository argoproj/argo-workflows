import * as React from 'react';

import {genres} from '../../../workflows/components/workflow-dag/genres';
import {WorkflowDagRenderOptionsPanel} from '../../../workflows/components/workflow-dag/workflow-dag-render-options-panel';
import {DAGTask, Template, Workflow} from '../../models';
import {GraphPanel} from '../graph/graph-panel';
import {Graph} from '../graph/types';

export function GraphViewer({workflow}: {workflow: Workflow}) {
    function populateGraphFromWorkflow(workflow: Workflow, name: string): Graph {
        const graph = new Graph();

        const templates = workflow.spec.templates;
        const templateMap = new Map();
        const templateLeafMap = new Map();

        templates.forEach((template: Template) => {
            templateMap.set(template.name, template);
            templateLeafMap.set(template.name, getLeafNodes(template));
        });

        const entrypoint = workflow.spec.entrypoint;

        function processTemplate(templateName: string, parentTaskName: string = null): Graph {
            const template = templateMap.get(templateName);

            if (!template) {
                console.error(`Template "${templateName}" not found.`);
                return;
            }

            if (template.dag) {
                graph.nodes.set(parentTaskName, {
                    label: parentTaskName,
                    genre: 'DAG',
                    classNames: 'Pending',
                    progress: 0,
                    icon: 'clock'
                });

                template.dag.tasks.forEach((task: DAGTask) => {
                    const taskName = parentTaskName ? `${parentTaskName}.${task.name}` : task.name;

                    graph.nodes.set(taskName, {
                        label: task.name,
                        genre: templateMap.has(task.template) && templateMap.get(task.template).dag ? 'DAG' : 'Pod',
                        classNames: 'Pending',
                        progress: 0,
                        icon: 'clock'
                    });

                    if (task.depends) {
                        const dependencies = parseDepends(task.depends);
                        dependencies.forEach((dep: string) => {
                            const dependancyName = `${parentTaskName}.${dep}`;
                            if (graph.nodes.get(dependancyName).genre !== 'DAG') {
                                graph.edges.set({v: dependancyName, w: taskName}, {});
                            } else {
                                // Override edge to be from last element of DAG if dep is a DAG.
                                const depTemplate = getTemplateNameFromTask(template.dag, dep);
                                const templateLeafNodes = templateLeafMap.get(depTemplate);
                                if (templateLeafNodes) {
                                    templateLeafNodes.forEach((leaf: string) => {
                                        graph.edges.set({v: `${dependancyName}.${leaf}`, w: taskName}, {});
                                    });
                                }
                            }
                        });
                    } else {
                        // Add an edge from the parent template to the task
                        graph.edges.set({v: parentTaskName, w: taskName}, {});
                    }

                    // Recursively process the task's template if it's a nested DAG
                    if (templateMap.has(task.template) && templateMap.get(task.template).dag) {
                        processTemplate(task.template, taskName);
                    }
                });
            } else {
                // Edge case when workflow consists of only one template
                if (templateMap.size === 1) {
                    templateName = parentTaskName;
                    parentTaskName = '';
                }

                graph.nodes.set(templateName, {
                    label: templateName,
                    genre: 'Pod',
                    classNames: 'Pending',
                    progress: 0,
                    icon: 'clock'
                });

                if (parentTaskName) {
                    graph.edges.set({v: parentTaskName, w: templateName}, {});
                }
            }
        }
        processTemplate(entrypoint, name);

        return graph;
    }
    function parseDepends(dependsString: string) {
        return dependsString.split(/&&|\|\|/).map(s => s.trim());
    }

    function generateNamePostfix(len: number): string {
        const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
        let result = '';
        for (let i = 0; i < len; i++) {
            const randomIndex = Math.floor(Math.random() * chars.length);
            result += chars[randomIndex];
        }
        return result;
    }

    function getLeafNodes(template: Template) {
        const allTaskNames = new Set<string>();
        const dependentTaskNames = new Set<string>();

        if (template.dag) {
            template.dag.tasks.forEach((task: DAGTask) => {
                allTaskNames.add(task.name);
                if (task.depends) {
                    const dependencies = parseDepends(task.depends);
                    dependencies.forEach((dep: string) => {
                        dependentTaskNames.add(dep);
                    });
                }
            });
        }
        const independentTasks: string[] = Array.from(allTaskNames).filter(taskName => !dependentTaskNames.has(taskName));

        return independentTasks ? independentTasks : null;
    }

    function getTemplateNameFromTask(dag: {tasks: {name: string; template: string}[]}, taskName: string): string | null {
        const task = dag.tasks.find(t => t.name === taskName);
        return task ? task.template : null;
    }
    const state = {
        expandNodes: new Set(''),
        showArtifacts: localStorage.getItem('showArtifacts') !== 'false',
        showInvokingTemplateName: localStorage.getItem('showInvokingTemplateName') === 'true',
        showTemplateRefsGrouping: localStorage.getItem('showTemplateRefsGrouping') === 'true'
    };

    const name = workflow.metadata.name ?? `${workflow.metadata.generateName}${generateNamePostfix(5)}`;
    const graph = populateGraphFromWorkflow(workflow, name);

    return (
        <GraphPanel
            storageScope='workflow-dag'
            graph={graph}
            nodeGenresTitle={'Node Type'}
            nodeGenres={genres}
            nodeClassNamesTitle={'Node Phase'}
            nodeClassNames={{Pending: true}}
            nodeTagsTitle={'Template'}
            nodeTags={{[name]: true}}
            nodeSize={64}
            defaultIconShape='circle'
            hideNodeTypes={true}
            hideOptions={true}
            options={<WorkflowDagRenderOptionsPanel {...state} onChange={workflowDagRenderOptions => this.saveOptions(workflowDagRenderOptions)} />}
        />
    );
}
