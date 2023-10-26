import * as dagre from 'dagre';
import {Graph, Node} from './types';

const minSize = 1;
export function layoutGraphPretty(graph: Graph, nodeSize: number, horizontal: boolean, hidden: (id: Node) => boolean) {
    const gap = nodeSize * 1.25;
    const g = new dagre.graphlib.Graph();
    g.setGraph({rankdir: horizontal ? 'LR' : 'TB', ranksep: gap, nodesep: gap, edgesep: gap});
    g.setDefaultEdgeLabel(() => ({}));
    graph.nodes.forEach((label, id) =>
        g.setNode(id, {
            width: hidden(id) ? minSize : nodeSize,
            height: hidden(id) ? minSize : nodeSize
        })
    );
    graph.edges.forEach((label, e) => {
        if (graph.nodes.has(e.v) && graph.nodes.has(e.w)) {
            g.setEdge(e.v, e.w);
        }
    });

    dagre.layout(g);

    graph.width = 0;
    graph.height = 0;
    graph.nodes.forEach((label, id) => {
        graph.nodes.get(id).x = g.node(id).x;
        graph.nodes.get(id).y = g.node(id).y;
        graph.width = Math.max(graph.width, label.x + nodeSize);
        graph.height = Math.max(graph.height, label.y + nodeSize);
    });
    graph.edges.forEach((label, e) => {
        if (graph.nodes.has(e.v) && graph.nodes.has(e.w)) {
            const points = g.edge(e).points;
            graph.edges.get(e).points = points;
            points.forEach(p => {
                graph.width = Math.max(graph.width, p.x + nodeSize);
                graph.height = Math.max(graph.height, p.y + nodeSize);
            });
        }
    });
}
