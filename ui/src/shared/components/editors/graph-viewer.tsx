import * as React from 'react';

import {genres} from '../../../workflows/components/workflow-dag/genres';
import {WorkflowDagRenderOptionsPanel} from '../../../workflows/components/workflow-dag/workflow-dag-render-options-panel';
import {DAGTask, Template, Workflow, WorkflowTemplate, CronWorkflow, ClusterWorkflowTemplate, WorkflowSpec} from '../../models';
import {GraphPanel} from '../graph/graph-panel';
import {Graph} from '../graph/types';
import {services} from '../../services';
import {useEffect, useState} from 'react';

export function GraphViewer({workflowDefinition}: {workflowDefinition: Workflow | WorkflowTemplate | ClusterWorkflowTemplate | CronWorkflow}) {
    const [workflow, setWorkflow] = useState<Workflow | WorkflowTemplate | ClusterWorkflowTemplate>(workflowDefinition);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {

        if ("workflowTemplateRef" in workflowDefinition.spec) {
            setWorkflowFromRefrence(workflowDefinition.spec.workflowTemplateRef.name).then(() => {
                setIsLoading(false);
            });
        } else if ("workflowSpec" in workflowDefinition.spec) {
            const convertedCronWorkflow = convertFromCronWorkflow(workflowDefinition as CronWorkflow);
            setWorkflow(convertedCronWorkflow);
            setIsLoading(false);
        }
        else {
            setIsLoading(false);
        }
    }, [workflow]);

    if (isLoading) {
        return <div>Loading...</div>;
    }

    const state = {
        expandNodes: new Set(''),
        showArtifacts: false,
        showInvokingTemplateName: false,
        showTemplateRefsGrouping: false,
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
            hideNodeTypes={false}
            hideOptions={true}
            options={<WorkflowDagRenderOptionsPanel {...state} onChange={workflowDagRenderOptions => this.saveOptions(workflowDagRenderOptions)} />}
        />
    );

    function setWorkflowFromRefrence(name: string): Promise<void> {
        return services.workflowTemplate.get(name, workflowDefinition.metadata.namespace).then((workflowTemplate) => {
            setWorkflow(workflowTemplate);
        });
    }

    function convertFromCronWorkflow(cronWorkflow: CronWorkflow): Workflow {
        return {
            apiVersion: "argoproj.io/v1alpha1",
            kind: "Workflow",
            metadata: cronWorkflow.metadata,
            spec: cronWorkflow.spec.workflowSpec,
        };
    }

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
                    const taskName = `${parentTaskName}.${task.name}`

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
}
