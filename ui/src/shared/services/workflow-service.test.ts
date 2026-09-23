/**
 * @jest-environment jsdom
 */
import {firstValueFrom, of} from 'rxjs';

import {Workflow} from '../models';
import {WorkflowsService} from './workflows-service';

jest.mock('./requests');

const workflow = (name: string, namespace: string, uid: string): Workflow => {
    return {
        metadata: {
            name,
            namespace,
            uid
        },
        spec: {}
    };
};

describe('workflow service', () => {
    const service = WorkflowsService;

    afterEach(() => jest.restoreAllMocks());

    test('getArtifactLogsUrl', () => {
        expect(service.getArtifactLogsPath(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-container', false)).toBe(
            'artifact-files/argo/workflows/hello-world/test-node/outputs/test-container-logs'
        );
        expect(service.getArtifactLogsPath(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-container', true)).toBe(
            'artifact-files/argo/archived-workflows/test-uid/test-node/outputs/test-container-logs'
        );
    });

    test('getArtifactDownloadUrl', () => {
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', false, false)).toBe(
            '/artifact-files/argo/workflows/hello-world/test-node/outputs/test-artifact'
        );
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', true, false)).toBe(
            '/artifact-files/argo/archived-workflows/test-uid/test-node/outputs/test-artifact'
        );
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', false, true)).toBe(
            '/artifact-files/argo/workflows/hello-world/test-node/inputs/test-artifact'
        );
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', true, true)).toBe(
            '/artifact-files/argo/archived-workflows/test-uid/test-node/inputs/test-artifact'
        );
    });

    test('getContainerLogs for completed container set', async () => {
        const completedWorkflow = {
            ...workflow('test-workflow-1', 'argo', 'test-uid'),
            spec: {
                templates: [
                    {
                        name: 'container-set',
                        containerSet: {containers: [{name: 'step-1'}]}
                    }
                ]
            },
            status: {
                nodes: {
                    'test-node': {
                        id: 'test-node',
                        phase: 'Succeeded',
                        templateName: 'container-set',
                        outputs: {artifacts: [{name: 'step-1-logs'}]}
                    }
                }
            }
        } as unknown as Workflow;
        const requestedWorkflow = workflow('hello-world', 'argo', 'test-uid');
        jest.spyOn(service, 'get').mockResolvedValue(completedWorkflow);
        const getArtifactLogs = jest.spyOn(service, 'getContainerLogsFromArtifact').mockReturnValue(of({content: 'from artifact', podName: 'test-pod'}));
        const getClusterLogs = jest.spyOn(service, 'getContainerLogsFromCluster').mockReturnValue(of({content: 'from cluster', podName: 'test-pod'}));

        const log = await firstValueFrom(service.getContainerLogs(requestedWorkflow, 'test-pod', 'test-node', 'step-1', '', false));

        expect(log.content).toBe('from artifact');
        expect(service.get).toHaveBeenCalledWith('argo', 'hello-world');
        expect(getArtifactLogs).toHaveBeenCalledWith(completedWorkflow, 'test-node', 'step-1', '', false);
        expect(getClusterLogs).not.toHaveBeenCalled();
    });

    test('getContainerLogs for completed main container', async () => {
        const completedWorkflow = {
            ...workflow('test-workflow-1', 'argo', 'test-uid'),
            spec: {templates: [{name: 'normal-container'}]},
            status: {
                nodes: {
                    'test-node': {
                        id: 'test-node',
                        phase: 'Succeeded',
                        templateName: 'normal-container',
                        outputs: {artifacts: [{name: 'main-logs'}]}
                    }
                }
            }
        } as unknown as Workflow;
        jest.spyOn(service, 'get').mockResolvedValue(completedWorkflow);
        const getArtifactLogs = jest.spyOn(service, 'getContainerLogsFromArtifact').mockReturnValue(of({content: 'from artifact', podName: 'test-pod'}));
        const getClusterLogs = jest.spyOn(service, 'getContainerLogsFromCluster').mockReturnValue(of({content: 'from cluster', podName: 'test-pod'}));

        const log = await firstValueFrom(service.getContainerLogs(workflow('test-workflow-1', 'argo', 'test-uid'), 'test-pod', 'test-node', 'main', '', false));

        expect(log.content).toBe('from artifact');
        expect(getArtifactLogs).toHaveBeenCalledWith(completedWorkflow, 'test-node', 'main', '', false);
        expect(getClusterLogs).not.toHaveBeenCalled();
        jest.spyOn(service, 'get').mockResolvedValue(completedWorkflow);
    });

    test('getContainerLogs for a container not in the node', async () => {
        const completedWorkflow = {
            ...workflow('test-workflow-1', 'argo', 'test-uid'),
            spec: {templates: [{name: 'normal-container'}]},
            status: {
                nodes: {
                    'test-node': {
                        id: 'test-node',
                        phase: 'Succeeded',
                        templateName: 'normal-container',
                        outputs: {artifacts: [{name: 'some-other-step-logs'}]}
                    }
                }
            }
        } as unknown as Workflow;
        jest.spyOn(service, 'get').mockResolvedValue(completedWorkflow);
        const getArtifactLogs = jest.spyOn(service, 'getContainerLogsFromArtifact').mockReturnValue(of({content: 'from artifact', podName: 'test-pod'}));
        const getClusterLogs = jest.spyOn(service, 'getContainerLogsFromCluster').mockReturnValue(of({content: 'from cluster', podName: 'test-pod'}));

        const log = await firstValueFrom(service.getContainerLogs(completedWorkflow, 'test-pod', 'test-node', 'missing-container', '', false));

        expect(log.content).toBe('from cluster');
        expect(getArtifactLogs).not.toHaveBeenCalled();
        expect(getClusterLogs).toHaveBeenCalledWith(completedWorkflow, 'test-pod', 'missing-container', '');
    });

    test('getContainerLogs for a container in inline template', async () => {
        const completedWorkflow = {
            ...workflow('test-workflow-1', 'argo', 'test-uid'),
            spec: {
                templates: [
                    {
                        name: 'inline-template',
                        dag: {
                            tasks: [
                                {
                                    name: 'some-template',
                                    inline: {
                                        containerSet: [
                                            {
                                                container: 'step-1'
                                            },
                                            {
                                                container: 'step-2'
                                            }
                                        ]
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
        jest.spyOn(service, 'get').mockResolvedValue(completedWorkflow);
        const getArtifactLogs = jest.spyOn(service, 'getContainerLogsFromArtifact').mockReturnValue(of({content: 'from artifact', podName: 'test-pod'}));
        const getClusterLogs = jest.spyOn(service, 'getContainerLogsFromCluster').mockReturnValue(of({content: 'from cluster', podName: 'test-pod'}));

        const log = await firstValueFrom(service.getContainerLogs(completedWorkflow, 'test-pod', 'test-node', 'step-1', '', false));

        expect(log.content).toBe('from artifact');
        expect(getArtifactLogs).toHaveBeenCalledWith(completedWorkflow, 'test-node', 'step-1', '', false);
        expect(getClusterLogs).not.toHaveBeenCalled();
    });
});
