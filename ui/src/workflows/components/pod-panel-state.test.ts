import {NodePhase, Pod} from '../../shared/models';
import {podPanelState} from './pod-panel-state';

const pod = {metadata: {name: 'my-pod'}} as Pod;

describe('podPanelState', () => {
    test.each([false, true])('shows a received pod with settled=%s', settled => {
        expect(podPanelState(pod, false, 'Running', settled)).toBe('shown');
        expect(podPanelState(pod, false, 'Succeeded', settled)).toBe('shown');
    });

    test.each([false, true])('keeps the last pod once it is deleted with settled=%s', settled => {
        expect(podPanelState(pod, true, 'Succeeded', settled)).toBe('deleted');
        expect(podPanelState(pod, true, 'Running', settled)).toBe('deleted');
    });

    test.each(['Pending', 'Running'] as NodePhase[])('waits for the pod while the node is %s', phase => {
        expect(podPanelState(undefined, false, phase, false)).toBe('waiting');
        expect(podPanelState(undefined, false, phase, true)).toBe('waiting');
    });

    test.each(['Succeeded', 'Failed', 'Error', 'Skipped', 'Omitted'] as NodePhase[])('shows loading while the %s node pod watch is unsettled', phase => {
        expect(podPanelState(undefined, false, phase, false)).toBe('loading');
    });

    test('shows loading while the unknown node pod watch is unsettled', () => {
        expect(podPanelState(undefined, false, undefined, false)).toBe('loading');
    });

    test.each(['Succeeded', 'Failed', 'Error', 'Skipped', 'Omitted'] as NodePhase[])('reports no pod once the %s node pod watch is settled', phase => {
        expect(podPanelState(undefined, false, phase, true)).toBe('not-found');
    });

    test('reports no pod when the unknown node pod watch is settled', () => {
        expect(podPanelState(undefined, false, undefined, true)).toBe('not-found');
    });
});
