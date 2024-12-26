import * as React from 'react';
import {useEffect, useState} from 'react';

import {genres} from '../../../workflows/components/workflow-dag/genres';
import {WorkflowDagRenderOptions} from '../../../workflows/components/workflow-dag/workflow-dag';
import {WorkflowDagRenderOptionsPanel} from '../../../workflows/components/workflow-dag/workflow-dag-render-options-panel';
import {ClusterWorkflowTemplate, CronWorkflow, DAGTask, Template, Workflow, WorkflowStep, WorkflowTemplate} from '../../models';
import {services} from '../../services';
import {GraphPanel} from '../graph/graph-panel';
import {Graph} from '../graph/types';
import {services} from '../../services';
import {useEffect, useState} from 'react';

export function GraphViewer({workflowDefinition}: {workflowDefinition: Workflow | WorkflowTemplate | ClusterWorkflowTemplate | CronWorkflow}) {
    const [workflow, setWorkflow] = useState<Workflow | WorkflowTemplate | ClusterWorkflowTemplate>(workflowDefinition);
    const [isLoading, setIsLoading] = useState(true);
    const [state, saveOptions] = useState<WorkflowDagRenderOptions>({
        expandNodes: new Set(),
        showArtifacts: false,
        showInvokingTemplateName: false,
        showTemplateRefsGrouping: false
    });

    useEffect(() => {
        if ('workflowTemplateRef' in workflowDefinition.spec) {
            setWorkflowFromRefrence(workflowDefinition.spec.workflowTemplateRef.name).then(() => {
                setIsLoading(false);
            });
        } else if ('workflowSpec' in workflowDefinition.spec) {
            const convertedCronWorkflow = convertFromCronWorkflow(workflowDefinition as CronWorkflow);
            setWorkflow(convertedCronWorkflow);
            setIsLoading(false);
        } else {
            setIsLoading(false);
        }
    }, [workflow]);

    if (isLoading) {
        return <div>Loading...</div>;
    }

    const name = workflow.metadata.name ?? `${workflow.metadata.generateName}${generateNamePostfix(5)}`;
    const graph = populateGraphFromWorkflow(workflow, name);

    return (
        <GraphPanel
            storageScope='graph-viewer'
            graph={graph}
            nodeGenresTitle={'Node Type'}
            nodeGenres={genres}
            nodeClassNamesTitle={'Node Phase'}
            nodeClassNames={{Skipped: true}}
            nodeTagsTitle={'Template'}
            nodeTags={{[name]: true}}
            nodeSize={64}
            defaultIconShape='circle'
            hideNodeTypes={false}
            hideOptions={false}
            options={<WorkflowDagRenderOptionsPanel {...state} onChange={workflowDagRenderOptions => saveOptions(workflowDagRenderOptions)} />}
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
        let previousSteps: string[] = [];

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
                    classNames: 'Skipped',
                    progress: 0,
                    icon: 'clock'
                });

                template.dag.tasks.forEach((task: DAGTask) => {
                    const taskName = `${parentTaskName}.${task.name}`
                    const taskLabel = generateTaskLabel(task)
                    const retryStrategy = getRetryStrategy(task)
                    if (retryStrategy) {
                        const retryNodeName = `${taskName}.retry`
                        graph.nodes.set(retryNodeName, {
                            label: `${taskName}${retryStrategy}}`,
                            genre: 'Retry',
                            classNames: 'Skipped',
                            progress: 0,
                            icon: 'clock'
                        });

                        graph.edges.set({v: parentTaskName, w: retryNodeName}, {});
                        parentTaskName = retryNodeName
                    }

                    graph.nodes.set(taskName, {
                        label: taskLabel,
                        genre: getTaskGenre(task),
                        classNames: 'Skipped',
                        progress: 0,
                        icon: 'clock'
                    });

                    if (task.depends || task.dependencies) {
                        const dependencies = task.dependencies ? task.dependencies : parseDepends(task.depends);
                        const dependencyLabel = task.depends ? task.depends : task.dependencies.join(' && ');
                        dependencies.forEach((dep: string) => {
                            const dependancyName = `${parentTaskName}.${dep}`;
                            if (graph.nodes.get(dependancyName).genre == 'Pod') {
                                graph.edges.set({v: dependancyName, w: taskName}, {label: dependencyLabel});
                            } else {
                                // Override edge to be from last element of DAG if dep is a DAG.
                                const depTemplate = getTemplateNameFromTask(template.dag, dep);
                                const templateLeafNodes = templateLeafMap.get(depTemplate);
                                if (templateLeafNodes) {
                                    templateLeafNodes.forEach((leaf: string) => {
                                        graph.edges.set({v: `${dependancyName}.${leaf}`, w: taskName}, {label: dependencyLabel});
                                    });
                                }
                            }
                        });
                    } else {
                        graph.edges.set({v: parentTaskName, w: taskName}, {});
                    }

                    // Recursively process the task's template if it's a nested DAG
                    if (isTemplateNested(task)) {
                        processTemplate(task.template, taskName);
                    }
                    previousSteps = [];
                });
            } else if (template.steps) {
                if (!graph.nodes.has(parentTaskName)) {

                    graph.nodes.set(parentTaskName, {
                        label: parentTaskName,
                        genre: 'Steps',
                        classNames: 'Skipped',
                        progress: 0,
                        icon: 'clock'
                    });
                }
                graph.edges.set({v: parentTaskName, w: `${parentTaskName}.0`}, {});

                template.steps.forEach((stepGroup: WorkflowStep[], stepGroupIndex: number) => {
                    let groupName = `${parentTaskName}.${stepGroupIndex}`
                    graph.nodes.set(groupName, {
                        label: `[${stepGroupIndex}]`,
                        genre: 'StepGroup',
                        classNames: 'Skipped',
                        progress: 0,
                        icon: 'clock'
                    });
                    previousSteps.forEach((prevStep: string) => {
                        graph.edges.set({v: prevStep, w: groupName}, {});
                    });
                    previousSteps = [];

                    stepGroup.forEach((step: WorkflowStep) => {
                        const stepName = `${groupName}.${step.name}`
                        const retryStrategy = getRetryStrategy(step)
                        if (retryStrategy) {
                            const retryNodeName = `${stepName}.retry`
                            graph.nodes.set(retryNodeName, {
                                label: `${stepName}${retryStrategy}}}`,
                                genre: 'Retry',
                                classNames: 'Skipped',
                                progress: 0,
                                icon: 'clock'
                            });

                            graph.edges.set({v: groupName, w: retryNodeName}, {});
                            groupName = retryNodeName
                        }


                        graph.nodes.set(stepName, {
                            label: generateTaskLabel(step),
                            genre: getTaskGenre(step),
                            classNames: 'Skipped',
                            progress: 0,
                            icon: 'clock'
                        });
                        graph.edges.set({v: groupName, w: stepName}, {});
                        previousSteps.push(stepName)

                        // Recursively process the step's template if it's a nested step
                        if (isTemplateNested(step)) {
                            processTemplate(step.template, stepName);
                        }
                    });


                });
            }
            else {
                // Edge case when workflow consists of only one template
                if (templateMap.size === 1) {
                    templateName = parentTaskName;
                    parentTaskName = '';
                }

                graph.edges.set({v: parentTaskName, w: templateName}, {});

                graph.nodes.set(templateName, {
                    label: templateName,
                    genre: 'Pod',
                    classNames: 'Skipped',
                    progress: 0,
                    icon: 'clock'
                });

            }
        }
        function getTaskGenre(task: DAGTask | WorkflowStep): 'Steps' | 'DAG' | 'Pod' {
            if (templateMap.has(task.template)) {
                const template = templateMap.get(task.template);
                if ('steps' in template) {
                    return 'Steps';
                } else if ('dag' in template) {
                    return 'DAG';
                }
            }
            return 'Pod';
        }

        function isTemplateNested(node: DAGTask | WorkflowStep): boolean {
            if (templateMap.has(node.template)) {
                const template = templateMap.get(node.template);
                return 'dag' in template || 'steps' in template;
            }
            return false;
        }

        function getRetryStrategy(node: DAGTask | WorkflowStep): string {
            if (templateMap.has(node.template)) {
                const template = templateMap.get(node.template);
                if ('retryStrategy' in template) {
                    return `\n{retryStrategy: ${String(template.retryStrategy)}}`;
                }
            }
            return ''; // Return an empty string if retryStrategy is not available
        }
        processTemplate(entrypoint, name);

        return graph;
    }
    function parseDepends(dependsString: string): string[] {
        const taskNameRegex = /([a-zA-Z0-9-_]+)(?:\.[a-zA-Z]+)?/g;
        const taskNames = new Set<string>();
        let match;
        while ((match = taskNameRegex.exec(dependsString)) !== null) {
            taskNames.add(match[1]);
        }

        return Array.from(taskNames);
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
        let independentTasks: string[] = []

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
            independentTasks = Array.from(allTaskNames).filter(taskName => !dependentTaskNames.has(taskName));
        }
        else if (template.steps) {
            const lastStepGroupIdx = template.steps.length - 1
            const lastStepGroup = template.steps[lastStepGroupIdx]

            lastStepGroup.forEach((step: WorkflowStep) => {
                independentTasks.push(`${lastStepGroupIdx}.${step.name}`)
            });
        }

        return independentTasks ? independentTasks : null;
    }

    function getTemplateNameFromTask(dag: {tasks: {name: string; template: string}[]}, taskName: string): string | null {
        const task = dag.tasks.find(t => t.name === taskName);
        return task ? task.template : null;
    }

    function generateTaskLabel(task: DAGTask | WorkflowStep) {
        let taskLabel = task.name;

        if (task.withItems) {
            taskLabel += `\n{withItems: ${String(task.withItems)}}`;
        } else if (task.withParam) {
            taskLabel += `\n{withParam: ${String(task.withParam)}}`;
        } else if (task.withSequence) {
            taskLabel += `\n{withSequence: ${String(task.withSequence)}}`;
        }

        return taskLabel;
    }


}
