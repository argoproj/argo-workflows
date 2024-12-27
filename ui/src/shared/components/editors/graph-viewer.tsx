import * as React from 'react';
import {useEffect, useState} from 'react';

import {genres} from '../../../workflows/components/workflow-dag/genres';
import {WorkflowDagRenderOptions} from '../../../workflows/components/workflow-dag/workflow-dag';
import {WorkflowDagRenderOptionsPanel} from '../../../workflows/components/workflow-dag/workflow-dag-render-options-panel';
import {ClusterWorkflowTemplate, CronWorkflow, DAGTask, Template, Workflow, WorkflowStep, WorkflowTemplate} from '../../models';
import {services} from '../../services';
import {GraphPanel} from '../graph/graph-panel';
import {Graph} from '../graph/types';
import {Icon} from '../icon';


export function GraphViewer({workflowDefinition}: {workflowDefinition: Workflow | WorkflowTemplate | ClusterWorkflowTemplate | CronWorkflow}) {
    const [workflow, setWorkflow] = useState<Workflow | WorkflowTemplate | ClusterWorkflowTemplate>(workflowDefinition);
    const [isLoading, setIsLoading] = useState(true);
    const [state, saveOptions] = useState<WorkflowDagRenderOptions>({
        expandNodes: new Set(),
        showArtifacts: false,
        showInvokingTemplateName: false,
        showTemplateRefsGrouping: false
    });
    const defaultNodeParams: {
        classNames: string;
        progress: number;
        icon: Icon;
    } = {
        classNames: 'Skipped',
        progress: 0,
        icon: 'clock',
    };

    useEffect(() => {
        if ('workflowTemplateRef' in workflowDefinition.spec && isLoading) {
            setWorkflowFromRefrence(workflowDefinition.spec.workflowTemplateRef.name).then(() => {
                setIsLoading(false);
            });
        } else if ('workflowSpec' in workflowDefinition.spec && isLoading) {
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

    const name = workflow.metadata.name ? `${workflow.metadata.name}-${generateNamePostfix(5)}` : `${workflow.metadata.generateName}${generateNamePostfix(5)}`;
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
        return services.workflowTemplate.get(name, workflowDefinition.metadata.namespace).then(workflowTemplate => {
            setWorkflow(workflowTemplate);
        });
    }

    function convertFromCronWorkflow(cronWorkflow: CronWorkflow): Workflow {
        return {
            apiVersion: 'argoproj.io/v1alpha1',
            kind: 'Workflow',
            metadata: cronWorkflow.metadata,
            spec: cronWorkflow.spec.workflowSpec
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

        function processTemplate(templateName: string, parentNodeName: string = null): Graph {
            const template = templateMap.get(templateName);

            if (!template) {
                console.error(`Template "${templateName}" not found.`);
                return;
            }

            if (template.dag) {
                graph.nodes.set(parentNodeName, {
                    label: getStringAfterDelimiter(parentNodeName),
                    genre: 'DAG',
                    ...defaultNodeParams
                });

                template.dag.tasks.forEach((task: DAGTask) => {
                    const nodeName = `${parentNodeName}.${task.name}`;
                    const retryNodeName = `${nodeName}.retry`;
                    const taskGroupName = `${nodeName}.TaskGroup`;
                    let nodeLabel = task.name;
                    const retryStrategy = getRetryStrategy(task);
                    const executionStrategy = getExecutionStrategy(task);
                    if (retryStrategy) {
                        graph.nodes.set(retryNodeName, {
                            label: nodeLabel,
                            genre: 'Retry',
                            ...defaultNodeParams
                        });
                    }
                    if (executionStrategy) {
                        graph.nodes.set(taskGroupName, {
                            label: nodeLabel,
                            genre: 'TaskGroup',
                            ...defaultNodeParams
                        });

                        nodeLabel = `${nodeLabel}${executionStrategy}`;
                    }

                    graph.nodes.set(nodeName, {
                        label: nodeLabel,
                        genre: getTaskGenre(task),
                        ...defaultNodeParams
                    });

                    if (task.depends || task.dependencies) {
                        const dependencies = task.dependencies ? task.dependencies : parseDepends(task.depends);
                        const dependencyLabel = task.depends ? task.depends : task.dependencies.join(' && ');
                        dependencies.forEach((dep: string) => {
                            let dependancyName = `${parentNodeName}.${dep}`;
                            if (graph.nodes.get(dependancyName).genre == 'Pod') {
                                if (retryStrategy) {
                                    graph.edges.set({v: dependancyName, w: retryNodeName}, {});
                                    dependancyName = retryNodeName;
                                }
                                if (executionStrategy) {
                                    graph.edges.set({v: dependancyName, w: taskGroupName}, {});
                                    dependancyName = taskGroupName;
                                }
                                graph.edges.set({v: dependancyName, w: nodeName}, {label: dependencyLabel});
                            } else {
                                // Override edge to be from last element of DAG if dep is a DAG.
                                const depTemplate = getTemplateNameFromTask(template.dag, dep);
                                const templateLeafNodes = templateLeafMap.get(depTemplate);
                                if (templateLeafNodes) {
                                    templateLeafNodes.forEach((leaf: string) => {
                                        let leafNode = `${dependancyName}.${leaf}`;
                                        if (retryStrategy) {
                                            graph.edges.set({v: parentNodeName, w: retryNodeName}, {});
                                            leafNode = retryNodeName;
                                        }
                                        if (executionStrategy) {
                                            graph.edges.set({v: parentNodeName, w: taskGroupName}, {});
                                            leafNode = taskGroupName;
                                        }
                                        graph.edges.set({v: leafNode, w: nodeName}, {label: dependencyLabel});
                                    });
                                }
                            }
                        });
                    } else {
                        let newParentTaskName = parentNodeName;
                        if (retryStrategy) {
                            graph.edges.set({v: parentNodeName, w: retryNodeName}, {});
                            newParentTaskName = retryNodeName;
                        }
                        if (executionStrategy) {
                            graph.edges.set({v: parentNodeName, w: taskGroupName}, {});
                            newParentTaskName = taskGroupName;
                        }
                        graph.edges.set({v: newParentTaskName, w: nodeName}, {});
                    }

                    // Recursively process the task's template if it's a nested DAG
                    if (isTemplateNested(task)) {
                        processTemplate(task.template, nodeName);
                    }
                    previousSteps = [];
                });
            } else if (template.steps) {
                if (!graph.nodes.has(parentNodeName)) {
                    graph.nodes.set(parentNodeName, {
                        label: parentNodeName,
                        genre: 'Steps',
                        ...defaultNodeParams
                    });
                }
                graph.edges.set({v: parentNodeName, w: `${parentNodeName}.0`}, {});

                template.steps.forEach((stepGroup: WorkflowStep[], stepGroupIndex: number) => {
                    let groupName = `${parentNodeName}.${stepGroupIndex}`;
                    graph.nodes.set(groupName, {
                        label: `[${stepGroupIndex}]`,
                        genre: 'StepGroup',
                        ...defaultNodeParams
                    });
                    previousSteps.forEach((prevStep: string) => {
                        graph.edges.set({v: prevStep, w: groupName}, {});
                    });
                    previousSteps = [];

                    stepGroup.forEach((step: WorkflowStep) => {
                        const nodeName = `${groupName}.${step.name}`;
                        const nodeLabel = `${step.name}${getExecutionStrategy(step)}`;
                        const retryStrategy = getRetryStrategy(step);
                        if (retryStrategy) {
                            const retryNodeName = `${nodeName}.retry`;
                            graph.nodes.set(retryNodeName, {
                                label: nodeLabel,
                                genre: 'Retry',
                                ...defaultNodeParams
                            });

                            graph.edges.set({v: groupName, w: retryNodeName}, {});
                            groupName = retryNodeName;
                        }

                        graph.nodes.set(nodeName, {
                            label: nodeLabel,
                            genre: getTaskGenre(step),
                            ...defaultNodeParams
                        });
                        graph.edges.set({v: groupName, w: nodeName}, {});
                        previousSteps.push(nodeName);

                        // Recursively process the step's template if it's a nested step
                        if (isTemplateNested(step)) {
                            processTemplate(step.template, nodeName);
                        }
                    });
                });
            } else {
                // Edge case when workflow consists of only one template
                if (templateMap.size === 1) {
                    templateName = parentNodeName;
                    parentNodeName = '';
                }

                graph.edges.set({v: parentNodeName, w: templateName}, {});

                graph.nodes.set(templateName, {
                    label: templateName,
                    genre: 'Pod',
                    ...defaultNodeParams
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
        let independentTasks: string[] = [];

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
            independentTasks = Array.from(allTaskNames).filter(nodeName => !dependentTaskNames.has(nodeName));
        } else if (template.steps) {
            const lastStepGroupIdx = template.steps.length - 1;
            const lastStepGroup = template.steps[lastStepGroupIdx];

            lastStepGroup.forEach((step: WorkflowStep) => {
                independentTasks.push(`${lastStepGroupIdx}.${step.name}`);
            });
        }

        return independentTasks ? independentTasks : null;
    }

    function getTemplateNameFromTask(dag: {tasks: {name: string; template: string}[]}, nodeName: string): string | null {
        const task = dag.tasks.find(t => t.name === nodeName);
        return task ? task.template : null;
    }

    function getExecutionStrategy(task: DAGTask | WorkflowStep) {
        let executionStrategy = '';

        if (task.withItems) {
            executionStrategy += `\n{withItems: ${String(task.withItems)}}`;
        } else if (task.withParam) {
            executionStrategy += `\n{withParam: ${String(task.withParam)}}`;
        } else if (task.withSequence) {
            executionStrategy += `\n{withSequence: ${String(task.withSequence)}}`;
        }

        return executionStrategy;
    }

    function getStringAfterDelimiter(input: string, delimiter: string = '.'): string {
        const lastIndex = input.lastIndexOf(delimiter);
        if (lastIndex === -1) {
            return input;
        }
        return input.substring(lastIndex + 1);
    }
}
