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

        // Map templates by their names for quick lookup
        templates.forEach((template: Template) => {
            templateMap.set(template.name, template);
        });

        // Start with the entrypoint
        const entrypoint = workflow.spec.entrypoint;

        // Recursive function to process a template
        function processTemplate(templateName: string, parentTaskName: string = null): Graph {
            const template = templateMap.get(templateName);

            if (!template) {
                console.error(`Template "${templateName}" not found.`);
                return;
            }

            // Check if the template is a DAG or a single pod
            if (template.dag) {
                // Add a DAG node
                graph.nodes.set(parentTaskName, {
                    label: parentTaskName,
                    genre: 'DAG',
                    classNames: 'Pending',
                    progress: 0,
                    icon: 'clock'
                });

                // Process each task in the DAG
                template.dag.tasks.forEach((task: DAGTask) => {
                    const taskName = parentTaskName ? `${parentTaskName}.${task.name}` : task.name;

                    // Add the task node
                    graph.nodes.set(taskName, {
                        label: taskName,
                        genre: templateMap.has(task.template) && templateMap.get(task.template).dag ? 'DAG' : 'Pod',
                        classNames: 'Pending',
                        progress: 0,
                        icon: 'clock'
                    });

                    // Add edges based on "depends" field
                    if (task.depends) {
                        const dependencies = parseDepends(task.depends);
                        dependencies.forEach((dep: string) => {
                            const dependentTaskName = parentTaskName ? `${parentTaskName}.${dep}` : dep;
                            graph.edges.set({v: dependentTaskName, w: taskName}, {});
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
                // If it's not a DAG, treat it as a Pod

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

                // If there's a parent task, add an edge
                if (parentTaskName) {
                    graph.edges.set({v: parentTaskName, w: templateName}, {});
                }
            }
        }
        processTemplate(entrypoint, name);

        return graph;

        function parseDepends(dependsString: string) {
            return dependsString.split(/&&|\|\|/).map(s => s.trim());
        }
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
            nodeClassNames={{'': true, 'Pending': true, 'Ready': true, 'Running': true, 'Failed': true, 'Succeeded': true, 'Error': true}}
            nodeTagsTitle={'Template'}
            nodeTags={{[name]: true}}
            nodeSize={64}
            defaultIconShape='circle'
            hideNodeTypes={true}
            hideOptions={false}
            options={<WorkflowDagRenderOptionsPanel {...state} onChange={workflowDagRenderOptions => this.saveOptions(workflowDagRenderOptions)} />}
        />
    );
}
