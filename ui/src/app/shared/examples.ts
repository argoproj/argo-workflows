import {CronWorkflow, Workflow, WorkflowTemplate} from '../../models';

const randomSillyName = () => {
    const adjectives = ['wonderful', 'fantastic', 'awesome', 'delightful', 'lovely'];
    const nouns = ['rhino', 'python', 'bear', 'dragon', 'octopus', 'tiger'];
    const random = (array: string[]) => array[Math.floor(Math.random() * array.length)];
    return `${random(adjectives)}-${random(nouns)}`;
};

// TODO - remove "name: 'main'" - we should not have it in these examples

export const exampleWorkflow = (namespace: string): Workflow => ({
    metadata: {
        name: randomSillyName(),
        namespace: namespace || 'default'
    },
    spec: {
        entrypoint: 'whalesay',
        templates: [
            {
                name: 'whalesay',
                container: {
                    name: 'main',
                    image: 'docker/whalesay:latest',
                    command: ['cowsay'],
                    args: ['hello world']
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
                name: 'whalesay',
                container: {
                    name: 'main',
                    image: 'docker/whalesay:latest',
                    command: ['cowsay'],
                    args: ['hello world']
                }
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
            entrypoint: 'whalesay',
            templates: [
                {
                    name: 'whalesay',
                    container: {
                        name: 'main',
                        image: 'docker/whalesay:latest',
                        command: ['cowsay'],
                        args: ['hello world']
                    }
                }
            ]
        }
    }
});
