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
    const service = new WorkflowsService();
    test('getArtifactLogsUrl', () => {
        expect(service.getArtifactLogsUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-container', false)).toBe(
            'artifacts/argo/hello-world/test-node/test-container-logs'
        );
        expect(service.getArtifactLogsUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-container', true)).toBe(
            'artifacts-by-uid/test-uid/test-node/test-container-logs'
        );
    });

    test('getArtifactDownloadUrl', () => {
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', false, false)).toBe(
            'artifacts/argo/hello-world/test-node/test-artifact'
        );
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', true, false)).toBe(
            'artifacts-by-uid/test-uid/test-node/test-artifact'
        );
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', false, true)).toBe(
            'input-artifacts/argo/hello-world/test-node/test-artifact'
        );
        expect(service.getArtifactDownloadUrl(workflow('hello-world', 'argo', 'test-uid'), 'test-node', 'test-artifact', true, true)).toBe(
            'input-artifacts-by-uid/test-uid/test-node/test-artifact'
        );
    });
});
