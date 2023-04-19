/**
 * @jest-environment jsdom
 */
import {Workflow} from '../../../models';
import {WorkflowsService} from './workflows-service';

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
            '/input-artifacts/argo/hello-world/test-node/test-artifact'
        );
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', true, true)).toBe(
            '/input-artifacts-by-uid/test-uid/test-node/test-artifact'
        );
    });
});
