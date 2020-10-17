import {ClusterWorkflowTemplate, CronWorkflow, Template, Workflow, WorkflowTemplate} from '../../models';

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

const templates: Template[] = [
    {
        name: entrypoint,
        inputs: {
            parameters: [{name: 'message', value: '{{workflow.parameters.message}}'}]
        },
        container: {
            name: 'main',
            image: 'argoproj/argosay:v2',
            command: ['/argosay'],
            args: ['echo', '{{inputs.parameters.message}}']
        }
    }
];

export const exampleWorkflow = (): Workflow => {
    return {
        metadata: {
            name: randomSillyName(),
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

export const exampleWorkflowTemplate = (): WorkflowTemplate => ({
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

export const exampleCronWorkflow = (): CronWorkflow => ({
    metadata: {
        name: randomSillyName(),
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
