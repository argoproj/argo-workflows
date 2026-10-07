import {NodePhase, Pod} from '../../shared/models';
import {podPanelState} from './pod-panel-state';

const pod = {metadata: {name: 'my-pod'}} as Pod;

describe('podPanelState', () => {
    test('shows a received pod', () => {
        expect(podPanelState(pod, false, 'Running')).toBe('shown');
        expect(podPanelState(pod, false, 'Succeeded')).toBe('shown');
    });

    test('keeps the last pod once it is deleted', () => {
        expect(podPanelState(pod, true, 'Succeeded')).toBe('deleted');
        expect(podPanelState(pod, true, 'Running')).toBe('deleted');
    });

    test.each(['Pending', 'Running'] as NodePhase[])('waits for the pod while the node is %s', phase => {
        expect(podPanelState(undefined, false, phase)).toBe('waiting');
    });

    test.each(['Succeeded', 'Failed', 'Error', 'Skipped', 'Omitted'] as NodePhase[])('reports no pod once the node is %s', phase => {
        expect(podPanelState(undefined, false, phase)).toBe('not-found');
    });

    test('reports no pod when the node is unknown', () => {
        expect(podPanelState(undefined, false, undefined)).toBe('not-found');
    });
});
