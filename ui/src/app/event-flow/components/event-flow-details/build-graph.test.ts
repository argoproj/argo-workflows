import {Node, NodeLabel} from '../../../shared/components/graph/types';
import {buildGraph} from './build-graph';

describe('build graph', () => {
    test('empty', () => {
        const g = buildGraph([], [], [], {}, false);
        expect(g.nodes).toEqual(new Map<Node, NodeLabel>());
    });
    test('event source', () => {
        const g = buildGraph([{metadata: {namespace: 'my-ns', name: 'my-es'}, spec: {calendar: {example: {}}}}], [], [], {}, false);
        expect(g.nodes).toEqual(
            new Map<Node, NodeLabel>(Object.entries({'my-ns/EventSource/my-es/example': {label: 'example', icon: 'clock', genre: 'calendar', classNames: ''}}))
        );
    });
});
