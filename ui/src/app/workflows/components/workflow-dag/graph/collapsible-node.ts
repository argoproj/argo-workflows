interface CollapsedNode {
    kind: string;
    parent: string;
    message: string;
}

export function getCollapsedNodeName(parent: string, message: string): string {
    return JSON.stringify({kind: 'collapsed', parent, message} as CollapsedNode);
}

export function getOutboundNodeName(parent: string, message: string): string {
    return JSON.stringify({kind: 'outbound', parent, message} as CollapsedNode);
}

export function isCollapsedNode(id: string): boolean {
    try {
        return (JSON.parse(id) as CollapsedNode).kind === 'collapsed';
    } catch (e) {
        return false;
    }
}

export function isOutboundNode(id: string): boolean {
    try {
        return (JSON.parse(id) as CollapsedNode).kind === 'outbound';
    } catch (e) {
        return false;
    }
}

export function getNodeParent(id: string): string {
    return (JSON.parse(id) as CollapsedNode).parent;
}

export function getMessage(id: string): string {
    return (JSON.parse(id) as CollapsedNode).message;
}
