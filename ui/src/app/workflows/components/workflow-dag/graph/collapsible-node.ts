interface CollapsedNode {
    is: boolean;
    parent: string;
    numHidden: number;
}

export function getCollapsedNodeName(parent: string, numHidden: number): string {
    return JSON.stringify({is: true, parent, numHidden} as CollapsedNode);
}

export function isCollapsedNode(id: string): boolean {
    try {
        return (JSON.parse(id) as CollapsedNode).is;
    } catch (e) {
        return false;
    }
}

export function getCollapsedNodeParent(id: string): string {
    return (JSON.parse(id) as CollapsedNode).parent;
}

export function getCollapsedNumHidden(id: string): number {
    return (JSON.parse(id) as CollapsedNode).numHidden;
}
