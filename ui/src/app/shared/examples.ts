import {CronWorkflow} from '../../models';

export const exampleCronWorkflow = (namespace: string) => {
    return {
        metadata: {
            generateName: 'hello-world-',
            namespace: namespace || 'default'
        },
        spec: {
            entrypoint: 'whalesay',
            schedule: '* * * * *',
            workflowSpec: {
                entrypoint: 'whalesay',
                templates: [
                    {
                        name: 'whalesay',
                        container: {
                            name: 'whalesay',
                            image: 'docker/whalesay:latest',
                            command: ['cowsay'],
                            args: ['hello world']
                        }
                    }
                ]
            }
        }
    } as CronWorkflow;
};
