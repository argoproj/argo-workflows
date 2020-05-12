import {ClusterWorkflowTemplate, CronWorkflow, Workflow, WorkflowTemplate} from '../../models';

const randomSillyName = () => {
    const adjectives = ['wonderful', 'fantastic', 'awesome', 'delightful', 'lovely'];
    const nouns = ['rhino', 'python', 'bear', 'dragon', 'octopus', 'tiger'];
    const random = (array: string[]) => array[Math.floor(Math.random() * array.length)];
    return `${random(adjectives)}-${random(nouns)}`;
};

// TODO - remove "name: 'main'" - we should not have it in these examples
const container = {
    name: 'main',
    image: 'argoproj/argosay:v2',
    command: ['/argosay'],
    args: ['echo', 'hello argo!']
};

export const exampleWorkflow = (namespace: string): Workflow => ({
    metadata: {
        name: randomSillyName(),
        namespace: namespace || 'default'
    },
    spec: {
        entrypoint: 'argosay',
        templates: [
            {
                name: 'argosay',
                container
            }
        ]
    }
});
export const exampleClusterWorkflowTemplate = (): ClusterWorkflowTemplate => ({
    metadata: {
        name: randomSillyName()
    },
    spec: {
        templates: [
            {
                name: 'argosay',
                container: {
                    name: 'main',
                    image: 'argoproj/argosay:v2',
                    command: ['argosay'],
                    args: ['echo', 'hello world']
                }
            }
        ]
    }
});

export const exampleWorkflowTemplate = (namespace: string): WorkflowTemplate => ({
    metadata: {
        name: randomSillyName(),
        namespace
    },
    spec: {
        templates: [
            {
                name: 'argosay',
                container
            }
        ]
    }
});

export const exampleCronWorkflow = (namespace: string): CronWorkflow => ({
    metadata: {
        name: randomSillyName(),
        namespace: namespace || 'default'
    },
    spec: {
        schedule: '* * * * *',
        workflowSpec: {
            entrypoint: 'argosay',
            templates: [
                {
                    name: 'argosay',
                    container
                }
            ]
        }
    }
});
