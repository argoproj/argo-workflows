import {CoffmanGrahamSorter} from './coffman-graham-sorter';
import {Edge, Graph, Node} from './types';

const minSize = 1;
export const layoutGraphFast = (graph: Graph, nodeSize: number, horizontal: boolean, hidden: (id: Node) => boolean) => {
    const gap = nodeSize * 1.25;
    const layers = new CoffmanGrahamSorter(graph).sort();

    // we have a lot of logic here about laying it out with suitable gaps - but what if we
    // would just translate it somehow?
    if (horizontal) {
        graph.width = layers.length * gap * 2;
        graph.height = 0;
    } else {
        graph.width = 0;
        graph.height = layers.length * gap * 2;
    }
    layers.forEach(level => {
        if (horizontal) {
            graph.height = Math.max(graph.height, level.length * gap * 2);
        } else {
            graph.width = Math.max(graph.width, level.length * gap * 2);
        }
    });
    layers.forEach((level, i) => {
        level.forEach((node, j) => {
            const l = horizontal ? minSize : graph.width / 2 - level.length * gap;
            const t = !horizontal ? minSize : graph.height / 2 - level.length * gap;
            const label = graph.nodes.get(node);
            label.x = (horizontal ? i : j) * gap * 2 + l;
            label.y = (horizontal ? j : i) * gap * 2 + t;
        });
    });
    graph.edges.forEach((label, e) => {
        if (graph.nodes.has(e.v) && graph.nodes.has(e.w)) {
            label.points = generateEdge(graph, e, nodeSize, horizontal, hidden);
        }
    });
};

const generateEdge = (graph: Graph, edge: Edge, nodeSize: number, horizontal: boolean, hiddenNode: (id: Node) => boolean) => {
    // `h` and `v` move the arrow heads to next to the node, otherwise they would be behind it
    const h = horizontal ? nodeSize / 2 : 0;
    const v = !horizontal ? nodeSize / 2 : 0;
    return [
        {
            // for hidden nodes, we want to size them zero
            x: graph.nodes.get(edge.v).x + (hiddenNode(edge.v) ? minSize : h),
            y: graph.nodes.get(edge.v).y + (hiddenNode(edge.v) ? minSize : v)
        },
        {
            x: graph.nodes.get(edge.w).x - (hiddenNode(edge.w) ? minSize : h),
            y: graph.nodes.get(edge.w).y - (hiddenNode(edge.w) ? minSize : v)
        }
    ];
};
