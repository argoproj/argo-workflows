import {NodeStatus, Template, Workflow} from './models';
import {getMainContainerNames} from './template-resolution';

const workflow = (templates: Template[] = []): Workflow =>
    ({
        metadata: {name: 'test-workflow', namespace: 'argo'},
        spec: {templates},
        status: {nodes: {}}
    }) as unknown as Workflow;

const node = (templateName: string): NodeStatus =>
    ({
        templateName
    }) as NodeStatus;

describe('getMainContainerNames', () => {
    test('returns main when no node is provided', () => {
        expect(getMainContainerNames(workflow())).toEqual(['main']);
    });

    test('returns main when the node template cannot be resolved', () => {
        const completedWorkflow = {
            ...workflow(),
            spec: {
                templates: [
                    {
                        name: 'inline-template',
                        dag: {
                            tasks: [
                                {
                                    name: 'some-template',
                                    inline: {
                                        containerSet: {
                                            containers: [{name: 'step-1'}, {name: 'step-2'}]
                                        }
                                    }
                                }
                            ]
                        }
                    }
                ]
            },
            status: {
                nodes: {
                    'test-node': {
                        id: 'test-node',
                        phase: 'Succeeded',
                        type: 'Pod',
                        templateName: '',
                        outputs: {artifacts: [{name: 'step-1-logs'}, {name: 'step-2-logs'}]}
                    }
                }
            }
        } as unknown as Workflow;
        jest.spyOn(console, 'error').mockImplementation();

        expect(getMainContainerNames(completedWorkflow, completedWorkflow.status!.nodes['test-node'])).toEqual(['main']);
    });

    test('returns container set names in declaration order', () => {
        const containerSetWorkflow = workflow([
            {
                name: 'container-set',
                containerSet: {containers: [{name: 'step-1'}, {name: 'step-2'}, {name: 'step-3'}, {name: 'step-4'}]}
            }
        ]);

        expect(getMainContainerNames(containerSetWorkflow, node('container-set'))).toEqual(['step-1', 'step-2', 'step-3', 'step-4']);
    });
});
