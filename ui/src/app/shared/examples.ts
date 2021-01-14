import {ClusterWorkflowTemplate, CronWorkflow, EventSource, Sensor, Template, Workflow, WorkflowTemplate} from '../../models';

const randomSillyName = () => {
    const adjectives = ['wonderful', 'fantastic', 'awesome', 'delightful', 'lovely', 'sparkly', 'omniscient'];
    const nouns = ['rhino', 'python', 'bear', 'dragon', 'octopus', 'tiger', 'whale', 'poochenheimer'];
    const random = (array: string[]) => array[Math.floor(Math.random() * array.length)];
    return `${random(adjectives)}-${random(nouns)}`;
};

// cannot be called `arguments` due to typescript
const argumentz = {parameters: [{name: 'message', value: 'hello argo'}]};
const entrypoint = 'argosay';
const labels = {example: 'true'};
const ttlStrategy = {secondsAfterCompletion: 5 * 60};
const podGC = {strategy: 'OnPodCompletion'};

const exampleTemplate = (name: string): Template => ({
    name,
    inputs: {
        parameters: [{name: 'message', value: '{{workflow.parameters.message}}'}]
    },
    container: {
        name: 'main',
        image: 'argoproj/argosay:v2',
        command: ['/argosay'],
        args: ['echo', '{{inputs.parameters.message}}']
    }
});

const templates: Template[] = [exampleTemplate(entrypoint)];

export const exampleWorkflow = (namespace: string): Workflow => {
    return {
        metadata: {
            name: randomSillyName(),
            namespace,
            labels
        },
        spec: {
            arguments: argumentz,
            entrypoint,
            templates,
            ttlStrategy,
            podGC
        }
    };
};
export const exampleClusterWorkflowTemplate = (): ClusterWorkflowTemplate => ({
    metadata: {
        name: randomSillyName(),
        labels
    },
    spec: {
        workflowMetadata: {labels},
        entrypoint,
        arguments: argumentz,
        templates,
        ttlStrategy,
        podGC
    }
});

export const exampleWorkflowTemplate = (namespace: string): WorkflowTemplate => ({
    metadata: {
        name: randomSillyName(),
        namespace,
        labels
    },
    spec: {
        workflowMetadata: {labels},
        entrypoint,
        arguments: argumentz,
        templates,
        ttlStrategy,
        podGC
    }
});

export const exampleCronWorkflow = (namespace: string): CronWorkflow => ({
    metadata: {
        name: randomSillyName(),
        namespace,
        labels
    },
    spec: {
        workflowMetadata: {labels},
        schedule: '* * * * *',
        workflowSpec: {
            entrypoint,
            arguments: argumentz,
            templates,
            ttlStrategy,
            podGC
        }
    }
});

const calender = {'example-with-interval': {interval: '10s'}};

export const exampleEventSource = (namespace: string): EventSource => ({
    metadata: {
        name: 'calendar',
        namespace,
        labels
    },
    spec: {
        calendar: calender
    }
});

export const exampleSensor = (namespace: string): Sensor => ({
    metadata: {
        name: 'workflow',
        namespace,
        labels
    },
    spec: {
        dependencies: [
            {
                name: 'dependency-1',
                eventSourceName: 'calendar',
                eventName: 'example-with-interval'
            }
        ],
        triggers: [
            {
                template: {
                    name: 'workflow-trigger-1',
                    k8s: {
                        group: 'argoproj.io',
                        version: 'v1alpha1',
                        resource: 'workflows',
                        operation: 'create',
                        source: {
                            resource: {
                                apiVersion: 'argoproj.io/v1alpha1',
                                kind: 'Workflow',
                                metadata: {
                                    generateName: 'workflow-from-sensor-'
                                },
                                spec: {
                                    entrypoint: 'main',
                                    templates: [
                                        {
                                            name: 'main',
                                            container: {
                                                image: 'argoproj/argosay:v2'
                                            }
                                        }
                                    ]
                                }
                            }
                        }
                    }
                }
            },
            {
                template: {
                    name: 'log-1',
                    log: {}
                }
            }
        ]
    }
});
